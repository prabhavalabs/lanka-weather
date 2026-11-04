package weather

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
)

// ErrRegionsUnavailable means no administrative regions are loaded yet; the
// API answers 503 not_available.
var ErrRegionsUnavailable = errors.New("administrative regions are not available")

// ErrUnknownLevel is an unknown region level.
var ErrUnknownLevel = errors.New("unknown region level")

// Levels are the administrative levels regions are served at.
var Levels = []string{"province", "district", "ds", "gn"}

// ValidLevel reports whether a level is one of Levels.
func ValidLevel(level string) bool {
	for _, known := range Levels {
		if known == level {
			return true
		}
	}
	return false
}

// Region is an administrative region as the weather aggregation needs it.
type Region struct {
	ID   string
	Name string
	// ProvinceID, DistrictID and DSID are the regions containing this one,
	// or the region itself at its own level; empty for levels below it.
	ProvinceID, DistrictID, DSID string
	// Lon and Lat are a point inside the region, used for regions too small
	// to aggregate cells over.
	Lon, Lat float64
	// Nodes are the lattice indexes of the grid nodes whose cells intersect
	// the region, and Weights the share of each cell inside it (0 to 1).
	// Both are empty for a region smaller than one cell, which takes the
	// value interpolated at its point instead.
	Nodes   []int
	Weights []float64
}

// Within reports whether the region lies in the region with the given id,
// which must be at a level above the region's own.
func (region Region) Within(ancestor string) bool {
	return ancestor != "" && (ancestor == region.ProvinceID || ancestor == region.DistrictID || ancestor == region.DSID)
}

// RegionSource yields the regions of one level with their grid nodes.
type RegionSource interface {
	Regions(ctx context.Context, level string) ([]Region, error)
}

// UnavailableRegions is the RegionSource when regions are switched off
// (LANKAWEATHER_WEATHER_REGIONS=none): every call fails with
// ErrRegionsUnavailable.
type UnavailableRegions struct{}

// Regions implements RegionSource.
func (UnavailableRegions) Regions(context.Context, string) ([]Region, error) {
	return nil, ErrRegionsUnavailable
}

// DefaultRegionsRelation is the view over the regions table that migration
// 00004 creates for PostGISRegions.
const DefaultRegionsRelation = "admin_regions"

// PostGISRegions reads regions from a PostGIS relation, the admin_regions
// view by default, and joins them to the weather grid's node cells
// (docs/api/weather-and-floods.md, "Regions"). The relation has the columns
// id, level (province, district, ds or gn), name, province_id, district_id,
// ds_id (text), area_km2 (double precision), label_point (a point inside the
// region) and geom (geometry(MultiPolygon or Polygon, 4326), GiST-indexed).
//
// The join is computed once per level and kept for regionsTTL: boundaries
// change a few times a year, so the weights are effectively precomputed.
// Call Warm at startup so no request waits for it.
type PostGISRegions struct {
	Pool *pgxpool.Pool
	// Relation is the table or view, DefaultRegionsRelation when empty.
	Relation string
	Grid     grid.Grid

	mutex  sync.Mutex
	levels map[string]*levelRegions
}

// levelRegions holds one level's regions. Its own mutex lets levels load
// concurrently while requests for the same level wait for one load.
type levelRegions struct {
	mutex   sync.Mutex
	regions []Region
	at      time.Time
}

// regionsTTL is how long a level's regions are kept: boundaries change
// rarely, and a reload is picked up within this time.
const regionsTTL = 6 * time.Hour

// Regions implements RegionSource. Regions at the gn level, and regions
// smaller than one grid cell, are given no nodes and take the value at
// their point. Without the relation, or before any region of the level is
// loaded, it returns ErrRegionsUnavailable; before the weather grid's cells
// are stored, ErrNoData. Either is asked again next time.
func (source *PostGISRegions) Regions(ctx context.Context, level string) ([]Region, error) {
	if !ValidLevel(level) {
		return nil, ErrUnknownLevel
	}
	source.mutex.Lock()
	if source.levels == nil {
		source.levels = map[string]*levelRegions{}
	}
	entry := source.levels[level]
	if entry == nil {
		entry = &levelRegions{}
		source.levels[level] = entry
	}
	source.mutex.Unlock()

	entry.mutex.Lock()
	defer entry.mutex.Unlock()
	if entry.regions != nil && time.Since(entry.at) < regionsTTL {
		return entry.regions, nil
	}
	regions, err := source.load(ctx, level)
	if err != nil {
		return nil, err
	}
	entry.regions, entry.at = regions, time.Now()
	return regions, nil
}

// Warm loads every level, so the first requests do not wait for the joins.
// Levels without regions or grid cells yet are skipped quietly.
func (source *PostGISRegions) Warm(ctx context.Context) error {
	var errs []error
	for _, level := range Levels {
		if _, err := source.Regions(ctx, level); err != nil && !errors.Is(err, ErrRegionsUnavailable) && !errors.Is(err, ErrNoData) {
			errs = append(errs, fmt.Errorf("%s: %w", level, err))
		}
	}
	return errors.Join(errs...)
}

// cellLevelQuery joins regions to the cells they intersect, with the share
// of each cell inside the region. Regions smaller than a cell
// (@min_area_m2) get no cells.
const cellLevelQuery = `
WITH r AS (
    SELECT id::text AS id, name::text AS name, province_id::text AS province_id,
           district_id::text AS district_id, ds_id::text AS ds_id, geom,
           ST_X(label_point)::float8 AS lon, ST_Y(label_point)::float8 AS lat,
           area_km2 * 1e6 >= @min_area_m2::float8 AS by_cells
    FROM %s
    WHERE level = @level::text
)
SELECT r.id, r.name, COALESCE(r.province_id, ''), COALESCE(r.district_id, ''), COALESCE(r.ds_id, ''), r.lon, r.lat,
       COALESCE(array_agg(n.node ORDER BY n.node) FILTER (WHERE n.node IS NOT NULL), '{}')::integer[],
       COALESCE(array_agg(ST_Area(ST_Intersection(n.cell, r.geom)) / ST_Area(n.cell) ORDER BY n.node)
                FILTER (WHERE n.node IS NOT NULL), '{}')::float8[]
FROM r
LEFT JOIN weather_grid_nodes AS n ON r.by_cells AND n.grid = @grid::text AND ST_Intersects(n.cell, r.geom)
GROUP BY r.id, r.name, r.province_id, r.district_id, r.ds_id, r.lon, r.lat
ORDER BY r.id`

// pointLevelQuery reads regions that take the value at their point: GN
// divisions, which are a few square kilometres against a cell's ~490.
const pointLevelQuery = `
SELECT id::text, name::text, COALESCE(province_id::text, ''), COALESCE(district_id::text, ''), COALESCE(ds_id::text, ''),
       ST_X(label_point)::float8, ST_Y(label_point)::float8, '{}'::integer[], '{}'::float8[]
FROM %s
WHERE level = @level::text
ORDER BY id`

func (source *PostGISRegions) load(ctx context.Context, level string) ([]Region, error) {
	relation := source.Relation
	if relation == "" {
		relation = DefaultRegionsRelation
	}
	var exists bool
	if err := source.Pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, relation).Scan(&exists); err != nil {
		return nil, fmt.Errorf("look up %s: %w", relation, err)
	}
	if !exists {
		return nil, ErrRegionsUnavailable
	}
	query := cellLevelQuery
	if level == "gn" {
		query = pointLevelQuery
	}
	// A cell is about 22 km square on the 0.2° grid; smaller regions are
	// better served by the value at their point than by a whole cell.
	rows, err := source.Pool.Query(ctx, fmt.Sprintf(query, pgx.Identifier{relation}.Sanitize()), pgx.NamedArgs{
		"level": level, "grid": source.Grid.ID, "min_area_m2": cellAreaM2(source.Grid.Step, 7.8),
	})
	if err != nil {
		return nil, fmt.Errorf("read %s regions: %w", level, err)
	}
	defer rows.Close()
	regions := []Region{}
	withCells := 0
	for rows.Next() {
		var region Region
		var nodes []int32
		if err := rows.Scan(&region.ID, &region.Name, &region.ProvinceID, &region.DistrictID, &region.DSID,
			&region.Lon, &region.Lat, &nodes, &region.Weights); err != nil {
			return nil, err
		}
		for _, node := range nodes {
			region.Nodes = append(region.Nodes, int(node))
		}
		if len(region.Weights) == 0 {
			region.Weights = nil
		} else {
			withCells++
		}
		regions = append(regions, region)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read %s regions: %w", level, err)
	}
	if len(regions) == 0 {
		return nil, ErrRegionsUnavailable
	}
	if query == cellLevelQuery && withCells == 0 {
		// Before the grid's cells are stored every region would fall back
		// to its point, and stay so until the cache expired.
		var nodes int
		if err := source.Pool.QueryRow(ctx, `SELECT count(*) FROM weather_grid_nodes WHERE grid = $1`, source.Grid.ID).Scan(&nodes); err != nil {
			return nil, fmt.Errorf("count grid nodes: %w", err)
		}
		if nodes == 0 {
			return nil, ErrNoData
		}
	}
	return regions, nil
}

// cellAreaM2 is the area of a grid cell of `step` degrees at a latitude.
func cellAreaM2(step, lat float64) float64 {
	const metresPerDegree = 111_320.0
	return step * metresPerDegree * step * metresPerDegree * math.Cos(lat*math.Pi/180)
}

// RegionWeather is one region's current weather and today's forecast range.
type RegionWeather struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Method is "cells" (aggregated over the cells the region intersects)
	// or "point" (interpolated at a point inside a small region).
	Method string `json:"method"`
	Cells  int    `json:"cells"`
	// TemperatureC is the current temperature: area-weighted mean.
	TemperatureC Value `json:"temperatureC"`
	// TemperatureMinC and TemperatureMaxC are today's forecast low and high
	// (local day): the lowest minimum and highest maximum over the cells.
	TemperatureMinC Value `json:"temperatureMinC"`
	TemperatureMaxC Value `json:"temperatureMaxC"`
	// PrecipitationMm is the current hour's precipitation (mean) and
	// PrecipitationTodayMm today's forecast total (mean).
	PrecipitationMm      Value `json:"precipitationMm"`
	PrecipitationTodayMm Value `json:"precipitationTodayMm"`
	// WindSpeedMaxKmh and WindGustsMaxKmh are the highest current values
	// over the cells.
	WindSpeedMaxKmh Value `json:"windSpeedMaxKmh"`
	WindGustsMaxKmh Value `json:"windGustsMaxKmh"`

	// region keeps the ancestors, to limit a list to one parent.
	region Region
}

// RegionsResponse is GET /v1/weather/regions.
type RegionsResponse struct {
	Level string `json:"level"`
	// Parent is the region the list is limited to, when one was asked for.
	Parent            string          `json:"parent,omitempty"`
	Regions           []RegionWeather `json:"regions"`
	ValidAt           *time.Time      `json:"validAt"`
	FetchedAt         *time.Time      `json:"fetchedAt"`
	ForecastFetchedAt *time.Time      `json:"forecastFetchedAt"`
	Source            string          `json:"source"`
	Attribution       string          `json:"attribution"`
}

// minimumShare is the smallest share of a cell inside a region for the cell
// to count towards the region's lowest and highest values; slivers along a
// boundary still count towards the mean, by their share.
const minimumShare = 0.1

type regionResult struct {
	response *RegionsResponse
}

// RegionWeather aggregates the snapshot over a level's regions, once per
// snapshot and level. A parent limits the list to the regions within it
// (its level must be above level; the caller checks).
func (service *Service) RegionWeather(ctx context.Context, level, parent string) (*RegionsResponse, error) {
	if !ValidLevel(level) {
		return nil, ErrUnknownLevel
	}
	if service.Regions == nil {
		return nil, ErrRegionsUnavailable
	}
	// Regions first: without them the answer is not_available whether or
	// not weather has been collected.
	regions, err := service.Regions.Regions(ctx, level)
	if err != nil {
		return nil, err
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	// The aggregate is kept with the snapshot: regions reloaded during an
	// hour show from the next snapshot.
	var response *RegionsResponse
	if cached, ok := snapshot.regions.Load(level); ok {
		response = cached.(*regionResult).response
	} else {
		response = AggregateRegions(snapshot, level, regions, service.now())
		snapshot.regions.Store(level, &regionResult{response: response})
	}
	if parent == "" {
		return response, nil
	}
	return WithinParent(response, parent), nil
}

// WithinParent returns a copy of an aggregate limited to the regions within
// parent.
func WithinParent(response *RegionsResponse, parent string) *RegionsResponse {
	limited := *response
	limited.Parent = parent
	limited.Regions = []RegionWeather{}
	for _, region := range response.Regions {
		if region.region.Within(parent) {
			limited.Regions = append(limited.Regions, region)
		}
	}
	return &limited
}

// AggregateRegions computes every region's values from a snapshot.
func AggregateRegions(snapshot *Snapshot, level string, regions []Region, now time.Time) *RegionsResponse {
	response := &RegionsResponse{Level: level, Regions: make([]RegionWeather, 0, len(regions)),
		Source: openmeteo.SourceName, Attribution: openmeteo.Attribution}
	var temperature, precipitation, wind, gusts []float32
	if current := snapshot.Current; current != nil {
		validAt, fetchedAt := current.SourceTime, current.FetchedAt
		response.ValidAt, response.FetchedAt = &validAt, &fetchedAt
		temperature = fieldAt(current, "temperature_2m", 0)
		precipitation = fieldAt(current, "precipitation", 0)
		wind = fieldAt(current, "wind_speed_10m", 0)
		gusts = fieldAt(current, "wind_gusts_10m", 0)
	}
	var low, high, rain []float32
	if forecast := snapshot.Forecast; forecast != nil {
		fetchedAt := forecast.FetchedAt
		response.ForecastFetchedAt = &fetchedAt
		if field := forecast.Fields[openmeteo.DailyPrefix+"temperature_2m_min"]; field != nil {
			if day := int(openmeteo.LocalDayStart(now).Sub(field.ValidFrom) / (24 * time.Hour)); day >= 0 && day < field.Steps {
				low = fieldAt(forecast, openmeteo.DailyPrefix+"temperature_2m_min", day)
				high = fieldAt(forecast, openmeteo.DailyPrefix+"temperature_2m_max", day)
				rain = fieldAt(forecast, openmeteo.DailyPrefix+"precipitation_sum", day)
			}
		}
	}
	for _, region := range regions {
		result := RegionWeather{ID: region.ID, Name: region.Name, region: Region{
			ProvinceID: region.ProvinceID, DistrictID: region.DistrictID, DSID: region.DSID}}
		nodes, weights := region.Nodes, region.Weights
		if len(nodes) == 0 || len(weights) != len(nodes) {
			around, ok := corners(snapshot.Lattice, region.Lon, region.Lat)
			if !ok {
				result.Method = "point"
				nan := Value(math.NaN())
				result.TemperatureC, result.TemperatureMinC, result.TemperatureMaxC = nan, nan, nan
				result.PrecipitationMm, result.PrecipitationTodayMm, result.WindSpeedMaxKmh, result.WindGustsMaxKmh = nan, nan, nan, nan
				response.Regions = append(response.Regions, result)
				continue
			}
			nodes, weights = nil, nil
			for _, corner := range around {
				if corner.Weight > 0 {
					nodes, weights = append(nodes, corner.Index), append(weights, corner.Weight)
				}
			}
			result.Method = "point"
		} else {
			result.Method, result.Cells = "cells", len(nodes)
		}
		result.TemperatureC = weightedMean(temperature, nodes, weights)
		result.PrecipitationMm = weightedMean(precipitation, nodes, weights)
		result.PrecipitationTodayMm = weightedMean(rain, nodes, weights)
		if result.Method == "cells" {
			result.TemperatureMinC = extreme(low, nodes, weights, math.Min)
			result.TemperatureMaxC = extreme(high, nodes, weights, math.Max)
			result.WindSpeedMaxKmh = extreme(wind, nodes, weights, math.Max)
			result.WindGustsMaxKmh = extreme(gusts, nodes, weights, math.Max)
		} else {
			// At a point, the interpolated value is its own extreme.
			result.TemperatureMinC = weightedMean(low, nodes, weights)
			result.TemperatureMaxC = weightedMean(high, nodes, weights)
			result.WindSpeedMaxKmh = weightedMean(wind, nodes, weights)
			result.WindGustsMaxKmh = weightedMean(gusts, nodes, weights)
		}
		response.Regions = append(response.Regions, result)
	}
	return response
}

func fieldAt(run *Run, name string, step int) []float32 {
	field := run.Fields[name]
	if field == nil || step >= field.Steps {
		return nil
	}
	return field.At(step)
}

func weightedMean(values []float32, nodes []int, weights []float64) Value {
	var sum, total float64
	for index, node := range nodes {
		if node >= len(values) || math.IsNaN(float64(values[node])) || weights[index] <= 0 {
			continue
		}
		sum += float64(values[node]) * weights[index]
		total += weights[index]
	}
	if total == 0 {
		return Value(math.NaN())
	}
	return Value(sum / total)
}

// extreme is the lowest or highest value over cells with at least
// minimumShare inside the region, or over all of them if none has.
func extreme(values []float32, nodes []int, weights []float64, pick func(a, b float64) float64) Value {
	best, found := math.NaN(), false
	for _, threshold := range []float64{minimumShare, 0} {
		for index, node := range nodes {
			if node >= len(values) || weights[index] <= threshold-1e-12 || weights[index] <= 0 {
				continue
			}
			value := float64(values[node])
			if math.IsNaN(value) {
				continue
			}
			if !found {
				best, found = value, true
				continue
			}
			best = pick(best, value)
		}
		if found {
			break
		}
	}
	return Value(best)
}
