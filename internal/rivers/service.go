package rivers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// ErrStationNotFound is returned for an unknown station identifier.
var ErrStationNotFound = errors.New("station not found")

// Service reads the river-levels layer from the database.
type Service struct {
	Pool *pgxpool.Pool
	// StaleAfter is the reading age after which a station is marked stale.
	StaleAfter time.Duration
	Now        func() time.Time
}

// FeatureCollection is a GeoJSON feature collection with layer metadata.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
	Meta     LayerMeta `json:"meta"`
}

// LayerMeta describes the freshness and origin of the layer as a whole.
type LayerMeta struct {
	Source      string     `json:"source"`
	Attribution string     `json:"attribution"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	GeneratedAt time.Time  `json:"generatedAt"`
	// Sources lists every collector behind the layer with when it last ran;
	// Source and FetchedAt above are the authoritative Irrigation Department feed.
	Sources []SourceMeta `json:"sources"`
}

// SourceMeta describes one collector behind the layer.
type SourceMeta struct {
	Source      string     `json:"source"`
	Attribution string     `json:"attribution"`
	FetchedAt   *time.Time `json:"fetchedAt"`
}

// Feature is one station as a GeoJSON point.
type Feature struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Geometry   Point             `json:"geometry"`
	Properties StationProperties `json:"properties"`
}

// Point is a GeoJSON point in longitude, latitude order.
type Point struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

// StationProperties are the properties of a station feature.
type StationProperties struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Basin           string     `json:"basin"`
	Unit            string     `json:"unit"`
	AlertLevel      *float64   `json:"alertLevel"`
	MinorFloodLevel *float64   `json:"minorFloodLevel"`
	MajorFloodLevel *float64   `json:"majorFloodLevel"`
	WaterLevel      *float64   `json:"waterLevel"`
	RainfallMM      *float64   `json:"rainfallMm"`
	ObservedAt      *time.Time `json:"observedAt"`
	Status          Status     `json:"status"`
	Trend           Trend      `json:"trend"`
	RatePerHour     *float64   `json:"ratePerHour"`
	Stale           bool       `json:"stale"`
	// Source is the collector the station comes from; Attribution says who to
	// credit; SourceURL is where to read more, when the source has a page.
	Source      string  `json:"source"`
	Attribution string  `json:"attribution"`
	SourceURL   *string `json:"sourceUrl"`
	// NextHour is an experimental forecast by a third party. It is shown beside
	// the status and never changes it; null when none is recent enough.
	NextHour *NextHour `json:"nextHour"`
}

// NextHour is the experimental next-hour forecast at a station, in the
// station's unit.
type NextHour struct {
	Level        float64   `json:"level"`
	Low          *float64  `json:"low"`
	High         *float64  `json:"high"`
	Unit         string    `json:"unit"`
	Direction    *string   `json:"direction"`
	Crossing     *string   `json:"crossing"`
	Method       *string   `json:"method"`
	IssuedAt     time.Time `json:"issuedAt"`
	Experimental bool      `json:"experimental"`
	Attribution  string    `json:"attribution"`
}

// NextHourMaxAge is how old a next-hour forecast may be and still be shown.
const NextHourMaxAge = 2 * time.Hour

// lookback bounds the readings read to build the layer: enough to find a
// stale station's last reading and every station's trend.
const lookback = 48 * time.Hour

// Layer returns every station with its latest reading, status and trend.
func (service *Service) Layer(ctx context.Context) (FeatureCollection, error) {
	now := service.now()
	queries := store.New(service.Pool)
	stations, err := queries.ListRiverStations(ctx)
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("list stations: %w", err)
	}
	rows, err := queries.RecentRiverReadings(ctx, now.Add(-lookback))
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("read recent readings: %w", err)
	}
	samples := map[string][]Sample{}
	rainfall := map[string]*float64{}
	for _, row := range rows {
		if _, seen := samples[row.StationID]; !seen {
			rainfall[row.StationID] = row.RainfallMm
		}
		samples[row.StationID] = append(samples[row.StationID], Sample{ObservedAt: row.ObservedAt, Level: row.WaterLevel})
	}

	forecasts, err := queries.LatestNextHour(ctx, now.Add(-NextHourMaxAge))
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("read next-hour forecasts: %w", err)
	}
	nextHours := make(map[string]store.RiverNextHour, len(forecasts))
	for _, forecast := range forecasts {
		nextHours[forecast.StationID] = forecast
	}

	collection := FeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0, len(stations)),
		Meta: LayerMeta{
			Source:      irrigation.SourceName,
			Attribution: IrrigationAttribution,
			GeneratedAt: now,
			Sources:     []SourceMeta{},
		},
	}
	for _, meta := range []SourceMeta{
		{Source: irrigation.SourceName, Attribution: IrrigationAttribution},
		{Source: waterlevels.GaugesSource, Attribution: waterlevels.Attribution},
	} {
		run, err := queries.LastSuccessfulRun(ctx, meta.Source)
		if err == nil {
			meta.FetchedAt = run.FinishedAt
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return FeatureCollection{}, fmt.Errorf("read last run: %w", err)
		}
		if meta.Source == irrigation.SourceName {
			collection.Meta.FetchedAt = meta.FetchedAt
		}
		collection.Meta.Sources = append(collection.Meta.Sources, meta)
	}

	for _, station := range stations {
		thresholds := Thresholds{Alert: station.AlertLevel, MinorFlood: station.MinorFloodLevel, MajorFlood: station.MajorFloodLevel}
		properties := StationProperties{
			ID: station.ID, Name: station.Name, Basin: station.Basin, Unit: station.Unit,
			AlertLevel: station.AlertLevel, MinorFloodLevel: station.MinorFloodLevel, MajorFloodLevel: station.MajorFloodLevel,
			Status: StatusUnknown, Trend: TrendUnknown, Stale: true,
			Source: station.Source, Attribution: attribution(station.Source, station.Upstream), SourceURL: station.SourceUrl,
		}
		if forecast, ok := nextHours[station.ID]; ok {
			properties.NextHour = nextHour(forecast, station.Unit)
		}
		if series := samples[station.ID]; len(series) > 0 {
			latest := series[0]
			level, observed := latest.Level, latest.ObservedAt
			properties.WaterLevel = &level
			properties.ObservedAt = &observed
			properties.RainfallMM = rainfall[station.ID]
			properties.Status = Classify(level, thresholds)
			properties.Trend, properties.RatePerHour = EstimateTrend(series)
			properties.Stale = now.Sub(observed) > service.StaleAfter
		}
		collection.Features = append(collection.Features, Feature{
			Type: "Feature", ID: station.ID,
			Geometry:   Point{Type: "Point", Coordinates: [2]float64{station.Lon, station.Lat}},
			Properties: properties,
		})
	}
	return collection, nil
}

// History is one station's readings over a period.
type History struct {
	Station  StationProperties `json:"station"`
	Readings []HistoryReading  `json:"readings"`
}

// HistoryReading is one reading in a station's history.
type HistoryReading struct {
	ObservedAt time.Time `json:"observedAt"`
	WaterLevel float64   `json:"waterLevel"`
	RainfallMM *float64  `json:"rainfallMm"`
}

// StationHistory returns a station's readings since the given duration ago,
// oldest first.
func (service *Service) StationHistory(ctx context.Context, id string, period time.Duration) (History, error) {
	queries := store.New(service.Pool)
	station, err := queries.GetRiverStation(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return History{}, ErrStationNotFound
	}
	if err != nil {
		return History{}, fmt.Errorf("read station: %w", err)
	}
	rows, err := queries.StationRiverReadings(ctx, store.StationRiverReadingsParams{StationID: id, ObservedAt: service.now().Add(-period)})
	if err != nil {
		return History{}, fmt.Errorf("read readings: %w", err)
	}
	history := History{
		Station: StationProperties{
			ID: station.ID, Name: station.Name, Basin: station.Basin, Unit: station.Unit,
			AlertLevel: station.AlertLevel, MinorFloodLevel: station.MinorFloodLevel, MajorFloodLevel: station.MajorFloodLevel,
			Status: StatusUnknown, Trend: TrendUnknown, Stale: true,
			Source: station.Source, Attribution: attribution(station.Source, station.Upstream), SourceURL: station.SourceUrl,
		},
		Readings: make([]HistoryReading, 0, len(rows)),
	}
	for _, row := range rows {
		history.Readings = append(history.Readings, HistoryReading{ObservedAt: row.ObservedAt, WaterLevel: row.WaterLevel, RainfallMM: row.RainfallMm})
	}
	if n := len(rows); n > 0 {
		latest := rows[n-1]
		level, observed := latest.WaterLevel, latest.ObservedAt
		history.Station.WaterLevel, history.Station.ObservedAt, history.Station.RainfallMM = &level, &observed, latest.RainfallMm
		history.Station.Status = Classify(level, Thresholds{Alert: station.AlertLevel, MinorFlood: station.MinorFloodLevel, MajorFlood: station.MajorFloodLevel})
		newestFirst := make([]Sample, 0, n)
		for i := n - 1; i >= 0; i-- {
			newestFirst = append(newestFirst, Sample{ObservedAt: rows[i].ObservedAt, Level: rows[i].WaterLevel})
		}
		history.Station.Trend, history.Station.RatePerHour = EstimateTrend(newestFirst)
		history.Station.Stale = service.now().Sub(observed) > service.StaleAfter
	}
	return history, nil
}

// IrrigationAttribution credits the department's own feed.
const IrrigationAttribution = "Irrigation Department of Sri Lanka"

// attribution credits a station: the department for its own feed; otherwise the
// site it was read from, naming the agency behind it where the site does.
func attribution(source string, upstream *string) string {
	if source != waterlevels.GaugesSource {
		return IrrigationAttribution
	}
	if upstream != nil && *upstream != "" {
		return *upstream + " via " + waterlevels.Attribution
	}
	return waterlevels.Attribution + " (upstream agency not stated)"
}

// nextHour converts a stored forecast to the station's unit, keeping a sensible
// precision. A station without a known unit cannot show it: a forecast in feet
// next to a level of unknown unit would invite a comparison that means nothing.
func nextHour(row store.RiverNextHour, unit string) *NextHour {
	convert := func(feet float64) float64 {
		value, _ := waterlevels.FromFeet(feet, unit)
		return value
	}
	if unit != "m" && unit != "ft" {
		return nil
	}
	forecast := &NextHour{
		Level: convert(row.LevelFt), Unit: unit, Direction: row.Direction, Crossing: row.Crossing, Method: row.Method,
		IssuedAt: row.IssuedAt, Experimental: true, Attribution: waterlevels.Attribution,
	}
	// A range is shown only as a whole.
	if row.LowFt != nil && row.HighFt != nil {
		low, high := convert(*row.LowFt), convert(*row.HighFt)
		forecast.Low, forecast.High = &low, &high
	}
	return forecast
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
