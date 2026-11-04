package stations

import (
	"testing"
	"time"
)

var now = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
var limits = Limits{ThreeHourly: 6 * time.Hour, Daily: DailyStaleAfter}

func number(value float64) *float64 { return &value }

func threeAt(age time.Duration, temperature, rain float64) Three {
	return Three{ObservedAt: now.Add(-age), TempC: number(temperature), RainMm: number(rain), Hum: number(80)}
}

func TestLatestPrefersACurrentThreeHourlyReading(t *testing.T) {
	readings := []Three{threeAt(2*time.Hour, 30, 1), threeAt(5*time.Hour, 28, 2)}
	day := &Day{ObservedAt: now.Add(-time.Hour), RainMm: number(12), Max: number(33)}
	got := Latest(Properties{}, readings, day, now, limits)
	if got.Reading != ReadingThreeHourly || got.Stale || *got.TemperatureC != 30 || got.RainHours != 3 {
		t.Fatalf("got %+v", got)
	}
}

func TestLatestFallsBackToANewerDailyReportWhenThreeHourlyIsOld(t *testing.T) {
	readings := []Three{threeAt(4*24*time.Hour, 30, 1)}
	day := &Day{ObservedAt: now.Add(-3 * time.Hour), RainMm: number(12), Min: number(24), Max: number(33)}
	got := Latest(Properties{}, readings, day, now, limits)
	if got.Reading != ReadingDaily || got.Stale || got.TemperatureC != nil || *got.Rain24hMm != 12 || got.RainHours != 24 || *got.TempMaxC != 33 {
		t.Fatalf("got %+v", got)
	}
	if !got.ReportsTemperature {
		t.Fatal("a station with a daily maximum reports temperature")
	}
}

func TestLatestMarksTheNewestOldReadingStaleWhenBothFeedsAreOld(t *testing.T) {
	readings := []Three{threeAt(4*24*time.Hour, 30, 1)}
	day := &Day{ObservedAt: now.Add(-10 * 24 * time.Hour), RainMm: number(12)}
	got := Latest(Properties{}, readings, day, now, limits)
	if got.Reading != ReadingThreeHourly || !got.Stale || *got.TemperatureC != 30 {
		t.Fatalf("got %+v", got)
	}
	old := Latest(Properties{}, nil, day, now, limits)
	if old.Reading != ReadingDaily || !old.Stale {
		t.Fatalf("daily alone = %+v", old)
	}
	none := Latest(Properties{}, nil, nil, now, limits)
	if none.Reading != ReadingNone || !none.Stale || none.ReportsTemperature {
		t.Fatalf("no readings = %+v", none)
	}
}

func TestDailyRainOnlyStationDoesNotReportTemperature(t *testing.T) {
	got := Latest(Properties{}, nil, &Day{ObservedAt: now.Add(-time.Hour), RainMm: number(5)}, now, limits)
	if got.ReportsTemperature {
		t.Fatalf("got %+v", got)
	}
}

func TestThreeHourlyDayTotalsNeedMostOfTheDay(t *testing.T) {
	var day []Three
	for slot := range 8 {
		day = append(day, threeAt(time.Duration(slot*3)*time.Hour, float64(20+slot), 1.5))
	}
	got := Latest(Properties{}, day, nil, now, limits)
	if got.Rain24hMm == nil || *got.Rain24hMm != 12 || *got.TempMinC != 20 || *got.TempMaxC != 27 {
		t.Fatalf("got %+v", got)
	}
	if partial := Latest(Properties{}, day[:5], nil, now, limits); partial.Rain24hMm != nil {
		t.Fatalf("five readings are not a day: %+v", partial)
	}
}

func TestGaugeTotalSumsTheDayAndCountsRepeatsOnce(t *testing.T) {
	end := now
	var readings []GaugeReading
	for hour := range 24 {
		readings = append(readings, GaugeReading{ObservedAt: end.Add(-time.Duration(hour) * time.Hour), RainMm: number(1)})
	}
	// Re-entered 17 seconds apart; and a reading a whole day old, outside the window.
	readings = append(readings[:3:3], append([]GaugeReading{{ObservedAt: end.Add(-2*time.Hour - 17*time.Second), RainMm: number(1)}}, readings[3:]...)...)
	readings = append(readings, GaugeReading{ObservedAt: end.Add(-24 * time.Hour), RainMm: number(50)})
	total, at, ok := GaugeTotal(readings)
	if !ok || total != 24 || !at.Equal(end) {
		t.Fatalf("total = %v at %s, %v", total, at, ok)
	}
}

func TestGaugeTotalNeedsMostOfTheDay(t *testing.T) {
	var readings []GaugeReading
	for hour := range 12 {
		readings = append(readings, GaugeReading{ObservedAt: now.Add(-time.Duration(hour) * time.Hour), RainMm: number(1)})
	}
	if _, _, ok := GaugeTotal(readings); ok {
		t.Fatal("twelve hours of readings are not a day")
	}
	if _, _, ok := GaugeTotal(nil); ok {
		t.Fatal("no readings are not a day")
	}
}

func TestNearestInPicksTheClosestWithATemperature(t *testing.T) {
	feature := func(id string, lon, lat float64, temperature *float64) Feature {
		return Feature{ID: id, Geometry: Point{Coordinates: [2]float64{lon, lat}},
			Properties: Properties{ID: id, Reading: ReadingThreeHourly, TemperatureC: temperature, ReportsTemperature: temperature != nil}}
	}
	layer := FeatureCollection{Features: []Feature{
		feature("rain-only", 79.87, 6.91, nil),
		feature("colombo", 79.867, 6.9, number(29)),
		feature("ratmalana", 79.886, 6.822, number(28)),
	}}
	got := NearestIn(layer, 79.86, 6.92)
	if got == nil || got.ID != "colombo" || got.DistanceKm > 3 {
		t.Fatalf("nearest = %+v", got)
	}
	if far := NearestIn(layer, 81.8, 9.5); far != nil {
		t.Fatalf("nothing should be within 75 km, got %+v", far)
	}
}

// Each pair is from a day the Irrigation Department's bulletin settles.
func TestGaugeTotalKeepsABacklogAndCountsAReEntryOnce(t *testing.T) {
	end := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	mm := func(value float64) *float64 { return &value }
	filler := func(readings []GaugeReading, from time.Time, count int) []GaugeReading {
		for index := range count {
			readings = append(readings, GaugeReading{ObservedAt: from.Add(-time.Duration(index) * time.Hour), WaterLevel: 1, RainMm: mm(0)})
		}
		return readings
	}
	start := end.Add(-2 * time.Hour)

	// Pitabeddara, 1 November: 1.67 m with 4.4 mm, then 1.69 m with 0, a second apart.
	backlog := filler([]GaugeReading{
		{ObservedAt: start.Add(time.Second), WaterLevel: 1.69, RainMm: mm(0)},
		{ObservedAt: start, WaterLevel: 1.67, RainMm: mm(4.4)},
	}, start.Add(-time.Hour), 20)
	if total, _, ok := GaugeTotal(backlog); !ok || total != 4.4 {
		t.Fatalf("a backlog should keep its rain: %v %v", total, ok)
	}

	// Ellagawa, 30 October: 8.72 m with 9.5 mm, twice, seconds apart.
	repeat := filler([]GaugeReading{
		{ObservedAt: start.Add(2 * time.Second), WaterLevel: 8.72, RainMm: mm(9.5)},
		{ObservedAt: start, WaterLevel: 8.72, RainMm: mm(9.5)},
	}, start.Add(-time.Hour), 20)
	if total, _, ok := GaugeTotal(repeat); !ok || total != 9.5 {
		t.Fatalf("a repeat should count once: %v %v", total, ok)
	}
}
