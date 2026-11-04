// Package metstations collects Department of Meteorology observations from
// nuuuwan's scraped copies, since www.meteo.gov.lk has no API and does not
// answer reliably (ADR 0012):
//
//   - lk_weather_3h: observations every three hours at the synoptic stations;
//   - weather_lk: the daily report, with rain at about 60 stations.
//
// Both are copies that can stop being updated, so every reading keeps the time
// it was observed and the service marks old readings out of date.
package metstations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

// Source names, as listed in docs/data-sources.md.
const (
	SourceThreeHourly = "nuuuwan-lk-weather-3h"
	SourceDaily       = "nuuuwan-weather-lk"
)

// Attribution credits the department, whose readings these are.
const Attribution = "Department of Meteorology, Sri Lanka, via nuuuwan/lk_weather_3h and nuuuwan/weather_lk"

// Addresses of the two feeds and the list of geocoded places.
const (
	EndpointThreeHourly = "https://raw.githubusercontent.com/nuuuwan/lk_weather_3h/main/data/alert_data.json"
	EndpointDaily       = "https://raw.githubusercontent.com/nuuuwan/weather_lk/data/flat.json"
	EndpointPlaces      = "https://raw.githubusercontent.com/nuuuwan/weather_lk/data/flat_places.json"
)

// LocalZone is Sri Lanka Standard Time. Both feeds write local times. The
// 3-hourly README heads its column "Time (UT)", but WMO's synoptic hours fall
// on the hour UTC, so they fall at minute 30 only in a UTC+05:30 clock, and
// the coolest reading is at 05:30 and the warmest at 14:30.
// TestThreeHourlyTimesAreLocal holds this to a recorded file.
var LocalZone = time.FixedZone("Asia/Colombo", 5*3600+30*60)

// synopticSlots are the local hours of the 3-hourly report: every three hours
// from 00:00 UTC, each at minute 30 local.
var synopticSlots = map[int]bool{2: true, 5: true, 8: true, 11: true, 14: true, 17: true, 20: true, 23: true}

// Reading3h is one 3-hourly observation. RainMm is the rain in the three hours
// ending at ObservedAt.
type Reading3h struct {
	StationID   string
	ObservedAt  time.Time
	RainMm      *float64
	TempC       *float64
	HumidityPct *float64
}

type wire3hReading struct {
	RainMm *float64 `json:"rain_mm"`
	TempC  *float64 `json:"temp_c"`
	RH     *float64 `json:"rh"`
}

type wire3h struct {
	Event     string                                         `json:"event"`
	EventData map[string]map[string]map[string]wire3hReading `json:"event_data"`
}

// ThreeHourly is a decoded 3-hourly file.
type ThreeHourly struct {
	Readings []Reading3h
	// Unknown lists station names no known station matches.
	Unknown []string
	// Dropped counts times that are not a report slot, or days that are not dates.
	Dropped int
	// Newest is the newest observation in the file, before any cut-off.
	Newest time.Time
}

// DecodeThreeHourly reads the 3-hourly file, keeping readings at or after
// `since`. It does no I/O.
func DecodeThreeHourly(payload []byte, since time.Time) (ThreeHourly, error) {
	var wire wire3h
	if err := json.Unmarshal(payload, &wire); err != nil {
		return ThreeHourly{}, fmt.Errorf("decode 3-hourly file: %w", err)
	}
	if wire.Event != "weather_report_3h" || wire.EventData == nil {
		return ThreeHourly{}, errors.New("3-hourly file is not a weather_report_3h")
	}
	var out ThreeHourly
	for name, days := range wire.EventData {
		id := ID(name)
		if _, known := KnownByID(id); !known {
			out.Unknown = append(out.Unknown, name)
			continue
		}
		for day, times := range days {
			for clock, reading := range times {
				observed, ok := parseSlot(day, clock)
				if !ok {
					out.Dropped++
					continue
				}
				if observed.After(out.Newest) {
					out.Newest = observed
				}
				if observed.Before(since) {
					continue
				}
				out.Readings = append(out.Readings, Reading3h{
					StationID: id, ObservedAt: observed.UTC(),
					RainMm:      inRange(reading.RainMm, 0, 1000),
					TempC:       inRange(reading.TempC, -5, 50),
					HumidityPct: humidity(reading.RH),
				})
			}
		}
	}
	return out, nil
}

// parseSlot reads a day and time as written in the file. A few readings are
// timed some minutes after the half hour, as when a scrape stamped them one by
// one, and belong to the report of that half hour.
func parseSlot(day, clock string) (time.Time, bool) {
	if len(clock) != 4 {
		return time.Time{}, false
	}
	hour, hourErr := strconv.Atoi(clock[:2])
	minute, minuteErr := strconv.Atoi(clock[2:])
	if hourErr != nil || minuteErr != nil || minute < 30 || minute > 59 || !synopticSlots[hour] {
		return time.Time{}, false
	}
	date, err := time.ParseInLocation("20060102", day, LocalZone)
	if err != nil {
		return time.Time{}, false
	}
	return date.Add(time.Duration(hour)*time.Hour + 30*time.Minute), true
}

// inRange drops a value outside what the quantity can be.
func inRange(value *float64, low, high float64) *float64 {
	if value == nil || math.IsNaN(*value) || *value < low || *value > high {
		return nil
	}
	return value
}

// humidity turns the file's fraction into a percentage.
func humidity(fraction *float64) *float64 {
	if inRange(fraction, 0, 1) == nil {
		return nil
	}
	percent := math.Round(*fraction * 100)
	return &percent
}

// ReadingDaily is one station's entry in a daily report: the rain over the 24
// hours ending at ObservedAt (08:30 local on Date), and temperature extremes.
type ReadingDaily struct {
	StationID  string
	Name       string
	Date       time.Time
	ObservedAt time.Time
	RainMm     *float64
	TempMinC   *float64
	TempMaxC   *float64
}

type wireDaily struct {
	ID       string   `json:"id"`
	TimeUT   float64  `json:"time_ut"`
	RainMm   *float64 `json:"rain_mm"`
	TempMinC *float64 `json:"temp_min_c"`
	TempMaxC *float64 `json:"temp_max_c"`
}

// Daily is a decoded stretch of the daily feed.
type Daily struct {
	Readings []ReadingDaily
	// Duplicates counts later entries for a station and day. Reports list a few
	// names twice for different gauges, with no way to tell them apart, so the
	// first entry stands.
	Duplicates int
	// Newest is the newest report's time.
	Newest time.Time
}

// reportHour and reportMinute are when a daily report's 24 hours end: rain
// totals agree with the sum of the eight 3-hourly readings to that time
// (Galle 14.6 mm and Anuradhapura 12.2 mm on 23 October 2026).
const (
	reportHour   = 8
	reportMinute = 30
)

// DecodeDaily reads the end of the daily file, a list of rows in time order
// that may start mid-row after a range request, keeping reports at or after
// `since`. A row's time_ut is local midnight of the report's date. It does no
// I/O.
func DecodeDaily(payload []byte, since time.Time) (Daily, error) {
	rows, err := parseDailyRows(payload)
	if err != nil {
		return Daily{}, err
	}
	var out Daily
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		id := ID(row.ID)
		if id == "" || row.TimeUT <= 0 {
			continue
		}
		local := time.Unix(int64(row.TimeUT), 0).In(LocalZone)
		date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
		observed := time.Date(local.Year(), local.Month(), local.Day(), reportHour, reportMinute, 0, 0, LocalZone).UTC()
		if observed.After(out.Newest) {
			out.Newest = observed
		}
		if observed.Before(since) {
			continue
		}
		key := id + date.Format(time.DateOnly)
		if seen[key] {
			out.Duplicates++
			continue
		}
		seen[key] = true
		out.Readings = append(out.Readings, ReadingDaily{
			StationID: id, Name: row.ID, Date: date, ObservedAt: observed,
			RainMm:   inRange(row.RainMm, 0, 1000),
			TempMinC: inRange(row.TempMinC, -5, 50),
			TempMaxC: inRange(row.TempMaxC, -5, 50),
		})
	}
	return out, nil
}

// parseDailyRows reads a whole list, or the end of one from a range request:
// the part-row at its start is dropped by starting at the first complete row.
func parseDailyRows(payload []byte) ([]wireDaily, error) {
	body := bytes.TrimSpace(payload)
	if len(body) == 0 {
		return nil, errors.New("daily file is empty")
	}
	if body[0] != '[' {
		start := firstRow(body)
		if start < 0 {
			return nil, errors.New("daily file holds no complete row")
		}
		body = append([]byte("["), body[start:]...)
	}
	var rows []wireDaily
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode daily file: %w", err)
	}
	return rows, nil
}

// firstRow finds the first row that starts within body: an opening brace
// followed by the "id" key.
func firstRow(body []byte) int {
	for offset := 0; offset < len(body); {
		next := bytes.IndexByte(body[offset:], '{')
		if next < 0 {
			return -1
		}
		at := offset + next
		if bytes.HasPrefix(bytes.TrimLeft(body[at+1:], " \t\r\n"), []byte(`"id"`)) {
			return at
		}
		offset = at + 1
	}
	return -1
}

// Position is where a geocoded place is.
type Position struct{ Lat, Lon float64 }

type wirePlace struct {
	ID     string     `json:"id"`
	LatLng [2]float64 `json:"lat_lng"`
}

// defaultPositionNames is how many names may share a position before it is
// taken to be the geocoder's answer for "somewhere in Sri Lanka". Names that
// could not be found all get the centre of the country (7.873054, 80.771797).
const defaultPositionNames = 3

// DecodePlaces reads the geocoded places, by station identifier, keeping only
// usable positions. The list holds names geocoded to (0, 0), to the centre of
// the country and to nowhere near Sri Lanka, and those are left out. It does
// no I/O.
func DecodePlaces(payload []byte) (map[string]Position, error) {
	var places []wirePlace
	if err := json.Unmarshal(payload, &places); err != nil {
		return nil, fmt.Errorf("decode places: %w", err)
	}
	shared := map[Position]map[string]bool{}
	for _, place := range places {
		position := Position{Lat: round(place.LatLng[0]), Lon: round(place.LatLng[1])}
		if shared[position] == nil {
			shared[position] = map[string]bool{}
		}
		shared[position][ID(place.ID)] = true
	}
	out := make(map[string]Position, len(places))
	for _, place := range places {
		id := ID(place.ID)
		lat, lon := place.LatLng[0], place.LatLng[1]
		if id == "" || !onSriLanka(lat, lon) {
			continue
		}
		if len(shared[Position{Lat: round(lat), Lon: round(lon)}]) >= defaultPositionNames {
			continue
		}
		if _, seen := out[id]; !seen {
			out[id] = Position{Lat: lat, Lon: lon}
		}
	}
	return out, nil
}

// round groups positions to about 11 m.
func round(value float64) float64 { return math.Round(value*1e4) / 1e4 }
