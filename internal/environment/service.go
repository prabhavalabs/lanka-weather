// Package environment serves air quality and sea conditions: model values on
// coarse grids, stored by the openmeteo environment collectors. It never calls
// a source for a request (ADR 0012).
package environment

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/store"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// recheck is how often the newest run is looked for; a new one is served
// within this time of being stored.
const recheck = 30 * time.Second

const (
	// coastalReach is how far inland, in degrees (about 16 km), a place still
	// has the sea state of its coast.
	coastalReach = 0.15
	// seaNodeKm bounds how far off the sea node read may be: on a 0.5° grid the
	// nearest one to a coast can be this far, across a cell and a headland.
	seaNodeKm = 75.0
)

// Service reads the two grids.
type Service struct {
	Pool *pgxpool.Pool
	// AirStaleAfter and SeaStaleAfter are the ages after which values are
	// marked stale.
	AirStaleAfter time.Duration
	SeaStaleAfter time.Duration
	Now           func() time.Time

	air, sea cache
}

type cache struct {
	mutex     sync.Mutex
	run       *run
	checkedAt time.Time
}

type run struct {
	id                int64
	source, model     string
	sourceTime        time.Time
	fetchedAt         time.Time
	lattice           grid.Lattice
	gridID            string
	nodes             []grid.Node
	fields            map[string][]float32
	once              sync.Once
	plain, compressed []byte
	encodeErr         error
}

// Grid is GET /v1/air/current or /v1/sea/current: a grid response as
// /v1/weather/current gives, read by the same rendering code.
type Grid struct {
	Source      string                    `json:"source"`
	Model       string                    `json:"model"`
	Attribution string                    `json:"attribution"`
	ValidAt     time.Time                 `json:"validAt"`
	FetchedAt   time.Time                 `json:"fetchedAt"`
	Grid        weather.GridSpec          `json:"grid"`
	Units       map[string]string         `json:"units"`
	Values      map[string]weather.Values `json:"values"`
	// Land is Sri Lanka's land, simplified, for clipping a sea field at the
	// coast; absent for air quality.
	Land [][][2]float64 `json:"land,omitempty"`
	Note string         `json:"note"`
}

// Response names and units of the stored variables.
type variable struct{ stored, name, unit string }

var airVariables = []variable{
	{"us_aqi", "usAqi", ""},
	{"pm2_5", "pm25", "µg/m³"},
}

var seaVariables = []variable{
	{"wave_height", "waveHeightM", "m"},
	{"wave_direction", "waveDirectionDeg", "°"},
	{"wave_period", "wavePeriodS", "s"},
	{"swell_wave_height", "swellHeightM", "m"},
	{"swell_wave_direction", "swellDirectionDeg", "°"},
	{"swell_wave_period", "swellPeriodS", "s"},
}

const (
	airNote = "Model estimate from the Copernicus Atmosphere Monitoring Service, about 40 km across. It does not measure the air at any place."
	seaNote = "Model values on a 0.5° grid (about 55 km), not observations; they do not resolve harbours or the shore. Follow the Department of Meteorology's sea-area warnings."
)

// AirJSON returns the air-quality grid encoded once per run.
func (service *Service) AirJSON(ctx context.Context) (plain, gzipped []byte, err error) {
	return service.encoded(ctx, &service.air, openmeteo.KindAirQuality, airVariables, openmeteo.AttributionAirQuality, airNote, nil)
}

// SeaJSON returns the sea grid encoded once per run.
func (service *Service) SeaJSON(ctx context.Context) (plain, gzipped []byte, err error) {
	return service.encoded(ctx, &service.sea, openmeteo.KindMarine, seaVariables, openmeteo.AttributionMarine, seaNote, grid.Land())
}

func (service *Service) encoded(ctx context.Context, into *cache, kind openmeteo.Kind, variables []variable, attribution, note string, land [][][2]float64) ([]byte, []byte, error) {
	current, err := service.load(ctx, into, kind)
	if err != nil {
		return nil, nil, err
	}
	current.once.Do(func() {
		body, err := json.Marshal(BuildGrid(current, variables, attribution, note, land))
		if err != nil {
			current.encodeErr = err
			return
		}
		current.plain, current.compressed = body, compress(body)
	})
	return current.plain, current.compressed, current.encodeErr
}

// BuildGrid assembles a grid response.
func BuildGrid(current *run, variables []variable, attribution, note string, land [][][2]float64) Grid {
	response := Grid{
		Source: current.source, Model: current.model, Attribution: attribution, ValidAt: current.sourceTime, FetchedAt: current.fetchedAt,
		Grid: weather.GridSpec{ID: current.gridID, Lon0: current.lattice.Lon0, Lat0: current.lattice.Lat0, Step: current.lattice.Step,
			Nx: current.lattice.Nx, Ny: current.lattice.Ny},
		Units: map[string]string{}, Values: map[string]weather.Values{}, Land: land, Note: note,
	}
	for _, item := range variables {
		field := current.fields[item.stored]
		if field == nil {
			continue
		}
		values := make(weather.Values, len(field))
		for index, value := range field {
			values[index] = weather.Value(value)
		}
		response.Values[item.name] = values
		response.Units[item.name] = item.unit
	}
	return response
}

// load returns the newest run of a kind.
func (service *Service) load(ctx context.Context, into *cache, kind openmeteo.Kind) (*run, error) {
	into.mutex.Lock()
	defer into.mutex.Unlock()
	if into.run != nil && time.Since(into.checkedAt) < recheck {
		return into.run, nil
	}
	queries := store.New(service.Pool)
	stored, err := queries.LatestWeatherRun(ctx, string(kind))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, weather.ErrNoData
	}
	if err != nil {
		return nil, fmt.Errorf("read %s run: %w", kind, err)
	}
	into.checkedAt = time.Now()
	if into.run != nil && into.run.id == stored.ID {
		return into.run, nil
	}
	rows, err := queries.WeatherRunFields(ctx, stored.ID)
	if err != nil {
		return nil, fmt.Errorf("read %s fields: %w", kind, err)
	}
	built := Grids[stored.Grid]
	if built.Size() == 0 {
		return nil, fmt.Errorf("run %d is on unknown grid %q", stored.ID, stored.Grid)
	}
	current := &run{id: stored.ID, source: stored.Source, model: stored.Model, sourceTime: stored.SourceTime.UTC(), fetchedAt: stored.FetchedAt.UTC(),
		lattice: built.Lattice, gridID: stored.Grid, nodes: built.Nodes, fields: map[string][]float32{}}
	for _, row := range rows {
		if len(row.NodeValues) != len(stored.Nodes) {
			return nil, fmt.Errorf("field %s of run %d has %d values for %d nodes", row.Variable, stored.ID, len(row.NodeValues), len(stored.Nodes))
		}
		values := make([]float32, built.Size())
		for index := range values {
			values[index] = float32(math.NaN())
		}
		for position, node := range stored.Nodes {
			if int(node) < len(values) {
				values[node] = row.NodeValues[position]
			}
		}
		current.fields[row.Variable] = values
	}
	into.run = current
	return current, nil
}

// Grids are the grids runs may be stored on.
var Grids = map[string]grid.Grid{grid.AirQualityID: grid.AirQuality(), grid.SeaID: grid.Sea()}

// AirPoint is air quality at a place.
type AirPoint struct {
	// USAQI is the US Air Quality Index, Category its band.
	USAQI       float64   `json:"usAqi"`
	Category    string    `json:"category"`
	PM25        *float64  `json:"pm25"`
	ValidAt     time.Time `json:"validAt"`
	FetchedAt   time.Time `json:"fetchedAt"`
	Stale       bool      `json:"stale"`
	Estimate    bool      `json:"estimate"`
	Attribution string    `json:"attribution"`
}

// AQI bands, as the US EPA names them.
const (
	AQIGood               = "good"
	AQIModerate           = "moderate"
	AQIUnhealthySensitive = "unhealthy_sensitive"
	AQIUnhealthy          = "unhealthy"
	AQIVeryUnhealthy      = "very_unhealthy"
	AQIHazardous          = "hazardous"
)

// AQICategory is the US EPA band of an index value.
func AQICategory(index float64) string {
	switch {
	case index <= 50:
		return AQIGood
	case index <= 100:
		return AQIModerate
	case index <= 150:
		return AQIUnhealthySensitive
	case index <= 200:
		return AQIUnhealthy
	case index <= 300:
		return AQIVeryUnhealthy
	default:
		return AQIHazardous
	}
}

// AirAt interpolates air quality at a place; nil outside the grid.
func (service *Service) AirAt(ctx context.Context, lon, lat float64) (*AirPoint, error) {
	current, err := service.load(ctx, &service.air, openmeteo.KindAirQuality)
	if errors.Is(err, weather.ErrNoData) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	corners, inside := current.lattice.Corners(lon, lat)
	if !inside {
		return nil, nil
	}
	index, ok := grid.Sample(current.fields["us_aqi"], corners)
	if !ok {
		return nil, nil
	}
	point := &AirPoint{USAQI: math.Round(index), Category: AQICategory(math.Round(index)), ValidAt: current.sourceTime, FetchedAt: current.fetchedAt,
		Stale: service.now().Sub(current.fetchedAt) > service.AirStaleAfter, Estimate: true, Attribution: openmeteo.AttributionAirQuality}
	if pm, ok := grid.Sample(current.fields["pm2_5"], corners); ok {
		rounded := math.Round(pm*10) / 10
		point.PM25 = &rounded
	}
	return point, nil
}

// SeaPoint is the sea state at or off a place.
type SeaPoint struct {
	WaveHeightM       float64  `json:"waveHeightM"`
	WavePeriodS       *float64 `json:"wavePeriodS"`
	WaveDirectionDeg  *float64 `json:"waveDirectionDeg"`
	SwellHeightM      *float64 `json:"swellHeightM"`
	SwellPeriodS      *float64 `json:"swellPeriodS"`
	SwellDirectionDeg *float64 `json:"swellDirectionDeg"`
	// State is the WMO sea-state word for the wave height.
	State string `json:"state"`
	// DistanceKm is how far the grid node read is from the place.
	DistanceKm  float64   `json:"distanceKm"`
	ValidAt     time.Time `json:"validAt"`
	FetchedAt   time.Time `json:"fetchedAt"`
	Stale       bool      `json:"stale"`
	Estimate    bool      `json:"estimate"`
	Attribution string    `json:"attribution"`
	Note        string    `json:"note"`
}

// SeaState is the WMO sea state code's word for a significant wave height.
func SeaState(heightM float64) string {
	switch {
	case heightM < 0.1:
		return "calm"
	case heightM < 0.5:
		return "smooth"
	case heightM < 1.25:
		return "slight"
	case heightM < 2.5:
		return "moderate"
	case heightM < 4:
		return "rough"
	case heightM < 6:
		return "very rough"
	case heightM < 9:
		return "high"
	default:
		return "very high"
	}
}

// SeaAt gives the sea state of the nearest sea node, for a place on the water
// or within about 16 km of it; nil further inland.
func (service *Service) SeaAt(ctx context.Context, lon, lat float64) (*SeaPoint, error) {
	if !grid.NearSea(lon, lat, coastalReach) {
		return nil, nil
	}
	current, err := service.load(ctx, &service.sea, openmeteo.KindMarine)
	if errors.Is(err, weather.ErrNoData) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	heights := current.fields["wave_height"]
	best, bestKm := -1, math.Inf(1)
	for _, node := range current.nodes {
		if node.Index >= len(heights) || math.IsNaN(float64(heights[node.Index])) {
			continue
		}
		if distance := distanceKm(lon, lat, node.Lon, node.Lat); distance < bestKm {
			best, bestKm = node.Index, distance
		}
	}
	if best < 0 || bestKm > seaNodeKm {
		return nil, nil
	}
	value := func(name string) *float64 {
		field := current.fields[name]
		if best >= len(field) || math.IsNaN(float64(field[best])) {
			return nil
		}
		rounded := math.Round(float64(field[best])*10) / 10
		return &rounded
	}
	height := math.Round(float64(heights[best])*10) / 10
	return &SeaPoint{WaveHeightM: height, WavePeriodS: value("wave_period"), WaveDirectionDeg: value("wave_direction"),
		SwellHeightM: value("swell_wave_height"), SwellPeriodS: value("swell_wave_period"), SwellDirectionDeg: value("swell_wave_direction"),
		State: SeaState(height), DistanceKm: math.Round(bestKm*10) / 10, ValidAt: current.sourceTime, FetchedAt: current.fetchedAt,
		Stale: service.now().Sub(current.fetchedAt) > service.SeaStaleAfter, Estimate: true, Attribution: openmeteo.AttributionMarine, Note: seaNote}, nil
}

func distanceKm(lon1, lat1, lon2, lat2 float64) float64 {
	const earthRadiusKm = 6371.0088
	toRadians := math.Pi / 180
	dLat, dLon := (lat2-lat1)*toRadians, (lon2-lon1)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

func compress(body []byte) []byte {
	var out bytes.Buffer
	writer, _ := gzip.NewWriterLevel(&out, gzip.BestCompression)
	_, _ = writer.Write(body)
	_ = writer.Close()
	return out.Bytes()
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
