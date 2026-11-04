package gdacs

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

var now = time.Date(2026, 11, 2, 19, 0, 0, 0, time.UTC)

func TestRelevantKeepsCurrentNorthIndianOceanEvents(t *testing.T) {
	events, skipped, err := DecodeEvents(read(t, "events.json"))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(events) != 4 {
		t.Fatalf("%d events, %d skipped", len(events), skipped)
	}
	relevant := Relevant(events, now)
	// Dropped: In-Fa is current but in the Pacific, Arnab ended, and Vance is
	// in the Pacific anyway.
	if len(relevant) != 1 {
		t.Fatalf("relevant = %+v", relevant)
	}
	murasu := relevant[0]
	if murasu.ID != 1001346 || murasu.Name != "MURASU-26" || murasu.AlertLevel != "Red" || murasu.AdvisorySource != "JTWC" {
		t.Fatalf("murasu = %+v", murasu)
	}
	if murasu.WindKmh == nil || *murasu.WindKmh != 120 {
		t.Fatalf("wind = %v", murasu.WindKmh)
	}
	if murasu.Lon != 80.7 || murasu.Lat != 14.7 || murasu.ReportURL != "https://www.gdacs.org/report.aspx?eventid=1001346&episodeid=7&eventtype=TC" {
		t.Fatalf("murasu = %+v", murasu)
	}
}

func TestIsCurrentIgnoresAFlagLeftSetOnAnOldEvent(t *testing.T) {
	event := Event{Current: true, LatestAt: now.Add(-72 * time.Hour)}
	if IsCurrent(event, now) {
		t.Fatal("an event with no advisory for three days is not current")
	}
	event.LatestAt = now.Add(-6 * time.Hour)
	if !IsCurrent(event, now) {
		t.Fatal("an event with a recent advisory is current")
	}
	event.Current = false
	if IsCurrent(event, now) {
		t.Fatal("an event GDACS calls ended is not current")
	}
}

func TestInBasin(t *testing.T) {
	cases := []struct {
		lon, lat float64
		want     bool
	}{
		{80.7, 14.7, true},  // Bay of Bengal
		{65, 18, true},      // Arabian Sea
		{140, 23, false},    // Western Pacific
		{48.6, -5.8, false}, // South Indian Ocean
		{110, 15, false},    // South China Sea
	}
	for _, c := range cases {
		if InBasin(c.lon, c.lat) != c.want {
			t.Errorf("InBasin(%v, %v) = %v", c.lon, c.lat, !c.want)
		}
	}
}

func TestDecodeGeometryKeepsWhatTheMapDraws(t *testing.T) {
	latest := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	features, err := DecodeGeometry(read(t, "geometry.json"), latest)
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	for _, feature := range features {
		count[feature.Properties.Kind]++
	}
	want := map[string]int{KindBuffer: 3, KindTrack: 6, KindForecast: 3, KindPosition: 10, KindCone: 1}
	for kind, number := range want {
		if count[kind] != number {
			t.Errorf("%s: %d features, want %d (%v)", kind, count[kind], number, count)
		}
	}
	for _, feature := range features {
		details := feature.Properties
		switch details.Kind {
		case KindBuffer:
			if details.WindKmh == 0 || details.Band == "" {
				t.Errorf("buffer = %+v", details)
			}
		case KindPosition:
			var point struct {
				Type        string     `json:"type"`
				Coordinates [2]float64 `json:"coordinates"`
			}
			if err := json.Unmarshal(feature.Geometry, &point); err != nil || point.Type != "Point" {
				t.Errorf("position geometry = %s", feature.Geometry)
			}
			if details.Time == nil {
				t.Errorf("position has no time: %+v", details)
				continue
			}
			if details.Forecast != details.Time.After(latest) {
				t.Errorf("position forecast flag wrong: %+v", details)
			}
		}
	}
}

func TestPositionTimeTakesTheYearNearestTheAdvisory(t *testing.T) {
	latest := time.Date(2027, 1, 1, 6, 0, 0, 0, time.UTC)
	at := positionTime("31/12 18:00 UTC", latest)
	if at == nil || !at.Equal(time.Date(2026, 12, 31, 18, 0, 0, 0, time.UTC)) {
		t.Fatalf("at = %v", at)
	}
	if positionTime("soon", latest) != nil {
		t.Fatal("an unreadable label has no time")
	}
}
