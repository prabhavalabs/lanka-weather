package meteo

import (
	"math"
	"testing"
	"time"
)

func near(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > tolerance {
		t.Errorf("%s = %.3f, want %.3f ± %.3f", name, got, want, tolerance)
	}
}

func TestDewPoint(t *testing.T) {
	near(t, "saturated air", DewPoint(25, 100), 25, 0.01)
	// Open-Meteo gave 19.6 °C for 21.8 °C and 88 % at Kandy.
	near(t, "Kandy", DewPoint(21.8, 88), 19.7, 0.15)
	near(t, "Colombo afternoon", DewPoint(31, 70), 24.9, 0.2)
	if !math.IsNaN(DewPoint(30, 0)) || !math.IsNaN(DewPoint(math.NaN(), 50)) {
		t.Fatal("no humidity or no temperature gives no dew point")
	}
}

func TestIsDay(t *testing.T) {
	// Colombo is UTC+5:30; the sun rises about 06:05 and sets about 17:55 in November.
	colombo := [2]float64{79.86, 6.93}
	cases := map[string]bool{
		"2026-11-02T06:30:00Z": true,  // noon local
		"2026-11-02T18:00:00Z": false, // 23:30 local
		"2026-11-02T00:00:00Z": false, // 05:30 local, before sunrise
		"2026-11-02T01:30:00Z": true,  // 07:00 local
		"2026-11-02T13:00:00Z": false, // 18:30 local, after sunset
	}
	for at, want := range cases {
		parsed, _ := time.Parse(time.RFC3339, at)
		if got := IsDay(parsed, colombo[0], colombo[1]); got != want {
			t.Errorf("IsDay(%s) = %v, want %v (elevation %.1f°)", at, got, want, SolarElevation(parsed, colombo[0], colombo[1]))
		}
	}
	noon, _ := time.Parse(time.RFC3339, "2026-03-21T06:48:00Z")
	near(t, "equinox noon elevation in Colombo", SolarElevation(noon, colombo[0], colombo[1]), 83, 1.5)
}

func TestHeatIndex(t *testing.T) {
	// NWS table: 90 °F at 70 % is 106 °F; 86 °F at 50 % is 88 °F; 104 °F at 40 % is 119 °F.
	near(t, "32.2 °C, 70 %", HeatIndex(32.22, 70), (106.0-32)*5/9, 0.6)
	near(t, "30 °C, 50 %", HeatIndex(30, 50), (88.0-32)*5/9, 0.6)
	near(t, "40 °C, 40 %", HeatIndex(40, 40), (119.0-32)*5/9, 0.8)
	// Below about 27 °C the heat index is close to the temperature.
	near(t, "22 °C, 60 %", HeatIndex(22, 60), 21.8, 0.6)
}

func TestWindComponentsRoundTrip(t *testing.T) {
	// A wind from the west blows eastward: u positive, v zero.
	u, v := WindComponents(10, 270)
	near(t, "westerly u", u, 10, 1e-9)
	near(t, "westerly v", v, 0, 1e-9)
	// From the north-east, the north-east monsoon, blows south-west.
	u, v = WindComponents(10, 45)
	if u >= 0 || v >= 0 {
		t.Fatalf("north-easterly components = %v, %v", u, v)
	}
	for _, from := range []float64{0, 10, 90, 179, 181, 270, 359} {
		u, v := WindComponents(12, from)
		speed, back := WindFrom(u, v)
		near(t, "speed", speed, 12, 1e-9)
		if diff := math.Abs(math.Mod(back-from+540, 360) - 180); diff > 1e-6 {
			t.Errorf("direction %v came back as %v", from, back)
		}
	}
	// Averaging components across north does not give south.
	u1, v1 := WindComponents(10, 350)
	u2, v2 := WindComponents(10, 10)
	if _, mean := WindFrom((u1+u2)/2, (v1+v2)/2); math.Abs(mean) > 1e-6 && math.Abs(mean-360) > 1e-6 {
		t.Fatalf("mean of 350° and 10° = %v, want 0°", mean)
	}
	if speed, from := WindFrom(0, 0); speed != 0 || from != 0 {
		t.Fatal("calm has no direction")
	}
}
