// Package gdacs collects tropical cyclones in the North Indian Ocean from the
// Global Disaster Alert and Coordination System: the event list, and each
// current event's track, wind buffers and uncertainty cone.
package gdacs

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// SourceName identifies the source in docs/data-sources.md.
const SourceName = "gdacs"

// Attribution is the credit GDACS asks for.
const Attribution = "Global Disaster Alert and Coordination System (GDACS), gdacs.org"

// The North Indian Ocean: the Arabian Sea and the Bay of Bengal, as far east
// as JTWC's Indian Ocean area reaches.
const (
	basinSouth = 0.0
	basinNorth = 35.0
	basinWest  = 40.0
	basinEast  = 100.0
)

// StaleEventAfter is when an event flagged current stops counting. GDACS leaves
// the flag set for days after the last advisory: on 2026-11-02 it still called
// a storm with a last advisory on 2026-10-30 current.
const StaleEventAfter = 48 * time.Hour

// Event is one tropical cyclone in the list.
type Event struct {
	ID             int64
	Episode        int
	Name           string
	AlertLevel     string
	Current        bool
	WindKmh        *float64
	SeverityText   string
	Lon            float64
	Lat            float64
	Country        string
	StartedAt      time.Time
	LatestAt       time.Time
	ModifiedAt     time.Time
	AdvisorySource string
	ReportURL      string
}

type wireList struct {
	Features []struct {
		Geometry struct {
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
		Properties struct {
			EventID    int64  `json:"eventid"`
			EpisodeID  int    `json:"episodeid"`
			EventName  string `json:"eventname"`
			Name       string `json:"name"`
			AlertLevel string `json:"alertlevel"`
			IsCurrent  string `json:"iscurrent"`
			Country    string `json:"country"`
			FromDate   string `json:"fromdate"`
			ToDate     string `json:"todate"`
			Modified   string `json:"datemodified"`
			Source     string `json:"source"`
			URL        struct {
				Report string `json:"report"`
			} `json:"url"`
			SeverityData struct {
				Severity     *float64 `json:"severity"`
				SeverityText string   `json:"severitytext"`
				SeverityUnit string   `json:"severityunit"`
			} `json:"severitydata"`
		} `json:"properties"`
	} `json:"features"`
}

// gdacsTime reads the list's times, which are UTC without a zone.
func gdacsTime(value string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05", strings.TrimSpace(value))
}

// DecodeEvents reads the event list. An event without a usable position or
// time is skipped and counted.
func DecodeEvents(payload []byte) (events []Event, skipped int, err error) {
	var wire wireList
	if err := json.Unmarshal(payload, &wire); err != nil {
		return nil, 0, fmt.Errorf("decode cyclone list: %w", err)
	}
	if wire.Features == nil {
		return nil, 0, errors.New("cyclone list has no features")
	}
	for _, feature := range wire.Features {
		properties := feature.Properties
		if properties.EventID == 0 || len(feature.Geometry.Coordinates) < 2 {
			skipped++
			continue
		}
		started, err := gdacsTime(properties.FromDate)
		if err != nil {
			skipped++
			continue
		}
		latest, err := gdacsTime(properties.ToDate)
		if err != nil {
			skipped++
			continue
		}
		modified, err := gdacsTime(properties.Modified)
		if err != nil {
			modified = latest
		}
		var wind *float64
		if value := properties.SeverityData.Severity; value != nil && strings.EqualFold(properties.SeverityData.SeverityUnit, "km/h") {
			rounded := math.Round(*value)
			wind = &rounded
		}
		name := strings.TrimSpace(properties.EventName)
		if name == "" {
			name = strings.TrimSpace(properties.Name)
		}
		report := properties.URL.Report
		if !strings.HasPrefix(report, "https://www.gdacs.org/") {
			report = fmt.Sprintf("https://www.gdacs.org/report.aspx?eventid=%d&episodeid=%d&eventtype=TC", properties.EventID, properties.EpisodeID)
		}
		events = append(events, Event{
			ID: properties.EventID, Episode: properties.EpisodeID, Name: name, AlertLevel: properties.AlertLevel,
			Current: strings.EqualFold(properties.IsCurrent, "true"), WindKmh: wind,
			SeverityText: properties.SeverityData.SeverityText,
			Lon:          feature.Geometry.Coordinates[0], Lat: feature.Geometry.Coordinates[1],
			Country: properties.Country, StartedAt: started, LatestAt: latest, ModifiedAt: modified,
			AdvisorySource: properties.Source, ReportURL: report,
		})
	}
	return events, skipped, nil
}

// InBasin reports whether a position is in the North Indian Ocean.
func InBasin(lon, lat float64) bool {
	return lat >= basinSouth && lat <= basinNorth && lon >= basinWest && lon <= basinEast
}

// IsCurrent reports whether an event counts as active now.
func IsCurrent(event Event, now time.Time) bool {
	return event.Current && now.Sub(event.LatestAt) <= StaleEventAfter
}

// Relevant keeps current events in the basin.
func Relevant(events []Event, now time.Time) []Event {
	var kept []Event
	for _, event := range events {
		if InBasin(event.Lon, event.Lat) && IsCurrent(event, now) {
			kept = append(kept, event)
		}
	}
	return kept
}

// Feature kinds in a cyclone's stored geometry.
const (
	KindTrack    = "track"
	KindForecast = "forecast-track"
	KindPosition = "track-point"
	KindBuffer   = "wind-buffer"
	KindCone     = "uncertainty-cone"
)

// Feature is a GeoJSON feature of a cyclone's geometry, in draw order.
type Feature struct {
	Type       string          `json:"type"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties FeatureDetails  `json:"properties"`
}

// FeatureDetails describes a feature.
type FeatureDetails struct {
	Kind string `json:"kind"`
	// Stage is the storm class along a track: TD, TS or HU.
	Stage string `json:"stage,omitempty"`
	// Label is GDACS's own: "60 km/h", "02/11 12:00 UTC".
	Label string `json:"label,omitempty"`
	// WindKmh is the wind speed a buffer encloses.
	WindKmh int `json:"windKmh,omitempty"`
	// Band is the buffer's GDACS colour band: green, orange or red.
	Band string `json:"band,omitempty"`
	// Time is when a track position applies.
	Time *time.Time `json:"time,omitempty"`
	// Forecast marks a position after the latest advisory.
	Forecast bool `json:"forecast"`
}

type wireGeometry struct {
	Features []struct {
		Geometry   json.RawMessage `json:"geometry"`
		Properties struct {
			Class        string `json:"Class"`
			PolygonLabel string `json:"polygonlabel"`
			Forecast     *bool  `json:"forecast"`
		} `json:"properties"`
	} `json:"features"`
}

// DecodeGeometry reads an event's geometry and keeps what the map draws: the
// observed and forecast track, a point for each track position, the overall
// wind buffers and the uncertainty cone. The per-step wind polygons GDACS also
// sends are dropped: the overall buffers already enclose them.
func DecodeGeometry(payload []byte, latest time.Time) ([]Feature, error) {
	var wire wireGeometry
	if err := json.Unmarshal(payload, &wire); err != nil {
		return nil, fmt.Errorf("decode cyclone geometry: %w", err)
	}
	if wire.Features == nil {
		return nil, errors.New("cyclone geometry has no features")
	}
	features := []Feature{}
	for _, item := range wire.Features {
		class, label := item.Properties.Class, strings.TrimSpace(item.Properties.PolygonLabel)
		switch {
		case class == "Poly_Cones":
			features = append(features, Feature{Type: "Feature", Geometry: item.Geometry, Properties: FeatureDetails{Kind: KindCone, Label: label}})
		case strings.HasPrefix(class, "Poly_") && strings.HasSuffix(label, "km/h"):
			wind, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(label, "km/h")))
			features = append(features, Feature{Type: "Feature", Geometry: item.Geometry, Properties: FeatureDetails{
				Kind: KindBuffer, Label: label, WindKmh: wind, Band: strings.ToLower(strings.TrimPrefix(class, "Poly_")),
			}})
		case strings.HasPrefix(class, "Line_"):
			forecast := item.Properties.Forecast != nil && *item.Properties.Forecast
			kind := KindTrack
			if forecast {
				kind = KindForecast
			}
			features = append(features, Feature{Type: "Feature", Geometry: item.Geometry, Properties: FeatureDetails{
				Kind: kind, Stage: label, Forecast: forecast,
			}})
		case strings.HasPrefix(class, "Point_Polygon_Point"):
			geometry, err := polygonCentre(item.Geometry)
			if err != nil {
				continue
			}
			at := positionTime(label, latest)
			details := FeatureDetails{Kind: KindPosition, Label: label, Time: at}
			if at != nil && at.After(latest) {
				details.Forecast = true
			}
			features = append(features, Feature{Type: "Feature", Geometry: geometry, Properties: details})
		}
	}
	return features, nil
}

// polygonCentre turns the small circle GDACS draws around a track position
// back into its point.
func polygonCentre(raw json.RawMessage) (json.RawMessage, error) {
	var polygon struct {
		Coordinates [][][2]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal(raw, &polygon); err != nil || len(polygon.Coordinates) == 0 || len(polygon.Coordinates[0]) == 0 {
		return nil, errors.New("not a polygon")
	}
	west, east, south, north := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, position := range polygon.Coordinates[0] {
		west, east = math.Min(west, position[0]), math.Max(east, position[0])
		south, north = math.Min(south, position[1]), math.Max(north, position[1])
	}
	round := func(value float64) float64 { return math.Round(value*100) / 100 }
	return json.Marshal(map[string]any{"type": "Point", "coordinates": [2]float64{round((west + east) / 2), round((south + north) / 2)}})
}

// positionTime reads a track label, "02/11 12:00 UTC" (day/month, no year),
// taking the year nearest the event's latest advisory.
func positionTime(label string, latest time.Time) *time.Time {
	at, err := time.Parse("02/01 15:04 MST", label)
	if err != nil {
		return nil
	}
	best := time.Date(latest.Year(), at.Month(), at.Day(), at.Hour(), at.Minute(), 0, 0, time.UTC)
	for _, year := range []int{latest.Year() - 1, latest.Year() + 1} {
		candidate := time.Date(year, at.Month(), at.Day(), at.Hour(), at.Minute(), 0, 0, time.UTC)
		if candidate.Sub(latest).Abs() < best.Sub(latest).Abs() {
			best = candidate
		}
	}
	return &best
}
