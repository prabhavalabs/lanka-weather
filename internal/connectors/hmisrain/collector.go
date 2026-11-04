package hmisrain

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// Interval matches how often the copy is refreshed. An unchanged file costs
	// one conditional request.
	Interval = 30 * time.Minute
	// retention is how long readings are kept for a gauge's chart; a gauge the
	// feed has dropped goes after gaugeRetention.
	retention      = 7 * 24 * time.Hour
	gaugeRetention = 30 * 24 * time.Hour
	// minimumGauges is the fewest a complete file has; it had 275 when written.
	minimumGauges = 100
)

// Doer sends a request to a source.
type Doer interface {
	Do(ctx context.Context, request source.Request) (source.Response, error)
}

// Collector stores the live rain gauges.
type Collector struct {
	Pool   *pgxpool.Pool
	Client Doer
	// Endpoint and Minimum override the defaults, for tests.
	Endpoint string
	Minimum  int
	Logger   *slog.Logger
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration { return Interval }

// Collect fetches the file unless it is unchanged, and returns the readings stored.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	queries := store.New(collector.Pool)
	endpoint := collector.Endpoint
	if endpoint == "" {
		endpoint = Endpoint
	}
	request := source.Request{URL: endpoint, Accept: "application/json", AcceptStatus: []int{http.StatusOK, http.StatusNotModified}}
	version, err := queries.GetSourceVersion(ctx, SourceName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("read source version: %w", err)
	}
	if err == nil && version.Version != "" {
		request.Header = http.Header{"If-None-Match": {version.Version}}
	}
	response, err := collector.Client.Do(ctx, request)
	if err != nil {
		return 0, fmt.Errorf("fetch rain gauges: %w", err)
	}
	if response.Status == http.StatusNotModified {
		return 0, nil
	}
	feed, err := Decode(response.Body)
	if err != nil {
		return 0, err
	}
	minimum := minimumGauges
	if collector.Minimum > 0 {
		minimum = collector.Minimum
	}
	if len(feed.Gauges) < minimum {
		return 0, fmt.Errorf("source returned %d rain gauges, want at least %d", len(feed.Gauges), minimum)
	}
	if feed.Skipped > 0 && collector.Logger != nil {
		collector.Logger.Warn("rain gauges left out for having no usable name or position", "skipped", feed.Skipped)
	}
	stored := 0
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		txQueries := store.New(tx)
		var readings []store.InsertRainGaugeReadingParams
		for _, gauge := range feed.Gauges {
			params := store.UpsertRainGaugeParams{ID: gauge.ID, Device: gauge.Device, Name: gauge.Name, Institution: gauge.Institution,
				Lon: gauge.Lon, Lat: gauge.Lat, Source: SourceName, SourceTime: feed.UpdatedAt, FetchedAt: now}
			if newest, ok := gauge.Newest(); ok {
				at, rain := newest.At, newest.RainMm
				params.LatestObservedAt, params.LatestRainMm = &at, &rain
			}
			if err := txQueries.UpsertRainGauge(ctx, params); err != nil {
				return fmt.Errorf("store gauge %s: %w", gauge.ID, err)
			}
			for _, point := range gauge.Points {
				readings = append(readings, store.InsertRainGaugeReadingParams{GaugeID: gauge.ID, ObservedAt: point.At, RainMm: point.RainMm,
					Source: SourceName, SourceTime: feed.UpdatedAt, FetchedAt: now})
			}
		}
		var batchErr error
		results := txQueries.InsertRainGaugeReading(ctx, readings)
		results.Exec(func(_ int, err error) {
			if err != nil && batchErr == nil {
				batchErr = err
			}
		})
		if err := results.Close(); err != nil && batchErr == nil {
			batchErr = err
		}
		if batchErr != nil {
			return fmt.Errorf("store readings: %w", batchErr)
		}
		stored = len(readings)
		return txQueries.PutSourceVersion(ctx, store.PutSourceVersionParams{Source: SourceName, Version: response.Header.Get("ETag"),
			SourceTime: feed.UpdatedAt, FetchedAt: now, Records: int32(len(feed.Gauges))})
	})
	if err != nil {
		return 0, err
	}
	return stored, nil
}

// Prune implements source.Pruner.
func (collector *Collector) Prune(ctx context.Context, now time.Time) error {
	queries := store.New(collector.Pool)
	if err := queries.DeleteRainGaugeReadingsBefore(ctx, now.Add(-retention)); err != nil {
		return err
	}
	return queries.DeleteRainGaugesFetchedBefore(ctx, now.Add(-gaugeRetention))
}
