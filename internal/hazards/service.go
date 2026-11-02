// Package hazards serves the earthquake and cyclone layers. Both are read from
// the database as their collectors left them, with the distance to Sri Lanka
// added. Distances are measurements, not warnings: this package generates no
// warning of its own.
package hazards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/gdacs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/usgs"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Service reads the hazard layers from the database.
type Service struct {
	Pool *pgxpool.Pool
	// EarthquakesStaleAfter and CyclonesStaleAfter are the catalogue's limits.
	EarthquakesStaleAfter time.Duration
	CyclonesStaleAfter    time.Duration
	Now                   func() time.Time
}

// EarthquakeLayer is the earthquakes layer as GeoJSON.
type EarthquakeLayer struct {
	Type     string              `json:"type"`
	Features []EarthquakeFeature `json:"features"`
	Meta     EarthquakeMeta      `json:"meta"`
}

// EarthquakeMeta is the layer's origin and freshness.
type EarthquakeMeta struct {
	Source      string     `json:"source"`
	Attribution string     `json:"attribution"`
	WindowDays  int        `json:"windowDays"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	GeneratedAt time.Time  `json:"generatedAt"`
	Stale       bool       `json:"stale"`
	Count       int        `json:"count"`
}

// EarthquakeFeature is one event.
type EarthquakeFeature struct {
	Type       string     `json:"type"`
	ID         string     `json:"id"`
	Geometry   point      `json:"geometry"`
	Properties Earthquake `json:"properties"`
}

type point struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

// Earthquake is one event's properties. GeoJSON properties must be flat for
// the map, so a missing alert is "".
type Earthquake struct {
	ID        string    `json:"id"`
	Magnitude float64   `json:"magnitude"`
	MagType   string    `json:"magType"`
	Place     string    `json:"place"`
	Time      time.Time `json:"time"`
	DepthKm   float64   `json:"depthKm"`
	// DistanceKm is to Sri Lanka's nearest coast.
	DistanceKm float64 `json:"distanceKm"`
	URL        string  `json:"url"`
	Alert      string  `json:"alert"`
	// Tsunami is USGS's flag: an indicator, not a warning for Sri Lanka.
	Tsunami      bool      `json:"tsunami"`
	Significance int32     `json:"significance"`
	Status       string    `json:"status"`
	UpdatedAt    time.Time `json:"updatedAt"`
	FetchedAt    time.Time `json:"fetchedAt"`
	Source       string    `json:"source"`
}

// Earthquakes returns the window's events, newest first.
func (service *Service) Earthquakes(ctx context.Context) (EarthquakeLayer, error) {
	now := service.now()
	queries := store.New(service.Pool)
	layer := EarthquakeLayer{Type: "FeatureCollection", Features: []EarthquakeFeature{}, Meta: EarthquakeMeta{
		Source: usgs.SourceName, Attribution: usgs.Attribution, WindowDays: int(usgs.Window / (24 * time.Hour)),
		GeneratedAt: now, Stale: true,
	}}
	if version, err := queries.GetSourceVersion(ctx, usgs.SourceName); err == nil {
		fetched := version.FetchedAt
		layer.Meta.FetchedAt = &fetched
		layer.Meta.Stale = now.Sub(fetched) > service.EarthquakesStaleAfter
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return layer, fmt.Errorf("read earthquake version: %w", err)
	}
	rows, err := queries.ListEarthquakes(ctx, now.Add(-usgs.Window))
	if err != nil {
		return layer, fmt.Errorf("list earthquakes: %w", err)
	}
	for _, row := range rows {
		alert := ""
		if row.Alert != nil {
			alert = *row.Alert
		}
		layer.Features = append(layer.Features, EarthquakeFeature{
			Type: "Feature", ID: row.ID, Geometry: point{Type: "Point", Coordinates: [2]float64{row.Lon, row.Lat}},
			Properties: Earthquake{
				ID: row.ID, Magnitude: row.Magnitude, MagType: row.MagType, Place: row.Place, Time: row.OccurredAt,
				DepthKm: row.DepthKm, DistanceKm: math.Round(grid.CoastDistanceKm(row.Lon, row.Lat)), URL: row.Url,
				Alert: alert, Tsunami: row.Tsunami, Significance: row.Significance, Status: row.Status,
				UpdatedAt: row.SourceTime, FetchedAt: row.FetchedAt, Source: row.Source,
			},
		})
	}
	layer.Meta.Count = len(layer.Features)
	return layer, nil
}

// Cyclone statuses. Only "none" reads as calm.
const (
	CycloneWaiting = "waiting"
	CycloneActive  = "active"
	CycloneNone    = "none"
	CycloneUnknown = "unknown"
)

// CycloneLayer is the cyclones layer as GeoJSON: for each system its position,
// then its track, forecast, buffers and cone.
type CycloneLayer struct {
	Type     string           `json:"type"`
	Features []CycloneFeature `json:"features"`
	Meta     CycloneMeta      `json:"meta"`
}

// CycloneMeta is the layer's origin, freshness and what can be said.
type CycloneMeta struct {
	Source      string `json:"source"`
	Attribution string `json:"attribution"`
	// Status is waiting, active, none or unknown.
	Status      string     `json:"status"`
	Note        string     `json:"note"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	GeneratedAt time.Time  `json:"generatedAt"`
	Stale       bool       `json:"stale"`
	Count       int        `json:"count"`
}

// CycloneFeature is a GeoJSON feature of a system.
type CycloneFeature struct {
	Type       string          `json:"type"`
	ID         string          `json:"id,omitempty"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties map[string]any  `json:"properties"`
}

// staleAdvisoryAfter marks a system whose last advisory is old. Advisories
// come about every six hours.
const staleAdvisoryAfter = 18 * time.Hour

// Cyclones returns the current systems in the North Indian Ocean.
func (service *Service) Cyclones(ctx context.Context) (CycloneLayer, error) {
	now := service.now()
	queries := store.New(service.Pool)
	layer := CycloneLayer{Type: "FeatureCollection", Features: []CycloneFeature{}, Meta: CycloneMeta{
		Source: gdacs.SourceName, Attribution: gdacs.Attribution, Status: CycloneWaiting,
		Note: "Cyclones have not been collected yet.", GeneratedAt: now, Stale: true,
	}}
	version, err := queries.GetSourceVersion(ctx, gdacs.SourceName)
	if errors.Is(err, pgx.ErrNoRows) {
		return layer, nil
	}
	if err != nil {
		return layer, fmt.Errorf("read cyclone version: %w", err)
	}
	fetched := version.FetchedAt
	layer.Meta.FetchedAt = &fetched
	layer.Meta.Stale = now.Sub(fetched) > service.CyclonesStaleAfter

	rows, err := queries.ListCurrentCyclones(ctx)
	if err != nil {
		return layer, fmt.Errorf("list cyclones: %w", err)
	}
	for _, row := range rows {
		features, err := cycloneFeatures(row, now)
		if err != nil {
			return layer, err
		}
		layer.Features = append(layer.Features, features...)
		layer.Meta.Count++
	}
	switch {
	case layer.Meta.Count > 0:
		layer.Meta.Status, layer.Meta.Note = CycloneActive, ""
		if layer.Meta.Stale {
			layer.Meta.Note = "We have not been able to check GDACS recently, so this may be out of date."
		}
	case layer.Meta.Stale:
		layer.Meta.Status = CycloneUnknown
		layer.Meta.Note = "We have not been able to check GDACS recently, so we cannot say whether a cyclone is active. Check with the Department of Meteorology."
	default:
		layer.Meta.Status, layer.Meta.Note = CycloneNone, "No active cyclone in the North Indian Ocean"
	}
	return layer, nil
}

func cycloneFeatures(row store.Cyclone, now time.Time) ([]CycloneFeature, error) {
	eventID := fmt.Sprint(row.EventID)
	position, err := json.Marshal(point{Type: "Point", Coordinates: [2]float64{row.Lon, row.Lat}})
	if err != nil {
		return nil, err
	}
	wind := any(nil)
	if row.WindKmh != nil {
		wind = *row.WindKmh
	}
	out := []CycloneFeature{{
		Type: "Feature", ID: eventID, Geometry: position,
		Properties: map[string]any{
			"kind": "position", "eventId": eventID, "name": row.Name, "alertLevel": row.AlertLevel, "windKmh": wind,
			"severityText": row.SeverityText, "country": row.Country, "startedAt": row.StartedAt, "latestAt": row.LatestAt,
			"advisorySource": row.AdvisorySource, "reportUrl": row.ReportUrl, "lon": row.Lon, "lat": row.Lat,
			"distanceKm": math.Round(grid.CoastDistanceKm(row.Lon, row.Lat)),
			"stale":      now.Sub(row.LatestAt) > staleAdvisoryAfter,
			"source":     row.Source, "sourceTime": row.SourceTime, "fetchedAt": row.FetchedAt,
		},
	}}
	var stored []CycloneFeature
	if err := json.Unmarshal(row.Features, &stored); err != nil {
		return nil, fmt.Errorf("read geometry of %s: %w", row.Name, err)
	}
	for _, feature := range stored {
		feature.Properties["eventId"] = eventID
		feature.Properties["name"] = row.Name
		feature.Properties["alertLevel"] = row.AlertLevel
		out = append(out, feature)
	}
	return out, nil
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
