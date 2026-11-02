// Package openmeteo collects current conditions and forecasts for every node
// of the weather grid from Open-Meteo's forecast API (open-meteo.com).
//
// Open-Meteo is keyless for non-commercial use, under 10,000 API calls a
// day, 5,000 an hour and 600 a minute. A request may carry many coordinates;
// it is weighted as nodes × max(1, variables / 10) × max(1, days / 14)
// calls. Two collectors share the budget (ADR 0007):
//
//   - current conditions every hour: 10 variables, weight 1 per node;
//   - the hourly forecast every three hours: 12 variables over 9 days
//     (yesterday for whole local days, and 8 ahead), weight 1.2 per node.
//
// Dew point and day or night are derived here rather than requested, and the
// daily summary is derived from the hourly forecast for whole local days in
// Sri Lanka (UTC+05:30), which keeps both requests at or near ten variables.
//
// Data is licensed CC BY 4.0: "Weather data by Open-Meteo.com".
package openmeteo

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
)

// SourceName identifies the source in stored records and docs/data-sources.md.
const SourceName = "open-meteo"

// Model is the model selection asked for: Open-Meteo's blend of the most
// suitable global models for each point (ECMWF IFS, DWD ICON and NOAA GFS
// over Sri Lanka).
const Model = "best_match"

// Attribution is the credit Open-Meteo's CC BY 4.0 licence requires.
const Attribution = "Weather data by Open-Meteo.com (CC BY 4.0)"

// Endpoint is the forecast API.
const Endpoint = "https://api.open-meteo.com/v1/forecast"

// Kind is what a collector requests.
type Kind string

const (
	KindCurrent  Kind = "current"
	KindForecast Kind = "forecast"
)

// CurrentVariables are requested for current conditions. Ten keep a node's
// weight at one call.
var CurrentVariables = []string{
	"temperature_2m", "relative_humidity_2m", "apparent_temperature", "precipitation", "weather_code",
	"cloud_cover", "pressure_msl", "wind_speed_10m", "wind_direction_10m", "wind_gusts_10m",
}

// HourlyVariables are requested for the hourly forecast.
var HourlyVariables = []string{
	"temperature_2m", "relative_humidity_2m", "apparent_temperature", "precipitation_probability",
	"precipitation", "weather_code", "pressure_msl", "cloud_cover", "wind_speed_10m",
	"wind_direction_10m", "wind_gusts_10m", "uv_index",
}

// Derived variables, computed for both kinds.
const (
	VariableDewPoint = "dew_point_2m"
	VariableIsDay    = "is_day"
)

// Forecast span: yesterday, so today's local day is whole from its start at
// 18:30 UTC, and eight days ahead, so the seventh local day is whole too.
const (
	pastDays     = 1
	forecastDays = 8
	// DailyDays is the number of local days summarised.
	DailyDays = 7
)

// Weight is what one request for `locations` nodes counts against the
// free-tier limits, by Open-Meteo's published weighting.
func Weight(kind Kind, locations int) float64 {
	variables, days := len(CurrentVariables), 1
	if kind == KindForecast {
		variables, days = len(HourlyVariables), pastDays+forecastDays
	}
	return float64(locations) * math.Max(1, float64(variables)/10) * math.Max(1, float64(days)/14)
}

// DailyCalls is the weighted calls a day of collecting costs the grid at the
// given intervals.
func DailyCalls(nodes int, currentEvery, forecastEvery time.Duration) float64 {
	day := 24 * time.Hour
	return float64(day/currentEvery)*Weight(KindCurrent, nodes) + float64(day/forecastEvery)*Weight(KindForecast, nodes)
}

// Chunk is one request: grid nodes that share a cell selection.
type Chunk struct {
	// Positions index the grid's Nodes.
	Positions []int
	Land      bool
}

// Chunks splits the grid into requests of at most size nodes. Land nodes
// ask Open-Meteo for a land model cell and sea nodes for a sea cell, so a
// coastal node is not given the climate of the wrong side of the coast.
func Chunks(nodes []grid.Node, size int) []Chunk {
	var chunks []Chunk
	for _, land := range []bool{true, false} {
		var current *Chunk
		for position, node := range nodes {
			if node.Land != land {
				continue
			}
			if current == nil || len(current.Positions) == size {
				chunks = append(chunks, Chunk{Land: land})
				current = &chunks[len(chunks)-1]
			}
			current.Positions = append(current.Positions, position)
		}
	}
	return chunks
}

// Query builds the request for one chunk.
func Query(kind Kind, nodes []grid.Node, chunk Chunk) url.Values {
	lats := make([]string, len(chunk.Positions))
	lons := make([]string, len(chunk.Positions))
	for k, position := range chunk.Positions {
		lats[k] = strconv.FormatFloat(nodes[position].Lat, 'f', -1, 64)
		lons[k] = strconv.FormatFloat(nodes[position].Lon, 'f', -1, 64)
	}
	selection := "sea"
	if chunk.Land {
		selection = "land"
	}
	query := url.Values{
		"latitude":        {strings.Join(lats, ",")},
		"longitude":       {strings.Join(lons, ",")},
		"cell_selection":  {selection},
		"timeformat":      {"unixtime"},
		"timezone":        {"GMT"},
		"wind_speed_unit": {"kmh"},
		"models":          {Model},
	}
	if kind == KindCurrent {
		query.Set("current", strings.Join(CurrentVariables, ","))
	} else {
		query.Set("hourly", strings.Join(HourlyVariables, ","))
		query.Set("past_days", strconv.Itoa(pastDays))
		query.Set("forecast_days", strconv.Itoa(forecastDays))
	}
	return query
}

// Location is one decoded location of a response. Missing values are NaN.
type Location struct {
	Elevation *float64
	// CurrentTime and Current are set for current conditions.
	CurrentTime time.Time
	Current     map[string]float32
	// Times and Hourly are set for a forecast; Hourly[variable][k] is valid
	// at Times[k].
	Times  []time.Time
	Hourly map[string][]float32
}

type wireLocation struct {
	Elevation *float64                   `json:"elevation"`
	Current   map[string]json.RawMessage `json:"current"`
	Hourly    map[string]json.RawMessage `json:"hourly"`
}

type wireError struct {
	Error  bool   `json:"error"`
	Reason string `json:"reason"`
}

// Decode reads a forecast API response for `count` locations. Open-Meteo
// answers several locations with a list, in request order, and one with an
// object. It does no I/O.
func Decode(kind Kind, payload []byte, count int) ([]Location, error) {
	trimmed := strings.TrimSpace(string(payload))
	var wire []wireLocation
	switch {
	case strings.HasPrefix(trimmed, "["):
		if err := json.Unmarshal(payload, &wire); err != nil {
			return nil, fmt.Errorf("decode Open-Meteo response: %w", err)
		}
	case strings.HasPrefix(trimmed, "{"):
		var failure wireError
		if err := json.Unmarshal(payload, &failure); err == nil && failure.Error {
			return nil, fmt.Errorf("Open-Meteo error: %s", failure.Reason)
		}
		var single wireLocation
		if err := json.Unmarshal(payload, &single); err != nil {
			return nil, fmt.Errorf("decode Open-Meteo response: %w", err)
		}
		wire = []wireLocation{single}
	default:
		return nil, errors.New("Open-Meteo response is not JSON")
	}
	if len(wire) != count {
		return nil, fmt.Errorf("Open-Meteo returned %d locations for %d asked", len(wire), count)
	}
	locations := make([]Location, len(wire))
	for index, item := range wire {
		location := Location{Elevation: item.Elevation}
		var err error
		if kind == KindCurrent {
			location.CurrentTime, location.Current, err = decodeCurrent(item.Current)
		} else {
			location.Times, location.Hourly, err = decodeHourly(item.Hourly)
		}
		if err != nil {
			return nil, fmt.Errorf("location %d: %w", index, err)
		}
		locations[index] = location
	}
	return locations, nil
}

func decodeCurrent(fields map[string]json.RawMessage) (time.Time, map[string]float32, error) {
	if fields == nil {
		return time.Time{}, nil, errors.New("no current block")
	}
	var unix int64
	if err := json.Unmarshal(fields["time"], &unix); err != nil || unix <= 0 {
		return time.Time{}, nil, errors.New("current block has no unix time")
	}
	values := make(map[string]float32, len(CurrentVariables))
	for _, variable := range CurrentVariables {
		value, err := number(fields[variable])
		if err != nil {
			return time.Time{}, nil, fmt.Errorf("current %s: %w", variable, err)
		}
		values[variable] = value
	}
	return time.Unix(unix, 0).UTC(), values, nil
}

func decodeHourly(fields map[string]json.RawMessage) ([]time.Time, map[string][]float32, error) {
	if fields == nil {
		return nil, nil, errors.New("no hourly block")
	}
	var unix []int64
	if err := json.Unmarshal(fields["time"], &unix); err != nil || len(unix) == 0 {
		return nil, nil, errors.New("hourly block has no unix times")
	}
	times := make([]time.Time, len(unix))
	for index, value := range unix {
		times[index] = time.Unix(value, 0).UTC()
		if index > 0 && !times[index].After(times[index-1]) {
			return nil, nil, errors.New("hourly times are not increasing")
		}
	}
	values := make(map[string][]float32, len(HourlyVariables))
	for _, variable := range HourlyVariables {
		var raw []json.RawMessage
		if err := json.Unmarshal(fields[variable], &raw); err != nil {
			return nil, nil, fmt.Errorf("hourly %s is not a list", variable)
		}
		if len(raw) != len(times) {
			return nil, nil, fmt.Errorf("hourly %s has %d values for %d times", variable, len(raw), len(times))
		}
		series := make([]float32, len(raw))
		for index, item := range raw {
			value, err := number(item)
			if err != nil {
				return nil, nil, fmt.Errorf("hourly %s: %w", variable, err)
			}
			series[index] = value
		}
		values[variable] = series
	}
	return times, values, nil
}

// number reads a JSON number; null or absent is NaN.
func number(raw json.RawMessage) (float32, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return float32(math.NaN()), nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%s is not a number", text)
	}
	return float32(value), nil
}
