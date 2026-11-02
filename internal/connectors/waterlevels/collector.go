package waterlevels

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// minimumReservoirs and minimumReports are the fewest records a complete
	// response has; the source had 142 and 62 when this was written.
	minimumReservoirs = 80
	minimumReports    = 30
	// nextHourRetention keeps forecasts long enough to compare them with what
	// happened; the map shows only those under two hours old.
	nextHourRetention = 30 * 24 * time.Hour
	// duplicateWithin is how close a report from the feed may lie to a station
	// of another source before it is taken for the same gauge under another
	// name, and left out rather than drawn twice.
	duplicateWithin = 150.0
)

// ReservoirCollector stores the day's reservoir storage and spill status. The
// API sends no ETag or Last-Modified, so there is nothing to make a request
// conditional on; the response is a small JSON document.
type ReservoirCollector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// Endpoint and Minimum override the defaults, for tests.
	Endpoint string
	Minimum  int
}

// Name implements source.Collector.
func (collector *ReservoirCollector) Name() string { return ReservoirsSource }

// Interval implements source.Collector. The agencies report daily; four
// fetches a day pick up a late sheet within hours.
func (collector *ReservoirCollector) Interval() time.Duration { return 6 * time.Hour }

// Collect fetches the current response and stores it as today's snapshot.
func (collector *ReservoirCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	endpoint := collector.Endpoint
	if endpoint == "" {
		endpoint = ReservoirsURL
	}
	payload, err := collector.Fetcher.Fetch(ctx, source.Request{URL: endpoint})
	if err != nil {
		return 0, fmt.Errorf("fetch reservoirs: %w", err)
	}
	snapshot, err := DecodeReservoirs(payload)
	if err != nil {
		return 0, err
	}
	minimum := minimumReservoirs
	if collector.Minimum > 0 {
		minimum = collector.Minimum
	}
	if len(snapshot.Reservoirs) < minimum {
		return 0, fmt.Errorf("source returned %d reservoirs, want at least %d", len(snapshot.Reservoirs), minimum)
	}
	day := SnapshotDay(snapshot.GeneratedAt)
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		for _, reservoir := range snapshot.Reservoirs {
			if err := queries.UpsertReservoir(ctx, store.UpsertReservoirParams{
				ID: reservoir.ID, Name: reservoir.Name, District: reservoir.District, River: reservoir.River,
				Kind: reservoir.Kind, Source: ReservoirsSource, SourceUrl: reservoir.URL, UpdatedAt: now,
			}); err != nil {
				return fmt.Errorf("upsert reservoir %s: %w", reservoir.ID, err)
			}
			params := store.UpsertReservoirReadingParams{
				ReservoirID: reservoir.ID, SnapshotDay: pgDate(&day), ReadingDay: pgDate(reservoir.ReadingDay),
				Status: reservoir.Status, StoragePercent: reservoir.StoragePercent, StorageFresh: reservoir.StorageFresh,
				Agency: reservoir.Agency, Source: ReservoirsSource, SourceTime: snapshot.GeneratedAt, FetchedAt: now,
			}
			if flow := reservoir.Flow; flow != nil {
				params.FlowKind, params.FlowCusecs, params.FlowAgency, params.FlowDay =
					text(flow.Kind), flow.Cusecs, text(flow.Agency), pgDate(flow.Day)
			}
			if err := queries.UpsertReservoirReading(ctx, params); err != nil {
				return fmt.Errorf("store reading of %s: %w", reservoir.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(snapshot.Reservoirs), nil
}

// SnapshotDay is the calendar day in Sri Lanka of a response, at midnight UTC
// as a date column holds it.
func SnapshotDay(generated time.Time) time.Time {
	local := generated.In(ColomboZone)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

func text(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// KnownStation is a station already stored, as Plan needs it.
type KnownStation struct {
	ID, Source string
	Lon, Lat   float64
}

// StationPlan says what to do with one response's reports.
type StationPlan struct {
	// Extra are the gauges this collector adds or updates, with their
	// identifiers. Nothing else from the feed enters river_stations.
	Extra []PlannedStation
	// Forecasts are the next-hour forecasts of every known station, by
	// identifier, whichever source owns it.
	Forecasts []PlannedForecast
	// Duplicates names reports dropped for lying on top of another source's
	// station under a name we do not know.
	Duplicates []string
}

// PlannedStation is a report with the identifier it is stored under.
type PlannedStation struct {
	ID     string
	Report Report
}

// PlannedForecast is a forecast with the station it belongs to.
type PlannedForecast struct {
	StationID string
	NextHour  NextHour
}

// Plan decides, for each report, whether it describes a station another
// source owns or a new gauge. A station of another source is never touched:
// the feed's name, its alias or its own identifier finding such a station is
// enough, and a report within 150 m of one is treated as the same gauge. Plan
// does no I/O.
func Plan(reports []Report, known []KnownStation) StationPlan {
	byID := make(map[string]KnownStation, len(known))
	for _, station := range known {
		byID[station.ID] = station
	}
	var plan StationPlan
	for _, report := range reports {
		id := StationID(report)
		owner, found := byID[id]
		if !found {
			if slug, ok := byID[report.Slug]; ok && slug.Source != GaugesSource {
				id, owner, found = slug.ID, slug, true
			}
		}
		if found && owner.Source != GaugesSource {
			if report.NextHour != nil {
				plan.Forecasts = append(plan.Forecasts, PlannedForecast{StationID: id, NextHour: *report.NextHour})
			}
			continue
		}
		if !found && nearOther(report, known) {
			plan.Duplicates = append(plan.Duplicates, report.Name)
			continue
		}
		plan.Extra = append(plan.Extra, PlannedStation{ID: id, Report: report})
		if report.NextHour != nil {
			plan.Forecasts = append(plan.Forecasts, PlannedForecast{StationID: id, NextHour: *report.NextHour})
		}
	}
	return plan
}

func nearOther(report Report, known []KnownStation) bool {
	for _, station := range known {
		if station.Source != GaugesSource && metresBetween(report.Lon, report.Lat, station.Lon, station.Lat) <= duplicateWithin {
			return true
		}
	}
	return false
}

func metresBetween(lon1, lat1, lon2, lat2 float64) float64 {
	const earth = 6371000.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earth * math.Asin(math.Sqrt(a))
}

// GaugeCollector adds the river gauges the Irrigation Department's feed does
// not have to river_stations and river_readings, and stores the experimental
// next-hour forecast of every station. The department's own feed stays
// authoritative and direct.
type GaugeCollector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	Logger  *slog.Logger
	// Endpoint and Minimum override the defaults, for tests.
	Endpoint string
	Minimum  int
}

// Name implements source.Collector.
func (collector *GaugeCollector) Name() string { return GaugesSource }

// Interval implements source.Collector.
func (collector *GaugeCollector) Interval() time.Duration { return 30 * time.Minute }

// ErrIrrigationNotLoaded stops the collector until the department's feed has
// run, because without its stations every report would look like a new gauge.
var ErrIrrigationNotLoaded = errors.New("no Irrigation Department stations stored yet")

// Collect fetches the current levels and stores the new gauges' readings and
// every station's forecast, returning the number of readings stored.
func (collector *GaugeCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	endpoint := collector.Endpoint
	if endpoint == "" {
		endpoint = LevelsURL
	}
	payload, err := collector.Fetcher.Fetch(ctx, source.Request{URL: endpoint})
	if err != nil {
		return 0, fmt.Errorf("fetch levels: %w", err)
	}
	snapshot, err := DecodeLevels(payload)
	if err != nil {
		return 0, err
	}
	minimum := minimumReports
	if collector.Minimum > 0 {
		minimum = collector.Minimum
	}
	if len(snapshot.Reports) < minimum {
		return 0, fmt.Errorf("source returned %d stations, want at least %d", len(snapshot.Reports), minimum)
	}

	rows, err := store.New(collector.Pool).RiverStationSources(ctx)
	if err != nil {
		return 0, fmt.Errorf("read stations: %w", err)
	}
	known := make([]KnownStation, 0, len(rows))
	irrigationStations := 0
	for _, row := range rows {
		known = append(known, KnownStation{ID: row.ID, Source: row.Source, Lon: row.Lon, Lat: row.Lat})
		if row.Source == irrigation.SourceName {
			irrigationStations++
		}
	}
	if irrigationStations == 0 {
		return 0, ErrIrrigationNotLoaded
	}

	plan := Plan(snapshot.Reports, known)
	for _, name := range plan.Duplicates {
		collector.logger().Warn("gauge left out: within 150 m of another source's station", "name", name)
	}
	stored := 0
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		for _, station := range plan.Extra {
			report := station.Report
			if err := queries.UpsertExtraRiverStation(ctx, store.UpsertExtraRiverStationParams{
				ID: station.ID, Name: report.Name, Basin: report.River, Lon: report.Lon, Lat: report.Lat, Unit: report.Unit,
				AlertLevel: report.AlertLevel, MinorFloodLevel: report.MinorFloodLevel, MajorFloodLevel: report.MajorFloodLevel,
				Source: GaugesSource, SourceUrl: report.URL, Upstream: text(Upstream(station.ID)), UpdatedAt: now,
			}); err != nil {
				return fmt.Errorf("upsert station %s: %w", station.ID, err)
			}
			if report.Level == nil || report.ObservedAt == nil {
				continue
			}
			readings := queries.InsertRiverReading(ctx, []store.InsertRiverReadingParams{{
				StationID: station.ID, ObservedAt: *report.ObservedAt, WaterLevel: *report.Level, FetchedAt: now,
			}})
			var batchErr error
			readings.Exec(func(_ int, err error) {
				if err != nil && batchErr == nil {
					batchErr = err
				}
			})
			if err := readings.Close(); err != nil && batchErr == nil {
				batchErr = err
			}
			if batchErr != nil {
				return fmt.Errorf("store reading of %s: %w", station.ID, batchErr)
			}
			stored++
		}
		params := make([]store.InsertNextHourParams, 0, len(plan.Forecasts))
		for _, forecast := range plan.Forecasts {
			next := forecast.NextHour
			params = append(params, store.InsertNextHourParams{
				StationID: forecast.StationID, IssuedAt: next.IssuedAt, LevelFt: next.LevelFt,
				LowFt: next.LowFt, HighFt: next.HighFt, ChangeFt: next.ChangeFt,
				Direction: next.Direction, Crossing: next.Crossing, Method: next.Method,
				Source: GaugesSource, FetchedAt: now,
			})
		}
		var batchErr error
		results := queries.InsertNextHour(ctx, params)
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
	if err != nil {
		return 0, err
	}
	return stored, nil
}

// Prune implements source.Pruner.
func (collector *GaugeCollector) Prune(ctx context.Context, now time.Time) error {
	return store.New(collector.Pool).DeleteNextHourBefore(ctx, now.Add(-nextHourRetention))
}

func (collector *GaugeCollector) logger() *slog.Logger {
	if collector.Logger != nil {
		return collector.Logger
	}
	return slog.Default()
}

func pgDate(day *time.Time) pgtype.Date {
	if day == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *day, Valid: true}
}
