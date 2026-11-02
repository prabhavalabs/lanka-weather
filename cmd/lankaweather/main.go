// Command lankaweather runs the lanka-weather service.
//
// Usage:
//
//	lankaweather migrate   apply database migrations
//	lankaweather serve     run the HTTP API
//	lankaweather worker    run the scheduler that collects data sources
//	lankaweather collect   run every collector once and exit
//	lankaweather collect NAME...   run the named collectors once and exit
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
	"slices"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/canals"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/adminregions"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/dmcbulletins"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/gdacs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/gfs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/hmisrain"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/hydrobasins"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/hydrorivers"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/imerg"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigationgis"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/landslides"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/metstations"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/osmplaces"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/osmreservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/usgs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
	"github.com/prabhavalabs/lanka-weather/internal/environment"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/httpapi"
	"github.com/prabhavalabs/lanka-weather/internal/ingest"
	"github.com/prabhavalabs/lanka-weather/internal/layers"
	"github.com/prabhavalabs/lanka-weather/internal/overview"
	"github.com/prabhavalabs/lanka-weather/internal/places"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/rainfall"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivernet"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
	"github.com/prabhavalabs/lanka-weather/internal/summary"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

const usage = "usage: lankaweather migrate|serve|worker|collect [NAME...]"

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
	if len(args) == 0 || (len(args) > 1 && args[0] != "collect") {
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
			chosen, err := chooseCollectors(scheduler.Collectors, args[1:])
			if err != nil {
				return err
			}
			for _, collector := range chosen {
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

// chooseCollectors picks collectors by name, or all of them with no names. Naming
// them spends one source's requests rather than every source's: a full run costs
// about 470 of Open-Meteo's weighted calls (ADR 0007).
func chooseCollectors(all []source.Collector, names []string) ([]source.Collector, error) {
	if len(names) == 0 {
		return all, nil
	}
	chosen := make([]source.Collector, 0, len(names))
	for _, name := range names {
		index := slices.IndexFunc(all, func(collector source.Collector) bool { return collector.Name() == name })
		if index < 0 {
			return nil, fmt.Errorf("no collector named %q", name)
		}
		chosen = append(chosen, all[index])
	}
	return chosen, nil
}

// collectors lists every data source the worker collects.
func collectors(pool *pgxpool.Pool, logger *slog.Logger) []source.Collector {
	client := source.NewClient(os.Getenv("LANKAWEATHER_CONTACT"))
	weatherGrid := grid.SriLanka()
	// Regions load before places, so places can be placed in their regions.
	list := []source.Collector{
		&irrigation.Collector{Pool: pool, Fetcher: client},
		&adminregions.Collector{Pool: pool, Fetcher: client},
		&osmplaces.Collector{Pool: pool, Fetcher: client},
		&waterlevels.GaugeCollector{Pool: pool, Fetcher: client, Logger: logger},
		&waterlevels.ReservoirCollector{Pool: pool, Fetcher: client},
		&waterlevels.CanalCollector{Pool: pool, Fetcher: client},
		&hmisrain.Collector{Pool: pool, Client: client, Logger: logger},
		&osmreservoirs.Collector{Pool: pool, Fetcher: client, Logger: logger},
		&openmeteo.Collector{Pool: pool, Client: client, Grid: weatherGrid, Kind: openmeteo.KindCurrent, Logger: logger},
		&openmeteo.Collector{Pool: pool, Client: client, Grid: weatherGrid, Kind: openmeteo.KindForecast, Logger: logger},
		&gfs.Collector{Pool: pool, Client: client, Logger: logger},
		&hydrorivers.Collector{Pool: pool, Fetcher: client},
		&irrigationgis.BasinsCollector{Pool: pool, Fetcher: client, Logger: logger},
		&irrigationgis.PastFloodsCollector{Pool: pool, Fetcher: client, Logger: logger},
		&hydrobasins.Collector{Pool: pool, Client: client, Logger: logger},
		&dmcbulletins.Collector{Pool: pool, Fetcher: client},
		&landslides.Collector{Pool: pool, Fetcher: client},
		&usgs.Collector{Pool: pool, Fetcher: client},
		&gdacs.Collector{Pool: pool, Fetcher: client},
		&imerg.Collector{Pool: pool, Client: client, Logger: logger},
		&metstations.ThreeHourlyCollector{Pool: pool, Client: client, Logger: logger},
		&metstations.DailyCollector{Pool: pool, Client: client, Logger: logger},
		&openmeteo.EnvironmentCollector{Pool: pool, Client: client, Grid: grid.AirQuality(), Kind: openmeteo.KindAirQuality, Logger: logger},
		&openmeteo.EnvironmentCollector{Pool: pool, Client: client, Grid: grid.Sea(), Kind: openmeteo.KindMarine, Logger: logger},
	}
	if key := os.Getenv("LANKAWEATHER_GOOGLE_API_KEY"); key != "" {
		list = append(list, &googleflood.Collector{Pool: pool, Client: client, APIKey: key, Logger: logger,
			IncludeUnverified: googleflood.IncludeUnverified(os.Getenv("LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED"))})
	} else {
		logger.Info("LANKAWEATHER_GOOGLE_API_KEY is not set; Google flood forecasts are not collected")
	}
	// Derived from the collectors above, so it runs last.
	return append(list, &rivernet.Collector{Pool: pool, Logger: logger})
}

// regionSource is the source of administrative regions for
// /v1/weather/regions: the admin_regions view over the regions table
// (migration 00004), or the relation named by
// LANKAWEATHER_WEATHER_REGIONS_RELATION. Until the regions collector has
// loaded a level, requests for it answer 503 not_available.
// LANKAWEATHER_WEATHER_REGIONS=none switches region weather off.
func regionSource(pool *pgxpool.Pool, logger *slog.Logger) weather.RegionSource {
	if os.Getenv("LANKAWEATHER_WEATHER_REGIONS") == "none" {
		logger.Info("weather regions are switched off")
		return weather.UnavailableRegions{}
	}
	relation := os.Getenv("LANKAWEATHER_WEATHER_REGIONS_RELATION")
	if relation == "" {
		relation = weather.DefaultRegionsRelation
	}
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
	weatherRegions := regionSource(pool, logger)
	weatherService := &weather.Service{Pool: pool, Regions: weatherRegions,
		CurrentStaleAfter: staleAfter("temperature"), ForecastStaleAfter: 12 * time.Hour}
	if warmer, ok := weatherRegions.(interface{ Warm(context.Context) error }); ok {
		// Join regions to grid cells now rather than in the first requests.
		go func() {
			started := time.Now()
			if err := warmer.Warm(ctx); err != nil {
				logger.Warn("weather regions warm-up failed", "error", err)
				return
			}
			logger.Info("weather regions ready", "duration_ms", time.Since(started).Milliseconds())
		}()
	}
	floodService := &floods.Service{Pool: pool, Configured: os.Getenv("LANKAWEATHER_GOOGLE_API_KEY") != "",
		StaleAfter: staleAfter("flood-forecast")}
	regionService := &regions.Service{Pool: pool}
	reservoirService := &reservoirs.Service{Pool: pool, StaleAfter: staleAfter("reservoirs")}
	riverRiskService := &riverrisk.Service{Pool: pool, Rivers: riverService, Floods: floodService, Weather: weatherService, Reservoirs: reservoirService, Logger: logger}
	warningService := &warnings.Service{Pool: pool, LandslideStaleAfter: staleAfter("landslide-warnings"),
		BulletinStaleAfter: staleAfter("official-warnings")}
	stationService := &stations.Service{Pool: pool, StaleAfter: staleAfter("weather-stations")}
	environmentService := &environment.Service{Pool: pool, AirStaleAfter: staleAfter("air-quality"), SeaStaleAfter: staleAfter("sea-conditions")}
	rainGaugeService := &raingauges.Service{Pool: pool, StaleAfter: staleAfter("rain-gauges")}
	canalService := &canals.Service{Pool: pool}
	hazardService := &hazards.Service{Pool: pool, EarthquakesStaleAfter: staleAfter("earthquakes"), CyclonesStaleAfter: staleAfter("cyclones")}
	server := &http.Server{
		Addr: addr,
		Handler: httpapi.NewRouter(httpapi.Dependencies{
			Logger:  logger,
			Rivers:  riverService,
			Regions: regionService,
			Places:  &places.Service{Pool: pool},
			Weather: weatherService,
			Floods:  floodService,
			Summary: &summary.Service{Weather: weatherService, Rivers: riverService, Floods: floodService,
				Regions: regionService, RiverRisk: riverRiskService, Official: warningService,
				Stations: stationService, Environment: environmentService, Reservoirs: reservoirService,
				RainGauges: rainGaugeService, Canals: canalService, Logger: logger},
			RiverRisk: riverRiskService,
			Warnings:  warningService,
			Overview: &overview.Service{Warnings: warningService, Risk: riverRiskService, Gauges: riverService,
				Reservoirs: reservoirService, Hazards: hazardService, Weather: weatherService, Rain: stationService, RainGauges: rainGaugeService,
				Environment: environmentService, Regions: regionService, WeatherStaleAfter: staleAfter("temperature"), Logger: logger},
			Hazards:     hazardService,
			Stations:    stationService,
			Rain:        &rainfall.Service{Pool: pool, StaleAfter: staleAfter("rainfall")},
			Environment: environmentService,
			Reservoirs:  reservoirService,
			RainGauges:  rainGaugeService,
			Canals:      canalService,
			Ready:       pool.Ping,
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
