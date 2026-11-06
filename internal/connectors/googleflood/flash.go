package googleflood

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// FlashSourceName is the flash-flood collector's name in source_runs. Its rows
// keep SourceName: the data is the same API's.
const FlashSourceName = "google-flash-floods"

// FlashInterval is how often flash floods are read. Google issues them a few
// times a day, and they matter within hours, so waiting up to an hour for the
// forecast collector's turn was too slow (on 6 November 2026 a Colombo forecast
// reached us only after the flooding). A calm run is one request: the search,
// whose single page holds Sri Lanka's flash floods, and no polygon, since a
// polygon already stored is not read again.
const FlashInterval = 10 * time.Minute

// FlashCollector keeps Google's flash-flood forecasts for Sri Lanka, apart from
// the hourly forecast collector.
type FlashCollector struct {
	Pool   *pgxpool.Pool
	Client Doer
	// APIKey is the Google Cloud API key; it is sent in a header only.
	APIKey string
	// BaseURL overrides DefaultBaseURL, for tests.
	BaseURL string
	Logger  *slog.Logger
}

// Name implements source.Collector.
func (collector *FlashCollector) Name() string { return FlashSourceName }

// Interval implements source.Collector.
func (collector *FlashCollector) Interval() time.Duration { return FlashInterval }

// RetryAfter retries a failed run at the next turn rather than after the
// scheduler's longer back-off: a missed flash flood is the costly failure.
func (collector *FlashCollector) RetryAfter() time.Duration { return FlashInterval }

// LongRunTimeout makes the scheduler run it beside the other collectors, so a
// slow download elsewhere never delays a flash flood.
func (collector *FlashCollector) LongRunTimeout() time.Duration { return 2 * time.Minute }

func (collector *FlashCollector) api() *Collector {
	return &Collector{Pool: collector.Pool, Client: collector.Client, APIKey: collector.APIKey, BaseURL: collector.BaseURL, Logger: collector.Logger}
}

// Collect reads the flash floods affecting Sri Lanka and replaces the stored
// ones. It returns the number of areas stored.
func (collector *FlashCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	if strings.TrimSpace(collector.APIKey) == "" {
		return 0, errors.New("LANKAWEATHER_GOOGLE_API_KEY is not set")
	}
	api := collector.api()
	areas, err := api.search(ctx, "/v1/flashFloods:search", []string{RegionCode}, DecodeFlashFloods)
	if err != nil {
		return 0, fmt.Errorf("flash floods: %w", err)
	}
	seen, err := firstSeen(ctx, store.New(collector.Pool), FlashKinds)
	if err != nil {
		return 0, err
	}
	known := make(map[string][]byte, len(seen))
	for id, sighting := range seen {
		known[id] = sighting.shape
	}
	shapes := api.polygons(ctx, areas, known)
	var fresh []Area
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		// Read again inside the transaction, so two runs cannot both claim a
		// first sighting.
		seen, err := firstSeen(ctx, queries, FlashKinds)
		if err != nil {
			return err
		}
		if err := queries.DeleteFloodAreasOfKinds(ctx, FlashKinds); err != nil {
			return err
		}
		fresh, err = insertAreas(ctx, queries, areas, shapes, seen, now)
		return err
	})
	if err != nil {
		return 0, err
	}
	for _, area := range fresh {
		collector.logFirstSight(area, now)
	}
	stored := 0
	for _, area := range areas {
		if shapes[area.PolygonID] != nil {
			stored++
		}
	}
	return stored, nil
}

// logFirstSight records how long after Google's issue time an area reached us,
// so the delay can be measured.
func (collector *FlashCollector) logFirstSight(area Area, now time.Time) {
	if collector.Logger == nil {
		return
	}
	attributes := []any{"kind", area.Kind}
	if area.IssuedAt != nil {
		attributes = append(attributes, "issued_at", area.IssuedAt.UTC(), "delay_minutes", int(now.Sub(*area.IssuedAt).Minutes()))
	}
	collector.Logger.Info("google flash flood first seen", attributes...)
}
