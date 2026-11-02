package openmeteo

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/meteo"
)

// Field is one variable over every node of the grid and every step: Values
// holds steps × nodes values, step-major, in the grid's node order, with NaN
// where the source gave none.
type Field struct {
	Variable  string
	ValidFrom time.Time
	Step      time.Duration
	Steps     int
	Values    []float32
}

// Prefixes of forecast field names: "hourly:temperature_2m",
// "daily:temperature_2m_max".
const (
	HourlyPrefix = "hourly:"
	DailyPrefix  = "daily:"
)

// LocalOffset is Sri Lanka Standard Time, UTC+05:30, with no daylight saving.
const LocalOffset = 5*time.Hour + 30*time.Minute

// LocalDayStart is the UTC instant at which the local day containing t began.
func LocalDayStart(t time.Time) time.Time {
	local := t.UTC().Add(LocalOffset)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC).Add(-LocalOffset)
}

var nan = float32(math.NaN())

// CurrentFields turns the current conditions of every node into one field
// per variable, adding dew point and day or night. The fields' time is the
// earliest the locations report (they differ only when a run crosses
// Open-Meteo's 15-minute step).
func CurrentFields(nodes []grid.Node, locations []Location) (time.Time, []Field, error) {
	if len(locations) != len(nodes) || len(nodes) == 0 {
		return time.Time{}, nil, fmt.Errorf("%d locations for %d nodes", len(locations), len(nodes))
	}
	var at time.Time
	for _, location := range locations {
		if location.Current == nil {
			return time.Time{}, nil, errors.New("a location has no current conditions")
		}
		if at.IsZero() || location.CurrentTime.Before(at) {
			at = location.CurrentTime
		}
	}
	fields := make([]Field, 0, len(CurrentVariables)+2)
	for _, variable := range CurrentVariables {
		values := make([]float32, len(nodes))
		for position, location := range locations {
			values[position] = location.Current[variable]
		}
		fields = append(fields, Field{Variable: variable, ValidFrom: at, Steps: 1, Values: values})
	}
	dew := make([]float32, len(nodes))
	day := make([]float32, len(nodes))
	for position, location := range locations {
		dew[position] = float32(meteo.DewPoint(float64(location.Current["temperature_2m"]), float64(location.Current["relative_humidity_2m"])))
		day[position] = boolValue(meteo.IsDay(location.CurrentTime, nodes[position].Lon, nodes[position].Lat))
	}
	fields = append(fields,
		Field{Variable: VariableDewPoint, ValidFrom: at, Steps: 1, Values: dew},
		Field{Variable: VariableIsDay, ValidFrom: at, Steps: 1, Values: day})
	return at, fields, nil
}

// ForecastFields turns every node's hourly forecast into hourly fields on
// one time axis, adds dew point and day or night for each hour, and derives
// the daily summary for DailyDays local days from the one containing now.
func ForecastFields(nodes []grid.Node, locations []Location, now time.Time) (time.Time, []Field, error) {
	if len(locations) != len(nodes) || len(nodes) == 0 {
		return time.Time{}, nil, fmt.Errorf("%d locations for %d nodes", len(locations), len(nodes))
	}
	axis := locations[0].Times
	if len(axis) == 0 {
		return time.Time{}, nil, errors.New("the forecast has no hours")
	}
	for index := 1; index < len(axis); index++ {
		if axis[index].Sub(axis[index-1]) != time.Hour {
			return time.Time{}, nil, errors.New("forecast hours are not one hour apart")
		}
	}
	steps, count := len(axis), len(nodes)
	first := axis[0]
	hourly := map[string][]float32{}
	for _, variable := range append(append([]string{}, HourlyVariables...), VariableDewPoint, VariableIsDay) {
		values := make([]float32, steps*count)
		for index := range values {
			values[index] = nan
		}
		hourly[variable] = values
	}
	for position, location := range locations {
		for k, at := range location.Times {
			step := int(at.Sub(first) / time.Hour)
			if at.Before(first) || step >= steps || !at.Equal(first.Add(time.Duration(step)*time.Hour)) {
				continue
			}
			offset := step*count + position
			for _, variable := range HourlyVariables {
				hourly[variable][offset] = location.Hourly[variable][k]
			}
			hourly[VariableDewPoint][offset] = float32(meteo.DewPoint(
				float64(location.Hourly["temperature_2m"][k]), float64(location.Hourly["relative_humidity_2m"][k])))
			hourly[VariableIsDay][offset] = boolValue(meteo.IsDay(at, nodes[position].Lon, nodes[position].Lat))
		}
	}

	fields := make([]Field, 0, len(hourly)+len(dailyVariables))
	for _, variable := range append(append([]string{}, HourlyVariables...), VariableDewPoint, VariableIsDay) {
		fields = append(fields, Field{Variable: HourlyPrefix + variable, ValidFrom: first, Step: time.Hour, Steps: steps, Values: hourly[variable]})
	}
	return first, append(fields, Daily(hourly, first, count, steps, LocalDayStart(now))...), nil
}

// dailyVariables are the daily summary's fields, in order.
var dailyVariables = []string{
	"temperature_2m_max", "temperature_2m_min", "apparent_temperature_max", "precipitation_sum",
	"precipitation_probability_max", "wind_speed_10m_max", "wind_gusts_10m_max", "wind_direction_10m_dominant",
	"uv_index_max", "weather_code",
}

// minimumHours is how many of a day's 24 hours must have a value for the
// day's summary to have one.
const minimumHours = 20

// Daily summarises hourly fields (laid out steps × nodes from first) over
// DailyDays local days from dayStart. A day covers the hours from its start
// to the next day's start; the weather code is the day's highest code, as
// Open-Meteo's own daily summary gives it.
func Daily(hourly map[string][]float32, first time.Time, nodes, steps int, dayStart time.Time) []Field {
	out := make(map[string][]float32, len(dailyVariables))
	for _, variable := range dailyVariables {
		values := make([]float32, DailyDays*nodes)
		for index := range values {
			values[index] = nan
		}
		out[variable] = values
	}
	for day := range DailyDays {
		start := dayStart.Add(time.Duration(day) * 24 * time.Hour)
		from := int(math.Ceil(start.Sub(first).Hours()))
		to := from + 24
		if from < 0 || to > steps {
			continue
		}
		for position := range nodes {
			series := func(variable string) []float64 {
				values := make([]float64, 0, 24)
				for step := from; step < to; step++ {
					if value := float64(hourly[variable][step*nodes+position]); !math.IsNaN(value) {
						values = append(values, value)
					}
				}
				return values
			}
			offset := day*nodes + position
			temperature := series("temperature_2m")
			out["temperature_2m_max"][offset] = maximum(temperature)
			out["temperature_2m_min"][offset] = minimum(temperature)
			out["apparent_temperature_max"][offset] = maximum(series("apparent_temperature"))
			out["precipitation_sum"][offset] = total(series("precipitation"))
			out["precipitation_probability_max"][offset] = maximum(series("precipitation_probability"))
			out["wind_speed_10m_max"][offset] = maximum(series("wind_speed_10m"))
			out["wind_gusts_10m_max"][offset] = maximum(series("wind_gusts_10m"))
			out["uv_index_max"][offset] = maximum(series("uv_index"))
			out["weather_code"][offset] = maximum(series("weather_code"))

			var u, v float64
			counted := 0
			for step := from; step < to; step++ {
				speed := float64(hourly["wind_speed_10m"][step*nodes+position])
				direction := float64(hourly["wind_direction_10m"][step*nodes+position])
				if math.IsNaN(speed) || math.IsNaN(direction) {
					continue
				}
				du, dv := meteo.WindComponents(speed, direction)
				u, v, counted = u+du, v+dv, counted+1
			}
			if counted >= minimumHours {
				if speed, direction := meteo.WindFrom(u, v); speed > 0 {
					out["wind_direction_10m_dominant"][offset] = float32(math.Round(direction))
				}
			}
		}
	}
	fields := make([]Field, 0, len(dailyVariables))
	for _, variable := range dailyVariables {
		fields = append(fields, Field{Variable: DailyPrefix + variable, ValidFrom: dayStart, Step: 24 * time.Hour, Steps: DailyDays, Values: out[variable]})
	}
	return fields
}

// maximum, minimum and total are NaN for a day with too few hours.
func maximum(values []float64) float32 {
	if len(values) < minimumHours {
		return nan
	}
	best := values[0]
	for _, value := range values[1:] {
		best = math.Max(best, value)
	}
	return float32(best)
}

func minimum(values []float64) float32 {
	if len(values) < minimumHours {
		return nan
	}
	best := values[0]
	for _, value := range values[1:] {
		best = math.Min(best, value)
	}
	return float32(best)
}

func total(values []float64) float32 {
	if len(values) < minimumHours {
		return nan
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return float32(math.Round(sum*100) / 100)
}

func boolValue(value bool) float32 {
	if value {
		return 1
	}
	return 0
}
