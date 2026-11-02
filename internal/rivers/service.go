package rivers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
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
}

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

	collection := FeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0, len(stations)),
		Meta: LayerMeta{
			Source:      irrigation.SourceName,
			Attribution: "Irrigation Department of Sri Lanka",
			GeneratedAt: now,
		},
	}
	if run, err := queries.LastSuccessfulRun(ctx, irrigation.SourceName); err == nil {
		collection.Meta.FetchedAt = run.FinishedAt
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return FeatureCollection{}, fmt.Errorf("read last run: %w", err)
	}

	for _, station := range stations {
		thresholds := Thresholds{Alert: station.AlertLevel, MinorFlood: station.MinorFloodLevel, MajorFlood: station.MajorFloodLevel}
		properties := StationProperties{
			ID: station.ID, Name: station.Name, Basin: station.Basin, Unit: station.Unit,
			AlertLevel: station.AlertLevel, MinorFloodLevel: station.MinorFloodLevel, MajorFloodLevel: station.MajorFloodLevel,
			Status: StatusUnknown, Trend: TrendUnknown, Stale: true,
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

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
