package alerts

import (
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
)

func intensities(at time.Time, gauges ...raingauges.Intensity) raingauges.Intensities {
	fetched := at
	for index := range gauges {
		if gauges[index].ObservedAt.IsZero() {
			gauges[index].ObservedAt = at.Add(-10 * time.Minute)
		}
	}
	return raingauges.Intensities{Gauges: gauges, FetchedAt: &fetched}
}

func inColombo(float64, float64) string { return "LK-11" }

func TestRainLevelTakesTheShortestWindowThatReachesIt(t *testing.T) {
	cases := []struct {
		gauge raingauges.Intensity
		want  RainMeasure
	}{
		{raingauges.Intensity{Rise1hMm: 67.5, Rise3hMm: 67.5, Rain24hMm: 83.5}, RainMeasure{2, "very_heavy", "1h", 67.5}},
		{raingauges.Intensity{Rise1hMm: 37.5, Rise3hMm: 112.5, Rain24hMm: 128.5}, RainMeasure{2, "very_heavy", "3h", 112.5}},
		{raingauges.Intensity{Rise1hMm: 5, Rise3hMm: 10, Rain24hMm: 160}, RainMeasure{2, "very_heavy", "24h", 160}},
		{raingauges.Intensity{Rise1hMm: 80}, RainMeasure{3, "extreme", "1h", 80}},
		{raingauges.Intensity{Rise1hMm: 31}, RainMeasure{1, "heavy", "1h", 31}},
		{raingauges.Intensity{Rise1hMm: 10, Rise3hMm: 20, Rain24hMm: 60}, RainMeasure{}},
	}
	for _, c := range cases {
		if got := RainLevel(c.gauge); got != c.want {
			t.Errorf("%+v: got %+v, want %+v", c.gauge, got, c.want)
		}
	}
}

func TestHeavyRainWouldHaveWarnedColomboAt1820(t *testing.T) {
	// The Department of Meteorology's head-office gauge, 6 November 2026.
	at := time.Date(2026, 11, 6, 13, 5, 0, 0, time.UTC)
	gauge := raingauges.Intensity{ID: "argcbo01", Name: "DOM HEAD OFFICE", Lon: 79.87, Lat: 6.9,
		ObservedAt: time.Date(2026, 11, 6, 12, 50, 0, 0, time.UTC), Rise1hMm: 67.5, Rise3hMm: 67.5, Rain24hMm: 83.5}
	dry := raingauges.Intensity{ID: "kesbewa", Name: "Kesbewa", Lon: 79.93, Lat: 6.8, Rise1hMm: 3.5, Rain24hMm: 48.5}
	reading := HeavyRain(intensities(at, dry, gauge), inColombo, at)
	if !reading.Complete || len(reading.Observations) != 1 {
		t.Fatalf("reading %+v", reading)
	}
	observation := reading.Observations[0]
	event := observation.Event
	if observation.Subject != "rain:district:LK-11" || observation.Level != 2 || event.Severity != push.SeverityWarning ||
		event.Facts["gauge"] != "DOM HEAD OFFICE" || event.Facts["mm"] != "67" || event.Facts["window"] != "1h" ||
		event.URL != "/?raingauge=argcbo01" || event.Topic != push.TopicHeavyRain {
		t.Fatalf("observation %+v", observation)
	}
	// Once alerted, the burst holding in the three-hour rise is the same episode.
	states := map[string]State{}
	for _, transition := range Apply(reading, states, at) {
		states[transition.State.Subject] = transition.State
	}
	later := at.Add(time.Hour)
	gauge.ObservedAt, gauge.Rise1hMm, gauge.Rise3hMm, gauge.Rain24hMm = later.Add(-5*time.Minute), 37.5, 112.5, 128.5
	for _, transition := range Apply(HeavyRain(intensities(later, gauge), inColombo, later), states, later) {
		if transition.Event != nil {
			t.Fatalf("alerted again: %+v", transition.Event)
		}
	}
}

func TestOutOfDateAndSuspectGaugesSayNothing(t *testing.T) {
	at := time.Date(2026, 11, 6, 13, 0, 0, 0, time.UTC)
	old := raingauges.Intensity{ID: "old", Lon: 79.87, Lat: 6.9, ObservedAt: at.Add(-3 * time.Hour), Rise1hMm: 90}
	broken := raingauges.Intensity{ID: "broken", Lon: 79.87, Lat: 6.9, Rise1hMm: 250, Suspect: true}
	reading := HeavyRain(intensities(at, old, broken), inColombo, at)
	if len(reading.Observations) != 1 || !reading.Observations[0].Hold {
		t.Fatalf("an out-of-date district is held: %+v", reading.Observations)
	}
	stale := at.Add(-2 * time.Hour)
	if HeavyRain(raingauges.Intensities{FetchedAt: &stale}, inColombo, at).Complete {
		t.Fatal("a feed not stored for two hours cannot lower anything")
	}
}

func TestTheWettestCurrentGaugeLeadsItsDistrict(t *testing.T) {
	at := time.Date(2026, 11, 6, 13, 0, 0, 0, time.UTC)
	light := raingauges.Intensity{ID: "a", Name: "A", Lon: 79.9, Lat: 6.9, Rise1hMm: 32}
	heavy := raingauges.Intensity{ID: "b", Name: "B", Lon: 79.9, Lat: 6.9, Rise3hMm: 120}
	stale := raingauges.Intensity{ID: "c", Name: "C", Lon: 79.9, Lat: 6.9, ObservedAt: at.Add(-5 * time.Hour), Rise1hMm: 99}
	reading := HeavyRain(intensities(at, light, stale, heavy), inColombo, at)
	if len(reading.Observations) != 1 || reading.Observations[0].Event.Facts["gauge"] != "B" || reading.Observations[0].Level != 2 {
		t.Fatalf("%+v", reading.Observations)
	}
}
