// Package waterlevels collects reservoir storage and extra river gauges from
// the public data API of Sri Lanka Water Levels (waterlevels.lk), an
// independent community site by @shakeeb_x. It compiles the Irrigation
// Department's reservoir sheet and river gauges, the Mahaweli Authority's daily
// report and RiverNet, the Disaster Management Centre's sensors (ADR 0013).
//
// The site states its data is free and allows any website to read it, and its
// robots.txt allows /api/levels and /api/reservoirs. It gives no formal
// licence, so everything taken from it is credited to it and to the agency
// behind it.
package waterlevels

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

const (
	// ReservoirsSource and GaugesSource identify the two collectors in
	// docs/data-sources.md and source_runs.
	ReservoirsSource = "waterlevels-lk-reservoirs"
	GaugesSource     = "waterlevels-lk-gauges"
	CanalsSource     = "waterlevels-lk-canals"

	// ReservoirsURL, LevelsURL and CanalsURL are the endpoints robots.txt allows.
	ReservoirsURL = "https://waterlevels.lk/api/reservoirs"
	LevelsURL     = "https://waterlevels.lk/api/levels"
	CanalsURL     = "https://waterlevels.lk/api/canals"

	// Attribution is the credit for anything read from the site.
	Attribution = "Sri Lanka Water Levels (waterlevels.lk)"
	// RiverNet is the credit for gauges the site reads from RiverNet.
	RiverNet = "RiverNet, DMC / Leecom SCADA (rivernet.lk)"
)

// The statuses the source gives a reservoir.
const (
	StatusSpilling  = "spilling"
	StatusReleasing = "releasing"
	StatusHolding   = "holding"
	StatusNoData    = "nodata"
)

// ColomboZone is Sri Lanka's time zone, in which the source's days are written.
var ColomboZone = time.FixedZone("Asia/Colombo", 5*3600+30*60)

// Flow is water leaving a reservoir. A null flow means unknown, never zero.
type Flow struct {
	// Kind is spill, outlets or downstream as the source writes it. A
	// Mahaweli Authority downstream flow can include power-station flow.
	Kind   string
	Cusecs *float64
	Agency string
	Day    *time.Time
}

// Reservoir is one reservoir with its reading on the day of the response.
type Reservoir struct {
	ID       string
	Name     string
	District *string
	River    *string
	// Kind is major, medium or mahaweli.
	Kind string
	// Status is spilling, releasing, holding or nodata; an unknown value is
	// treated as nodata rather than guessed.
	Status         string
	StoragePercent *float64
	StorageFresh   *bool
	Flow           *Flow
	ReadingDay     *time.Time
	// Agency is the agency that reported the reading.
	Agency *string
	URL    *string
}

// ReservoirSnapshot is one decoded /api/reservoirs response.
type ReservoirSnapshot struct {
	GeneratedAt time.Time
	Reservoirs  []Reservoir
	// Skipped counts records that could not be used (no identifier or name).
	Skipped int
}

type reservoirResponse struct {
	GeneratedAt string `json:"generatedAt"`
	Reservoirs  []struct {
		ID             string   `json:"id"`
		Name           string   `json:"name"`
		River          *string  `json:"river"`
		District       *string  `json:"district"`
		Type           string   `json:"type"`
		Status         string   `json:"status"`
		StoragePercent *float64 `json:"storagePercent"`
		StorageFresh   *bool    `json:"storageFresh"`
		Flow           *struct {
			Kind   string   `json:"kind"`
			Cusecs *float64 `json:"cusecs"`
			Source string   `json:"source"`
			Day    string   `json:"day"`
		} `json:"flow"`
		ReadingDay string  `json:"readingDay"`
		Source     *string `json:"source"`
		URL        *string `json:"url"`
	} `json:"reservoirs"`
}

// DecodeReservoirs turns an /api/reservoirs response into reservoirs. It does
// no I/O.
func DecodeReservoirs(payload []byte) (ReservoirSnapshot, error) {
	var body reservoirResponse
	if err := json.Unmarshal(payload, &body); err != nil {
		return ReservoirSnapshot{}, fmt.Errorf("decode reservoirs: %w", err)
	}
	generated, err := time.Parse(time.RFC3339, body.GeneratedAt)
	if err != nil {
		return ReservoirSnapshot{}, errors.New("reservoirs response has no generatedAt")
	}
	snapshot := ReservoirSnapshot{GeneratedAt: generated.UTC()}
	for _, record := range body.Reservoirs {
		id, name := strings.TrimSpace(record.ID), strings.TrimSpace(record.Name)
		if id == "" || name == "" {
			snapshot.Skipped++
			continue
		}
		reservoir := Reservoir{
			ID: id, Name: name,
			District: optional(record.District), River: optional(record.River),
			Kind:           strings.TrimSpace(record.Type),
			Status:         reservoirStatus(record.Status),
			StoragePercent: finite(record.StoragePercent),
			StorageFresh:   record.StorageFresh,
			ReadingDay:     parseDay(record.ReadingDay),
			Agency:         optional(record.Source),
			URL:            optional(record.URL),
		}
		if record.Flow != nil {
			reservoir.Flow = &Flow{
				Kind: strings.TrimSpace(record.Flow.Kind), Cusecs: nonNegative(record.Flow.Cusecs),
				Agency: strings.TrimSpace(record.Flow.Source), Day: parseDay(record.Flow.Day),
			}
		}
		snapshot.Reservoirs = append(snapshot.Reservoirs, reservoir)
	}
	return snapshot, nil
}

func reservoirStatus(raw string) string {
	switch status := strings.TrimSpace(raw); status {
	case StatusSpilling, StatusReleasing, StatusHolding:
		return status
	default:
		return StatusNoData
	}
}

// NextHour is the site's forecast of a gauge's level about an hour after
// IssuedAt, in feet. It is experimental and never an official status.
type NextHour struct {
	IssuedAt            time.Time
	LevelFt             float64
	LowFt, HighFt       *float64
	ChangeFt            *float64
	Direction, Crossing *string
	Method              *string
}

// Report is one station in an /api/levels response.
type Report struct {
	// Slug is the last part of the source's identifier, usually the
	// station's own spelling ("rathnapura" in "kalu-ganga/rathnapura").
	Slug     string
	Name     string
	River    string
	Lon, Lat float64
	// Unit is m, ft or unknown: the unit the agency publishes in.
	Unit string
	// Level is the figure as the agency released it, in Unit.
	Level *float64
	// AlertLevel, MinorFloodLevel and MajorFloodLevel are the source's marks
	// converted from feet to Unit; all three are set, or none.
	AlertLevel, MinorFloodLevel, MajorFloodLevel *float64
	ObservedAt                                   *time.Time
	URL                                          *string
	NextHour                                     *NextHour
}

// LevelsSnapshot is one decoded /api/levels response.
type LevelsSnapshot struct {
	GeneratedAt time.Time
	Reports     []Report
	// Skipped counts records without a name or a position in Sri Lanka.
	Skipped int
}

type levelsResponse struct {
	GeneratedAt string `json:"generatedAt"`
	Stations    []struct {
		ID        string   `json:"id"`
		Station   string   `json:"station"`
		River     string   `json:"river"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		Published *struct {
			Level *float64 `json:"level"`
			Unit  string   `json:"unit"`
		} `json:"published"`
		Marks *struct {
			Alert *float64 `json:"alert"`
			Minor *float64 `json:"minor"`
			Major *float64 `json:"major"`
		} `json:"marks"`
		ObservedAt string  `json:"observedAt"`
		URL        *string `json:"url"`
		NextHour   *struct {
			LevelFt   *float64 `json:"level_ft"`
			LowFt     *float64 `json:"low_ft"`
			HighFt    *float64 `json:"high_ft"`
			ChangeFt  *float64 `json:"change_ft"`
			Direction *string  `json:"direction"`
			Crossing  *string  `json:"crossing"`
			IssuedAt  string   `json:"issuedAt"`
			Method    *string  `json:"method"`
		} `json:"nextHour"`
	} `json:"stations"`
}

// Sri Lanka with a margin, as for the Irrigation Department's gauges.
const minLon, maxLon, minLat, maxLat = 79.0, 82.5, 5.5, 10.5

// DecodeLevels turns an /api/levels response into reports. It does no I/O.
func DecodeLevels(payload []byte) (LevelsSnapshot, error) {
	var body levelsResponse
	if err := json.Unmarshal(payload, &body); err != nil {
		return LevelsSnapshot{}, fmt.Errorf("decode levels: %w", err)
	}
	generated, err := time.Parse(time.RFC3339, body.GeneratedAt)
	if err != nil {
		return LevelsSnapshot{}, errors.New("levels response has no generatedAt")
	}
	snapshot := LevelsSnapshot{GeneratedAt: generated.UTC()}
	for _, record := range body.Stations {
		name := strings.TrimSpace(record.Station)
		if name == "" || record.Latitude == nil || record.Longitude == nil ||
			*record.Longitude < minLon || *record.Longitude > maxLon ||
			*record.Latitude < minLat || *record.Latitude > maxLat {
			snapshot.Skipped++
			continue
		}
		report := Report{
			Slug: lastSegment(record.ID), Name: name, River: strings.TrimSpace(record.River),
			Lon: *record.Longitude, Lat: *record.Latitude, Unit: UnitUnknown,
			URL: optional(record.URL),
		}
		if record.Published != nil {
			report.Unit = publishedUnit(record.Published.Unit)
			report.Level = roundLevel(finite(record.Published.Level))
		}
		if record.Marks != nil {
			report.AlertLevel, report.MinorFloodLevel, report.MajorFloodLevel = Marks(
				record.Marks.Alert, record.Marks.Minor, record.Marks.Major, report.Unit)
		}
		if observed, err := time.Parse(time.RFC3339, record.ObservedAt); err == nil {
			utc := observed.UTC()
			report.ObservedAt = &utc
		}
		if report.ObservedAt == nil {
			report.Level = nil
		}
		if next := record.NextHour; next != nil && next.LevelFt != nil && !isBad(*next.LevelFt) {
			if issued, err := time.Parse(time.RFC3339, next.IssuedAt); err == nil {
				report.NextHour = &NextHour{
					IssuedAt: issued.UTC(), LevelFt: *next.LevelFt,
					LowFt: finite(next.LowFt), HighFt: finite(next.HighFt), ChangeFt: finite(next.ChangeFt),
					Direction: optional(next.Direction), Crossing: optional(next.Crossing), Method: optional(next.Method),
				}
			}
		}
		snapshot.Reports = append(snapshot.Reports, report)
	}
	return snapshot, nil
}

// UnitUnknown is a station's unit when the agency publishes none.
const UnitUnknown = "unknown"

func publishedUnit(raw string) string {
	switch unit := strings.TrimSpace(raw); unit {
	case "m", "ft":
		return unit
	default:
		return UnitUnknown
	}
}

const metresPerFoot = 0.3048

// FromFeet converts a mark from the source's feet into a station's unit. The
// source rounds converted marks to 0.01 ft, so a mark in metres comes back to
// the agency's figure when rounded to the centimetre: 13.12 ft is 4.00 m.
// With no known unit there is nothing to convert to.
func FromFeet(feet float64, unit string) (float64, bool) {
	switch unit {
	case "ft":
		return round(feet, 2), true
	case "m":
		return round(feet*metresPerFoot, 2), true
	default:
		return 0, false
	}
}

// Marks converts the source's alert, minor-flood and major-flood marks to a
// station's unit. They are used only when all three are present, positive and
// increasing, as for the Irrigation Department's own thresholds; otherwise the
// station has no flood verdict.
func Marks(alert, minor, major *float64, unit string) (*float64, *float64, *float64) {
	if alert == nil || minor == nil || major == nil || isBad(*alert) || isBad(*minor) || isBad(*major) {
		return nil, nil, nil
	}
	a, okA := FromFeet(*alert, unit)
	b, okB := FromFeet(*minor, unit)
	c, okC := FromFeet(*major, unit)
	if !okA || !okB || !okC || a <= 0 || b < a || c < b {
		return nil, nil, nil
	}
	return &a, &b, &c
}

// stationAliases maps the identifiers of spellings the feed uses to the
// identifier our Irrigation Department collector gives the same gauge, so a
// station is never stored twice. "Ratnapura" on the Kalu Ganga is the
// department's "Rathnapura": the same gauge at the same position, and the feed's
// own identifier is kalu-ganga/rathnapura.
var stationAliases = map[string]string{"ratnapura": "rathnapura"}

// riverNetStations are the gauges whose page on the site credits "RiverNet –
// DMC / Leecom SCADA (rivernet.lk)", read from each page on 2026-11-02. The API
// has no per-station credit. Gauges not listed name no agency beyond the site.
var riverNetStations = map[string]bool{
	"denawaka-ganga-pelmadulla": true,
	"galathura-oya-ayagama":     true,
	"kukule-ganga-kalawana":     true,
	"kuru-ganga-kuruvita":       true,
	"lavanai-aru-bridge":        true,
	"maalayarkattu-bridge":      true,
	"mapitigama":                true,
	"miriswatta":                true,
	"mullamunai-bridge":         true,
	"niriella-ganga-elapatha":   true,
	"pugoda":                    true,
	"pulipanchakal-bridge":      true,
	"punani-anicut":             true,
	"rajagalathenna-bridge":     true,
	"vadamunai-bridge":          true,
	"wattala":                   true,
	"wey-ganga-kahawaththa":     true,
}

// Upstream is the credit for who measured a gauge, or "" when the site names
// nobody.
func Upstream(id string) string {
	if riverNetStations[id] {
		return RiverNet
	}
	return ""
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// stationSlug derives an identifier from a name the way the Irrigation
// Department collector does, so the same name gives the same identifier.
func stationSlug(name string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// StationID is the identifier a report's station is stored under.
func StationID(report Report) string {
	id := stationSlug(report.Name)
	if canonical, ok := stationAliases[id]; ok {
		return canonical
	}
	return id
}

func lastSegment(id string) string {
	id = strings.TrimSpace(id)
	if index := strings.LastIndex(id, "/"); index >= 0 {
		return id[index+1:]
	}
	return id
}

func optional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseDay(raw string) *time.Time {
	day, err := time.ParseInLocation(time.DateOnly, strings.TrimSpace(raw), time.UTC)
	if err != nil {
		return nil
	}
	return &day
}

func finite(value *float64) *float64 {
	if value == nil || isBad(*value) {
		return nil
	}
	v := *value
	return &v
}

func nonNegative(value *float64) *float64 {
	if value == nil || isBad(*value) || *value < 0 {
		return nil
	}
	v := *value
	return &v
}

// roundLevel removes the floating-point noise some published levels carry
// (2.6000000000000014) while keeping every digit an agency reads: millimetres.
func roundLevel(value *float64) *float64 {
	if value == nil {
		return nil
	}
	v := round(*value, 3)
	return &v
}

func round(value float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(value*scale) / scale
}

func isBad(value float64) bool { return math.IsNaN(value) || math.IsInf(value, 0) }
