package summary

import (
	"math"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

// calmHours is a day of quiet weather.
func calmHours() []Hour {
	hours := make([]Hour, 24)
	for index := range hours {
		hours[index] = Hour{At: start.Add(time.Duration(index) * time.Hour), PrecipitationMm: 0.5, TemperatureC: 26, HumidityPct: 60,
			WindSpeedKmh: 12, WindGustsKmh: 25}
	}
	return hours
}

func calmDays() []Day {
	return []Day{
		{Start: start.Add(-time.Hour), PrecipitationMm: 5},
		{Start: start.Add(23 * time.Hour), PrecipitationMm: 5, WindSpeedKmh: 15, WindGustsKmh: 30},
		{Start: start.Add(47 * time.Hour), PrecipitationMm: 5, WindSpeedKmh: 15, WindGustsKmh: 30},
	}
}

func find(alerts []Alert, kind string) *Alert {
	for index := range alerts {
		if alerts[index].Type == kind {
			return &alerts[index]
		}
	}
	return nil
}

func TestBand(t *testing.T) {
	cases := []struct {
		value float64
		want  Level
	}{{49.9, ""}, {50, LevelYellow}, {99, LevelYellow}, {100, LevelOrange}, {150, LevelRed}, {math.NaN(), ""}}
	for _, c := range cases {
		if got, _ := RainBand.Of(c.value); got != c.want {
			t.Errorf("RainBand.Of(%v) = %q, want %q", c.value, got, c.want)
		}
	}
}

func TestCalmWeatherRaisesNoAlerts(t *testing.T) {
	if alerts := Evaluate(Inputs{Hours: calmHours(), Days: calmDays()}); len(alerts) != 0 {
		t.Fatalf("alerts = %+v", alerts)
	}
}

func TestHeavyRain(t *testing.T) {
	hours := calmHours()
	for index := range 10 {
		hours[index].PrecipitationMm = 11 // 110 + 7 = 117 mm
	}
	alert := find(Evaluate(Inputs{Hours: hours, Days: calmDays()}), "heavy_rain")
	if alert == nil || alert.Level != LevelOrange || *alert.Value != 117 || *alert.Threshold != 100 || !alert.Estimate {
		t.Fatalf("alert = %+v", alert)
	}
	if !alert.ValidFrom.Equal(start) || !alert.ValidUntil.Equal(start.Add(24*time.Hour)) {
		t.Fatalf("window = %v to %v", alert.ValidFrom, alert.ValidUntil)
	}

	// The day after tomorrow's 160 mm outranks today's 60 mm.
	hours = calmHours()
	hours[0].PrecipitationMm = 60
	days := calmDays()
	days[2].PrecipitationMm = 160
	alert = find(Evaluate(Inputs{Hours: hours, Days: days}), "heavy_rain")
	if alert == nil || alert.Level != LevelRed || !alert.ValidFrom.Equal(days[2].Start) {
		t.Fatalf("alert = %+v", alert)
	}

	// Today's daily total is not used: the next 24 hours cover it.
	days = calmDays()
	days[0].PrecipitationMm = 200
	if alert := find(Evaluate(Inputs{Hours: calmHours(), Days: days}), "heavy_rain"); alert != nil {
		t.Fatalf("today's total raised %+v", alert)
	}

	// Too few hours with values give no 24-hour total.
	hours = calmHours()
	for index := range 6 {
		hours[index].PrecipitationMm = math.NaN()
	}
	hours[10].PrecipitationMm = 300
	if alert := find(Evaluate(Inputs{Hours: hours}), "heavy_rain"); alert != nil {
		t.Fatalf("partial day raised %+v", alert)
	}
}

func TestStrongWindUsesGustsOrSustainedWind(t *testing.T) {
	hours := calmHours()
	hours[5].WindGustsKmh = 85
	alert := find(Evaluate(Inputs{Hours: hours}), "strong_wind")
	if alert == nil || alert.Level != LevelOrange || *alert.Value != 85 || *alert.Threshold != 80 {
		t.Fatalf("gust alert = %+v", alert)
	}
	hours = calmHours()
	hours[5].WindSpeedKmh = 95
	alert = find(Evaluate(Inputs{Hours: hours}), "strong_wind")
	if alert == nil || alert.Level != LevelRed || *alert.Threshold != 90 {
		t.Fatalf("sustained alert = %+v", alert)
	}
	days := calmDays()
	days[1].WindGustsKmh = 62
	alert = find(Evaluate(Inputs{Hours: calmHours(), Days: days}), "strong_wind")
	if alert == nil || alert.Level != LevelYellow || !alert.ValidFrom.Equal(days[1].Start) {
		t.Fatalf("tomorrow's gusts = %+v", alert)
	}
}

func TestHeat(t *testing.T) {
	hours := calmHours()
	hours[4].TemperatureC, hours[4].HumidityPct = 32.2, 70 // heat index about 41 °C
	alert := find(Evaluate(Inputs{Hours: hours}), "heat")
	if alert == nil || alert.Level != LevelYellow || math.Abs(*alert.Value-41) > 1 || !alert.ValidFrom.Equal(hours[4].At) {
		t.Fatalf("alert = %+v", alert)
	}
	hours[4].TemperatureC, hours[4].HumidityPct = 35, 75 // about 52 °C
	if alert := find(Evaluate(Inputs{Hours: hours}), "heat"); alert == nil || alert.Level != LevelOrange {
		t.Fatalf("alert = %+v", alert)
	}
}

func TestRiverAndFloodAlerts(t *testing.T) {
	forecastEnd := start.Add(48 * time.Hour)
	alerts := Evaluate(Inputs{
		Rivers: []NearbyRiver{
			{Name: "Hanwella", DistanceKm: 12, Status: "minor_flood"},
			{Name: "Glencourse", DistanceKm: 30, Status: "major_flood"}, // too far
			{Name: "Kitulgala", DistanceKm: 5, Status: "alert", Stale: true},
			{Name: "Nagalagam Street", DistanceKm: 3, Status: "normal"},
		},
		Floods: []NearbyFloodGauge{
			{GaugeID: "hybas_1", DistanceKm: 20, Severity: "extreme", ForecastEnd: &forecastEnd},
			{GaugeID: "hybas_2", DistanceKm: 10, Severity: "no_flooding"},
		},
		Areas: []FloodArea{{Kind: "flash_highly_likely", Severity: "highly_likely"}},
	})
	if len(alerts) != 3 {
		t.Fatalf("alerts = %+v", alerts)
	}
	if alerts[0].Type != "flood_forecast" || alerts[0].Level != LevelRed || alerts[0].ValidUntil != &forecastEnd {
		t.Fatalf("most serious first: %+v", alerts[0])
	}
	river := find(alerts, "river_level")
	if river == nil || river.Level != LevelOrange || *river.Value != 12 {
		t.Fatalf("river = %+v", river)
	}
	for _, alert := range alerts {
		if !alert.Estimate || alert.Basis == "" {
			t.Fatalf("every derived alert is an estimate with a basis: %+v", alert)
		}
	}
}

func TestDistanceKm(t *testing.T) {
	// Colombo to Kandy is about 94 km in a straight line.
	if got := DistanceKm(79.8612, 6.9271, 80.6337, 7.2906); math.Abs(got-94) > 2 {
		t.Fatalf("distance = %v", got)
	}
}
