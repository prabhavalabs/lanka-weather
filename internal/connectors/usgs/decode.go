// Package usgs collects earthquakes from the USGS FDSN event service. Events
// are revised after they are first published: magnitudes and places change,
// events merge, and some are deleted. So the whole window is read on every
// run and stored as USGS now has it.
package usgs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// SourceName identifies the source in docs/data-sources.md.
const SourceName = "usgs-earthquakes"

// Attribution credits the source.
const Attribution = "U.S. Geological Survey"

// statusDeleted is what includedeleted=true reports for a withdrawn event.
const statusDeleted = "deleted"

// Event is one seismic event.
type Event struct {
	ID           string
	Magnitude    float64
	MagType      string
	Place        string
	OccurredAt   time.Time
	UpdatedAt    time.Time
	Lon          float64
	Lat          float64
	DepthKm      float64
	URL          string
	Alert        string
	Tsunami      bool
	Significance int
	Type         string
	Status       string
}

type wireResponse struct {
	Type     string `json:"type"`
	Metadata struct {
		Status int `json:"status"`
	} `json:"metadata"`
	Features []struct {
		ID         string `json:"id"`
		Properties struct {
			Mag     *float64 `json:"mag"`
			MagType string   `json:"magType"`
			Place   string   `json:"place"`
			Time    *int64   `json:"time"`
			Updated *int64   `json:"updated"`
			URL     string   `json:"url"`
			Alert   *string  `json:"alert"`
			Tsunami int      `json:"tsunami"`
			Sig     int      `json:"sig"`
			Type    string   `json:"type"`
			Status  string   `json:"status"`
		} `json:"properties"`
		Geometry *struct {
			Coordinates []*float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

// Decode reads a GeoJSON response. It returns the events to keep, and the
// identifiers USGS says are deleted. A feature missing a magnitude, time or
// position is skipped and counted: it cannot be drawn honestly.
func Decode(payload []byte) (events []Event, deleted []string, skipped int, err error) {
	var wire wireResponse
	if err := json.Unmarshal(payload, &wire); err != nil {
		return nil, nil, 0, fmt.Errorf("decode earthquakes: %w", err)
	}
	if wire.Type != "FeatureCollection" {
		return nil, nil, 0, errors.New("earthquake response is not a feature collection")
	}
	for _, feature := range wire.Features {
		properties := feature.Properties
		if feature.ID == "" {
			skipped++
			continue
		}
		if properties.Status == statusDeleted {
			deleted = append(deleted, feature.ID)
			continue
		}
		if properties.Mag == nil || properties.Time == nil || feature.Geometry == nil ||
			len(feature.Geometry.Coordinates) < 3 || feature.Geometry.Coordinates[0] == nil ||
			feature.Geometry.Coordinates[1] == nil || feature.Geometry.Coordinates[2] == nil {
			skipped++
			continue
		}
		// Quarry blasts and the like are not earthquakes.
		if !strings.EqualFold(properties.Type, "earthquake") {
			skipped++
			continue
		}
		occurred := time.UnixMilli(*properties.Time).UTC()
		updated := occurred
		if properties.Updated != nil {
			updated = time.UnixMilli(*properties.Updated).UTC()
		}
		link := properties.URL
		if parsed, err := url.Parse(link); err != nil || parsed.Scheme != "https" {
			link = ""
		}
		alert := ""
		if properties.Alert != nil {
			alert = *properties.Alert
		}
		events = append(events, Event{
			ID: feature.ID, Magnitude: *properties.Mag, MagType: properties.MagType, Place: properties.Place,
			OccurredAt: occurred, UpdatedAt: updated,
			Lon: *feature.Geometry.Coordinates[0], Lat: *feature.Geometry.Coordinates[1], DepthKm: *feature.Geometry.Coordinates[2],
			URL: link, Alert: alert, Tsunami: properties.Tsunami != 0, Significance: properties.Sig,
			Type: properties.Type, Status: properties.Status,
		})
	}
	return events, deleted, skipped, nil
}

// Merge combines the answers of several queries; an event in more than one
// keeps its most recently updated version. Deleted identifiers win.
func Merge(deleted []string, lists ...[]Event) []Event {
	gone := make(map[string]bool, len(deleted))
	for _, id := range deleted {
		gone[id] = true
	}
	index := map[string]int{}
	var out []Event
	for _, list := range lists {
		for _, event := range list {
			if gone[event.ID] {
				continue
			}
			if at, ok := index[event.ID]; ok {
				if event.UpdatedAt.After(out[at].UpdatedAt) {
					out[at] = event
				}
				continue
			}
			index[event.ID] = len(out)
			out = append(out, event)
		}
	}
	return out
}
