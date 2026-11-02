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

	"github.com/prabhavalabs/lanka-weather/internal/connectors/adminregions"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/osmplaces"
	"github.com/prabhavalabs/lanka-weather/internal/httpapi"
	"github.com/prabhavalabs/lanka-weather/internal/ingest"
	"github.com/prabhavalabs/lanka-weather/internal/layers"
	"github.com/prabhavalabs/lanka-weather/internal/places"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/source"
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
		scheduler := &ingest.Scheduler{Pool: pool, Collectors: collectors(pool), Logger: logger}
		if args[0] == "collect" {
			for _, collector := range scheduler.Collectors {
				scheduler.RunOne(ctx, collector)
			}
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
func collectors(pool *pgxpool.Pool) []source.Collector {
	client := source.NewClient(os.Getenv("LANKAWEATHER_CONTACT"))
	// Regions load before places, so places can be placed in their regions.
	return []source.Collector{
		&irrigation.Collector{Pool: pool, Fetcher: client},
		&adminregions.Collector{Pool: pool, Fetcher: client},
		&osmplaces.Collector{Pool: pool, Fetcher: client},
	}
}

func serve(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	addr := os.Getenv("LANKAWEATHER_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	riverLayer, _ := layers.ByID("river-levels")
	server := &http.Server{
		Addr: addr,
		Handler: httpapi.NewRouter(httpapi.Dependencies{
			Logger:  logger,
			Rivers:  &rivers.Service{Pool: pool, StaleAfter: time.Duration(riverLayer.StaleAfterSeconds) * time.Second},
			Regions: &regions.Service{Pool: pool},
			Places:  &places.Service{Pool: pool},
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
