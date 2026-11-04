package usgs

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// DefaultEndpoint is the USGS FDSN event query.
	DefaultEndpoint = "https://earthquake.usgs.gov/fdsnws/event/1/query"
	// Window is how far back the layer reaches. USGS revises recent events,
	// so all of it is read on every run.
	Window = 30 * 24 * time.Hour
	// retention keeps a year of events after they leave the window.
	retention = 400 * 24 * time.Hour
)

// box is a query area and the smallest magnitude asked for in it.
type box struct {
	south, north, west, east, minMagnitude float64
}

var boxes = []box{
	// The Indian Ocean basin wide enough to matter for tsunamis.
	{south: -15, north: 25, west: 60, east: 105, minMagnitude: 4.5},
	// Closer to Sri Lanka, whatever smaller events USGS has. Its catalogue
	// is thin here; it may return nothing.
	{south: 3, north: 13, west: 75, east: 87, minMagnitude: 2.5},
}

// Collector stores the window's events as USGS now has them.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// Endpoint overrides DefaultEndpoint, for tests.
	Endpoint string
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration { return 15 * time.Minute }

// Collect reads both areas and stores the events, returning how many. An
// event USGS no longer lists leaves the table; an empty answer changes
// nothing, because a calm month and a broken answer look alike.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	since := now.Add(-Window)
	var lists [][]Event
	var deleted []string
	for _, area := range boxes {
		payload, err := collector.Fetcher.Fetch(ctx, source.Request{
			URL: collector.endpoint(), Query: query(area, since), Accept: "application/geo+json, application/json",
		})
		if err != nil {
			return 0, fmt.Errorf("fetch earthquakes: %w", err)
		}
		events, gone, _, err := Decode(payload)
		if err != nil {
			return 0, err
		}
		lists = append(lists, events)
		deleted = append(deleted, gone...)
	}
	events := Merge(deleted, lists...)
	err := pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		keep := make([]string, 0, len(events))
		for _, event := range events {
			keep = append(keep, event.ID)
			var alert *string
			if event.Alert != "" {
				alert = &event.Alert
			}
			if err := queries.UpsertEarthquake(ctx, store.UpsertEarthquakeParams{
				ID: event.ID, Magnitude: event.Magnitude, MagType: event.MagType, Place: event.Place,
				OccurredAt: event.OccurredAt, Lon: event.Lon, Lat: event.Lat, DepthKm: event.DepthKm, Url: event.URL,
				Alert: alert, Tsunami: event.Tsunami, Significance: int32(event.Significance), EventType: event.Type,
				Status: event.Status, Source: SourceName, SourceTime: event.UpdatedAt, FetchedAt: now,
			}); err != nil {
				return fmt.Errorf("store event %s: %w", event.ID, err)
			}
		}
		for _, id := range deleted {
			if err := queries.DeleteEarthquake(ctx, id); err != nil {
				return fmt.Errorf("delete event %s: %w", id, err)
			}
		}
		if len(events) > 0 {
			if _, err := queries.DeleteEarthquakesMissingSince(ctx, store.DeleteEarthquakesMissingSinceParams{Since: since, Keep: keep}); err != nil {
				return fmt.Errorf("remove events USGS no longer lists: %w", err)
			}
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: SourceName, Version: now.UTC().Format(time.RFC3339), SourceTime: now,
			FetchedAt: now, Records: int32(len(events)),
		})
	})
	if err != nil {
		return 0, err
	}
	return len(events), nil
}

// Prune implements source.Pruner.
func (collector *Collector) Prune(ctx context.Context, now time.Time) error {
	return store.New(collector.Pool).DeleteEarthquakesBefore(ctx, now.Add(-retention))
}

func (collector *Collector) endpoint() string {
	if collector.Endpoint != "" {
		return collector.Endpoint
	}
	return DefaultEndpoint
}

func query(area box, since time.Time) url.Values {
	number := func(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }
	return url.Values{
		"format":         {"geojson"},
		"starttime":      {since.UTC().Format("2006-01-02T15:04:05")},
		"minlatitude":    {number(area.south)},
		"maxlatitude":    {number(area.north)},
		"minlongitude":   {number(area.west)},
		"maxlongitude":   {number(area.east)},
		"minmagnitude":   {number(area.minMagnitude)},
		"orderby":        {"time"},
		"includedeleted": {"true"},
	}
}
