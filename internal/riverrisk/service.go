package riverrisk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/hydrobasins"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/hydrorivers"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigationgis"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivernet"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/store"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// ErrNotFound is returned for an unknown river or basin.
var ErrNotFound = errors.New("not found")

// SeedSource marks rows written by the local test seed (package riverseed).
// While any exist, every response says so, so a seeded flood can never pass as
// a real one.
const SeedSource = floods.SeedSource

const (
	// recompute is how long an assessment is reused. Gauges are polled every
	// 15 minutes and nothing else moves faster, so a minute costs nothing and
	// keeps this cheap under load.
	recompute = time.Minute
	// rainStaleAfter matches the point forecast's limit.
	rainStaleAfter = 12 * time.Hour
	// nearDegrees is how far /v1/summary looks for rivers, about 17 km.
	nearDegrees = 0.15
	maxNear     = 3
	historyHrs  = 72
	tileCache   = 48 << 20
)

// Note is said with every estimate.
const Note = "Estimate from gauge readings, Google flood forecasts, forecast rain and reported reservoir spills. Thresholds are the Irrigation Department's. Follow official warnings from the Irrigation Department, the Department of Meteorology and the Disaster Management Centre."

// Attribution credits what the estimate and the map are built from.
const Attribution = irrigationgis.Attribution + "; " + hydrorivers.Attribution + "; " + hydrobasins.Attribution + "; " + googleflood.Attribution + "; weather data by Open-Meteo.com (CC BY 4.0); reservoir figures from " + reservoirs.Attribution

// RiverSource serves gauge readings.
type RiverSource interface {
	Layer(ctx context.Context) (rivers.FeatureCollection, error)
	StationHistory(ctx context.Context, id string, period time.Duration) (rivers.History, error)
}

// FloodSource serves Google's forecasts, no older than its terms allow.
type FloodSource interface {
	Gauges(ctx context.Context) ([]floods.Gauge, error)
}

// WeatherSource serves the forecast grid.
type WeatherSource interface {
	Snapshot(ctx context.Context) (*weather.Snapshot, error)
}

// ReservoirSource serves the latest reservoir readings.
type ReservoirSource interface {
	Layer(ctx context.Context) (reservoirs.FeatureCollection, error)
}

// Service computes the risk documents at request time from fresh tables.
type Service struct {
	Pool    *pgxpool.Pool
	Rivers  RiverSource
	Floods  FloodSource
	Weather WeatherSource
	// Reservoirs adds spills upstream; without it, or when it fails, a stretch
	// simply has no such evidence.
	Reservoirs ReservoirSource
	Logger     *slog.Logger
	Now        func() time.Time

	mu       sync.Mutex
	network  *network
	links    *linkState
	current  *assessed
	computed time.Time

	tiles *regions.ByteCache
	once  sync.Once
}

// network is what the rebuildable tables hold; reloaded when their version moves.
type network struct {
	version   string
	builtAt   time.Time
	rivers    []store.ListRiversRow
	reaches   []store.ListRiverReachesRow
	gauges    map[int64][]store.RiverReachGauge
	topology  []store.ListSegmentTopologyRow
	basins    []store.ListOfficialBasinsRow
	subbasins []store.ListSubbasinsRow
	stationAt map[string]store.GaugeBasinsAtPointsRow
	googleAt  map[string]store.GaugeBasinsAtPointsRow
	stations  int
	google    int
}

type assessed struct {
	risk   *RiskDocument
	basins *BasinsDocument
	reach  map[int64]*ReachRisk
	river  map[string]*RiverRisk
}

// RiskDocument is GET /v1/rivers/risk.
type RiskDocument struct {
	// Estimate is always true: this is lanka-weather's estimate, not a warning.
	Estimate bool        `json:"estimate"`
	Note     string      `json:"note"`
	Rivers   []RiverRisk `json:"rivers"`
	Meta     RiskMeta    `json:"meta"`
}

// RiskMeta says how fresh the inputs are.
type RiskMeta struct {
	GeneratedAt time.Time `json:"generatedAt"`
	// NetworkVersion changes with the tiles' reach ids; clients fetch tiles
	// with it, so a rebuild can never mix old lines with new ids.
	NetworkVersion string     `json:"networkVersion"`
	NetworkBuiltAt *time.Time `json:"networkBuiltAt"`
	// GaugesObservedAt is the newest reading; GaugesCurrent and GaugesTotal
	// count stations with a usable reading.
	GaugesObservedAt  *time.Time `json:"gaugesObservedAt"`
	GaugesCurrent     int        `json:"gaugesCurrent"`
	GaugesTotal       int        `json:"gaugesTotal"`
	ForecastIssuedAt  *time.Time `json:"forecastIssuedAt"`
	ForecastFetchedAt *time.Time `json:"forecastFetchedAt"`
	RainFetchedAt     *time.Time `json:"rainFetchedAt"`
	RainStale         bool       `json:"rainStale"`
	// ReservoirsFetchedAt is when the readings used were read; null when none
	// were available, so spills upstream are not known.
	ReservoirsFetchedAt *time.Time `json:"reservoirsFetchedAt"`
	// ReservoirLinks says how many reservoirs are placed on the network, and why
	// the others are not. Those count for nothing.
	ReservoirLinks LinkReport `json:"reservoirLinks"`
	Attribution    string     `json:"attribution"`
	// Seeded means local test data is in the tables; never true in production.
	Seeded bool `json:"seeded"`
}

// RiverRisk is one river's state: its worst stretch's level and reasons.
type RiverRisk struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	NameSI   *string     `json:"nameSi"`
	Aliases  []string    `json:"aliases"`
	Named    bool        `json:"named"`
	ParentID *string     `json:"parentId"`
	Level    Level       `json:"level"`
	Reasons  []Reason    `json:"reasons"`
	UplandKM float64     `json:"uplandKm2"`
	LengthKM float64     `json:"lengthKm"`
	BBox     [4]float64  `json:"bbox"`
	Reaches  []ReachRisk `json:"reaches"`
	// GaugedReaches of Reaches have a gauge; the rest are unassessed.
	GaugedReaches int `json:"gaugedReaches"`
	// ObservedAt is its newest gauge reading.
	ObservedAt *time.Time `json:"observedAt"`
}

// ReachRisk is one stretch of river.
type ReachRisk struct {
	FID      int64        `json:"fid"`
	RiverID  string       `json:"riverId"`
	Name     string       `json:"name"`
	Level    Level        `json:"level"`
	Reasons  []Reason     `json:"reasons"`
	Gauged   bool         `json:"gauged"`
	LengthKM float64      `json:"lengthKm"`
	Gauges   []ReachGauge `json:"gauges"`
	Rain     *RainSummary `json:"rain"`
}

// ReachGauge is a gauge on a stretch, as far as the list needs.
type ReachGauge struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Name string `json:"name"`
	// Role is upstream, downstream or next; next is named but not used.
	Role       string     `json:"role"`
	WaterLevel *float64   `json:"waterLevel,omitempty"`
	Unit       string     `json:"unit,omitempty"`
	Status     string     `json:"status,omitempty"`
	Trend      string     `json:"trend,omitempty"`
	ObservedAt *time.Time `json:"observedAt,omitempty"`
	Stale      bool       `json:"stale"`
}

// RainSummary is forecast rain averaged over a stretch's catchment.
type RainSummary struct {
	Next24hMM float64 `json:"next24hMm"`
	Next72hMM float64 `json:"next72hMm"`
	// CatchmentKM2 is the catchment's area, for perspective.
	CatchmentKM2 float64 `json:"catchmentKm2"`
	Stale        bool    `json:"stale"`
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}

func (service *Service) warn(what string, err error) {
	if service.Logger != nil {
		service.Logger.Warn("river risk input failed", "input", what, "error", err)
	}
}

// loadNetwork reads the network once per version.
func (service *Service) loadNetwork(ctx context.Context, queries *store.Queries) (*network, error) {
	row, err := queries.GetSourceVersion(ctx, rivernet.SourceName)
	if errors.Is(err, pgx.ErrNoRows) {
		return &network{gauges: map[int64][]store.RiverReachGauge{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read network version: %w", err)
	}
	basinVersion, err := queries.GetSourceVersion(ctx, irrigationgis.BasinsSource)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("read basin version: %w", err)
	}
	version := row.Version + "-" + basinVersion.Version
	if service.network != nil && service.network.version == version {
		return service.network, nil
	}
	loaded := &network{version: version, builtAt: row.FetchedAt, gauges: map[int64][]store.RiverReachGauge{}}
	if loaded.rivers, err = queries.ListRivers(ctx); err != nil {
		return nil, fmt.Errorf("list rivers: %w", err)
	}
	if loaded.reaches, err = queries.ListRiverReaches(ctx); err != nil {
		return nil, fmt.Errorf("list reaches: %w", err)
	}
	gauges, err := queries.ListRiverReachGauges(ctx)
	if err != nil {
		return nil, fmt.Errorf("list reach gauges: %w", err)
	}
	for _, gauge := range gauges {
		loaded.gauges[gauge.ReachFid] = append(loaded.gauges[gauge.ReachFid], gauge)
	}
	if loaded.topology, err = queries.ListSegmentTopology(ctx); err != nil {
		return nil, fmt.Errorf("list segments: %w", err)
	}
	if loaded.basins, err = queries.ListOfficialBasins(ctx); err != nil {
		return nil, fmt.Errorf("list basins: %w", err)
	}
	if loaded.subbasins, err = queries.ListSubbasins(ctx); err != nil {
		return nil, fmt.Errorf("list sub-basins: %w", err)
	}
	service.network = loaded
	return loaded, nil
}

// compute produces the documents, reusing them for a minute.
func (service *Service) compute(ctx context.Context) (*assessed, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	now := service.now()
	if service.current != nil && now.Sub(service.computed) < recompute {
		return service.current, nil
	}
	queries := store.New(service.Pool)
	net, err := service.loadNetwork(ctx, queries)
	if err != nil {
		return nil, err
	}
	layer, err := service.Rivers.Layer(ctx)
	if err != nil {
		return nil, fmt.Errorf("read gauges: %w", err)
	}
	var google []floods.Gauge
	if service.Floods != nil {
		if google, err = service.Floods.Gauges(ctx); err != nil {
			// A forecast outage is treated as no forecast, and said so per stretch.
			service.warn("google forecasts", err)
			google = nil
		}
	}
	rain, rainMeta := service.catchmentRain(ctx, net, now)
	seeded, err := queries.SeedActive(ctx, SeedSource)
	if err != nil {
		return nil, fmt.Errorf("check for seeded data: %w", err)
	}
	spills, links, reservoirsAt := service.reservoirEvidence(ctx, queries, net)
	result := assemble(net, layer, google, rain, rainMeta, spills, now, seeded)
	result.risk.Meta.ReservoirLinks, result.risk.Meta.ReservoirsFetchedAt = links, reservoirsAt
	basins, err := service.buildBasins(ctx, queries, net, layer, google, now, seeded)
	if err != nil {
		return nil, err
	}
	result.basins = basins
	service.current, service.computed = result, now
	return result, nil
}

// reservoirEvidence reads each stretch's reservoirs. A failure leaves no such
// evidence and is logged: the estimate carries on with what it has.
func (service *Service) reservoirEvidence(ctx context.Context, queries *store.Queries, net *network) (map[int64][]Reservoir, LinkReport, *time.Time) {
	report := LinkReport{Unlinked: []UnlinkedReservoir{}}
	if service.Reservoirs == nil || len(net.topology) == 0 {
		return nil, report, nil
	}
	layer, err := service.Reservoirs.Layer(ctx)
	if err != nil {
		service.warn("reservoirs", err)
		return nil, report, nil
	}
	state, err := service.loadLinks(ctx, queries, net, layer)
	if err != nil {
		service.warn("reservoir links", err)
		return nil, report, nil
	}
	return reachReservoirs(state.links, layer), state.report, layer.Meta.FetchedAt
}

// rainMeta is when the rain forecast was fetched.
type rainMeta struct {
	fetchedAt *time.Time
	stale     bool
}

// catchmentRain totals forecast rain per segment's local catchment and
// accumulates it down the network, giving each segment the mean over everything
// draining through it. A failure leaves no rain: the stretches say nothing about
// rain rather than implying it is dry.
func (service *Service) catchmentRain(ctx context.Context, net *network, now time.Time) (map[int64]RainSummary, rainMeta) {
	if service.Weather == nil || len(net.topology) == 0 {
		return nil, rainMeta{}
	}
	snapshot, err := service.Weather.Snapshot(ctx)
	if err != nil {
		if !errors.Is(err, weather.ErrNoData) {
			service.warn("rain forecast", err)
		}
		return nil, rainMeta{}
	}
	if snapshot == nil || snapshot.Forecast == nil {
		return nil, rainMeta{}
	}
	field := snapshot.Forecast.Fields[openmeteo.HourlyPrefix+"precipitation"]
	if field == nil || field.Steps == 0 {
		return nil, rainMeta{}
	}
	fetched := snapshot.Forecast.FetchedAt
	meta := rainMeta{fetchedAt: &fetched, stale: now.Sub(fetched) > rainStaleAfter}
	next24, next72 := NodeTotals(field, now)
	if next24 == nil {
		return nil, meta
	}
	segments := make([]CatchmentSegment, len(net.topology))
	for index, row := range net.topology {
		segments[index] = CatchmentSegment{ID: row.HyrivID, NextDown: row.NextDown, CatchKM2: row.CatchKm2,
			Next24hMM: sampleLattice(snapshot.Lattice, next24, row.MidLon, row.MidLat),
			Next72hMM: sampleLattice(snapshot.Lattice, next72, row.MidLon, row.MidLat)}
	}
	summaries := AccumulateRain(segments)
	for id, summary := range summaries {
		summary.Stale = meta.stale
		summaries[id] = summary
	}
	return summaries, meta
}

func sampleLattice(lattice grid.Lattice, values []float32, lon, lat float64) float64 {
	corners, ok := lattice.Corners(lon, lat)
	if !ok {
		return math.NaN()
	}
	value, ok := grid.Sample(values, corners)
	if !ok {
		return math.NaN()
	}
	return value
}

// NodeTotals adds up each lattice node's next 24 and 72 hours from an hourly
// field. Hours the forecast does not reach simply do not add. Nil when the
// forecast does not cover now.
func NodeTotals(field *weather.Field, now time.Time) (next24, next72 []float32) {
	start, ok := field.StepAt(now)
	if !ok || field.Step != time.Hour {
		return nil, nil
	}
	size := len(field.At(start))
	next24, next72 = make([]float32, size), make([]float32, size)
	seen := make([]bool, size)
	for step := start; step < min(start+72, field.Steps); step++ {
		for node, value := range field.At(step) {
			if math.IsNaN(float64(value)) {
				continue
			}
			seen[node] = true
			next72[node] += value
			if step < start+24 {
				next24[node] += value
			}
		}
	}
	nan := float32(math.NaN())
	for node, covered := range seen {
		if !covered {
			next24[node], next72[node] = nan, nan
		}
	}
	return next24, next72
}

// CatchmentSegment is a segment's local rain and catchment.
type CatchmentSegment struct {
	ID, NextDown         int64
	CatchKM2             float64
	Next24hMM, Next72hMM float64
}

// AccumulateRain walks the network from the sources down, giving each segment
// the area-weighted mean over its whole upstream catchment. Local catchments
// with no value (outside the grid) are left out of the mean. Pure.
func AccumulateRain(segments []CatchmentSegment) map[int64]RainSummary {
	type total struct{ area, sum24, sum72, all float64 }
	totals := make(map[int64]*total, len(segments))
	pending := make(map[int64]int, len(segments))
	byID := make(map[int64]*CatchmentSegment, len(segments))
	for index := range segments {
		segment := &segments[index]
		byID[segment.ID] = segment
		totals[segment.ID] = &total{}
	}
	for _, segment := range segments {
		if _, ok := byID[segment.NextDown]; ok {
			pending[segment.NextDown]++
		}
	}
	queue := make([]int64, 0, len(segments))
	for _, segment := range segments {
		if pending[segment.ID] == 0 {
			queue = append(queue, segment.ID)
		}
	}
	out := make(map[int64]RainSummary, len(segments))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		segment, sum := byID[id], totals[id]
		sum.all += segment.CatchKM2
		if !math.IsNaN(segment.Next24hMM) && !math.IsNaN(segment.Next72hMM) {
			sum.area += segment.CatchKM2
			sum.sum24 += segment.Next24hMM * segment.CatchKM2
			sum.sum72 += segment.Next72hMM * segment.CatchKM2
		}
		if sum.area > 0 {
			out[id] = RainSummary{Next24hMM: round1(sum.sum24 / sum.area), Next72hMM: round1(sum.sum72 / sum.area), CatchmentKM2: math.Round(sum.all)}
		}
		if next, ok := totals[segment.NextDown]; ok {
			next.area += sum.area
			next.sum24 += sum.sum24
			next.sum72 += sum.sum72
			next.all += sum.all
			pending[segment.NextDown]--
			if pending[segment.NextDown] == 0 {
				queue = append(queue, segment.NextDown)
			}
		}
	}
	return out
}

func round1(value float64) float64 { return math.Round(value*10) / 10 }

// assemble turns the inputs into the risk document. Pure.
func assemble(net *network, layer rivers.FeatureCollection, google []floods.Gauge, rain map[int64]RainSummary, meta rainMeta, spills map[int64][]Reservoir, now time.Time, seeded bool) *assessed {
	stations := make(map[string]rivers.StationProperties, len(layer.Features))
	document := &RiskDocument{Estimate: true, Note: Note, Rivers: []RiverRisk{}, Meta: RiskMeta{
		GeneratedAt: now, NetworkVersion: net.version, Attribution: Attribution, Seeded: seeded,
		GaugesTotal: len(layer.Features), RainFetchedAt: meta.fetchedAt, RainStale: meta.stale,
		ReservoirLinks: LinkReport{Unlinked: []UnlinkedReservoir{}},
	}}
	if !net.builtAt.IsZero() {
		built := net.builtAt
		document.Meta.NetworkBuiltAt = &built
	}
	for _, feature := range layer.Features {
		properties := feature.Properties
		stations[properties.ID] = properties
		if properties.WaterLevel != nil && !properties.Stale {
			document.Meta.GaugesCurrent++
		}
		document.Meta.GaugesObservedAt = later(document.Meta.GaugesObservedAt, properties.ObservedAt)
	}
	forecasts := make(map[string]floods.Gauge, len(google))
	for _, gauge := range google {
		forecasts[gauge.GaugeID] = gauge
		issued, fetched := gauge.IssuedAt, gauge.FetchedAt
		document.Meta.ForecastIssuedAt = later(document.Meta.ForecastIssuedAt, &issued)
		document.Meta.ForecastFetchedAt = later(document.Meta.ForecastFetchedAt, &fetched)
	}

	result := &assessed{risk: document, reach: map[int64]*ReachRisk{}, river: map[string]*RiverRisk{}}
	byRiver := map[string][]ReachRisk{}
	for _, reach := range net.reaches {
		risk := assessReach(reach, net.gauges[reach.Fid], stations, forecasts, rain, spills[reach.Fid])
		byRiver[reach.RiverID] = append(byRiver[reach.RiverID], risk)
	}
	for _, river := range net.rivers {
		entry := RiverRisk{
			ID: river.ID, Name: river.Name, NameSI: river.NameSi, Aliases: river.Aliases, Named: river.Named,
			ParentID: river.ParentID, UplandKM: math.Round(river.UplandKm2), LengthKM: math.Round(river.LengthKm),
			BBox:  [4]float64{round(river.West), round(river.South), round(river.East), round(river.North)},
			Level: LevelUnknown, Reasons: []Reason{}, Reaches: byRiver[river.ID],
		}
		if entry.Reaches == nil {
			entry.Reaches = []ReachRisk{}
		}
		if entry.Aliases == nil {
			entry.Aliases = []string{}
		}
		for index := range entry.Reaches {
			reach := entry.Reaches[index]
			if reach.Gauged {
				entry.GaugedReaches++
			}
			if reach.Level.Rank() > entry.Level.Rank() || len(entry.Reasons) == 0 {
				entry.Level, entry.Reasons = reach.Level, reach.Reasons
			}
			for _, gauge := range reach.Gauges {
				if gauge.Role != "next" && !gauge.Stale {
					entry.ObservedAt = later(entry.ObservedAt, gauge.ObservedAt)
				}
			}
		}
		document.Rivers = append(document.Rivers, entry)
	}
	sort.SliceStable(document.Rivers, func(i, j int) bool {
		a, b := document.Rivers[i], document.Rivers[j]
		if a.Level.Rank() != b.Level.Rank() {
			return a.Level.Rank() > b.Level.Rank()
		}
		return a.UplandKM > b.UplandKM
	})
	for index := range document.Rivers {
		river := &document.Rivers[index]
		result.river[river.ID] = river
		for reach := range river.Reaches {
			result.reach[river.Reaches[reach].FID] = &river.Reaches[reach]
		}
	}
	return result
}

func round(value float64) float64 { return math.Round(value*1e5) / 1e5 }

func later(a, b *time.Time) *time.Time {
	if b == nil {
		return a
	}
	if a == nil || b.After(*a) {
		value := *b
		return &value
	}
	return a
}

// assessReach gathers a stretch's evidence and assesses it. Only verified
// Google gauges are placed on reaches, by the network build.
func assessReach(reach store.ListRiverReachesRow, gauges []store.RiverReachGauge, stations map[string]rivers.StationProperties, forecasts map[string]floods.Gauge, rain map[int64]RainSummary, spills []Reservoir) ReachRisk {
	risk := ReachRisk{FID: reach.Fid, RiverID: reach.RiverID, Name: reach.Name, Gauged: reach.Gauged,
		LengthKM: math.Round(reach.LengthKm*10) / 10, Gauges: []ReachGauge{}}
	inputs := Inputs{}
	for _, gauge := range gauges {
		entry := ReachGauge{Kind: gauge.Kind, ID: gauge.GaugeID, Role: gauge.Role, Name: gauge.GaugeID}
		switch gauge.Kind {
		case "station":
			station, ok := stations[gauge.GaugeID]
			if !ok {
				continue
			}
			entry.Name, entry.Unit, entry.Status, entry.Trend = station.Name, station.Unit, string(station.Status), string(station.Trend)
			entry.WaterLevel, entry.ObservedAt, entry.Stale = station.WaterLevel, station.ObservedAt, station.Stale
			if gauge.Role == "next" {
				inputs.NextGauge = station.Name
				break
			}
			inputs.Gauges = append(inputs.Gauges, Gauge{ID: station.ID, Name: station.Name, Unit: station.Unit,
				Level: station.WaterLevel, RatePerHour: station.RatePerHour, Alert: station.AlertLevel,
				Minor: station.MinorFloodLevel, Major: station.MajorFloodLevel, ObservedAt: station.ObservedAt, Stale: station.Stale})
		case "google":
			forecast, ok := forecasts[gauge.GaugeID]
			if !ok {
				// Google's data lapsed since the network was built.
				continue
			}
			issued := forecast.IssuedAt
			entry.Name = "Flood Hub point"
			entry.Status, entry.Trend, entry.ObservedAt, entry.Stale = forecast.Severity, forecast.Trend, &issued, forecast.Stale
			if gauge.Role == "next" {
				inputs.NextGauge = entry.Name
				break
			}
			inputs.Forecasts = append(inputs.Forecasts, Forecast{GaugeID: forecast.GaugeID, Severity: forecast.Severity, Trend: forecast.Trend, Stale: forecast.Stale})
		}
		risk.Gauges = append(risk.Gauges, entry)
	}
	if summary, ok := rain[reach.Fid]; ok {
		copied := summary
		risk.Rain = &copied
		inputs.Rain = &Rain{Next24hMM: summary.Next24hMM, Next72hMM: summary.Next72hMM, Stale: summary.Stale}
	}
	inputs.Reservoirs = spills
	assessment := Assess(inputs)
	risk.Level, risk.Reasons = assessment.Level, assessment.Reasons
	return risk
}

// Risk returns every river's state.
func (service *Service) Risk(ctx context.Context) (*RiskDocument, error) {
	current, err := service.compute(ctx)
	if err != nil {
		return nil, err
	}
	return current.risk, nil
}

// RiverGauge is a gauge in a river's detail, with its recent levels.
type RiverGauge struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Basin      string                  `json:"basin"`
	ReachFID   int64                   `json:"reachFid"`
	Unit       string                  `json:"unit"`
	Status     string                  `json:"status"`
	Trend      string                  `json:"trend"`
	WaterLevel *float64                `json:"waterLevel"`
	Alert      *float64                `json:"alertLevel"`
	Minor      *float64                `json:"minorFloodLevel"`
	Major      *float64                `json:"majorFloodLevel"`
	ObservedAt *time.Time              `json:"observedAt"`
	Stale      bool                    `json:"stale"`
	Readings   []rivers.HistoryReading `json:"readings"`
}

// RiverForecast is a Google forecast on a river.
type RiverForecast struct {
	GaugeID       string     `json:"gaugeId"`
	ReachFID      int64      `json:"reachFid"`
	Severity      string     `json:"severity"`
	Trend         string     `json:"trend"`
	IssuedAt      time.Time  `json:"issuedAt"`
	ForecastStart *time.Time `json:"forecastStart"`
	ForecastEnd   *time.Time `json:"forecastEnd"`
	Stale         bool       `json:"stale"`
}

// Affected is an at-risk stretch and the areas it passes through.
type Affected struct {
	ReachFID    int64         `json:"reachFid"`
	ReachName   string        `json:"reachName"`
	Level       Level         `json:"level"`
	Districts   []regions.Ref `json:"districts"`
	DSDivisions []regions.Ref `json:"dsDivisions"`
}

// RiverRef names a related river.
type RiverRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level Level  `json:"level"`
}

// RiverDetail is GET /v1/rivers/{id}.
type RiverDetail struct {
	RiverRisk
	Estimate    bool            `json:"estimate"`
	Note        string          `json:"note"`
	Parent      *RiverRef       `json:"parent"`
	Tributaries []RiverRef      `json:"tributaries"`
	Gauges      []RiverGauge    `json:"gauges"`
	Forecasts   []RiverForecast `json:"forecasts"`
	// Rain is over the catchment of its lowest stretch: the whole river so far.
	Rain     *RainSummary `json:"rain"`
	Affected []Affected   `json:"affected"`
	Meta     RiskMeta     `json:"meta"`
}

// River returns one river's detail.
func (service *Service) River(ctx context.Context, id string) (*RiverDetail, error) {
	current, err := service.compute(ctx)
	if err != nil {
		return nil, err
	}
	river, ok := current.river[id]
	if !ok {
		return nil, ErrNotFound
	}
	detail := &RiverDetail{RiverRisk: *river, Estimate: true, Note: Note, Tributaries: []RiverRef{},
		Gauges: []RiverGauge{}, Forecasts: []RiverForecast{}, Affected: []Affected{}, Meta: current.risk.Meta}
	if river.ParentID != nil {
		if parent, ok := current.river[*river.ParentID]; ok {
			detail.Parent = &RiverRef{ID: parent.ID, Name: parent.Name, Level: parent.Level}
		}
	}
	for index := range current.risk.Rivers {
		other := &current.risk.Rivers[index]
		if other.ParentID != nil && *other.ParentID == id {
			detail.Tributaries = append(detail.Tributaries, RiverRef{ID: other.ID, Name: other.Name, Level: other.Level})
		}
	}
	queries := store.New(service.Pool)
	google := map[string]floods.Gauge{}
	if service.Floods != nil {
		list, err := service.Floods.Gauges(ctx)
		if err != nil {
			service.warn("google forecasts", err)
		}
		for _, gauge := range list {
			google[gauge.GaugeID] = gauge
		}
	}
	seen := map[string]bool{}
	var atRisk []int64
	for _, reach := range river.Reaches {
		if reach.Rain != nil {
			detail.Rain = reach.Rain // reaches run source to sea; the lowest is last
		}
		if reach.Level.AtRisk() {
			atRisk = append(atRisk, reach.FID)
		}
		for _, gauge := range reach.Gauges {
			if gauge.Role == "next" || seen[gauge.Kind+gauge.ID] {
				continue
			}
			seen[gauge.Kind+gauge.ID] = true
			if gauge.Kind == "google" {
				if forecast, ok := google[gauge.ID]; ok {
					detail.Forecasts = append(detail.Forecasts, RiverForecast{GaugeID: forecast.GaugeID, ReachFID: reach.FID,
						Severity: forecast.Severity, Trend: forecast.Trend, IssuedAt: forecast.IssuedAt,
						ForecastStart: forecast.ForecastStart, ForecastEnd: forecast.ForecastEnd, Stale: forecast.Stale})
				}
				continue
			}
			history, err := service.Rivers.StationHistory(ctx, gauge.ID, historyHrs*time.Hour)
			if err != nil {
				service.warn("station history", err)
				continue
			}
			station := history.Station
			detail.Gauges = append(detail.Gauges, RiverGauge{ID: station.ID, Name: station.Name, Basin: station.Basin, ReachFID: reach.FID,
				Unit: station.Unit, Status: string(station.Status), Trend: string(station.Trend), WaterLevel: station.WaterLevel,
				Alert: station.AlertLevel, Minor: station.MinorFloodLevel, Major: station.MajorFloodLevel,
				ObservedAt: station.ObservedAt, Stale: station.Stale, Readings: history.Readings})
		}
	}
	if len(atRisk) > 0 {
		rows, err := queries.ReachRegions(ctx, atRisk)
		if err != nil {
			return nil, fmt.Errorf("areas along the river: %w", err)
		}
		affected := map[int64]*Affected{}
		for _, fid := range atRisk {
			reach := current.reach[fid]
			affected[fid] = &Affected{ReachFID: fid, ReachName: reach.Name, Level: reach.Level, Districts: []regions.Ref{}, DSDivisions: []regions.Ref{}}
		}
		for _, row := range rows {
			entry := affected[row.ReachFid]
			if entry == nil {
				continue
			}
			ref := regions.Ref{ID: row.ID, Level: regions.Level(row.Level), Name: row.Name, NameSI: row.NameSi}
			if row.Level == "district" {
				entry.Districts = append(entry.Districts, ref)
			} else {
				entry.DSDivisions = append(entry.DSDivisions, ref)
			}
		}
		for _, fid := range atRisk {
			detail.Affected = append(detail.Affected, *affected[fid])
		}
		sort.SliceStable(detail.Affected, func(i, j int) bool {
			return detail.Affected[i].Level.Rank() > detail.Affected[j].Level.Rank()
		})
	}
	return detail, nil
}

// NearbyRiver is a river near a place, for /v1/summary.
type NearbyRiver struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Named      bool    `json:"named"`
	Level      Level   `json:"level"`
	ReachName  string  `json:"reachName"`
	Reason     string  `json:"reason"`
	DistanceKm float64 `json:"distanceKm"`
	// Estimate is always true: follow official warnings.
	Estimate bool `json:"estimate"`
}

// Near lists the nearest rivers to a place with the risk of the nearest stretch
// of each, nearest first.
func (service *Service) Near(ctx context.Context, lon, lat float64) ([]NearbyRiver, error) {
	current, err := service.compute(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := store.New(service.Pool).RiversNear(ctx, store.RiversNearParams{Lon: lon, Lat: lat, Degrees: nearDegrees})
	if err != nil {
		return nil, fmt.Errorf("rivers near a point: %w", err)
	}
	out := make([]NearbyRiver, 0, len(rows))
	for _, row := range rows {
		river, ok := current.river[row.RiverID]
		reach, reachOK := current.reach[row.Fid]
		if !ok || !reachOK {
			continue
		}
		entry := NearbyRiver{ID: river.ID, Name: river.Name, Named: river.Named, Level: reach.Level, ReachName: reach.Name,
			DistanceKm: round1(row.DistanceM / 1000), Estimate: true}
		if len(reach.Reasons) > 0 {
			entry.Reason = reach.Reasons[0].Text
		}
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DistanceKm < out[j].DistanceKm })
	if len(out) > maxNear {
		out = out[:maxNear]
	}
	return out, nil
}

// tileOf returns a cached tile, rendering it on a miss.
func (service *Service) tileOf(key, tag string, render func() ([]byte, error)) (regions.Document, error) {
	service.once.Do(func() { service.tiles = regions.NewByteCache(tileCache) })
	etag := `W/"` + tag + `"`
	if body, ok := service.tiles.Get(key); ok {
		return regions.Document{Body: body, ETag: etag}, nil
	}
	body, err := render()
	if err != nil {
		return regions.Document{}, err
	}
	service.tiles.Put(key, body)
	return regions.Document{Body: body, ETag: etag}, nil
}

// minStreamUpland is the smallest stream drawn at a zoom: tributaries appear as
// you zoom in.
func minStreamUpland(z int) float64 {
	switch {
	case z < 7:
		return math.Inf(1)
	case z == 7:
		return 200
	case z == 8:
		return 100
	case z == 9:
		return 40
	case z == 10:
		return 15
	}
	return 0
}

func versionTag(ctx context.Context, queries *store.Queries, source string) (string, error) {
	row, err := queries.GetSourceVersion(ctx, source)
	if errors.Is(err, pgx.ErrNoRows) {
		return "empty", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s version: %w", source, err)
	}
	return row.Version, nil
}

// RiverTile renders one tile of the river network.
func (service *Service) RiverTile(ctx context.Context, z, x, y int) (regions.Document, error) {
	queries := store.New(service.Pool)
	tag, err := versionTag(ctx, queries, rivernet.SourceName)
	if err != nil {
		return regions.Document{}, err
	}
	return service.tileOf(tileKey("rivers", tag, z, x, y), tag, func() ([]byte, error) {
		upland := minStreamUpland(z)
		if math.IsInf(upland, 1) {
			upland = 1e9
		}
		return queries.RiverTile(ctx, store.RiverTileParams{Z: int32(z), X: int32(x), Y: int32(y), MinUpland: upland})
	})
}

// subbasinsFromZoom is where the finer boundaries enter the tiles.
const subbasinsFromZoom = 9

// BasinTile renders one tile of the flood-forecast basins.
func (service *Service) BasinTile(ctx context.Context, z, x, y int) (regions.Document, error) {
	queries := store.New(service.Pool)
	first, err := versionTag(ctx, queries, irrigationgis.BasinsSource)
	if err != nil {
		return regions.Document{}, err
	}
	second, err := versionTag(ctx, queries, hydrobasins.SourceName)
	if err != nil {
		return regions.Document{}, err
	}
	tag := first + "-" + second
	tolerance, minArea := 0.0, 0.0
	switch {
	case z <= 6:
		tolerance, minArea = 500, 25
	case z == 7:
		tolerance, minArea = 250, 10
	case z == 8:
		tolerance, minArea = 120, 3
	case z == 9:
		tolerance = 60
	}
	return service.tileOf(tileKey("basins", tag, z, x, y), tag, func() ([]byte, error) {
		return queries.BasinTile(ctx, store.BasinTileParams{Z: int32(z), X: int32(x), Y: int32(y), Tolerance: tolerance,
			MinArea: minArea, WithSubbasins: z >= subbasinsFromZoom})
	})
}

func tileKey(kind, tag string, z, x, y int) string {
	return kind + "/" + tag + "/" + strconv.Itoa(z) + "/" + strconv.Itoa(x) + "/" + strconv.Itoa(y)
}

// PastFlood is one mapped past flood. It describes an event that has ended and
// is not a forecast.
type PastFlood struct {
	FID        int64      `json:"fid"`
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	SourceName string     `json:"sourceName"`
	River      string     `json:"river"`
	Year       int32      `json:"year"`
	Date       *string    `json:"date"`
	Magnitude  *string    `json:"magnitude"`
	AreaKM2    float64    `json:"areaKm2"`
	Districts  []string   `json:"districts"`
	BBox       [4]float64 `json:"bbox"`
	Service    string     `json:"service"`
}

// PastFloodsDocument is GET /v1/floods/past.
type PastFloodsDocument struct {
	// Note says what these are and are not.
	Note   string      `json:"note"`
	Events []PastFlood `json:"events"`
	Meta   struct {
		Source      string     `json:"source"`
		Attribution string     `json:"attribution"`
		SourceTime  *time.Time `json:"sourceTime"`
		FetchedAt   *time.Time `json:"fetchedAt"`
	} `json:"meta"`
}

// PastFloodNote is said wherever past extents are shown.
const PastFloodNote = "Past flood extents mapped by the Irrigation Department. These show where water reached in past events, not a current forecast."

// PastFloods lists the mapped events, newest first.
func (service *Service) PastFloods(ctx context.Context) (*PastFloodsDocument, error) {
	queries := store.New(service.Pool)
	rows, err := queries.ListPastFloodEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("list past floods: %w", err)
	}
	document := &PastFloodsDocument{Note: PastFloodNote, Events: make([]PastFlood, 0, len(rows))}
	document.Meta.Source = irrigationgis.PastFloodsSource
	document.Meta.Attribution = irrigationgis.Attribution
	for _, row := range rows {
		event := PastFlood{FID: row.Fid, ID: row.ID, Name: row.Name, SourceName: row.SourceName, River: row.River, Year: row.Year,
			Magnitude: row.Magnitude, AreaKM2: round1(row.AreaKm2), Districts: row.Districts,
			BBox:    [4]float64{round(row.West), round(row.South), round(row.East), round(row.North)},
			Service: row.Service}
		if row.EventDate.Valid {
			date := row.EventDate.Time.Format(time.DateOnly)
			event.Date = &date
		}
		if event.Districts == nil {
			event.Districts = []string{}
		}
		document.Events = append(document.Events, event)
		sourceTime, fetched := row.SourceTime, row.FetchedAt
		document.Meta.SourceTime = later(document.Meta.SourceTime, &sourceTime)
		document.Meta.FetchedAt = later(document.Meta.FetchedAt, &fetched)
	}
	return document, nil
}

// pastPartsFromZoom is where past flood tiles read the full extents.
const pastPartsFromZoom = 11

// PastFloodTile renders one tile of past flood extents.
func (service *Service) PastFloodTile(ctx context.Context, z, x, y int) (regions.Document, error) {
	queries := store.New(service.Pool)
	tag, err := versionTag(ctx, queries, irrigationgis.PastFloodsSource)
	if err != nil {
		return regions.Document{}, err
	}
	tolerance := 0.0
	switch {
	case z <= 7:
		tolerance = 200
	case z == 8:
		tolerance = 100
	case z == 9:
		tolerance = 50
	}
	return service.tileOf(tileKey("past-floods", tag, z, x, y), tag, func() ([]byte, error) {
		return queries.PastFloodTile(ctx, store.PastFloodTileParams{Z: int32(z), X: int32(x), Y: int32(y), Tolerance: tolerance, UseParts: z >= pastPartsFromZoom})
	})
}
