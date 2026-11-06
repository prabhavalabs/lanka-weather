package floodrisk

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
)

func TestAssessNeedsTwoFamiliesToAgree(t *testing.T) {
	cases := []struct {
		name     string
		in       Inputs
		level    int
		scenario string
	}{
		{"nothing", Inputs{}, LevelNone, ScenarioGeneral},
		{"a burst alone is the heavy-rain alert's, not this", Inputs{Rain: &ObservedRain{Rise1hMm: 67.5, Rise3hMm: 67.5}}, LevelNone, ScenarioFlash},
		{"Google's flash flood alone", Inputs{FlashLevel: 1}, LevelNone, ScenarioFlash},
		{"Google's flash flood and a shower", Inputs{FlashLevel: 1, Rain: &ObservedRain{Rise1hMm: 16}}, LevelModerate, ScenarioFlash},
		{"Google's flash flood and a downpour", Inputs{FlashLevel: 1, Rain: &ObservedRain{Rise1hMm: 33, Rise3hMm: 45}}, LevelHigh, ScenarioFlash},
		{"a burst on soaked ground", Inputs{Rain: &ObservedRain{Rise1hMm: 55, PriorMm: 60}}, LevelHigh, ScenarioFlash},
		{"heavy rain forecast on soaked ground", Inputs{Rain: &ObservedRain{PriorMm: 120}, Forecast: &ForecastRain{Next6hMaxMm: 65}}, LevelHigh, ScenarioFlash},
		{"two notable signals are only moderate", Inputs{RiverLevel: 1, Forecast: &ForecastRain{Next6hMaxMm: 35}, OfficialLevel: 1}, LevelModerate, ScenarioBoth},
		{"a river at minor flood with rain forecast", Inputs{RiverLevel: 2, Forecast: &ForecastRain{Next24hMeanMm: 80}}, LevelModerate, ScenarioBoth},
		{"rivers and Google agree", Inputs{RiverLevel: 2, GaugeForecastLevel: 2}, LevelHigh, ScenarioRiver},
		{"everything at once", Inputs{FlashLevel: 2, Rain: &ObservedRain{Rise1hMm: 60, PriorMm: 110}, RiverLevel: 2, OfficialLevel: 2}, LevelVeryHigh, ScenarioBoth},
		{"an official warning on wet ground", Inputs{OfficialLevel: 2, Rain: &ObservedRain{PriorMm: 55}}, LevelModerate, ScenarioGeneral},
	}
	for _, c := range cases {
		got := Assess(c.in)
		if got.Level != c.level || got.Scenario != c.scenario {
			t.Errorf("%s: level %d %s, want %d %s; signals %+v", c.name, got.Level, got.Scenario, c.level, c.scenario, got.Signals)
		}
		if (got.Level == LevelNone) != (got.Variant == "") {
			t.Errorf("%s: variant %q at level %d", c.name, got.Variant, got.Level)
		}
	}
}

func TestSignalsAreStrongestFirst(t *testing.T) {
	got := Assess(Inputs{FlashLevel: 1, Rain: &ObservedRain{Rise1hMm: 16, PriorMm: 120}, Spilling: 1})
	families := got.Top(3)
	if strings.Join(families, ",") != "google_flash,wet_ground,rain_now" {
		t.Fatalf("top %v of %+v", families, got.Signals)
	}
}

// readings loads the fixture: every Colombo district gauge's rolling totals.
func readings(t *testing.T) map[string][]raingauges.Reading {
	t.Helper()
	file, err := os.Open("testdata/colombo-2026-11-06.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	out := map[string][]raingauges.Reading{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		clock, err := time.Parse("15:04", parts[1])
		if err != nil {
			t.Fatal(err)
		}
		mm, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 11, 6, clock.Hour(), clock.Minute(), 0, 0, time.UTC)
		out[parts[0]] = append(out[parts[0]], raingauges.Reading{At: at, Rain24: mm})
	}
	return out
}

// colomboAt is what the sources said about Colombo at a time on 6 November
// 2026: the gauges as stored, and Google's flash-flood areas as we held them
// (one issued 00:40 UTC and dropped by Google before 10:43, one issued 14:45
// and first stored at 18:45). Rivers stayed below alert level and no bulletin
// named the district; the rain forecast of that day is not kept, so it is left
// out, which can only lower the estimate.
func colomboAt(series map[string][]raingauges.Reading, at time.Time) Layers {
	intensities := raingauges.Intensities{FetchedAt: &at}
	for id, all := range series {
		known := []raingauges.Reading{}
		for _, reading := range all {
			if !reading.At.After(at) {
				known = append(known, reading)
			}
		}
		if len(known) == 0 {
			continue
		}
		newest := known[len(known)-1]
		intensities.Gauges = append(intensities.Gauges, raingauges.Intensity{ID: id, Name: id, Lon: 79.9, Lat: 6.9,
			ObservedAt: newest.At, Rain24hMm: newest.Rain24,
			Rise1hMm: raingauges.RiseOver(known, raingauges.Window1h), Rise3hMm: raingauges.RiseOver(known, raingauges.Window3h)})
	}
	clock := func(hour, minute int) time.Time { return time.Date(2026, 11, 6, hour, minute, 0, 0, time.UTC) }
	layer := floods.FeatureCollection{Meta: floods.Meta{Configured: true}}
	if !at.Before(clock(1, 39)) && at.Before(clock(10, 43)) {
		layer.Features = append(layer.Features, floods.Feature{Properties: floods.Area{ID: "morning", Kind: "flash_likely"}})
	}
	if !at.Before(clock(18, 45)) {
		layer.Features = append(layer.Features, floods.Feature{Properties: floods.Area{ID: "evening", Kind: "flash_likely"}})
	}
	return Layers{Rain: &intensities, Floods: &layer, Rivers: &rivers.FeatureCollection{}}
}

var colombo = Places{
	DistrictOf:    func(float64, float64) string { return "LK-11" },
	AreaDistricts: func(string) []string { return []string{"LK-11", "LK-12", "LK-13"} },
	DistrictNamed: func(string) string { return "" },
}

// TestColomboOnSixNovemberWouldHaveBeenHighByTwoInTheAfternoon replays the
// day the Colombo alert came late. Google's flash flood of the morning and a
// 33 mm hour at Kesbewa put the district at high at 08:30 UTC (14:00 in
// Colombo), four hours before the cloudburst and ten before Google's evening
// forecast reached us.
func TestColomboOnSixNovemberWouldHaveBeenHighByTwoInTheAfternoon(t *testing.T) {
	series := readings(t)
	levels := map[string]int{}
	firstHigh := ""
	for minutes := 0; minutes <= 19*60; minutes += 30 {
		at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC).Add(time.Duration(minutes) * time.Minute)
		assessment := Assess(Gather(colomboAt(series, at), colombo, at)["LK-11"])
		key := at.Format("15:04")
		levels[key] = assessment.Level
		if firstHigh == "" && assessment.Level >= LevelHigh {
			firstHigh = key
		}
	}
	if firstHigh != "08:30" {
		t.Fatalf("first high at %s; levels %v", firstHigh, levels)
	}
	// Before any rain, Google's forecast alone is not enough.
	if levels["06:00"] != LevelNone {
		t.Errorf("06:00 level %d", levels["06:00"])
	}
	// After the burst the ground is soaked, so Google's evening forecast is
	// high at once.
	if levels["19:00"] != LevelHigh {
		t.Errorf("19:00 level %d", levels["19:00"])
	}
}
