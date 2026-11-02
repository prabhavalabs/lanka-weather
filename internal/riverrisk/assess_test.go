package riverrisk

import (
	"strings"
	"testing"
)

func ptr(value float64) *float64 { return &value }

// hanwella has the department's thresholds: alert 6.5, minor 7.5, major 9.5 m.
func hanwella(level float64, rate *float64) Gauge {
	return Gauge{ID: "hanwella", Name: "Hanwella", Unit: "m", Level: &level, RatePerHour: rate,
		Alert: ptr(6.5), Minor: ptr(7.5), Major: ptr(9.5)}
}

func TestAssess(t *testing.T) {
	stale := hanwella(8, nil)
	stale.Stale = true
	unthresholded := Gauge{ID: "ampara", Name: "Ampara", Unit: "m", Level: ptr(3)}
	noReading := Gauge{ID: "x", Name: "X", Unit: "m", Alert: ptr(1), Minor: ptr(2), Major: ptr(3)}
	alertOnly := Gauge{ID: "y", Name: "Y", Unit: "m", Level: ptr(1), Alert: ptr(4)}
	alertOnlyAbove := Gauge{ID: "y", Name: "Y", Unit: "m", Level: ptr(5), Alert: ptr(4)}
	majorOnly := Gauge{ID: "z", Name: "Z", Unit: "m", Level: ptr(1), Major: ptr(4)}
	majorOnlyAbove := Gauge{ID: "z", Name: "Z", Unit: "m", Level: ptr(5), Major: ptr(4)}
	feet := Gauge{ID: "nagalagam", Name: "Nagalagam Street", Unit: "ft", Level: ptr(4.2), Alert: ptr(4), Minor: ptr(5), Major: ptr(7)}
	cases := []struct {
		name   string
		inputs Inputs
		want   Level
		says   string
	}{
		// Nothing known: honest, never safe.
		{"no evidence at all", Inputs{}, LevelUnknown, "No gauge on this stretch"},
		{"names the next gauge", Inputs{NextGauge: "Hanwella"}, LevelUnknown, "The next gauge downstream is Hanwella"},
		{"a stale gauge counts for nothing", Inputs{Gauges: []Gauge{stale}}, LevelUnknown, "not used"},
		{"a station with no thresholds says nothing", Inputs{Gauges: []Gauge{unthresholded}}, LevelUnknown, "publishes no flood thresholds"},
		{"a station with no reading says nothing", Inputs{Gauges: []Gauge{noReading}}, LevelUnknown, "no recent reading"},
		{"dry rain forecast alone is not safety", Inputs{Rain: &Rain{Next24hMM: 2, Next72hMM: 8}}, LevelUnknown, "No gauge"},

		// Thresholds from the gauge.
		{"below alert", Inputs{Gauges: []Gauge{hanwella(4, nil)}}, LevelNormal, "below its alert level"},
		{"at alert", Inputs{Gauges: []Gauge{hanwella(6.5, nil)}}, LevelWatch, "above its alert level"},
		{"at minor", Inputs{Gauges: []Gauge{hanwella(7.5, nil)}}, LevelHigh, "above its minor flood level"},
		{"at major", Inputs{Gauges: []Gauge{hanwella(9.5, nil)}}, LevelSevere, "above its major flood level"},
		{"a gauge read in feet", Inputs{Gauges: []Gauge{feet}}, LevelWatch, "4.2 ft"},

		// Null and partial thresholds, as other sources may have.
		{"only an alert level, below", Inputs{Gauges: []Gauge{alertOnly}}, LevelNormal, "below its alert level"},
		{"only an alert level, above", Inputs{Gauges: []Gauge{alertOnlyAbove}}, LevelWatch, "above its alert level"},
		{"only a major level, below, is no comfort", Inputs{Gauges: []Gauge{majorOnly}}, LevelUnknown, "no alert level"},
		{"only a major level, above", Inputs{Gauges: []Gauge{majorOnlyAbove}}, LevelSevere, "major flood"},

		// How close and how fast.
		{"close below alert and steady", Inputs{Gauges: []Gauge{hanwella(6.4, ptr(0))}}, LevelWatch, "within 0.10 m"},
		{"close below alert and falling is fine", Inputs{Gauges: []Gauge{hanwella(6.4, ptr(-0.1))}}, LevelNormal, "below its alert"},
		{"not close and steady", Inputs{Gauges: []Gauge{hanwella(6.0, ptr(0))}}, LevelNormal, "below its alert"},
		{"rising fast towards alert", Inputs{Gauges: []Gauge{hanwella(5.5, ptr(0.5))}}, LevelWatch, "would reach alert level in about 2 hours"},
		{"rising slowly is not projected", Inputs{Gauges: []Gauge{hanwella(4.0, ptr(0.1))}}, LevelNormal, "below its alert"},
		{"at alert, rising to minor", Inputs{Gauges: []Gauge{hanwella(6.8, ptr(0.5))}}, LevelHigh, "would reach minor flood level"},
		{"at minor, rising to major", Inputs{Gauges: []Gauge{hanwella(9.0, ptr(0.5))}}, LevelSevere, "about an hour"},
		{"above major has nowhere further to go", Inputs{Gauges: []Gauge{hanwella(10, ptr(1))}}, LevelSevere, "major flood"},

		// Google's forecast.
		{"no flooding forecast is direct evidence", Inputs{Forecasts: []Forecast{{Severity: "no_flooding", Trend: "rise"}}}, LevelNormal, "no flooding"},
		{"above normal", Inputs{Forecasts: []Forecast{{Severity: "above_normal", Trend: "rise"}}}, LevelWatch, "warning level"},
		{"severe, steady", Inputs{Forecasts: []Forecast{{Severity: "severe", Trend: "no_change"}}}, LevelHigh, "danger level"},
		{"severe, rising", Inputs{Forecasts: []Forecast{{Severity: "severe", Trend: "rise"}}}, LevelSevere, "still rising"},
		{"extreme", Inputs{Forecasts: []Forecast{{Severity: "extreme", Trend: "fall"}}}, LevelSevere, "extreme danger"},
		{"a stale forecast counts for nothing", Inputs{Forecasts: []Forecast{{Severity: "extreme", Stale: true}}}, LevelUnknown, "out of date"},
		{"a forecast with no severity counts for nothing", Inputs{Forecasts: []Forecast{{Severity: "unknown"}}}, LevelUnknown, "no severity"},

		// Rain over the catchment.
		{"fairly heavy rain, no gauge", Inputs{Rain: &Rain{Next24hMM: 55, Next72hMM: 60}}, LevelWatch, "Fairly heavy"},
		{"heavy rain, no gauge", Inputs{Rain: &Rain{Next24hMM: 120, Next72hMM: 130}}, LevelHigh, "Heavy rain"},
		{"very heavy rain alone stops at high", Inputs{Rain: &Rain{Next24hMM: 400, Next72hMM: 600}}, LevelHigh, "Heavy rain"},
		{"days of rain", Inputs{Rain: &Rain{Next24hMM: 40, Next72hMM: 110}}, LevelWatch, "Fairly heavy"},
		{"days of heavy rain", Inputs{Rain: &Rain{Next24hMM: 70, Next72hMM: 210}}, LevelHigh, "Heavy rain"},
		{"light rain with a calm gauge", Inputs{Gauges: []Gauge{hanwella(4, nil)}, Rain: &Rain{Next24hMM: 10, Next72hMM: 20}}, LevelNormal, "below its alert"},
		{"a stale rain forecast counts for nothing", Inputs{Gauges: []Gauge{hanwella(4, nil)}, Rain: &Rain{Next24hMM: 300, Next72hMM: 500, Stale: true}}, LevelNormal, "out of date"},

		// Evidence together.
		{"the worst gauge decides", Inputs{Gauges: []Gauge{hanwella(4, nil), hanwella(8, nil)}}, LevelHigh, "minor flood"},
		{"a calm gauge beside a stale one is still normal", Inputs{Gauges: []Gauge{stale, hanwella(4, nil)}}, LevelNormal, "below its alert"},
		{"heavy rain over a calm gauge", Inputs{Gauges: []Gauge{hanwella(4, nil)}, Rain: &Rain{Next24hMM: 120, Next72hMM: 130}}, LevelHigh, "Heavy rain"},
		{"rain on a river already at alert", Inputs{Gauges: []Gauge{hanwella(6.6, nil)}, Rain: &Rain{Next24hMM: 60, Next72hMM: 80}}, LevelHigh, "Fairly heavy"},
		{"rain on a river in minor flood", Inputs{Gauges: []Gauge{hanwella(8, nil)}, Rain: &Rain{Next24hMM: 60, Next72hMM: 80}}, LevelSevere, "minor flood"},
		{"rain never goes past severe", Inputs{Gauges: []Gauge{hanwella(10, nil)}, Rain: &Rain{Next24hMM: 300, Next72hMM: 400}}, LevelSevere, "major flood"},
		{"a forecast over a calm gauge", Inputs{Gauges: []Gauge{hanwella(4, nil)}, Forecasts: []Forecast{{Severity: "severe", Trend: "fall"}}}, LevelHigh, "danger level"},
		{"a calm forecast does not excuse a high gauge", Inputs{Gauges: []Gauge{hanwella(8, nil)}, Forecasts: []Forecast{{Severity: "no_flooding"}}}, LevelHigh, "minor flood"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Assess(c.inputs)
			if got.Level != c.want {
				t.Fatalf("level = %s, want %s; reasons %+v", got.Level, c.want, got.Reasons)
			}
			joined := ""
			for _, reason := range got.Reasons {
				joined += reason.Text + " | "
			}
			if !strings.Contains(joined, c.says) {
				t.Fatalf("reasons %q do not say %q", joined, c.says)
			}
		})
	}
}

func TestAnUngaugedStretchSaysSoEvenWhenRainRaisesIt(t *testing.T) {
	got := Assess(Inputs{Rain: &Rain{Next24hMM: 120, Next72hMM: 130}})
	if got.Level != LevelHigh {
		t.Fatalf("level = %s", got.Level)
	}
	found := false
	for _, reason := range got.Reasons {
		if reason.Kind == "coverage" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reasons = %+v", got.Reasons)
	}
}

func TestReasonsListTheWorstFirst(t *testing.T) {
	got := Assess(Inputs{Gauges: []Gauge{hanwella(4, nil), hanwella(8, nil)}})
	if got.Reasons[0].Level != LevelHigh {
		t.Fatalf("reasons = %+v", got.Reasons)
	}
}

// The next input is the owner's to wire up: an upstream reservoir spilling. It
// must take a field and a rule, and nothing more.
func TestANewRuleNeedsOnlyAFinding(t *testing.T) {
	spilling := func(Inputs) []Finding {
		return []Finding{{Compounds: true, Cap: LevelHigh, Reason: Reason{Level: LevelHigh, Kind: "reservoir", Text: "A reservoir upstream is spilling."}}}
	}
	list := append(append([]Rule{}, rules...), spilling)

	if got := assessWith(list, Inputs{Gauges: []Gauge{hanwella(4, nil)}}); got.Level != LevelHigh {
		t.Fatalf("over a calm gauge: %s", got.Level)
	}
	if got := assessWith(list, Inputs{Gauges: []Gauge{hanwella(8, nil)}}); got.Level != LevelSevere {
		t.Fatalf("it should compound with high water: %s", got.Level)
	}
	got := assessWith(list, Inputs{})
	if got.Level != LevelHigh {
		t.Fatalf("with no gauge: %s", got.Level)
	}
	if got.Reasons[len(got.Reasons)-1].Kind != "coverage" {
		t.Fatalf("an ungauged stretch must still say so: %+v", got.Reasons)
	}
}

func TestLevelRank(t *testing.T) {
	order := []Level{LevelUnknown, LevelNormal, LevelWatch, LevelHigh, LevelSevere}
	for index, level := range order {
		if level.Rank() != index {
			t.Fatalf("%s rank %d", level, level.Rank())
		}
	}
	if LevelNormal.AtRisk() || LevelUnknown.AtRisk() || !LevelWatch.AtRisk() {
		t.Fatal("at risk starts at watch")
	}
	if raise(LevelSevere, 1) != LevelSevere {
		t.Fatal("severe is the top")
	}
}
