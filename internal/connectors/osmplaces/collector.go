package osmplaces

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Endpoints are public Overpass API instances, tried in order. The main
// instance asks for no more than 10,000 queries and 1 GB a day; this
// collector sends one query of about 2 MB a week.
var Endpoints = []string{
	"https://overpass-api.de/api/interpreter",
	"https://overpass.kumi.systems/api/interpreter",
}

// Query selects named place nodes in a box around Sri Lanka. The box is used
// rather than the country's boundary relation because an area query is far
// more expensive for the server; the box excludes the Indian coast.
var Query = fmt.Sprintf(`[out:json][timeout:180];
node[place~"^(%s)$"][name](%g,%g,%g,%g);
out body qt;`, strings.Join(Kinds, "|"), minLat, minLon, maxLat, maxLon)

const (
	queryTimeout = 4 * time.Minute
	// MinimumPlaces is the fewest places a complete response has; Sri Lanka
	// has about 7,000 named place nodes.
	MinimumPlaces = 3000
	loadTimeout   = "2min"
)

// Collector loads populated places weekly.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// Endpoints overrides the package's Endpoints, for tests.
	Endpoints []string
	// Minimum overrides MinimumPlaces, for tests with small fixtures.
	Minimum int
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration { return 7 * 24 * time.Hour }

// RetryAfter implements source.Retrier. Public Overpass instances are often
// busy, so a failed run is retried after a few hours.
func (collector *Collector) RetryAfter() time.Duration { return 3 * time.Hour }

// Collect replaces the stored places with the current OpenStreetMap ones.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	payload, err := collector.fetch(ctx)
	if err != nil {
		return 0, err
	}
	snapshot, err := Decode(payload)
	if err != nil {
		return 0, err
	}
	minimum := MinimumPlaces
	if collector.Minimum > 0 {
		minimum = collector.Minimum
	}
	if len(snapshot.Places) < minimum {
		return 0, fmt.Errorf("overpass returned %d places, want at least %d", len(snapshot.Places), minimum)
	}
	if err := collector.save(ctx, snapshot, now); err != nil {
		return 0, err
	}
	return len(snapshot.Places), nil
}

// fetch tries each endpoint in turn and returns the first complete answer.
func (collector *Collector) fetch(ctx context.Context) ([]byte, error) {
	endpoints := collector.Endpoints
	if len(endpoints) == 0 {
		endpoints = Endpoints
	}
	var errs []error
	for _, endpoint := range endpoints {
		payload, err := collector.Fetcher.Fetch(ctx, source.Request{
			Method:  "POST",
			URL:     endpoint,
			Form:    url.Values{"data": {Query}},
			Timeout: queryTimeout,
		})
		if err == nil {
			return payload, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", endpoint, err))
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("query overpass: %w", errors.Join(errs...))
}

func (collector *Collector) save(ctx context.Context, snapshot Snapshot, now time.Time) error {
	rows := make([]store.StagePlacesParams, len(snapshot.Places))
	for index, place := range snapshot.Places {
		alt := place.AltNames
		if alt == nil {
			alt = []string{}
		}
		rows[index] = store.StagePlacesParams{
			ID: place.ID, Name: place.Name, NameSi: optional(place.NameSI), NameTa: optional(place.NameTA),
			AltNames: alt, Kind: place.Kind, Population: place.Population, Lon: place.Lon, Lat: place.Lat,
		}
	}
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '"+loadTimeout+"'"); err != nil {
			return err
		}
		queries := store.New(tx)
		if err := queries.ClearPlaceStaging(ctx); err != nil {
			return err
		}
		if _, err := queries.StagePlaces(ctx, rows); err != nil {
			return fmt.Errorf("stage places: %w", err)
		}
		if _, err := queries.MergeStagedPlaces(ctx, store.MergeStagedPlacesParams{
			Source: SourceName, SourceTime: snapshot.SourceTime, FetchedAt: now,
		}); err != nil {
			return fmt.Errorf("merge places: %w", err)
		}
		if _, err := queries.DeletePlacesNotStaged(ctx); err != nil {
			return fmt.Errorf("delete removed places: %w", err)
		}
		if err := queries.ClearPlaceStaging(ctx); err != nil {
			return err
		}
		if _, err := queries.AssignPlaceRegions(ctx); err != nil {
			return fmt.Errorf("assign places to regions: %w", err)
		}
		if err := queries.DeleteSearchNames(ctx, "place"); err != nil {
			return err
		}
		if _, err := queries.InsertPlaceSearchNames(ctx); err != nil {
			return fmt.Errorf("index place names: %w", err)
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: SourceName, Version: snapshot.SourceTime.Format(time.RFC3339),
			SourceTime: snapshot.SourceTime, FetchedAt: now, Records: int32(len(rows)),
		})
	})
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
