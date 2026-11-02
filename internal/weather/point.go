package weather

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/meteo"
)

// ErrOutsideGrid means a point is outside the weather grid.
var ErrOutsideGrid = errors.New("the point is outside the weather grid")

// dailyVariables are the daily summary's fields, in response order.
var dailyVariables = []variable{
	{Name: "temperatureMaxC", Hourly: "temperature_2m_max", Unit: "°C"},
	{Name: "temperatureMinC", Hourly: "temperature_2m_min", Unit: "°C"},
	{Name: "apparentTemperatureMaxC", Hourly: "apparent_temperature_max", Unit: "°C"},
	{Name: "precipitationSumMm", Hourly: "precipitation_sum", Unit: "mm"},
	{Name: "precipitationProbabilityMaxPct", Hourly: "precipitation_probability_max", Unit: "%"},
	{Name: "windSpeedMaxKmh", Hourly: "wind_speed_10m_max", Unit: "km/h"},
	{Name: "windGustsMaxKmh", Hourly: "wind_gusts_10m_max", Unit: "km/h"},
	{Name: "windDirectionDominantDeg", Hourly: "wind_direction_10m_dominant", Unit: "°"},
	{Name: "uvIndexMax", Hourly: "uv_index_max", Unit: ""},
	{Name: "weatherCode", Hourly: "weather_code", Categorical: true, Unit: "wmo"},
}

// Point is the weather at one place, interpolated from the grid.
type Point struct {
	Location Location `json:"location"`
	// Current holds validAt and one value per current variable; null
	// without a current run.
	Current map[string]any `json:"current"`
	// Hourly holds "times" and one array per variable; null without a
	// forecast run.
	Hourly map[string]any `json:"hourly"`
	// Daily holds "dates" (local, YYYY-MM-DD) and one array per variable.
	Daily map[string]any `json:"daily"`
	Meta  PointMeta      `json:"meta"`
}

// Location is a point asked for.
type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// PointMeta describes where a point's values come from.
type PointMeta struct {
	Source            string            `json:"source"`
	Model             string            `json:"model"`
	Attribution       string            `json:"attribution"`
	Grid              GridSpec          `json:"grid"`
	Interpolation     string            `json:"interpolation"`
	Units             map[string]string `json:"units"`
	CurrentFetchedAt  *time.Time        `json:"currentFetchedAt"`
	ForecastFetchedAt *time.Time        `json:"forecastFetchedAt"`
	CurrentStale      bool              `json:"currentStale"`
	ForecastStale     bool              `json:"forecastStale"`
}

// Interpolation describes how point values are read from the grid.
const Interpolation = "bilinear between the four surrounding grid nodes; weather code and day/night from the nearest node; " +
	"wind direction from interpolated wind components"

// PointOptions bound a point forecast.
type PointOptions struct {
	Hours              int
	Days               int
	CurrentStaleAfter  time.Duration
	ForecastStaleAfter time.Duration
}

// sampler reads values at one point.
type sampler struct{ corners [4]grid.Weight }

func (sampler sampler) scalar(values []float32) Value {
	value, ok := grid.Sample(values, sampler.corners)
	if !ok {
		return Value(math.NaN())
	}
	return Value(value)
}

func (sampler sampler) nearest(values []float32) Value {
	value, ok := grid.Nearest(values, sampler.corners)
	if !ok {
		return Value(math.NaN())
	}
	return Value(value)
}

// direction interpolates wind components and returns the direction they
// blow from; averaging angles directly would turn 350° and 10° into 180°.
// Without speeds, unit vectors are averaged.
func (sampler sampler) direction(speeds, directions []float32) Value {
	var u, v, weights float64
	for _, corner := range sampler.corners {
		if corner.Weight <= 0 || corner.Index >= len(directions) {
			continue
		}
		direction := float64(directions[corner.Index])
		speed := 1.0
		if speeds != nil {
			speed = float64(speeds[corner.Index])
		}
		if math.IsNaN(direction) || math.IsNaN(speed) {
			continue
		}
		du, dv := meteo.WindComponents(speed, direction)
		u, v, weights = u+du*corner.Weight, v+dv*corner.Weight, weights+corner.Weight
	}
	if weights == 0 {
		return Value(math.NaN())
	}
	speed, from := meteo.WindFrom(u/weights, v/weights)
	if speed == 0 {
		return Value(math.NaN())
	}
	return Value(math.Round(from))
}

// read reads one variable from a step's values, choosing the method.
func (sampler sampler) read(variable variable, values []float32, speeds []float32) Value {
	switch {
	case variable.Categorical:
		return sampler.nearest(values)
	case variable.Name == "windDirectionDeg" || variable.Name == "windDirectionDominantDeg":
		return sampler.direction(speeds, values)
	}
	return sampler.scalar(values)
}

// BuildPoint interpolates the current conditions, the hourly forecast from
// the current hour and the daily summary from today at a point.
func BuildPoint(snapshot *Snapshot, lon, lat float64, now time.Time, options PointOptions) (*Point, error) {
	around, ok := corners(snapshot.Lattice, lon, lat)
	if !ok {
		return nil, ErrOutsideGrid
	}
	at := sampler{corners: around}
	point := &Point{
		Location: Location{Lat: lat, Lon: lon},
		Meta: PointMeta{
			Source: openmeteo.SourceName, Model: openmeteo.Model, Attribution: openmeteo.Attribution,
			Grid: GridSpec{ID: snapshot.GridID, Lon0: snapshot.Lattice.Lon0, Lat0: snapshot.Lattice.Lat0,
				Step: snapshot.Lattice.Step, Nx: snapshot.Lattice.Nx, Ny: snapshot.Lattice.Ny},
			Interpolation: Interpolation, Units: Units(),
		},
	}
	if current := snapshot.Current; current != nil {
		fetched := current.FetchedAt
		point.Meta.CurrentFetchedAt = &fetched
		point.Meta.CurrentStale = now.Sub(current.SourceTime) > options.CurrentStaleAfter
		values := map[string]any{"validAt": current.SourceTime}
		speeds := currentValues(snapshot, variable{Current: "wind_speed_10m"})
		for _, variable := range currentVariables {
			if source := currentValues(snapshot, variable); source != nil {
				values[variable.Name] = at.read(variable, source, speeds)
			}
		}
		point.Current = values
	}
	forecast := snapshot.Forecast
	if forecast == nil {
		return point, nil
	}
	fetched := forecast.FetchedAt
	point.Meta.ForecastFetchedAt = &fetched
	point.Meta.ForecastStale = now.Sub(forecast.FetchedAt) > options.ForecastStaleAfter
	point.Hourly = hourly(forecast, at, now, options.Hours)
	point.Daily = daily(forecast, at, now, options.Days)
	return point, nil
}

func hourly(forecast *Run, at sampler, now time.Time, hours int) map[string]any {
	reference := forecast.Fields[openmeteo.HourlyPrefix+"temperature_2m"]
	if reference == nil || hours <= 0 {
		return nil
	}
	first := int(now.Sub(reference.ValidFrom) / time.Hour)
	if now.Before(reference.ValidFrom) {
		first = 0
	}
	last := min(first+hours, reference.Steps)
	if first >= last {
		return nil
	}
	times := make([]time.Time, 0, last-first)
	for step := first; step < last; step++ {
		times = append(times, reference.ValidFrom.Add(time.Duration(step)*time.Hour))
	}
	out := map[string]any{"times": times}
	speeds := forecast.Fields[openmeteo.HourlyPrefix+"wind_speed_10m"]
	for _, variable := range currentVariables {
		field := forecast.Fields[openmeteo.HourlyPrefix+variable.Hourly]
		if field == nil || field.Steps != reference.Steps {
			continue
		}
		series := make(Values, 0, last-first)
		for step := first; step < last; step++ {
			var speedValues []float32
			if speeds != nil {
				speedValues = speeds.At(step)
			}
			series = append(series, at.read(variable, field.At(step), speedValues))
		}
		out[variable.Name] = series
	}
	return out
}

func daily(forecast *Run, at sampler, now time.Time, days int) map[string]any {
	reference := forecast.Fields[openmeteo.DailyPrefix+"temperature_2m_max"]
	if reference == nil || days <= 0 {
		return nil
	}
	first := int(openmeteo.LocalDayStart(now).Sub(reference.ValidFrom) / (24 * time.Hour))
	first = max(first, 0)
	last := min(first+days, reference.Steps)
	if first >= last {
		return nil
	}
	dates := make([]string, 0, last-first)
	for step := first; step < last; step++ {
		dates = append(dates, reference.ValidFrom.Add(time.Duration(step)*24*time.Hour).Add(openmeteo.LocalOffset).Format(time.DateOnly))
	}
	out := map[string]any{"dates": dates}
	for _, variable := range dailyVariables {
		field := forecast.Fields[openmeteo.DailyPrefix+variable.Hourly]
		if field == nil || field.Steps != reference.Steps {
			continue
		}
		series := make(Values, 0, last-first)
		for step := first; step < last; step++ {
			series = append(series, at.read(variable, field.At(step), nil))
		}
		out[variable.Name] = series
	}
	return out
}

// Point interpolates the stored grid at a point.
func (service *Service) Point(ctx context.Context, lon, lat float64, hours, days int) (*Point, error) {
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return BuildPoint(snapshot, lon, lat, service.now(), PointOptions{
		Hours: hours, Days: days, CurrentStaleAfter: service.CurrentStaleAfter, ForecastStaleAfter: service.ForecastStaleAfter,
	})
}
