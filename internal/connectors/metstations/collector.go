package metstations

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// threeHourlyEvery: reports come every three hours and the copy is polled
	// often enough to catch one soon after it appears. An unchanged file costs
	// one conditional request (304).
	threeHourlyEvery = time.Hour
	// dailyEvery: reports are daily, at irregular hours.
	dailyEvery = 3 * time.Hour
	// tailBytes is what is read from the end of the 10 MB daily file: rows are
	// in time order at about 130 bytes each and 60 a report, so about 30 reports.
	tailBytes = 256 << 10
	// threeHourlyBackfill is how far a first run reaches, and threeHourlyWindow
	// how far later runs re-read, to pick up late corrections.
	threeHourlyBackfill = 35 * 24 * time.Hour
	threeHourlyWindow   = 3 * 24 * time.Hour
	// dailyWindow is what is stored from the tail, which stops being all of the
	// tail when the source falls behind.
	dailyWindow = 120 * 24 * time.Hour
	// Retention.
	keep3h    = 45 * 24 * time.Hour
	keepDaily = 400 * 24 * time.Hour
)

// Doer sends a request to a source.
type Doer interface {
	Do(ctx context.Context, request source.Request) (source.Response, error)
}

// ThreeHourlyCollector stores the 3-hourly observations.
type ThreeHourlyCollector struct {
	Pool   *pgxpool.Pool
	Client Doer
	// Endpoint overrides the file's address, for tests.
	Endpoint string
	Logger   *slog.Logger
}

// Name implements source.Collector.
func (collector *ThreeHourlyCollector) Name() string { return SourceThreeHourly }

// Interval implements source.Collector.
func (collector *ThreeHourlyCollector) Interval() time.Duration { return threeHourlyEvery }

// Collect fetches the file unless it has not changed since the last run.
func (collector *ThreeHourlyCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	queries := store.New(collector.Pool)
	request := source.Request{URL: firstNonEmpty(collector.Endpoint, EndpointThreeHourly), Accept: "application/json",
		AcceptStatus: []int{http.StatusOK, http.StatusNotModified}}
	version, err := queries.GetSourceVersion(ctx, SourceThreeHourly)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("read source version: %w", err)
	}
	if err == nil && version.Version != "" {
		request.Header = http.Header{"If-None-Match": {version.Version}}
	}
	response, err := collector.Client.Do(ctx, request)
	if err != nil {
		return 0, fmt.Errorf("fetch 3-hourly file: %w", err)
	}
	if response.Status == http.StatusNotModified {
		return 0, nil
	}
	count, err := queries.CountMetReadings3h(ctx)
	if err != nil {
		return 0, err
	}
	window := threeHourlyWindow
	if count == 0 {
		window = threeHourlyBackfill
	}
	decoded, err := DecodeThreeHourly(response.Body, now.Add(-window))
	if err != nil {
		return 0, err
	}
	if len(decoded.Unknown) > 0 && collector.Logger != nil {
		sort.Strings(decoded.Unknown)
		collector.Logger.Warn("3-hourly file names unknown stations", "stations", decoded.Unknown)
	}
	stored := len(decoded.Readings)
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		if err := upsertKnownStations(ctx, queries, SourceThreeHourly, now); err != nil {
			return err
		}
		params := make([]store.InsertMetReading3hParams, len(decoded.Readings))
		for index, reading := range decoded.Readings {
			params[index] = store.InsertMetReading3hParams{
				StationID: reading.StationID, ObservedAt: reading.ObservedAt, RainMm: reading.RainMm,
				TempC: reading.TempC, HumidityPct: reading.HumidityPct, Source: SourceThreeHourly, FetchedAt: now,
			}
		}
		if err := batch(queries.InsertMetReading3h(ctx, params)); err != nil {
			return fmt.Errorf("store readings: %w", err)
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{Source: SourceThreeHourly,
			Version: response.Header.Get("ETag"), SourceTime: decoded.Newest, FetchedAt: now, Records: int32(stored)})
	})
	if err != nil {
		return 0, err
	}
	return stored, nil
}

// Prune implements source.Pruner.
func (collector *ThreeHourlyCollector) Prune(ctx context.Context, now time.Time) error {
	return store.New(collector.Pool).DeleteMetReadings3hBefore(ctx, now.Add(-keep3h))
}

// DailyCollector stores the daily report's rows, with the stations they name.
type DailyCollector struct {
	Pool   *pgxpool.Pool
	Client Doer
	// Endpoint and PlacesEndpoint override the files' addresses, for tests.
	Endpoint, PlacesEndpoint string
	Logger                   *slog.Logger
}

// Name implements source.Collector.
func (collector *DailyCollector) Name() string { return SourceDaily }

// Interval implements source.Collector.
func (collector *DailyCollector) Interval() time.Duration { return dailyEvery }

// Collect reads the end of the daily file by HTTP range and the list of
// places. A server that ignores the range sends the whole file, which decodes
// the same way.
func (collector *DailyCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	response, err := collector.Client.Do(ctx, source.Request{
		URL: firstNonEmpty(collector.Endpoint, EndpointDaily), Accept: "application/json",
		Header:       http.Header{"Range": {fmt.Sprintf("bytes=-%d", tailBytes)}},
		AcceptStatus: []int{http.StatusOK, http.StatusPartialContent},
	})
	if err != nil {
		return 0, fmt.Errorf("fetch daily file: %w", err)
	}
	decoded, err := DecodeDaily(response.Body, now.Add(-dailyWindow))
	if err != nil {
		return 0, err
	}
	placesResponse, err := collector.Client.Do(ctx, source.Request{URL: firstNonEmpty(collector.PlacesEndpoint, EndpointPlaces), Accept: "application/json"})
	if err != nil {
		return 0, fmt.Errorf("fetch places: %w", err)
	}
	places, err := DecodePlaces(placesResponse.Body)
	if err != nil {
		return 0, err
	}

	stations := Resolve(decoded.Readings, places)
	var readings []store.InsertMetReadingDailyParams
	for _, reading := range decoded.Readings {
		if !stations.Matched[reading.StationID] {
			continue
		}
		readings = append(readings, store.InsertMetReadingDailyParams{
			StationID: reading.StationID, ReportDate: pgtype.Date{Time: reading.Date, Valid: true},
			ObservedAt: reading.ObservedAt, RainMm: reading.RainMm, TempMinC: reading.TempMinC, TempMaxC: reading.TempMaxC,
			Source: SourceDaily, FetchedAt: now,
		})
	}
	if collector.Logger != nil {
		collector.Logger.Info("daily report matched to stations", "rows", len(decoded.Readings), "stored", len(readings),
			"matched", len(stations.Matched), "unmatched", stations.Unmatched, "duplicates", decoded.Duplicates,
			"newest", decoded.Newest)
	}
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		if err := upsertKnownStations(ctx, queries, SourceDaily, now); err != nil {
			return err
		}
		params := make([]store.UpsertMetStationParams, len(stations.Geocoded))
		for index, station := range stations.Geocoded {
			params[index] = store.UpsertMetStationParams{ID: station.ID, Name: station.Name, Lon: station.Lon, Lat: station.Lat,
				PositionSource: station.PositionSource, Source: SourceDaily, UpdatedAt: now}
		}
		if err := batch(queries.UpsertMetStation(ctx, params)); err != nil {
			return fmt.Errorf("store stations: %w", err)
		}
		if err := batch(queries.InsertMetReadingDaily(ctx, readings)); err != nil {
			return fmt.Errorf("store readings: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(readings), nil
}

// Prune implements source.Pruner.
func (collector *DailyCollector) Prune(ctx context.Context, now time.Time) error {
	return store.New(collector.Pool).DeleteMetReadingsDailyBefore(ctx, now.Add(-keepDaily))
}

// Stations is the result of matching a daily report's names to positions.
type Stations struct {
	// Matched holds every station identifier with a position.
	Matched map[string]bool
	// Geocoded are the matched stations not in the known table.
	Geocoded []Station
	// Unmatched names have no usable position; their readings are not stored.
	Unmatched []string
}

// Resolve matches reports' station names to the known table, then to the
// geocoded list.
func Resolve(readings []ReadingDaily, places map[string]Position) Stations {
	out := Stations{Matched: map[string]bool{}}
	seen := map[string]bool{}
	for _, reading := range readings {
		id := reading.StationID
		if seen[id] {
			continue
		}
		seen[id] = true
		if _, known := KnownByID(id); known {
			out.Matched[id] = true
			continue
		}
		position, ok := places[id]
		if !ok {
			out.Unmatched = append(out.Unmatched, reading.Name)
			continue
		}
		out.Matched[id] = true
		out.Geocoded = append(out.Geocoded, Station{ID: id, Name: reading.Name, Lat: position.Lat, Lon: position.Lon, PositionSource: PositionGeocoded})
	}
	sort.Strings(out.Unmatched)
	return out
}

func upsertKnownStations(ctx context.Context, queries *store.Queries, sourceName string, now time.Time) error {
	params := make([]store.UpsertMetStationParams, len(knownStations))
	for index, station := range knownStations {
		var wmo *string
		if station.WMOID != "" {
			number := station.WMOID
			wmo = &number
		}
		params[index] = store.UpsertMetStationParams{ID: station.ID, Name: station.Name, WmoID: wmo, Lon: station.Lon, Lat: station.Lat,
			ElevationM: station.ElevationM, PositionSource: station.PositionSource, Source: sourceName, UpdatedAt: now}
	}
	if err := batch(queries.UpsertMetStation(ctx, params)); err != nil {
		return fmt.Errorf("store known stations: %w", err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// batchResults is what sqlc's :batchexec queries return.
type batchResults interface {
	Exec(func(int, error))
	Close() error
}

func batch(results batchResults) error {
	var first error
	results.Exec(func(_ int, err error) {
		if err != nil && first == nil {
			first = err
		}
	})
	if err := results.Close(); err != nil && first == nil {
		first = err
	}
	return first
}
