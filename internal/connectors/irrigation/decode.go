// Package irrigation collects river water levels published by the Irrigation
// Department of Sri Lanka's Hydrology and Disaster Management Division.
//
// The department's gauge dashboard reads a public ArcGIS feature service.
// Each feature is one reading at one gauging station: the level, the
// rainfall, the station's alert, minor-flood and major-flood thresholds, and
// the station's position.
package irrigation

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// SourceName identifies this source in docs/data-sources.md and source_runs.
const SourceName = "irrigation-gauges"

// Station is a gauging station as described by its latest reading.
type Station struct {
	ID              string
	Name            string
	Basin           string
	Lon, Lat        float64
	Unit            string
	AlertLevel      *float64
	MinorFloodLevel *float64
	MajorFloodLevel *float64
	// ObservedAt is the time of the reading this description came from.
	ObservedAt time.Time
}

// Reading is one water-level observation.
type Reading struct {
	StationID  string
	ObservedAt time.Time
	WaterLevel float64
	RainfallMM *float64
}

// Page is one decoded response page.
type Page struct {
	Stations []Station
	Readings []Reading
	// More reports that the service truncated the page and more records follow.
	More bool
	// Skipped counts features that could not be used (no gauge or level).
	Skipped int
}

// feetStations lists gauges whose levels and thresholds the department
// publishes in feet; all others are in metres. Nagalagam Street on the Kelani
// Ganga in Colombo has historically been read in feet (minor flood 5 ft, major
// flood 7 ft).
var feetStations = map[string]bool{"nagalagam-street": true}

type response struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	ExceededTransferLimit bool `json:"exceededTransferLimit"`
	Features              []struct {
		Attributes struct {
			Basin        string   `json:"basin"`
			Gauge        string   `json:"gauge"`
			WaterLevel   *float64 `json:"water_level"`
			RainFall     *float64 `json:"rain_fall"`
			CreationDate *int64   `json:"CreationDate"`
			AlertPull    *float64 `json:"alertpull"`
			MinorPull    *float64 `json:"minorpull"`
			MajorPull    *float64 `json:"majorpull"`
		} `json:"attributes"`
		Geometry *struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"geometry"`
	} `json:"features"`
}

// Sri Lanka with a margin; positions outside it are treated as missing.
const minLon, maxLon, minLat, maxLat = 79.0, 82.5, 5.5, 10.5

// Decode turns one page of the feature service's JSON response into stations
// and readings. It does no I/O.
func Decode(payload []byte) (Page, error) {
	var body response
	if err := json.Unmarshal(payload, &body); err != nil {
		return Page{}, fmt.Errorf("decode gauge response: %w", err)
	}
	if body.Error != nil {
		return Page{}, fmt.Errorf("gauge service error %d: %s", body.Error.Code, body.Error.Message)
	}

	page := Page{More: body.ExceededTransferLimit}
	latest := map[string]int{}
	for _, feature := range body.Features {
		attributes := feature.Attributes
		name := strings.TrimSpace(attributes.Gauge)
		id := StationID(name)
		if id == "" || attributes.WaterLevel == nil || attributes.CreationDate == nil || isBad(*attributes.WaterLevel) {
			page.Skipped++
			continue
		}
		observed := time.UnixMilli(*attributes.CreationDate).UTC()
		page.Readings = append(page.Readings, Reading{
			StationID:  id,
			ObservedAt: observed,
			WaterLevel: *attributes.WaterLevel,
			RainfallMM: nonNegative(attributes.RainFall),
		})

		if feature.Geometry == nil || !inSriLanka(feature.Geometry.X, feature.Geometry.Y) {
			continue
		}
		unit := "m"
		if feetStations[id] {
			unit = "ft"
		}
		station := Station{
			ID: id, Name: name, Basin: strings.TrimSpace(attributes.Basin),
			Lon: feature.Geometry.X, Lat: feature.Geometry.Y, Unit: unit,
			ObservedAt: observed,
		}
		station.AlertLevel, station.MinorFloodLevel, station.MajorFloodLevel =
			thresholds(attributes.AlertPull, attributes.MinorPull, attributes.MajorPull)
		if index, seen := latest[id]; seen {
			if observed.After(page.Stations[index].ObservedAt) {
				page.Stations[index] = station
			}
			continue
		}
		latest[id] = len(page.Stations)
		page.Stations = append(page.Stations, station)
	}
	return page, nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// StationID derives a stable identifier from a gauge name, for example
// "Kalawellawa (Millakanda)" becomes "kalawellawa-millakanda".
func StationID(gauge string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(gauge), "-"), "-")
}

// thresholds returns the three levels only when they are present, positive
// and in increasing order; the service reports missing thresholds as null or
// zero, and partial or inverted sets cannot classify a reading.
func thresholds(alert, minor, major *float64) (*float64, *float64, *float64) {
	if alert == nil || minor == nil || major == nil {
		return nil, nil, nil
	}
	if *alert <= 0 || *minor < *alert || *major < *minor || isBad(*alert) || isBad(*major) {
		return nil, nil, nil
	}
	a, b, c := *alert, *minor, *major
	return &a, &b, &c
}

func nonNegative(value *float64) *float64 {
	if value == nil || *value < 0 || isBad(*value) {
		return nil
	}
	v := *value
	return &v
}

func isBad(value float64) bool { return math.IsNaN(value) || math.IsInf(value, 0) }

func inSriLanka(lon, lat float64) bool {
	return lon >= minLon && lon <= maxLon && lat >= minLat && lat <= maxLat
}

// ErrNoData is returned when a fetch yields no usable readings at all.
var ErrNoData = errors.New("gauge service returned no usable readings")
