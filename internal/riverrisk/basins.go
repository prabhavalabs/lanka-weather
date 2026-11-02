package riverrisk

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/hydrobasins"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigationgis"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/rivernet"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// BasinState is how a basin is drawn on the flood layer.
type BasinState string

const (
	// BasinNoForecast means no gauge or forecast covers the basin: nothing is
	// known, which is not the same as calm.
	BasinNoForecast BasinState = "no_forecast"
	// BasinNoFlooding means a current gauge or forecast covers it, and none is
	// above normal.
	BasinNoFlooding  BasinState = "no_flooding"
	BasinAboveNormal BasinState = "above_normal"
	BasinSevere      BasinState = "severe"
	BasinExtreme     BasinState = "extreme"
)

// Rank orders states from no forecast (0) to extreme (4).
func (state BasinState) Rank() int {
	switch state {
	case BasinNoFlooding:
		return 1
	case BasinAboveNormal:
		return 2
	case BasinSevere:
		return 3
	case BasinExtreme:
		return 4
	}
	return 0
}

// BasinGauge is a gauge in a basin.
type BasinGauge struct {
	// Kind is station or google.
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Name string `json:"name"`
	// State is what this gauge says, in the basin's scale; empty when it says
	// nothing (out of date, or no thresholds).
	State      BasinState `json:"state"`
	Status     string     `json:"status"`
	Trend      string     `json:"trend"`
	WaterLevel *float64   `json:"waterLevel,omitempty"`
	Unit       string     `json:"unit,omitempty"`
	ObservedAt *time.Time `json:"observedAt"`
	Stale      bool       `json:"stale"`
	// Verified is false for a Google gauge it has not quality-verified. They
	// are read only with LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED=true.
	Verified bool `json:"verified"`
}

// BasinArea is a Google flood area touching a basin.
type BasinArea struct {
	ID    string     `json:"id"`
	Kind  string     `json:"kind"`
	State BasinState `json:"state"`
}

// BasinStatus is one official river basin's state.
type BasinStatus struct {
	FID     int64      `json:"fid"`
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Named   bool       `json:"named"`
	State   BasinState `json:"state"`
	AreaKM2 float64    `json:"areaKm2"`
	// RiverID is the river draining it, where one is known.
	RiverID *string      `json:"riverId"`
	BBox    [4]float64   `json:"bbox"`
	Label   [2]float64   `json:"labelPoint"`
	Gauges  []BasinGauge `json:"gauges"`
	Areas   []BasinArea  `json:"areas"`
	// OnlyUnverified says its state comes only from gauges Google has not
	// verified, and must be shown as such.
	OnlyUnverified bool `json:"onlyUnverified"`
	// ObservedAt is its newest reading or forecast.
	ObservedAt *time.Time `json:"observedAt"`
}

// SubbasinStatus is one HydroBASINS sub-basin's state. Only sub-basins with
// something to say are listed; the rest have no forecast.
type SubbasinStatus struct {
	FID            int64      `json:"fid"`
	State          BasinState `json:"state"`
	OnlyUnverified bool       `json:"onlyUnverified"`
}

// BasinsDocument is GET /v1/floods/basins.
type BasinsDocument struct {
	Basins    []BasinStatus    `json:"basins"`
	Subbasins []SubbasinStatus `json:"subbasins"`
	Meta      BasinsMeta       `json:"meta"`
}

// BasinsMeta says where the boundaries and states come from.
type BasinsMeta struct {
	GeneratedAt time.Time `json:"generatedAt"`
	Attribution string    `json:"attribution"`
	// BoundarySource names the official basins, and SubbasinSource the finer
	// ones.
	BoundarySource    string     `json:"boundarySource"`
	SubbasinSource    string     `json:"subbasinSource"`
	BoundaryTime      *time.Time `json:"boundaryTime"`
	GaugesObservedAt  *time.Time `json:"gaugesObservedAt"`
	ForecastIssuedAt  *time.Time `json:"forecastIssuedAt"`
	ForecastFetchedAt *time.Time `json:"forecastFetchedAt"`
	// IncludesUnverified says unverified Google gauges were read.
	IncludesUnverified bool `json:"includesUnverified"`
	Seeded             bool `json:"seeded"`
}

// StationState puts a station's status on the basins' scale.
func StationState(status rivers.Status) (BasinState, bool) {
	switch status {
	case rivers.StatusMajorFlood:
		return BasinExtreme, true
	case rivers.StatusMinorFlood:
		return BasinSevere, true
	case rivers.StatusAlert:
		return BasinAboveNormal, true
	case rivers.StatusNormal:
		return BasinNoFlooding, true
	}
	return "", false
}

// ForecastState puts a Google severity on the basins' scale.
func ForecastState(severity string) (BasinState, bool) {
	switch severity {
	case "extreme":
		return BasinExtreme, true
	case "severe":
		return BasinSevere, true
	case "above_normal":
		return BasinAboveNormal, true
	case "no_flooding":
		return BasinNoFlooding, true
	}
	return "", false
}

// AreaState puts a Google flood area on the basins' scale. A gauge's inundation
// and notification areas carry the gauge's severity. A flash flood or
// significant event carries its own band instead: a flash flood highly likely
// reads severe.
func AreaState(kind, severity string) BasinState {
	if state, ok := ForecastState(severity); ok && state != BasinNoFlooding {
		return state
	}
	switch kind {
	case "flash_highly_likely":
		return BasinSevere
	case "inundation_high":
		return BasinSevere
	}
	return BasinAboveNormal
}

// Spread gives each sub-basin the state of the gauges governing it: the gauge's
// own, and those below it down to the next gauge, since a forecast at a gauge
// describes the water leaving it. Sub-basins upstream of every gauge have no
// forecast. Where two gauges reach one sub-basin the worse state counts. Pure.
func Spread(nextDown map[int64]int64, gauges map[int64]BasinState) map[int64]BasinState {
	out := make(map[int64]BasinState, len(gauges))
	for start, state := range gauges {
		for current := start; ; {
			if existing, ok := out[current]; !ok || state.Rank() > existing.Rank() {
				out[current] = state
			}
			next := nextDown[current]
			if next == 0 {
				break
			}
			if _, known := nextDown[next]; !known {
				break
			}
			if _, gauged := gauges[next]; gauged {
				break
			}
			current = next
		}
	}
	return out
}

// buildBasins states every official basin and sub-basin.
func (service *Service) buildBasins(ctx context.Context, queries *store.Queries, net *network, layer rivers.FeatureCollection, google []floods.Gauge, now time.Time, seeded bool) (*BasinsDocument, error) {
	document := &BasinsDocument{Basins: make([]BasinStatus, 0, len(net.basins)), Subbasins: []SubbasinStatus{}, Meta: BasinsMeta{
		GeneratedAt: now, Attribution: irrigationgis.Attribution + "; " + hydrobasins.Attribution + "; " + googleflood.Attribution,
		BoundarySource: irrigationgis.BasinsSource, SubbasinSource: hydrobasins.SourceName, Seeded: seeded,
	}}
	if len(net.basins) == 0 {
		return document, nil
	}
	if err := service.placeGauges(ctx, queries, net, layer, google); err != nil {
		return nil, err
	}
	statusByFID := make(map[int64]*BasinStatus, len(net.basins))
	riverOfBasin := map[int64]string{}
	for _, river := range net.rivers {
		if river.BasinFid != nil && river.ParentID == nil {
			riverOfBasin[*river.BasinFid] = river.ID
		}
	}
	for _, basin := range net.basins {
		entry := BasinStatus{FID: basin.Fid, ID: basin.ID, Name: basin.Name, Named: basin.Named, State: BasinNoForecast,
			AreaKM2: round1(basin.AreaKm2), Gauges: []BasinGauge{}, Areas: []BasinArea{},
			BBox:  [4]float64{round(basin.West), round(basin.South), round(basin.East), round(basin.North)},
			Label: [2]float64{round(basin.LabelLon), round(basin.LabelLat)}}
		if river, ok := riverOfBasin[basin.Fid]; ok {
			entry.RiverID = &river
		}
		sourceTime := basin.SourceTime
		document.Meta.BoundaryTime = later(document.Meta.BoundaryTime, &sourceTime)
		document.Basins = append(document.Basins, entry)
	}
	for index := range document.Basins {
		statusByFID[document.Basins[index].FID] = &document.Basins[index]
	}

	verifiedSub, unverifiedSub := map[int64]BasinState{}, map[int64]BasinState{}
	add := func(place store.GaugeBasinsAtPointsRow, hybas int64, gauge BasinGauge) {
		if basin := statusByFID[place.BasinFid]; basin != nil {
			basin.Gauges = append(basin.Gauges, gauge)
			basin.ObservedAt = later(basin.ObservedAt, gauge.ObservedAt)
		}
		if gauge.State == "" || hybas == 0 {
			return
		}
		target := verifiedSub
		if !gauge.Verified {
			target = unverifiedSub
		}
		if existing, ok := target[hybas]; !ok || gauge.State.Rank() > existing.Rank() {
			target[hybas] = gauge.State
		}
	}
	for _, feature := range layer.Features {
		station := feature.Properties
		place, ok := net.stationAt[station.ID]
		if !ok {
			continue
		}
		gauge := BasinGauge{Kind: "station", ID: station.ID, Name: station.Name, Status: string(station.Status),
			Trend: string(station.Trend), WaterLevel: station.WaterLevel, Unit: station.Unit, ObservedAt: station.ObservedAt,
			Stale: station.Stale, Verified: true}
		if state, ok := StationState(station.Status); ok && !station.Stale && station.WaterLevel != nil {
			gauge.State = state
		}
		document.Meta.GaugesObservedAt = later(document.Meta.GaugesObservedAt, station.ObservedAt)
		add(place, place.HybasID, gauge)
	}
	subbasinKnown := make(map[int64]bool, len(net.subbasins))
	for _, sub := range net.subbasins {
		subbasinKnown[sub.HybasID] = true
	}
	for _, forecast := range google {
		place, ok := net.googleAt[forecast.GaugeID]
		if !ok {
			continue
		}
		issued, fetched := forecast.IssuedAt, forecast.FetchedAt
		gauge := BasinGauge{Kind: "google", ID: forecast.GaugeID, Name: "Flood Hub point", Status: forecast.Severity,
			Trend: forecast.Trend, ObservedAt: &issued, Stale: forecast.Stale, Verified: forecast.QualityVerified}
		if state, ok := ForecastState(forecast.Severity); ok && !forecast.Stale {
			gauge.State = state
		}
		if !forecast.QualityVerified {
			document.Meta.IncludesUnverified = true
		}
		document.Meta.ForecastIssuedAt = later(document.Meta.ForecastIssuedAt, &issued)
		document.Meta.ForecastFetchedAt = later(document.Meta.ForecastFetchedAt, &fetched)
		hybas := place.HybasID
		if id := rivernet.HybasOfGauge(forecast.GaugeID); id != 0 && subbasinKnown[id] {
			hybas = id
		}
		add(place, hybas, gauge)
	}
	if len(net.basins) > 0 {
		areas, err := queries.FloodAreasInBasins(ctx, now.Add(-googleflood.Retention))
		if err != nil {
			return nil, fmt.Errorf("flood areas in basins: %w", err)
		}
		for _, area := range areas {
			if basin := statusByFID[area.BasinFid]; basin != nil {
				basin.Areas = append(basin.Areas, BasinArea{ID: area.ID, Kind: area.Kind, State: AreaState(area.Kind, area.Severity)})
			}
		}
	}
	for index := range document.Basins {
		settleBasin(&document.Basins[index])
	}

	nextDown := make(map[int64]int64, len(net.subbasins))
	for _, sub := range net.subbasins {
		nextDown[sub.HybasID] = sub.NextDown
	}
	verified, unverified := Spread(nextDown, verifiedSub), Spread(nextDown, unverifiedSub)
	for _, sub := range net.subbasins {
		state, known := verified[sub.HybasID]
		other, otherKnown := unverified[sub.HybasID]
		switch {
		case known && (!otherKnown || state.Rank() >= other.Rank()):
			document.Subbasins = append(document.Subbasins, SubbasinStatus{FID: sub.HybasID, State: state})
		case otherKnown:
			document.Subbasins = append(document.Subbasins, SubbasinStatus{FID: sub.HybasID, State: other, OnlyUnverified: true})
		}
	}
	return document, nil
}

// settleBasin takes the worst of what a basin's gauges and areas say. Verified
// evidence decides unless unverified gauges say something worse, which is shown
// and marked as such.
func settleBasin(basin *BasinStatus) {
	verified, unverified := BasinNoForecast, BasinNoForecast
	for _, gauge := range basin.Gauges {
		if gauge.State == "" {
			continue
		}
		if gauge.Verified {
			if gauge.State.Rank() > verified.Rank() {
				verified = gauge.State
			}
		} else if gauge.State.Rank() > unverified.Rank() {
			unverified = gauge.State
		}
	}
	for _, area := range basin.Areas {
		if area.State.Rank() > verified.Rank() {
			verified = area.State
		}
	}
	basin.State = verified
	if unverified.Rank() > verified.Rank() {
		basin.State, basin.OnlyUnverified = unverified, true
	}
	sort.SliceStable(basin.Gauges, func(i, j int) bool { return basin.Gauges[i].State.Rank() > basin.Gauges[j].State.Rank() })
}

// placeGauges finds each gauge's official basin and sub-basin once per set of
// gauges. Part D's stations and Google's come and go, so the set is watched.
func (service *Service) placeGauges(ctx context.Context, queries *store.Queries, net *network, layer rivers.FeatureCollection, google []floods.Gauge) error {
	if net.stationAt != nil && net.stations == len(layer.Features) && net.google == len(google) && allPlaced(net, layer, google) {
		return nil
	}
	var positions []int32
	var lons, lats []float64
	var keys []string
	for _, feature := range layer.Features {
		positions = append(positions, int32(len(positions)))
		lons, lats = append(lons, feature.Geometry.Coordinates[0]), append(lats, feature.Geometry.Coordinates[1])
		keys = append(keys, "s:"+feature.Properties.ID)
	}
	for _, gauge := range google {
		positions = append(positions, int32(len(positions)))
		lons, lats = append(lons, gauge.Lon), append(lats, gauge.Lat)
		keys = append(keys, "g:"+gauge.GaugeID)
	}
	net.stationAt, net.googleAt = map[string]store.GaugeBasinsAtPointsRow{}, map[string]store.GaugeBasinsAtPointsRow{}
	net.stations, net.google = len(layer.Features), len(google)
	if len(positions) == 0 {
		return nil
	}
	rows, err := queries.GaugeBasinsAtPoints(ctx, store.GaugeBasinsAtPointsParams{Positions: positions, Lons: lons, Lats: lats})
	if err != nil {
		return fmt.Errorf("place gauges in basins: %w", err)
	}
	for _, row := range rows {
		key := keys[row.Position]
		if key[0] == 's' {
			net.stationAt[key[2:]] = row
		} else {
			net.googleAt[key[2:]] = row
		}
	}
	return nil
}

func allPlaced(net *network, layer rivers.FeatureCollection, google []floods.Gauge) bool {
	for _, feature := range layer.Features {
		if _, ok := net.stationAt[feature.Properties.ID]; !ok {
			return false
		}
	}
	for _, gauge := range google {
		if _, ok := net.googleAt[gauge.GaugeID]; !ok {
			return false
		}
	}
	return true
}

// Basins returns every basin's state.
func (service *Service) Basins(ctx context.Context) (*BasinsDocument, error) {
	current, err := service.compute(ctx)
	if err != nil {
		return nil, err
	}
	return current.basins, nil
}

// Basin returns one basin's state, by its id or feature id.
func (service *Service) Basin(ctx context.Context, id string) (*BasinStatus, *BasinsMeta, error) {
	current, err := service.compute(ctx)
	if err != nil {
		return nil, nil, err
	}
	fid, _ := strconv.ParseInt(id, 10, 64)
	for index := range current.basins.Basins {
		basin := &current.basins.Basins[index]
		if basin.ID == id || (fid != 0 && basin.FID == fid) {
			return basin, &current.basins.Meta, nil
		}
	}
	return nil, nil, ErrNotFound
}
