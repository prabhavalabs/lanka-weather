// Package riverseed writes a pretend flood into a local database, so that flood
// rendering can be checked when no real flood is happening. It exists only for
// local work and tests:
//
//   - it refuses to touch a database named lankaweather;
//   - it marks everything it writes with the source "local-seed", which the API
//     reports as `seeded`, and the web app shows as a banner across the map;
//   - nothing in the service calls it. Its only callers are cmd/riverseed and the
//     integration tests.
//
// The polygons are real past flood extents from the Irrigation Department, used
// here as the most realistic shapes available: raster-traced, hugely detailed
// and often overlapping.
package riverseed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Source marks seeded rows.
const Source = floods.SeedSource

// protected databases are never seeded.
var protected = map[string]bool{"lankaweather": true}

// marker is the fetch time of seeded readings, so they can be removed exactly.
var marker = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// stationLevels are elevated readings: every level on the risk scale, and one
// stretch still calm beside them. The last value is where each stops rising.
var stationLevels = []struct {
	id   string
	from float64
	to   float64
}{
	{"nagalagam-street", 5.0, 7.6},
	{"hanwella", 6.0, 8.1},
	{"glencourse", 13.0, 15.3},
	{"rathnapura", 6.5, 9.9},
	{"ellagawa", 8.0, 10.2},
	{"peradeniya", 3.0, 5.4},
	{"thalgahagoda", 1.1, 1.9},
	{"baddegama", 3.0, 3.6},
	{"horowpothana", 4.0, 6.2},
}

// googleGauges are Google gauges to raise, by sub-basin.
var googleGauges = []struct {
	id, severity, trend string
}{
	{"hybas_4121156400", "extreme", "rise"},
	{"hybas_4121160080", "severe", "rise"},
	{"hybas_4121600660", "severe", "no_change"},
	{"hybas_4121156830", "above_normal", "rise"},
}

// seededAreas are Google polygon kinds drawn from past extents, one of each, so
// every kind can be checked against real shapes.
var seededAreas = []struct {
	event, kind, severity string
}{
	{"kelani-ganga-flood-1989-critical", "inundation_low", "severe"},
	{"kelani-ganga-flood-2016-major", "inundation_medium", "severe"},
	{"kelani-ganga-flood-2012-minor", "inundation_high", "severe"},
	{"kalu-ganga-flood-2017", "notification", "above_normal"},
	{"nilwala-ganga-flood-2017", "flash_highly_likely", "severe"},
	{"kirama-oya-flood-2017", "flash_likely", "above_normal"},
	{"yan-oya-flood-2016", "significant_event", "severe"},
}

func guard(ctx context.Context, pool *pgxpool.Pool) error {
	var name string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return err
	}
	if protected[name] {
		return fmt.Errorf("refusing to seed the %s database", name)
	}
	return nil
}

// Apply writes the pretend flood, replacing any earlier one.
func Apply(ctx context.Context, pool *pgxpool.Pool, now time.Time) error {
	if err := guard(ctx, pool); err != nil {
		return err
	}
	if err := Clear(ctx, pool); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, station := range stationLevels {
			tag, err := tx.Exec(ctx, `UPDATE river_stations SET source = $1 WHERE id = $2`, Source, station.id)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			// Six hours of readings, rising to the seeded level; steep at the end.
			for step := range 13 {
				share := math.Pow(float64(step)/12, 2)
				level := station.from + (station.to-station.from)*share
				at := now.Add(-time.Duration(12-step) * 30 * time.Minute).Truncate(time.Minute)
				if _, err := tx.Exec(ctx, `INSERT INTO river_readings (station_id, observed_at, water_level, fetched_at)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (station_id, observed_at) DO UPDATE SET water_level = EXCLUDED.water_level, fetched_at = EXCLUDED.fetched_at`,
					station.id, at, level, marker); err != nil {
					return err
				}
			}
		}
		for _, gauge := range googleGauges {
			if _, err := tx.Exec(ctx, `UPDATE flood_gauges
				SET source = $1, severity = $2, trend = $3, issued_at = $4, fetched_at = $4,
				    forecast_start = $4, forecast_end = $5
				WHERE gauge_id = $6`,
				Source, gauge.severity, gauge.trend, now, now.Add(72*time.Hour), gauge.id); err != nil {
				return err
			}
		}
		// A gauge Google has not verified, to check how the map marks one.
		if _, err := tx.Exec(ctx, `INSERT INTO flood_gauges (gauge_id, source, gauge_source, site_name, river, lon, lat,
				quality_verified, severity, trend, issued_at, value_unit, fetched_at)
			SELECT 'hybas_' || q.hybas_id, $1, 'HYBAS', '', '', ST_X(q.c), ST_Y(q.c), false, 'severe', 'rise', $2, 'm3/s', $2
			FROM (SELECT s.hybas_id, ST_PointOnSurface(s.geom) AS c FROM river_subbasins s
			      JOIN official_basins b ON b.fid = s.basin_fid AND b.id = 'heda-oya' ORDER BY s.hybas_id LIMIT 1) q
			ON CONFLICT (gauge_id) DO NOTHING`, Source, now); err != nil {
			return err
		}
		for _, area := range seededAreas {
			if _, err := tx.Exec(ctx, `INSERT INTO flood_areas (id, kind, source, severity, issued_at, valid_from, valid_until,
					properties, geom, fetched_at, first_seen_at)
				SELECT 'seed-' || e.id, $1, $2, $3, $4, $4, $5, '{"seed":true}'::jsonb, ST_Transform(e.geom_simple_3857, 4326), $4, $4
				FROM past_flood_events e WHERE e.id = $6
				ON CONFLICT (id) DO NOTHING`, area.kind, Source, area.severity, now, now.Add(48*time.Hour), area.event); err != nil {
				return err
			}
		}
		return nil
	})
}

// Polygon is a real serialized polygon to draw as if it were Sri Lanka's, from a
// local manifest. Real Google polygons are never committed: Google's terms limit
// how long they are kept, so a manifest is fetched for the test and thrown away.
type Polygon struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Severity string `json:"severity"`
	// Raw is the response of serializedPolygons/{id}.
	Raw json.RawMessage `json:"raw"`
	// ShiftLon and ShiftLat move the polygon, to see it over Sri Lanka.
	ShiftLon float64 `json:"shiftLon"`
	ShiftLat float64 `json:"shiftLat"`
}

// Polygons writes polygons through the decoder production uses, so they are drawn
// exactly as they would be. It keeps whatever else is seeded.
func Polygons(ctx context.Context, pool *pgxpool.Pool, polygons []Polygon, now time.Time) error {
	if err := guard(ctx, pool); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		for _, polygon := range polygons {
			shape, err := googleflood.ParsePolygon(polygon.Raw)
			if err != nil {
				return fmt.Errorf("polygon %s: %w", polygon.ID, err)
			}
			if shape, err = shifted(shape, polygon.ShiftLon, polygon.ShiftLat); err != nil {
				return fmt.Errorf("polygon %s: %w", polygon.ID, err)
			}
			until := now.Add(24 * time.Hour)
			if err := queries.InsertFloodArea(ctx, store.InsertFloodAreaParams{ID: "seed-" + polygon.ID, Kind: polygon.Kind, Source: Source,
				Severity: polygon.Severity, IssuedAt: &now, ValidFrom: &now, ValidUntil: &until,
				Properties: []byte(`{"seed":true,"forecastPeriodHours":24}`), Geojson: string(shape), FetchedAt: now, FirstSeenAt: now}); err != nil {
				return fmt.Errorf("store polygon %s: %w", polygon.ID, err)
			}
		}
		return nil
	})
}

func shifted(shape []byte, lon, lat float64) ([]byte, error) {
	var geometry struct {
		Type        string           `json:"type"`
		Coordinates [][][][2]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal(shape, &geometry); err != nil {
		return nil, err
	}
	for _, polygon := range geometry.Coordinates {
		for _, ring := range polygon {
			for index := range ring {
				ring[index][0] += lon
				ring[index][1] += lat
			}
		}
	}
	return json.Marshal(geometry)
}

// Clear removes what Apply wrote. Raised Google gauges go back to "no flooding"
// until the next collection replaces them with what Google really says.
func Clear(ctx context.Context, pool *pgxpool.Pool) error {
	if err := guard(ctx, pool); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		statements := []struct {
			sql  string
			args []any
		}{
			{`DELETE FROM river_readings WHERE fetched_at = $1`, []any{marker}},
			{`UPDATE river_stations SET source = 'irrigation-gauges' WHERE source = $1`, []any{Source}},
			{`DELETE FROM flood_gauges WHERE source = $1 AND NOT quality_verified`, []any{Source}},
			{`UPDATE flood_gauges SET source = 'google-flood-forecasting', severity = 'no_flooding', trend = 'unknown' WHERE source = $1`, []any{Source}},
			{`DELETE FROM flood_areas WHERE source = $1`, []any{Source}},
		}
		for _, statement := range statements {
			if _, err := tx.Exec(ctx, statement.sql, statement.args...); err != nil {
				return errors.Join(fmt.Errorf("clear seed: %s", statement.sql), err)
			}
		}
		return nil
	})
}
