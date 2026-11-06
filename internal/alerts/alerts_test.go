package alerts

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

var start = time.Date(2026, 11, 3, 6, 0, 0, 0, time.UTC)

// replay feeds a gauge's statuses, one every 15 minutes, through the engine
// and returns the levels alerted.
func replay(t *testing.T, statuses []rivers.Status, stale map[int]bool) []int {
	t.Helper()
	states := map[string]State{}
	alerted := []int{}
	keys := map[string]bool{}
	for index, status := range statuses {
		now := start.Add(time.Duration(index) * 15 * time.Minute)
		collection := rivers.FeatureCollection{Features: []rivers.Feature{{ID: "nagalagam",
			Geometry:   rivers.Point{Type: "Point", Coordinates: [2]float64{79.87, 6.96}},
			Properties: rivers.StationProperties{ID: "nagalagam", Name: "Nagalagam Street", Basin: "Kelani Ganga", Status: status, Stale: stale[index], Source: irrigation.SourceName}}}}
		reading := Rivers(collection, func(float64, float64) string { return "LK-11" }, now)
		for _, transition := range Apply(reading, states, now) {
			if transition.State.Idle() {
				delete(states, transition.State.Subject)
			} else {
				states[transition.State.Subject] = transition.State
			}
			if event := transition.Event; event != nil {
				if keys[event.Key] {
					t.Fatalf("event key %s raised twice", event.Key)
				}
				keys[event.Key] = true
				alerted = append(alerted, event.Severity.Rank())
			}
		}
	}
	return alerted
}

func repeat(status rivers.Status, times int) []rivers.Status {
	out := make([]rivers.Status, times)
	for index := range out {
		out[index] = status
	}
	return out
}

func concat(parts ...[]rivers.Status) []rivers.Status {
	out := []rivers.Status{}
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

const (
	watch   = 2 // push.SeverityWatch.Rank()
	warning = 3
	danger  = 4
)

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func TestRisingAcrossLevelsAlertsEachEscalation(t *testing.T) {
	got := replay(t, []rivers.Status{rivers.StatusNormal, rivers.StatusAlert, rivers.StatusAlert, rivers.StatusMinorFlood, rivers.StatusMajorFlood, rivers.StatusMajorFlood}, nil)
	if !equal(got, []int{watch, warning, danger}) {
		t.Fatalf("alerted %v", got)
	}
}

func TestJumpingStraightToMajorFloodAlertsOnce(t *testing.T) {
	if got := replay(t, []rivers.Status{rivers.StatusNormal, rivers.StatusMajorFlood, rivers.StatusMajorFlood}, nil); !equal(got, []int{danger}) {
		t.Fatalf("alerted %v", got)
	}
}

func TestHoveringAtAThresholdDoesNotAlertAgain(t *testing.T) {
	// Minor flood, then in and out of it every 15 minutes for three hours.
	flapping := []rivers.Status{rivers.StatusMinorFlood}
	for range 6 {
		flapping = append(flapping, rivers.StatusAlert, rivers.StatusMinorFlood)
	}
	if got := replay(t, flapping, nil); !equal(got, []int{warning}) {
		t.Fatalf("alerted %v", got)
	}
}

func TestFallingBelowForTheHoldTimeRearms(t *testing.T) {
	// Six hours and a quarter below minor flood (25 readings) is longer than the hold.
	statuses := concat([]rivers.Status{rivers.StatusMinorFlood}, repeat(rivers.StatusNormal, 26), []rivers.Status{rivers.StatusMinorFlood})
	if got := replay(t, statuses, nil); !equal(got, []int{warning, warning}) {
		t.Fatalf("alerted %v", got)
	}
	// Five hours below is not.
	statuses = concat([]rivers.Status{rivers.StatusMinorFlood}, repeat(rivers.StatusNormal, 20), []rivers.Status{rivers.StatusMinorFlood})
	if got := replay(t, statuses, nil); !equal(got, []int{warning}) {
		t.Fatalf("alerted %v", got)
	}
}

func TestStaleReadingsAreNeitherARiseNorAFall(t *testing.T) {
	// Stale for seven hours in the middle: no rearm, no alert.
	statuses := concat([]rivers.Status{rivers.StatusMinorFlood}, repeat(rivers.StatusNormal, 28), []rivers.Status{rivers.StatusMinorFlood})
	stale := map[int]bool{}
	for index := 1; index <= 28; index++ {
		stale[index] = true
	}
	if got := replay(t, statuses, stale); !equal(got, []int{warning}) {
		t.Fatalf("alerted %v", got)
	}
	if got := replay(t, []rivers.Status{rivers.StatusUnknown, rivers.StatusUnknown}, nil); len(got) != 0 {
		t.Fatalf("a gauge without thresholds alerted %v", got)
	}
}

func TestStepRearmsToTheLowerLevel(t *testing.T) {
	state := State{Subject: "river:x"}
	state, alert := Step(state, 3, false, time.Hour, start)
	if !alert || state.AlertedLevel != 3 {
		t.Fatalf("first %+v", state)
	}
	state, _ = Step(state, 1, false, time.Hour, start.Add(time.Minute))
	state, _ = Step(state, 1, false, time.Hour, start.Add(2*time.Hour))
	if state.AlertedLevel != 1 || state.BelowSince != nil {
		t.Fatalf("after the hold %+v", state)
	}
	if _, alert = Step(state, 2, false, time.Hour, start.Add(3*time.Hour)); !alert {
		t.Fatal("rising above the rearmed level alerts")
	}
	if key := (State{Subject: "river:x", AlertedLevel: 2, AlertedAt: &start}).EventKey(); key != "river:x:2:1793685600" {
		t.Fatalf("key %s", key)
	}
}

func TestEventsCarryTheirEpisodeKey(t *testing.T) {
	reading := Reading{Kind: messages.KindRiver, HoldFor: HoldRiver, Complete: true, Observations: []Observation{{Subject: "river:a", Level: 2,
		Event: push.Event{Kind: messages.KindRiver, Variant: "minor_flood", Topic: push.TopicRiver, Severity: push.SeverityWarning}}}}
	first := Apply(reading, map[string]State{}, start)
	if len(first) != 1 || first[0].Event == nil || first[0].Event.Key != "river:a:2:1793685600" || !first[0].Event.DetectedAt.Equal(start) {
		t.Fatalf("first %+v", first)
	}
	again := Apply(reading, map[string]State{"river:a": first[0].State}, start.Add(time.Minute))
	if len(again) != 1 || again[0].Event != nil {
		t.Fatalf("same level again %+v", again)
	}
}

func TestACompleteReadingLowersSubjectsItNoLongerMentions(t *testing.T) {
	alertedAt := start
	states := map[string]State{
		"landslide:LK-91": {Subject: "landslide:LK-91", Kind: messages.KindLandslide, Level: 3, AlertedLevel: 3, AlertedAt: &alertedAt},
		"river:a":         {Subject: "river:a", Kind: messages.KindRiver, Level: 2, AlertedLevel: 2, AlertedAt: &alertedAt},
	}
	none := Landslides(warnings.LandslideLayer{Meta: warnings.LandslideMeta{Status: warnings.StatusNone}}, start.Add(time.Hour))
	transitions := Apply(none, states, start.Add(time.Hour))
	if len(transitions) != 1 || transitions[0].State.Subject != "landslide:LK-91" || transitions[0].State.Level != 0 || transitions[0].State.BelowSince == nil {
		t.Fatalf("none %+v", transitions)
	}
	lapsed := Landslides(warnings.LandslideLayer{Meta: warnings.LandslideMeta{Status: warnings.StatusLapsed}}, start.Add(time.Hour))
	if transitions := Apply(lapsed, states, start.Add(time.Hour)); len(transitions) != 0 {
		t.Fatalf("a lapsed report says nothing: %+v", transitions)
	}
}

func division(id, district string, level int) warnings.DivisionFeature {
	return warnings.DivisionFeature{ID: id, Properties: warnings.DivisionAt{ID: id, DistrictID: district, LevelWords: warnings.WordsFor(level)}}
}

func TestLandslidesTakeEachDistrictsHighestLevel(t *testing.T) {
	until := start.Add(20 * time.Hour)
	layer := warnings.LandslideLayer{Features: []warnings.DivisionFeature{division("LK-9103", "LK-91", 2), division("LK-9109", "LK-91", 3), division("LK-9203", "LK-92", 1)},
		Meta: warnings.LandslideMeta{Status: warnings.StatusActive, ValidUntil: &until}}
	reading := Landslides(layer, start)
	if !reading.Complete || len(reading.Observations) != 2 {
		t.Fatalf("reading %+v", reading)
	}
	ratnapura, kegalle := reading.Observations[0], reading.Observations[1]
	if ratnapura.Level != 3 || ratnapura.Event.Severity != push.SeverityDanger || ratnapura.Event.Topic != push.TopicLandslide ||
		ratnapura.Event.Facts["divisions"] != "2" || ratnapura.Event.Tag != "landslide:district:LK-91" || !ratnapura.Event.ExpiresAt.Equal(until) {
		t.Fatalf("Ratnapura %+v", ratnapura)
	}
	if kegalle.Level != 1 || kegalle.Event.Topic != push.TopicLandslideLevel1 || kegalle.Event.Severity != push.SeverityWatch {
		t.Fatalf("Level 1 is its own opt-in topic: %+v", kegalle)
	}
}

func colour(value string) *string { return &value }

func TestBulletinsInForceAreOfficialAndMatchedByDistrict(t *testing.T) {
	until := start.Add(6 * time.Hour)
	overview := &warnings.Overview{InForce: []warnings.Bulletin{
		{ID: "red", Type: "weather", Title: "Red Notice for Heavy Rain", Severity: "warning", Colour: colour("Red"), ValidUntil: &until,
			Districts: []warnings.DistrictRef{{ID: "LK-91"}, {ID: "LK-31"}}},
		{ID: "amber", Type: "weather", Title: "Weather Advisory", Severity: "advisory", Colour: colour("Amber")},
		{ID: "water", Type: "water_level", Title: "Flood Warning for Kalu River - No 2", Severity: "warning", Districts: []warnings.DistrictRef{{ID: "LK-91"}}},
		{ID: "landslide", Type: "landslide", Title: "Landslide Early Warning", Severity: "warning"},
	}}
	reading := Bulletins(overview, start)
	if len(reading.Observations) != 3 || !reading.Complete {
		t.Fatalf("reading %+v", reading)
	}
	red, amber, water := reading.Observations[0], reading.Observations[1], reading.Observations[2]
	if red.Level != 3 || red.Event.Severity != push.SeverityDanger || red.Event.National || red.Event.Facts["districts"] != "LK-31,LK-91" ||
		red.Event.Topic != push.TopicOfficial || !red.Event.ExpiresAt.Equal(until) || red.Event.Facts["title"] != "Red Notice for Heavy Rain" {
		t.Fatalf("red %+v", red)
	}
	if amber.Level != 2 || amber.Event.Variant != "advisory" || !amber.Event.National {
		t.Fatalf("an amber advisory with no district goes to everyone: %+v", amber)
	}
	if water.Event.URL != "/?warnings=water_level&district=LK-91" || water.Event.Variant != "warning" {
		t.Fatalf("water %+v", water)
	}
	// The next update of the same warning is the same subject.
	update := *overview
	update.InForce = []warnings.Bulletin{{ID: "water3", Type: "water_level", Title: "Flood Warning for Kalu River - No 3", Severity: "warning"}}
	if Bulletins(&update, start).Observations[0].Subject != water.Subject {
		t.Fatal("a numbered update is the same warning")
	}
	overview.Meta.Stale = true
	if Bulletins(overview, start).Complete {
		t.Fatal("a stale feed cannot end a warning")
	}
}

func TestFloodForecastsAreVerifiedEstimates(t *testing.T) {
	collection := floods.FeatureCollection{Meta: floods.Meta{Configured: true, FetchedAt: &start}, Features: []floods.Feature{
		{ID: "g1", Properties: floods.Gauge{GaugeID: "g1", SiteName: "Putupaula", River: "Kalu Ganga", QualityVerified: true, Severity: "severe", Lon: 80.1, Lat: 6.6}},
		{ID: "g2", Properties: floods.Gauge{GaugeID: "g2", QualityVerified: false, Severity: "extreme"}},
		{ID: "g3", Properties: floods.Gauge{GaugeID: "g3", QualityVerified: true, Severity: "extreme", Seeded: true}},
		{ID: "g4", Properties: floods.Gauge{GaugeID: "g4", QualityVerified: true, Severity: "extreme", Lon: 80.5, Lat: 7}},
		{ID: "a1", Properties: floods.Area{ID: "a1", Kind: "flash_highly_likely"}},
	}}
	reading := FloodForecasts(collection, func(float64, float64) string { return "LK-91" }, start)
	if len(reading.Observations) != 2 {
		t.Fatalf("only verified, unseeded gauges: %+v", reading.Observations)
	}
	severe, extreme := reading.Observations[0], reading.Observations[1]
	if severe.Level != 2 || severe.Event.Severity != push.SeverityWarning || severe.Event.Variant != "danger" || severe.Event.URL != "/?gauge=g1" {
		t.Fatalf("severe %+v", severe)
	}
	if extreme.Event.Severity != push.SeverityDanger {
		t.Fatalf("extreme %+v", extreme)
	}
	flash := FlashFloods(collection, func(string) []string { return []string{"LK-91", "LK-92"} }, start)
	if len(flash.Observations) != 2 || flash.Observations[0].Level != 2 || flash.Observations[0].Subject != "flash:district:LK-91" ||
		flash.Observations[1].Event.Districts[0] != "LK-92" {
		t.Fatalf("flash %+v", flash)
	}
	if flash.Complete {
		t.Fatal("a reading with no flash-flood run cannot be complete")
	}
}

// flashRun is one read of Google's flash floods: the areas in force and the
// districts each overlaps.
type flashRun struct {
	at    time.Time
	areas map[string][]string
	kinds map[string]string
}

// replayFlash feeds runs through the engine and returns the districts alerted
// at each run.
func replayFlash(t *testing.T, runs []flashRun) [][]string {
	t.Helper()
	states := map[string]State{}
	alerted := [][]string{}
	for _, run := range runs {
		fetched := run.at
		collection := floods.FeatureCollection{Meta: floods.Meta{Configured: true, FetchedAt: &fetched, FlashFetchedAt: &fetched}}
		for id := range run.areas {
			kind := run.kinds[id]
			if kind == "" {
				kind = "flash_likely"
			}
			issued := run.at.Add(-time.Hour)
			collection.Features = append(collection.Features, floods.Feature{ID: id, Properties: floods.Area{ID: id, Kind: kind, IssuedAt: &issued}})
		}
		reading := FlashFloods(collection, func(id string) []string { return run.areas[id] }, run.at)
		districts := []string{}
		for _, transition := range Apply(reading, states, run.at) {
			if transition.State.Idle() {
				delete(states, transition.State.Subject)
			} else {
				states[transition.State.Subject] = transition.State
			}
			if transition.Event != nil {
				districts = append(districts, transition.Event.Districts...)
			}
		}
		sort.Strings(districts)
		alerted = append(alerted, districts)
	}
	return alerted
}

func TestFlashFloodsAlertADistrictOnceAcrossGooglesReissues(t *testing.T) {
	hour := func(h int) time.Time { return start.Add(time.Duration(h) * time.Hour) }
	colombo := []string{"LK-11", "LK-12"}
	wider := []string{"LK-11", "LK-12", "LK-13"}
	got := replayFlash(t, []flashRun{
		{at: hour(0), areas: map[string][]string{"area-1": colombo}},
		// Reissued under a new ID: no new notice for Colombo or Gampaha,
		// one for Kalutara, which the new area adds.
		{at: hour(6), areas: map[string][]string{"area-2": wider}},
		{at: hour(11), areas: map[string][]string{"area-3": wider}},
		// Withdrawn: nothing is sent.
		{at: hour(12), areas: map[string][]string{}},
		// Back within the hold: still the same episode.
		{at: hour(16), areas: map[string][]string{"area-4": wider}},
		// Raised to highly likely: an escalation always alerts.
		{at: hour(17), areas: map[string][]string{"area-5": {"LK-11"}}, kinds: map[string]string{"area-5": "flash_highly_likely"}},
	})
	want := [][]string{{"LK-11", "LK-12"}, {"LK-13"}, {}, {}, {}, {"LK-11"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("alerted %v, want %v", got, want)
	}
}

func TestAFlashFloodGoneLongerThanTheHoldAlertsAgain(t *testing.T) {
	hour := func(h int) time.Time { return start.Add(time.Duration(h) * time.Hour) }
	runs := []flashRun{{at: hour(0), areas: map[string][]string{"area-1": {"LK-11"}}}}
	for h := 1; h <= 8; h++ {
		runs = append(runs, flashRun{at: hour(h), areas: map[string][]string{}})
	}
	runs = append(runs, flashRun{at: hour(9), areas: map[string][]string{"area-2": {"LK-11"}}})
	got := replayFlash(t, runs)
	if len(got[0]) != 1 || len(got[9]) != 1 {
		t.Fatalf("alerted %v", got)
	}
}

func TestAStaleFlashReadCannotEndAForecast(t *testing.T) {
	old := start.Add(-time.Hour)
	collection := floods.FeatureCollection{Meta: floods.Meta{Configured: true, FetchedAt: &start, FlashFetchedAt: &old}}
	if FlashFloods(collection, func(string) []string { return nil }, start).Complete {
		t.Fatal("an hour-old flash read must not lower anything")
	}
}

func cycloneLayer(t *testing.T, distanceKm float64, forecast [][2]float64, stale bool) hazards.CycloneLayer {
	t.Helper()
	line, _ := json.Marshal(map[string]any{"type": "LineString", "coordinates": forecast})
	return hazards.CycloneLayer{Meta: hazards.CycloneMeta{Status: hazards.CycloneActive}, Features: []hazards.CycloneFeature{
		{Properties: map[string]any{"kind": "position", "eventId": "1001", "name": "MONTHA", "distanceKm": distanceKm, "stale": stale}},
		{Geometry: line, Properties: map[string]any{"kind": "forecast-track", "eventId": "1001"}},
	}}
}

func TestCyclonesByDistanceAndForecastTrack(t *testing.T) {
	farTrack := [][2]float64{{95, 15}, {93, 16}}
	nearTrack := [][2]float64{{90, 12}, {84, 9}}
	cases := []struct {
		name     string
		layer    hazards.CycloneLayer
		level    int
		variant  string
		distance string
	}{
		{"far away", cycloneLayer(t, 2000, farTrack, false), 0, "", ""},
		{"forecast to approach", cycloneLayer(t, 2000, nearTrack, false), 1, "track", ""},
		{"within 1,000 km", cycloneLayer(t, 800, farTrack, false), 2, "near", "800"},
		{"within 300 km", cycloneLayer(t, 250, farTrack, false), 3, "close", "250"},
	}
	for _, test := range cases {
		reading := Cyclones(test.layer, start)
		observation := reading.Observations[0]
		if observation.Level != test.level || observation.Event.Variant != test.variant || !observation.Event.National {
			t.Errorf("%s: %+v", test.name, observation)
		}
		if test.distance != "" && observation.Event.Facts["distance"] != test.distance {
			t.Errorf("%s distance %s", test.name, observation.Event.Facts["distance"])
		}
	}
	if observation := Cyclones(cycloneLayer(t, 250, farTrack, true), start).Observations[0]; !observation.Hold {
		t.Fatal("an old advisory is held")
	}
}

func TestQuakeThresholds(t *testing.T) {
	cases := []struct {
		magnitude, distance float64
		tsunami             bool
		want                int
	}{
		{4.6, 2500, false, 0}, {5.2, 450, false, 1}, {5.2, 900, false, 0}, {6.1, 3000, false, 1},
		{4.2, 250, false, 1}, {6.7, 1400, false, 2}, {6.7, 2500, false, 1}, {7.2, 3500, false, 2}, {6.2, 2500, true, 2},
	}
	for _, test := range cases {
		if got := QuakeLevel(test.magnitude, test.distance, test.tsunami); got != test.want {
			t.Errorf("M%.1f at %.0f km: %d, want %d", test.magnitude, test.distance, got, test.want)
		}
	}
	layer := hazards.EarthquakeLayer{Features: []hazards.EarthquakeFeature{
		{Properties: hazards.Earthquake{ID: "new", Magnitude: 6.8, DistanceKm: 1200, Place: "off the west coast of northern Sumatra", Time: start.Add(-time.Hour)}},
		{Properties: hazards.Earthquake{ID: "old", Magnitude: 7.5, DistanceKm: 1200, Time: start.Add(-7 * time.Hour)}},
	}}
	reading := Earthquakes(layer, start)
	if len(reading.Observations) != 1 || reading.Observations[0].Event.Variant != "major" || reading.Observations[0].Event.Facts["magnitude"] != "6.8" {
		t.Fatalf("only recent quakes: %+v", reading.Observations)
	}
}

func TestReservoirsSpillingOnCurrentReadings(t *testing.T) {
	district := func(value string) *string { return &value }
	collection := reservoirs.FeatureCollection{Meta: reservoirs.Meta{FetchedAt: &start}, Features: []reservoirs.Feature{
		{Properties: reservoirs.Properties{ID: "kalawewa", Name: "Kala Wewa", District: district("Anuradhapura"), Status: "spilling"}},
		{Properties: reservoirs.Properties{ID: "old", Name: "Old", District: district("Anuradhapura"), Status: "spilling", Stale: true}},
		{Properties: reservoirs.Properties{ID: "nowhere", Name: "Nowhere", Status: "spilling"}},
		{Properties: reservoirs.Properties{ID: "holding", Name: "Holding", District: district("Nuwara-Eliya"), Status: "holding"}},
	}}
	named := func(name string) string {
		return map[string]string{normaliseName("Anuradhapura"): "LK-71", normaliseName("Nuwara Eliya"): "LK-23"}[normaliseName(name)]
	}
	reading := Reservoirs(collection, named, start)
	if len(reading.Observations) != 3 {
		t.Fatalf("observations %+v", reading.Observations)
	}
	if spill := reading.Observations[0]; spill.Level != 1 || spill.Event.Districts[0] != "LK-71" || spill.Event.Topic != push.TopicReservoir {
		t.Fatalf("spill %+v", spill)
	}
	if !reading.Observations[1].Hold || reading.Observations[2].Level != 0 {
		t.Fatalf("stale and holding %+v", reading.Observations[1:])
	}
}

// TestEveryKindRendersInEveryLanguage checks that what each detector raises
// has text in all three languages.
func TestEveryKindRendersInEveryLanguage(t *testing.T) {
	events := []push.Event{}
	for _, reading := range []Reading{
		Landslides(warnings.LandslideLayer{Features: []warnings.DivisionFeature{division("LK-9103", "LK-91", 3)}, Meta: warnings.LandslideMeta{Status: warnings.StatusActive}}, start),
		Cyclones(cycloneLayer(t, 250, nil, false), start),
		Earthquakes(hazards.EarthquakeLayer{Features: []hazards.EarthquakeFeature{{Properties: hazards.Earthquake{ID: "q", Magnitude: 7.1, DistanceKm: 1300, Place: "Sumatra", Time: start}}}}, start),
	} {
		for _, observation := range reading.Observations {
			events = append(events, observation.Event)
		}
	}
	for _, event := range events {
		for _, locale := range messages.Locales {
			title, body := messages.Render(locale, messages.Input{Kind: event.Kind, Variant: event.Variant, Count: 1, Facts: event.Facts}, push.DistrictNames{})
			if title == "" || body == "" {
				t.Errorf("%s %s.%s is empty", locale, event.Kind, event.Variant)
			}
		}
	}
}
