package irrigation

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Endpoint is the query URL of the department's public gauge feature service.
const Endpoint = "https://services3.arcgis.com/J7ZFXmR8rSmQ3FGf/arcgis/rest/services/gauges_2_view/FeatureServer/0/query"

const (
	// pageSize is below the service's maxRecordCount of 1000.
	pageSize = 1000
	// maxPages bounds one run; a week of readings is about 7 pages.
	maxPages = 20
	// backfill is how far back the first run reaches.
	backfill = 7 * 24 * time.Hour
	// overlap re-reads recent readings so ones entered late are not missed.
	overlap = 3 * time.Hour
)

// Fetcher performs a GET request and returns the response body.
type Fetcher interface {
	GetJSON(ctx context.Context, endpoint string, query url.Values) ([]byte, error)
}

// Collector fetches new gauge readings and stores them.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher Fetcher
	// Endpoint overrides the service URL, for tests.
	Endpoint string
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector. Gauges report about hourly, some
// several times an hour during floods.
func (collector *Collector) Interval() time.Duration { return 15 * time.Minute }

// Collect fetches readings since shortly before the newest stored one and
// stores stations and readings in one transaction.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	queries := store.New(collector.Pool)
	since := now.Add(-backfill)
	if latest, err := queries.LatestRiverObservation(ctx, SourceName); err == nil {
		if candidate := latest.Add(-overlap); candidate.After(since) {
			since = candidate
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("read latest observation: %w", err)
	}

	var stations []Station
	var readings []Reading
	for page := 0; ; page++ {
		if page == maxPages {
			return 0, fmt.Errorf("more than %d pages since %s", maxPages, since.Format(time.RFC3339))
		}
		payload, err := collector.Fetcher.GetJSON(ctx, collector.endpoint(), pageQuery(since, page))
		if err != nil {
			return 0, fmt.Errorf("fetch page %d: %w", page, err)
		}
		decoded, err := Decode(payload)
		if err != nil {
			return 0, err
		}
		stations = append(stations, decoded.Stations...)
		readings = append(readings, decoded.Readings...)
		if !decoded.More {
			break
		}
	}
	if len(readings) == 0 {
		return 0, nil
	}
	return len(readings), collector.save(ctx, latestStations(stations), readings, now)
}

func (collector *Collector) endpoint() string {
	if collector.Endpoint != "" {
		return collector.Endpoint
	}
	return Endpoint
}

// pageQuery selects readings created at or after since, oldest first, so
// offsets stay stable while new readings arrive during a run.
func pageQuery(since time.Time, page int) url.Values {
	return url.Values{
		"where":             {"CreationDate >= TIMESTAMP '" + since.UTC().Format("2006-01-02 15:04:05") + "'"},
		"outFields":         {"objectid,basin,gauge,water_level,rain_fall,CreationDate,alertpull,minorpull,majorpull"},
		"orderByFields":     {"CreationDate ASC,objectid ASC"},
		"returnGeometry":    {"true"},
		"outSR":             {"4326"},
		"resultOffset":      {strconv.Itoa(page * pageSize)},
		"resultRecordCount": {strconv.Itoa(pageSize)},
		"f":                 {"json"},
	}
}

// latestStations keeps the most recent description of each station.
func latestStations(stations []Station) []Station {
	index := map[string]int{}
	var out []Station
	for _, station := range stations {
		if i, ok := index[station.ID]; ok {
			if station.ObservedAt.After(out[i].ObservedAt) {
				out[i] = station
			}
			continue
		}
		index[station.ID] = len(out)
		out = append(out, station)
	}
	return out
}

func (collector *Collector) save(ctx context.Context, stations []Station, readings []Reading, now time.Time) error {
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		for _, station := range stations {
			if err := queries.UpsertRiverStation(ctx, store.UpsertRiverStationParams{
				ID: station.ID, Name: station.Name, Basin: station.Basin,
				Lon: station.Lon, Lat: station.Lat, Unit: station.Unit,
				AlertLevel: station.AlertLevel, MinorFloodLevel: station.MinorFloodLevel,
				MajorFloodLevel: station.MajorFloodLevel, Source: SourceName, UpdatedAt: now,
			}); err != nil {
				return fmt.Errorf("upsert station %s: %w", station.ID, err)
			}
		}
		known, err := queries.ListRiverStations(ctx)
		if err != nil {
			return err
		}
		exists := make(map[string]bool, len(known))
		for _, station := range known {
			exists[station.ID] = true
		}
		params := make([]store.InsertRiverReadingParams, 0, len(readings))
		for _, reading := range readings {
			// A reading for a station never seen with a position cannot be mapped.
			if !exists[reading.StationID] {
				continue
			}
			params = append(params, store.InsertRiverReadingParams{
				StationID: reading.StationID, ObservedAt: reading.ObservedAt,
				WaterLevel: reading.WaterLevel, RainfallMm: reading.RainfallMM, FetchedAt: now,
			})
		}
		var batchErr error
		results := queries.InsertRiverReading(ctx, params)
		results.Exec(func(_ int, err error) {
			if err != nil && batchErr == nil {
				batchErr = err
			}
		})
		if err := results.Close(); err != nil && batchErr == nil {
			batchErr = err
		}
		return batchErr
	})
}
