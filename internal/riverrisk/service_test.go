package riverrisk

import (
	"math"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/store"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

func TestAccumulateRainIsTheMeanOverTheWholeCatchment(t *testing.T) {
	// 1 and 2 flow into 3, which reaches the sea.
	got := AccumulateRain([]CatchmentSegment{
		{ID: 3, NextDown: 0, CatchKM2: 10, Next24hMM: 0, Next72hMM: 0},
		{ID: 1, NextDown: 3, CatchKM2: 30, Next24hMM: 100, Next72hMM: 200},
		{ID: 2, NextDown: 3, CatchKM2: 60, Next24hMM: 50, Next72hMM: 100},
	})
	if got[1].Next24hMM != 100 || got[1].CatchmentKM2 != 30 {
		t.Fatalf("headwater = %+v", got[1])
	}
	// (30×100 + 60×50 + 10×0) / 100
	if got[3].Next24hMM != 60 || got[3].Next72hMM != 120 || got[3].CatchmentKM2 != 100 {
		t.Fatalf("mouth = %+v", got[3])
	}
}

func TestAccumulateRainLeavesOutPlacesOffTheGrid(t *testing.T) {
	nan := math.NaN()
	got := AccumulateRain([]CatchmentSegment{
		{ID: 2, NextDown: 0, CatchKM2: 50, Next24hMM: 20, Next72hMM: 40},
		{ID: 1, NextDown: 2, CatchKM2: 50, Next24hMM: nan, Next72hMM: nan},
	})
	if _, ok := got[1]; ok {
		t.Fatalf("a catchment wholly off the grid says nothing: %+v", got[1])
	}
	if got[2].Next24hMM != 20 || got[2].CatchmentKM2 != 100 {
		t.Fatalf("mouth = %+v", got[2])
	}
}

func TestNodeTotals(t *testing.T) {
	start := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	const steps, nodes = 100, 2
	values := make([]float32, steps*nodes)
	for step := range steps {
		values[step*nodes] = 1
		values[step*nodes+1] = float32(math.NaN())
	}
	field := &weather.Field{ValidFrom: start, Step: time.Hour, Steps: steps, Values: values}
	next24, next72 := NodeTotals(field, start.Add(10*time.Hour))
	if next24[0] != 24 || next72[0] != 72 {
		t.Fatalf("totals %v %v", next24[0], next72[0])
	}
	if !math.IsNaN(float64(next24[1])) {
		t.Fatalf("a node with no values should have none, got %v", next24[1])
	}
	if a, b := NodeTotals(field, start.Add(-10*24*time.Hour)); a != nil || b != nil {
		t.Fatal("a forecast that does not cover now should give nothing")
	}
	// Near the end of the forecast, what is left is what is added.
	_, last := NodeTotals(field, start.Add(90*time.Hour))
	if last[0] != 10 {
		t.Fatalf("end of forecast = %v", last[0])
	}
}

func stationFeature(id string, level float64, status rivers.Status, stale bool, observed time.Time) rivers.Feature {
	return rivers.Feature{Properties: rivers.StationProperties{ID: id, Name: id, Unit: "m", WaterLevel: &level,
		Status: status, Stale: stale, ObservedAt: &observed, AlertLevel: ptr(5), MinorFloodLevel: ptr(6), MajorFloodLevel: ptr(7)}}
}

func testNetwork() *network {
	return &network{
		version: "v1",
		rivers: []store.ListRiversRow{
			{ID: "kelani-ganga", Name: "Kelani Ganga", Named: true, UplandKm2: 2325},
			{ID: "gin-ganga", Name: "Gin Ganga", Named: true, UplandKm2: 927},
			{ID: "river-9", Name: "Unnamed river near Mannar", Named: false, UplandKm2: 200},
		},
		reaches: []store.ListRiverReachesRow{
			{Fid: 1, RiverID: "kelani-ganga", Name: "Above Hanwella", Gauged: false},
			{Fid: 2, RiverID: "kelani-ganga", Name: "Below Hanwella", Gauged: true, Position: 1},
			{Fid: 3, RiverID: "gin-ganga", Name: "Below Baddegama", Gauged: true},
			{Fid: 4, RiverID: "river-9", Name: "Whole river", Gauged: false},
		},
		gauges: map[int64][]store.RiverReachGauge{
			1: {{ReachFid: 1, Kind: "station", GaugeID: "hanwella", Role: "next"}},
			2: {{ReachFid: 2, Kind: "station", GaugeID: "hanwella", Role: "upstream"}},
			3: {{ReachFid: 3, Kind: "station", GaugeID: "baddegama", Role: "upstream"},
				{ReachFid: 3, Kind: "google", GaugeID: "hybas_1", Role: "upstream"}},
		},
	}
}

func TestAssembleRanksRiversAndStaysHonestAboutCoverage(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	layer := rivers.FeatureCollection{Features: []rivers.Feature{
		stationFeature("hanwella", 5.5, rivers.StatusAlert, false, now.Add(-time.Hour)),
		stationFeature("baddegama", 2, rivers.StatusNormal, true, now.Add(-9*time.Hour)),
	}}
	google := []floods.Gauge{{GaugeID: "hybas_1", Severity: "no_flooding", Trend: "rise", IssuedAt: now.Add(-2 * time.Hour), FetchedAt: now}}
	result := assemble(testNetwork(), layer, google, nil, rainMeta{}, now, false)
	document := result.risk
	if !document.Estimate || document.Note == "" {
		t.Fatalf("every document must say it is an estimate: %+v", document)
	}
	if document.Rivers[0].ID != "kelani-ganga" || document.Rivers[0].Level != LevelWatch {
		t.Fatalf("first river = %+v", document.Rivers[0])
	}
	if document.Rivers[0].GaugedReaches != 1 || len(document.Rivers[0].Reaches) != 2 {
		t.Fatalf("coverage = %+v", document.Rivers[0])
	}
	// The stretch above the gauge reads unknown, and names the next gauge.
	above := result.reach[1]
	if above.Level != LevelUnknown || above.Gauged {
		t.Fatalf("above = %+v", above)
	}
	if above.Reasons[0].Kind != "coverage" {
		t.Fatalf("above reasons = %+v", above.Reasons)
	}
	// A stale station is no evidence, but Google's fresh calm forecast is.
	gin := result.river["gin-ganga"]
	if gin.Level != LevelNormal {
		t.Fatalf("gin = %+v", gin)
	}
	// The ungauged river is unknown, never normal.
	if result.river["river-9"].Level != LevelUnknown {
		t.Fatalf("unnamed = %+v", result.river["river-9"])
	}
	if document.Meta.GaugesCurrent != 1 || document.Meta.GaugesTotal != 2 {
		t.Fatalf("meta = %+v", document.Meta)
	}
	if document.Meta.GaugesObservedAt == nil || !document.Meta.GaugesObservedAt.Equal(now.Add(-time.Hour)) {
		t.Fatalf("meta = %+v", document.Meta)
	}
}

func TestAStationFromAnotherSourceWithNoThresholdsIsTolerated(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	level := 3.0
	observed := now.Add(-time.Hour)
	layer := rivers.FeatureCollection{Features: []rivers.Feature{{Properties: rivers.StationProperties{
		ID: "hanwella", Name: "Other source", Unit: "m", WaterLevel: &level, Status: rivers.StatusUnknown, ObservedAt: &observed,
	}}}}
	result := assemble(testNetwork(), layer, nil, nil, rainMeta{}, now, false)
	if reach := result.reach[2]; reach.Level != LevelUnknown {
		t.Fatalf("a gauge with no thresholds proves nothing: %+v", reach)
	}
}

func TestAssembleCarriesRainToTheStretch(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	layer := rivers.FeatureCollection{Features: []rivers.Feature{stationFeature("hanwella", 2, rivers.StatusNormal, false, now)}}
	result := assemble(testNetwork(), layer, nil, map[int64]RainSummary{2: {Next24hMM: 120, Next72hMM: 150, CatchmentKM2: 1800}}, rainMeta{}, now, false)
	reach := result.reach[2]
	if reach.Rain == nil || reach.Rain.Next24hMM != 120 || reach.Level != LevelHigh {
		t.Fatalf("reach = %+v", reach)
	}
}

func TestSpreadFollowsTheWaterDownToTheNextGauge(t *testing.T) {
	// 1 → 2 → 3 → 4 → 5, gauges at 2 and 4.
	next := map[int64]int64{1: 2, 2: 3, 3: 4, 4: 5, 5: 0}
	got := Spread(next, map[int64]BasinState{2: BasinSevere, 4: BasinNoFlooding})
	want := map[int64]BasinState{2: BasinSevere, 3: BasinSevere, 4: BasinNoFlooding, 5: BasinNoFlooding}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for id, state := range want {
		if got[id] != state {
			t.Fatalf("sub-basin %d = %s, want %s", id, got[id], state)
		}
	}
	if _, ok := got[1]; ok {
		t.Fatal("upstream of every gauge there is no forecast")
	}
}

func TestSpreadTakesTheWorseWhereTwoGaugesReachOneSubbasin(t *testing.T) {
	next := map[int64]int64{1: 3, 2: 3, 3: 0}
	got := Spread(next, map[int64]BasinState{1: BasinNoFlooding, 2: BasinExtreme})
	if got[3] != BasinExtreme {
		t.Fatalf("got %v", got)
	}
}

func TestStatesFromGaugesAndForecasts(t *testing.T) {
	for status, want := range map[rivers.Status]BasinState{
		rivers.StatusMajorFlood: BasinExtreme, rivers.StatusMinorFlood: BasinSevere,
		rivers.StatusAlert: BasinAboveNormal, rivers.StatusNormal: BasinNoFlooding,
	} {
		if got, ok := StationState(status); !ok || got != want {
			t.Fatalf("%s = %s", status, got)
		}
	}
	if _, ok := StationState(rivers.StatusUnknown); ok {
		t.Fatal("a station with no thresholds says nothing")
	}
	if _, ok := ForecastState("unknown"); ok {
		t.Fatal("an unknown severity says nothing")
	}
	if got := AreaState("flash_highly_likely", ""); got != BasinSevere {
		t.Fatalf("flash = %s", got)
	}
	if got := AreaState("notification", "extreme"); got != BasinExtreme {
		t.Fatalf("notification = %s", got)
	}
	if got := AreaState("flash_likely", ""); got != BasinAboveNormal {
		t.Fatalf("flash likely = %s", got)
	}
}

func TestSettleBasin(t *testing.T) {
	cases := []struct {
		name       string
		basin      BasinStatus
		state      BasinState
		unverified bool
	}{
		{"nothing known is not calm", BasinStatus{}, BasinNoForecast, false},
		{"a calm gauge", BasinStatus{Gauges: []BasinGauge{{State: BasinNoFlooding, Verified: true}}}, BasinNoFlooding, false},
		{"a stale gauge says nothing", BasinStatus{Gauges: []BasinGauge{{State: "", Verified: true}}}, BasinNoForecast, false},
		{"the worst gauge counts", BasinStatus{Gauges: []BasinGauge{{State: BasinNoFlooding, Verified: true}, {State: BasinSevere, Verified: true}}}, BasinSevere, false},
		{"unverified raises and is marked", BasinStatus{Gauges: []BasinGauge{{State: BasinNoFlooding, Verified: true}, {State: BasinSevere}}}, BasinSevere, true},
		{"unverified alone is marked", BasinStatus{Gauges: []BasinGauge{{State: BasinNoFlooding}}}, BasinNoFlooding, true},
		{"verified at least as bad is not marked", BasinStatus{Gauges: []BasinGauge{{State: BasinSevere, Verified: true}, {State: BasinSevere}}}, BasinSevere, false},
		{"a flood area counts", BasinStatus{Areas: []BasinArea{{State: BasinExtreme}}, Gauges: []BasinGauge{{State: BasinNoFlooding, Verified: true}}}, BasinExtreme, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			basin := c.basin
			settleBasin(&basin)
			if basin.State != c.state || basin.OnlyUnverified != c.unverified {
				t.Fatalf("state %s, unverified %v", basin.State, basin.OnlyUnverified)
			}
		})
	}
}
