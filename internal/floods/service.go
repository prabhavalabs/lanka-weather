// Package floods serves the Google Flood Forecasting layer for Sri Lanka:
// gauges with their forecast severity, trend and window, and the flood
// areas Google publishes for significant events.
package floods

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Service reads the flood-forecast layer from the database.
type Service struct {
	Pool *pgxpool.Pool
	// Configured is false when no API key is set, so the layer is empty
	// because it is switched off rather than because nothing is forecast.
	Configured bool
	// StaleAfter is the issue age after which a gauge is marked stale.
	StaleAfter time.Duration
	Now        func() time.Time
}

// FeatureCollection is the layer as GeoJSON with metadata.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
	Meta     Meta      `json:"meta"`
}

// Meta describes the layer's origin and freshness.
type Meta struct {
	Source      string     `json:"source"`
	Attribution string     `json:"attribution"`
	Region      string     `json:"region"`
	Configured  bool       `json:"configured"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	GeneratedAt time.Time  `json:"generatedAt"`
	Gauges      int        `json:"gauges"`
	Areas       int        `json:"areas"`
}

// Feature is a gauge (Point) or a flood area (MultiPolygon).
type Feature struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties any             `json:"properties"`
}

// point is a GeoJSON point in longitude, latitude order.
type point struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

// Gauge is a gauge feature's properties.
type Gauge struct {
	Kind            string     `json:"kind"`
	GaugeID         string     `json:"gaugeId"`
	GaugeSource     string     `json:"gaugeSource"`
	SiteName        string     `json:"siteName"`
	River           string     `json:"river"`
	QualityVerified bool       `json:"qualityVerified"`
	Severity        string     `json:"severity"`
	Trend           string     `json:"trend"`
	IssuedAt        time.Time  `json:"issuedAt"`
	ForecastStart   *time.Time `json:"forecastStart"`
	ForecastEnd     *time.Time `json:"forecastEnd"`
	Thresholds      Thresholds `json:"thresholds"`
	FetchedAt       time.Time  `json:"fetchedAt"`
	Stale           bool       `json:"stale"`
	// Lon and Lat repeat the geometry for callers that do not read GeoJSON.
	Lon float64 `json:"-"`
	Lat float64 `json:"-"`
}

// Thresholds are the levels of a gauge's model, in Unit (m3/s or m).
type Thresholds struct {
	Warning       *float64 `json:"warning"`
	Danger        *float64 `json:"danger"`
	ExtremeDanger *float64 `json:"extremeDanger"`
	Unit          string   `json:"unit"`
}

// Area is a flood area feature's properties.
type Area struct {
	Kind       string          `json:"kind"`
	GaugeID    *string         `json:"gaugeId"`
	Severity   string          `json:"severity"`
	IssuedAt   *time.Time      `json:"issuedAt"`
	ValidFrom  *time.Time      `json:"validFrom"`
	ValidUntil *time.Time      `json:"validUntil"`
	Details    json.RawMessage `json:"details"`
	FetchedAt  time.Time       `json:"fetchedAt"`
}

// Layer returns every gauge and flood area fetched within the last day.
func (service *Service) Layer(ctx context.Context) (FeatureCollection, error) {
	now := service.now()
	queries := store.New(service.Pool)
	collection := FeatureCollection{Type: "FeatureCollection", Features: []Feature{}, Meta: Meta{
		Source: googleflood.SourceName, Attribution: googleflood.Attribution, Region: googleflood.RegionCode,
		Configured: service.Configured, GeneratedAt: now,
	}}
	if run, err := queries.LastSuccessfulRun(ctx, googleflood.SourceName); err == nil {
		collection.Meta.FetchedAt = run.FinishedAt
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return FeatureCollection{}, fmt.Errorf("read last run: %w", err)
	}
	gauges, err := service.Gauges(ctx)
	if err != nil {
		return FeatureCollection{}, err
	}
	for _, gauge := range gauges {
		geometry, _ := json.Marshal(point{Type: "Point", Coordinates: [2]float64{gauge.Lon, gauge.Lat}})
		collection.Features = append(collection.Features, Feature{Type: "Feature", ID: gauge.GaugeID, Geometry: geometry, Properties: gauge})
	}
	areas, err := queries.ListFloodAreas(ctx)
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("list flood areas: %w", err)
	}
	for _, area := range areas {
		if now.Sub(area.FetchedAt) > googleflood.Retention {
			continue
		}
		collection.Features = append(collection.Features, Feature{
			Type: "Feature", ID: area.ID, Geometry: json.RawMessage(area.Geojson),
			Properties: Area{Kind: area.Kind, GaugeID: area.GaugeID, Severity: area.Severity, IssuedAt: area.IssuedAt,
				ValidFrom: area.ValidFrom, ValidUntil: area.ValidUntil, Details: area.Properties, FetchedAt: area.FetchedAt},
		})
		collection.Meta.Areas++
	}
	collection.Meta.Gauges = len(gauges)
	return collection, nil
}

// Gauges returns the gauges fetched within the last day: Google's terms let
// search results be kept no longer, whether or not pruning has run.
func (service *Service) Gauges(ctx context.Context) ([]Gauge, error) {
	now := service.now()
	rows, err := store.New(service.Pool).ListFloodGauges(ctx)
	if err != nil {
		return nil, fmt.Errorf("list flood gauges: %w", err)
	}
	gauges := make([]Gauge, 0, len(rows))
	for _, row := range rows {
		if now.Sub(row.FetchedAt) > googleflood.Retention {
			continue
		}
		gauges = append(gauges, Gauge{
			Kind: "gauge", GaugeID: row.GaugeID, GaugeSource: row.GaugeSource, SiteName: row.SiteName, River: row.River,
			QualityVerified: row.QualityVerified, Severity: row.Severity, Trend: row.Trend, IssuedAt: row.IssuedAt,
			ForecastStart: row.ForecastStart, ForecastEnd: row.ForecastEnd,
			Thresholds: Thresholds{Warning: row.WarningLevel, Danger: row.DangerLevel, ExtremeDanger: row.ExtremeDangerLevel, Unit: row.ValueUnit},
			FetchedAt:  row.FetchedAt, Stale: now.Sub(row.IssuedAt) > service.StaleAfter,
			Lon: row.Lon, Lat: row.Lat,
		})
	}
	return gauges, nil
}

// AreaAt is a flood area containing a point.
type AreaAt struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	GaugeID    *string    `json:"gaugeId"`
	Severity   string     `json:"severity"`
	ValidFrom  *time.Time `json:"validFrom"`
	ValidUntil *time.Time `json:"validUntil"`
}

// AreasAt returns the flood areas containing a point.
func (service *Service) AreasAt(ctx context.Context, lon, lat float64) ([]AreaAt, error) {
	rows, err := store.New(service.Pool).FloodAreasAt(ctx, store.FloodAreasAtParams{
		Lon: lon, Lat: lat, Since: service.now().Add(-googleflood.Retention),
	})
	if err != nil {
		return nil, fmt.Errorf("flood areas at a point: %w", err)
	}
	areas := make([]AreaAt, 0, len(rows))
	for _, row := range rows {
		areas = append(areas, AreaAt{ID: row.ID, Kind: row.Kind, GaugeID: row.GaugeID, Severity: row.Severity,
			ValidFrom: row.ValidFrom, ValidUntil: row.ValidUntil})
	}
	return areas, nil
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
