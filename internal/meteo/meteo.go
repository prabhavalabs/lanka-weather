// Package meteo holds the small meteorological formulas lanka-weather uses
// to derive values a source does not send: dew point, day or night, heat
// index and the wind's components. Every function is pure.
package meteo

import (
	"math"
	"time"
)

// DewPoint is the dew point in °C from air temperature in °C and relative
// humidity in percent, by the Magnus formula with Alduchov and Eskridge's
// (1996) coefficients, accurate to about 0.1 °C over -40 to 50 °C. It is NaN
// when either input is missing or the humidity is not positive.
func DewPoint(temperatureC, humidityPct float64) float64 {
	if math.IsNaN(temperatureC) || math.IsNaN(humidityPct) || humidityPct <= 0 {
		return math.NaN()
	}
	const a, b = 17.625, 243.04
	gamma := math.Log(math.Min(humidityPct, 100)/100) + a*temperatureC/(b+temperatureC)
	return b * gamma / (a - gamma)
}

// SolarElevation is the sun's elevation above the horizon in degrees at a
// time and place, by NOAA's low-precision algorithm (within about 0.5°).
func SolarElevation(at time.Time, lon, lat float64) float64 {
	at = at.UTC()
	day := float64(at.YearDay())
	hours := float64(at.Hour()) + float64(at.Minute())/60 + float64(at.Second())/3600
	// Fractional year in radians.
	gamma := 2 * math.Pi / 365 * (day - 1 + (hours-12)/24)
	equation := 229.18 * (0.000075 + 0.001868*math.Cos(gamma) - 0.032077*math.Sin(gamma) -
		0.014615*math.Cos(2*gamma) - 0.040849*math.Sin(2*gamma))
	declination := 0.006918 - 0.399912*math.Cos(gamma) + 0.070257*math.Sin(gamma) -
		0.006758*math.Cos(2*gamma) + 0.000907*math.Sin(2*gamma) -
		0.002697*math.Cos(3*gamma) + 0.00148*math.Sin(3*gamma)
	// True solar time in minutes, and the hour angle.
	solarMinutes := hours*60 + equation + 4*lon
	hourAngle := (solarMinutes/4 - 180) * math.Pi / 180
	phi := lat * math.Pi / 180
	cosZenith := math.Sin(phi)*math.Sin(declination) + math.Cos(phi)*math.Cos(declination)*math.Cos(hourAngle)
	return 90 - math.Acos(math.Max(-1, math.Min(1, cosZenith)))*180/math.Pi
}

// IsDay reports whether the sun is up: its centre above -0.833°, the
// standard allowance for refraction and the sun's radius at sunrise.
func IsDay(at time.Time, lon, lat float64) bool {
	return SolarElevation(at, lon, lat) > -0.833
}

// HeatIndex is the US National Weather Service heat index in °C from air
// temperature in °C and relative humidity in percent: Steadman's simple
// formula, and the Rothfusz regression with its adjustments where the
// simple result is 80 °F (26.7 °C) or more. It is NaN when an input is
// missing.
func HeatIndex(temperatureC, humidityPct float64) float64 {
	if math.IsNaN(temperatureC) || math.IsNaN(humidityPct) {
		return math.NaN()
	}
	t := temperatureC*9/5 + 32
	rh := math.Max(0, math.Min(100, humidityPct))
	simple := 0.5 * (t + 61.0 + (t-68.0)*1.2 + rh*0.094)
	if (simple+t)/2 < 80 {
		return (simple - 32) * 5 / 9
	}
	index := -42.379 + 2.04901523*t + 10.14333127*rh - 0.22475541*t*rh - 0.00683783*t*t -
		0.05481717*rh*rh + 0.00122874*t*t*rh + 0.00085282*t*rh*rh - 0.00000199*t*t*rh*rh
	switch {
	case rh < 13 && t >= 80 && t <= 112:
		index -= (13 - rh) / 4 * math.Sqrt((17-math.Abs(t-95))/17)
	case rh > 85 && t >= 80 && t <= 87:
		index += (rh - 85) / 10 * (87 - t) / 5
	}
	return (index - 32) * 5 / 9
}

// WindComponents returns the eastward (u) and northward (v) components of a
// wind of the given speed blowing from the given direction in degrees
// clockwise from north, in the speed's unit.
func WindComponents(speed, fromDegrees float64) (u, v float64) {
	radians := fromDegrees * math.Pi / 180
	return -speed * math.Sin(radians), -speed * math.Cos(radians)
}

// WindFrom returns the speed of a wind with components u and v, and the
// direction it blows from in degrees clockwise from north, 0 to 360.
func WindFrom(u, v float64) (speed, fromDegrees float64) {
	speed = math.Hypot(u, v)
	if speed == 0 {
		return 0, 0
	}
	fromDegrees = math.Mod(math.Atan2(-u, -v)*180/math.Pi+360, 360)
	return speed, fromDegrees
}
