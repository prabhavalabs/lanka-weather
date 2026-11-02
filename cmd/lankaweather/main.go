// Command lankaweather runs the lanka-weather service.
//
// Usage:
//
//	lankaweather migrate   apply database migrations
//	lankaweather serve     run the HTTP API
//	lankaweather worker    run the scheduler that collects data sources
//	lankaweather collect   run every collector once and exit
//
// Configuration comes from the environment; see .env.example.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/gfs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/httpapi"
	"github.com/prabhavalabs/lanka-weather/internal/ingest"
	"github.com/prabhavalabs/lanka-weather/internal/layers"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/summary"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

const usage = "usage: lankaweather migrate|serve|worker|collect"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], logger); err != nil {
		logger.Error("exiting", "error", err)
		stop()
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, logger *slog.Logger) error {
	if len(args) != 1 {
		return errors.New(usage)
	}
	databaseURL := os.Getenv("LANKAWEATHER_DATABASE_URL")
	switch args[0] {
	case "migrate":
		if databaseURL == "" {
			return errors.New("LANKAWEATHER_DATABASE_URL is not set")
		}
		if err := database.Migrate(ctx, databaseURL); err != nil {
			return err
		}
		logger.Info("migrations applied")
		return nil
	case "serve":
		pool, err := database.Open(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		return serve(ctx, pool, logger)
	case "worker", "collect":
		pool, err := database.Open(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		scheduler := &ingest.Scheduler{Pool: pool, Collectors: collectors(pool, logger), Logger: logger}
		if args[0] == "collect" {
			for _, collector := range scheduler.Collectors {
				scheduler.RunOne(ctx, collector)
			}
			scheduler.Prune(ctx)
			return nil
		}
		logger.Info("worker started")
		scheduler.Run(ctx)
		return nil
	default:
		return fmt.Errorf("unknown command %q; %s", args[0], usage)
	}
}

// collectors lists every data source the worker collects.
func collectors(pool *pgxpool.Pool, logger *slog.Logger) []source.Collector {
	client := source.NewClient(os.Getenv("LANKAWEATHER_CONTACT"))
	weatherGrid := grid.SriLanka()
	list := []source.Collector{
		&irrigation.Collector{Pool: pool, Fetcher: client},
		&openmeteo.Collector{Pool: pool, Client: client, Grid: weatherGrid, Kind: openmeteo.KindCurrent, Logger: logger},
		&openmeteo.Collector{Pool: pool, Client: client, Grid: weatherGrid, Kind: openmeteo.KindForecast, Logger: logger},
		&gfs.Collector{Pool: pool, Client: client, Logger: logger},
	}
	if key := os.Getenv("LANKAWEATHER_GOOGLE_API_KEY"); key != "" {
		list = append(list, &googleflood.Collector{Pool: pool, Client: client, APIKey: key, Logger: logger,
			IncludeUnverified: os.Getenv("LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED") == "true"})
	} else {
		logger.Info("LANKAWEATHER_GOOGLE_API_KEY is not set; Google flood forecasts are not collected")
	}
	return list
}

// regionSource is the source of administrative regions for
// /v1/weather/regions. Until the regions table is merged it is a stub that
// answers 503 not_available; LANKAWEATHER_WEATHER_REGIONS=postgis reads the
// relation named by LANKAWEATHER_WEATHER_REGIONS_RELATION (default
// admin_regions). See docs/api/weather-and-floods.md.
func regionSource(pool *pgxpool.Pool, logger *slog.Logger) weather.RegionSource {
	if os.Getenv("LANKAWEATHER_WEATHER_REGIONS") != "postgis" {
		return weather.UnavailableRegions{}
	}
	relation := os.Getenv("LANKAWEATHER_WEATHER_REGIONS_RELATION")
	logger.Info("weather regions read from PostGIS", "relation", relation)
	return &weather.PostGISRegions{Pool: pool, Relation: relation, Grid: grid.SriLanka()}
}

func serve(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	addr := os.Getenv("LANKAWEATHER_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	staleAfter := func(id string) time.Duration {
		layer, _ := layers.ByID(id)
		return time.Duration(layer.StaleAfterSeconds) * time.Second
	}
	riverService := &rivers.Service{Pool: pool, StaleAfter: staleAfter("river-levels")}
	weatherService := &weather.Service{Pool: pool, Regions: regionSource(pool, logger),
		CurrentStaleAfter: staleAfter("temperature"), ForecastStaleAfter: 12 * time.Hour}
	floodService := &floods.Service{Pool: pool, Configured: os.Getenv("LANKAWEATHER_GOOGLE_API_KEY") != "",
		StaleAfter: staleAfter("flood-forecast")}
	server := &http.Server{
		Addr: addr,
		Handler: httpapi.NewRouter(httpapi.Dependencies{
			Logger:  logger,
			Rivers:  riverService,
			Weather: weatherService,
			Floods:  floodService,
			Summary: &summary.Service{Weather: weatherService, Rivers: riverService, Floods: floodService, Logger: logger},
			Ready:   pool.Ping,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", addr)
		errs <- server.ListenAndServe()
	}()

	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Info("shutting down")
		return server.Shutdown(shutdown)
	}
}
