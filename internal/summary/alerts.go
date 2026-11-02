package summary

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/meteo"
)

// Level is an alert's colour, from least to most serious: yellow (be
// aware), orange (be prepared), red (take action).
type Level string

const (
	LevelYellow Level = "yellow"
	LevelOrange Level = "orange"
	LevelRed    Level = "red"
)

func (level Level) rank() int {
	switch level {
	case LevelYellow:
		return 1
	case LevelOrange:
		return 2
	case LevelRed:
		return 3
	}
	return 0
}

// Alert is a derived alert. Every derived alert is an estimate made by
// lanka-weather from forecasts and observations; official Department of
// Meteorology and Disaster Management Centre warnings are authoritative.
type Alert struct {
	// Type is heavy_rain, strong_wind, heat, river_level or flood_forecast.
	Type       string     `json:"type"`
	Level      Level      `json:"level"`
	Title      string     `json:"title"`
	Detail     string     `json:"detail"`
	Value      *float64   `json:"value"`
	Threshold  *float64   `json:"threshold"`
	Unit       string     `json:"unit"`
	ValidFrom  *time.Time `json:"validFrom"`
	ValidUntil *time.Time `json:"validUntil"`
	// Estimate is always true for derived alerts.
	Estimate bool `json:"estimate"`
	// Basis names the data the alert was derived from.
	Basis string `json:"basis"`
}

// Band holds the thresholds of the three levels; a value at or above a
// threshold reaches that level.
type Band struct {
	Yellow, Orange, Red float64
}

// Of returns the level a value reaches and its threshold, or "" below
// yellow.
func (band Band) Of(value float64) (Level, float64) {
	switch {
	case math.IsNaN(value):
		return "", 0
	case value >= band.Red:
		return LevelRed, band.Red
	case value >= band.Orange:
		return LevelOrange, band.Orange
	case value >= band.Yellow:
		return LevelYellow, band.Yellow
	}
	return "", 0
}

// Thresholds of the derived alerts (docs/api/weather-and-floods.md).
var (
	// RainBand is rain in 24 hours, in mm: the Department of Meteorology's
	// advisory steps of "above 50", "above 100" and "above 150 mm".
	RainBand = Band{Yellow: 50, Orange: 100, Red: 150}
	// WindBand is the sustained wind at 10 m, in km/h.
	WindBand = Band{Yellow: 40, Orange: 60, Red: 90}
	// GustBand is the wind gust, in km/h.
	GustBand = Band{Yellow: 60, Orange: 80, Red: 100}
	// HeatBand is the heat index, in °C: the "extreme caution", "danger"
	// and "extreme danger" categories the Department of Meteorology's heat
	// advisories use.
	HeatBand = Band{Yellow: 33, Orange: 42, Red: 54}
)

const (
	// RiverRadiusKm and FloodRadiusKm are how near a gauge must be to raise
	// an alert for a place.
	RiverRadiusKm = 25.0
	FloodRadiusKm = 30.0
	// minimumHours of the next 24 must have values for a 24-hour total.
	minimumHours = 20
)

// Hour is one forecast hour at the place.
type Hour struct {
	At                         time.Time
	PrecipitationMm            float64
	TemperatureC, HumidityPct  float64
	WindSpeedKmh, WindGustsKmh float64
}

// Day is one local day's forecast at the place.
type Day struct {
	Start           time.Time
	PrecipitationMm float64
	WindSpeedKmh    float64
	WindGustsKmh    float64
}

// NearbyRiver is a river gauge near the place.
type NearbyRiver struct {
	Name       string
	DistanceKm float64
	// Status is normal, alert, minor_flood, major_flood or unknown.
	Status string
	Stale  bool
}

// NearbyFloodGauge is a Google flood-forecast gauge near the place.
type NearbyFloodGauge struct {
	GaugeID       string
	DistanceKm    float64
	Severity      string
	ForecastStart *time.Time
	ForecastEnd   *time.Time
}

// FloodArea is a Google flood area containing the place.
type FloodArea struct {
	Kind       string
	Severity   string
	ValidFrom  *time.Time
	ValidUntil *time.Time
}

// Inputs are what alerts are derived from.
type Inputs struct {
	// Hours are the forecast from the current hour, at least 24 for the
	// 24-hour rules; Days are local days from today.
	Hours  []Hour
	Days   []Day
	Rivers []NearbyRiver
	Floods []NearbyFloodGauge
	Areas  []FloodArea
}

const (
	basisForecast = "Open-Meteo forecast, interpolated from the weather grid"
	basisRivers   = "Irrigation Department river gauge reading and the station's own thresholds"
	basisFloods   = "Google Flood Forecasting"
)

// Evaluate derives the alerts for a place, most serious first.
func Evaluate(inputs Inputs) []Alert {
	alerts := []Alert{}
	if alert, ok := rain(inputs); ok {
		alerts = append(alerts, alert)
	}
	if alert, ok := wind(inputs); ok {
		alerts = append(alerts, alert)
	}
	if alert, ok := heat(inputs); ok {
		alerts = append(alerts, alert)
	}
	alerts = append(alerts, riverAlerts(inputs.Rivers)...)
	alerts = append(alerts, floodAlerts(inputs.Floods, inputs.Areas)...)
	sort.SliceStable(alerts, func(i, j int) bool { return alerts[i].Level.rank() > alerts[j].Level.rank() })
	return alerts
}

// window is a span with a value, compared by level, then value.
type window struct {
	value      float64
	from, to   time.Time
	level      Level
	threshold  float64
	what, unit string
}

func better(candidate, best window) bool {
	if candidate.level.rank() != best.level.rank() {
		return candidate.level.rank() > best.level.rank()
	}
	return candidate.value > best.value
}

// rain checks the next 24 hours and the next two local days.
func rain(inputs Inputs) (Alert, bool) {
	var best window
	if len(inputs.Hours) >= 24 {
		total, counted := 0.0, 0
		for _, hour := range inputs.Hours[:24] {
			if !math.IsNaN(hour.PrecipitationMm) {
				total, counted = total+hour.PrecipitationMm, counted+1
			}
		}
		if counted >= minimumHours {
			level, threshold := RainBand.Of(total)
			best = window{value: total, from: inputs.Hours[0].At, to: inputs.Hours[0].At.Add(24 * time.Hour), level: level, threshold: threshold, what: "in the next 24 hours"}
		}
	}
	for index, day := range inputs.Days {
		if index == 0 || index > 2 {
			continue // today is covered by the next 24 hours
		}
		level, threshold := RainBand.Of(day.PrecipitationMm)
		candidate := window{value: day.PrecipitationMm, from: day.Start, to: day.Start.Add(24 * time.Hour), level: level, threshold: threshold, what: dayName(index)}
		if better(candidate, best) {
			best = candidate
		}
	}
	if best.level == "" {
		return Alert{}, false
	}
	titles := map[Level]string{LevelYellow: "Heavy rain possible", LevelOrange: "Heavy rain likely", LevelRed: "Very heavy rain likely"}
	return newAlert("heavy_rain", best, titles[best.level],
		fmt.Sprintf("About %.0f mm of rain is forecast %s (threshold %.0f mm).", best.value, best.what, best.threshold), "mm", basisForecast), true
}

// wind checks sustained winds and gusts over the next 24 hours and the next
// two local days.
func wind(inputs Inputs) (Alert, bool) {
	var best window
	consider := func(speed, gust float64, from, to time.Time, what string) {
		for _, candidate := range []window{
			{value: gust, from: from, to: to, what: what, unit: "gusts"},
			{value: speed, from: from, to: to, what: what, unit: "winds"},
		} {
			band := GustBand
			if candidate.unit == "winds" {
				band = WindBand
			}
			candidate.level, candidate.threshold = band.Of(candidate.value)
			if candidate.level != "" && better(candidate, best) {
				best = candidate
			}
		}
	}
	if len(inputs.Hours) >= 24 {
		speed, gust := math.NaN(), math.NaN()
		for _, hour := range inputs.Hours[:24] {
			speed, gust = nanMax(speed, hour.WindSpeedKmh), nanMax(gust, hour.WindGustsKmh)
		}
		consider(speed, gust, inputs.Hours[0].At, inputs.Hours[0].At.Add(24*time.Hour), "in the next 24 hours")
	}
	for index, day := range inputs.Days {
		if index >= 1 && index <= 2 {
			consider(day.WindSpeedKmh, day.WindGustsKmh, day.Start, day.Start.Add(24*time.Hour), dayName(index))
		}
	}
	if best.level == "" {
		return Alert{}, false
	}
	titles := map[Level]string{LevelYellow: "Strong wind possible", LevelOrange: "Strong wind likely", LevelRed: "Damaging wind likely"}
	return newAlert("strong_wind", best, titles[best.level],
		fmt.Sprintf("%s up to %.0f km/h are forecast %s (threshold %.0f km/h).", capitalise(best.unit), best.value, best.what, best.threshold),
		"km/h", basisForecast), true
}

// heat checks the highest heat index over the next 24 hours.
func heat(inputs Inputs) (Alert, bool) {
	if len(inputs.Hours) < 24 {
		return Alert{}, false
	}
	best := window{value: math.NaN()}
	for _, hour := range inputs.Hours[:24] {
		index := meteo.HeatIndex(hour.TemperatureC, hour.HumidityPct)
		if !math.IsNaN(index) && (math.IsNaN(best.value) || index > best.value) {
			best.value, best.from = index, hour.At
		}
	}
	best.level, best.threshold = HeatBand.Of(best.value)
	if best.level == "" {
		return Alert{}, false
	}
	best.to = best.from.Add(time.Hour)
	titles := map[Level]string{LevelYellow: "Heat: extreme caution", LevelOrange: "Heat: danger", LevelRed: "Heat: extreme danger"}
	return newAlert("heat", best, titles[best.level],
		fmt.Sprintf("The heat index may reach %.0f °C (threshold %.0f °C). Drink water, rest in the shade and check on others.", best.value, best.threshold),
		"°C", basisForecast+"; heat index from temperature and humidity"), true
}

var riverLevels = map[string]Level{"alert": LevelYellow, "minor_flood": LevelOrange, "major_flood": LevelRed}

func riverAlerts(rivers []NearbyRiver) []Alert {
	var alerts []Alert
	for _, river := range rivers {
		level, ok := riverLevels[river.Status]
		if !ok || river.Stale || river.DistanceKm > RiverRadiusKm {
			continue
		}
		words := map[Level]string{LevelYellow: "above its alert level", LevelOrange: "at minor flood level", LevelRed: "at major flood level"}
		distance := river.DistanceKm
		alerts = append(alerts, Alert{
			Type: "river_level", Level: level, Title: "River " + words[level],
			Detail: fmt.Sprintf("The %s gauge, %.0f km away, is %s.", river.Name, distance, words[level]),
			Value:  &distance, Unit: "km", Estimate: true, Basis: basisRivers,
		})
	}
	return alerts
}

var floodLevels = map[string]Level{
	"above_normal": LevelYellow, "severe": LevelOrange, "extreme": LevelRed,
	"likely": LevelYellow, "highly_likely": LevelOrange, "minor": LevelYellow, "moderate": LevelOrange,
}

func floodAlerts(gauges []NearbyFloodGauge, areas []FloodArea) []Alert {
	var alerts []Alert
	for _, gauge := range gauges {
		level, ok := floodLevels[gauge.Severity]
		if !ok || gauge.DistanceKm > FloodRadiusKm {
			continue
		}
		distance := gauge.DistanceKm
		alerts = append(alerts, Alert{
			Type: "flood_forecast", Level: level, Title: "River flooding forecast nearby",
			Detail: fmt.Sprintf("Google forecasts %s flooding at a river gauge %.0f km away.", humanise(gauge.Severity), distance),
			Value:  &distance, Unit: "km", ValidFrom: gauge.ForecastStart, ValidUntil: gauge.ForecastEnd,
			Estimate: true, Basis: basisFloods,
		})
	}
	for _, area := range areas {
		level, ok := floodLevels[area.Severity]
		if area.Kind == "significant_event" && area.Severity == "severe" {
			level, ok = LevelRed, true
		}
		if !ok {
			continue
		}
		alerts = append(alerts, Alert{
			Type: "flood_forecast", Level: level, Title: "Inside a forecast flood area",
			Detail:    fmt.Sprintf("This place is inside a %s area published by Google Flood Forecasting.", humanise(area.Kind)),
			ValidFrom: area.ValidFrom, ValidUntil: area.ValidUntil, Estimate: true, Basis: basisFloods,
		})
	}
	return alerts
}

func newAlert(kind string, best window, title, detail, unit, basis string) Alert {
	value, threshold := math.Round(best.value*10)/10, best.threshold
	from, to := best.from, best.to
	return Alert{Type: kind, Level: best.level, Title: title, Detail: detail, Value: &value, Threshold: &threshold,
		Unit: unit, ValidFrom: &from, ValidUntil: &to, Estimate: true, Basis: basis}
}

func dayName(index int) string {
	if index == 1 {
		return "tomorrow"
	}
	return "the day after tomorrow"
}

func nanMax(a, b float64) float64 {
	switch {
	case math.IsNaN(a):
		return b
	case math.IsNaN(b):
		return a
	}
	return math.Max(a, b)
}

func humanise(value string) string {
	out := []rune(value)
	for index, r := range out {
		if r == '_' {
			out[index] = ' '
		}
	}
	return string(out)
}

func capitalise(value string) string {
	if value == "" {
		return value
	}
	return string(value[0]-'a'+'A') + value[1:]
}
