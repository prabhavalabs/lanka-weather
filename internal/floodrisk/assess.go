// Package floodrisk estimates, for each district, how likely flooding is now
// and in the hours ahead, by combining what every source says: rain measured
// at the live gauges, how wet the ground already is, rain forecast, Google's
// flash-flood and river forecasts, river gauges, official bulletins and
// spilling reservoirs (ADR 0019).
//
// Each kind of evidence is a family scored 0 to 3 on its own terms (notable,
// significant, severe). A district's level needs at least two families to
// agree: one source alone is already an alert of its own (package alerts), and
// the estimate's job is corroboration and lead time. It is an estimate
// throughout and says so; official warnings remain the DMC's and the
// Department of Meteorology's.
//
// Assess is pure and table-tested; Gather builds its inputs from the layers the
// map serves.
package floodrisk

import (
	"slices"
	"sort"
)

// Families of evidence, in the order they are listed when tied.
const (
	FamilyOfficial     = "official"
	FamilyGoogleFlash  = "google_flash"
	FamilyRainNow      = "rain_now"
	FamilyRiver        = "river"
	FamilyGoogleGauge  = "google_gauge"
	FamilyRainForecast = "rain_forecast"
	FamilyWetGround    = "wet_ground"
	FamilyReservoir    = "reservoir"
)

var familyOrder = []string{FamilyOfficial, FamilyGoogleFlash, FamilyRainNow, FamilyRiver, FamilyGoogleGauge,
	FamilyRainForecast, FamilyWetGround, FamilyReservoir}

// Levels of the estimate.
const (
	LevelNone     = 0
	LevelModerate = 1
	LevelHigh     = 2
	LevelVeryHigh = 3
)

// Variants name the levels in notices and the API.
var Variants = map[int]string{LevelModerate: "moderate", LevelHigh: "high", LevelVeryHigh: "very_high"}

// Scenarios say what could happen, from which side the evidence comes.
const (
	// ScenarioFlash: bursts of rain flooding roads and low ground within hours.
	ScenarioFlash = "flash"
	// ScenarioRiver: rivers overflowing onto their banks.
	ScenarioRiver = "river"
	// ScenarioBoth: both at once.
	ScenarioBoth = "both"
	// ScenarioGeneral: evidence that names neither, such as an official
	// warning on wet ground.
	ScenarioGeneral = "general"
)

// ObservedRain is the rain measured in a district: the largest rises of its
// current gauges (lower bounds, raingauges.RiseOver) and the most rain that had
// already fallen in the 24 hours before the last three.
type ObservedRain struct {
	Rise1hMm float64 `json:"rise1hMm"`
	Rise3hMm float64 `json:"rise3hMm"`
	// Gauge and GaugeID name the gauge with the largest rise.
	Gauge   string `json:"gauge"`
	GaugeID string `json:"gaugeId"`
	// PriorMm is the rolling 24-hour total less the three-hour rise: rain
	// that had soaked the ground before the latest burst.
	PriorMm float64 `json:"priorMm"`
}

// ForecastRain is the rain forecast ahead (weather.RainOutlook).
type ForecastRain struct {
	Next6hMaxMm   float64 `json:"next6hMaxMm"`
	Next24hMeanMm float64 `json:"next24hMeanMm"`
}

// Inputs are what the sources say about one district now. Nil and zero mean
// nothing current: never "dry" or "calm".
type Inputs struct {
	District string        `json:"-"`
	Rain     *ObservedRain `json:"rain"`
	Forecast *ForecastRain `json:"forecast"`
	// FlashLevel is Google's flash flood over the district: likely 1, highly
	// likely 2.
	FlashLevel int `json:"flashLevel"`
	// GaugeForecastLevel is Google's highest verified river forecast in the
	// district: Warning 1, Danger 2, Extreme 3.
	GaugeForecastLevel int `json:"gaugeForecastLevel"`
	// RiverLevel is the highest river gauge: alert 1, minor flood 2, major 3.
	RiverLevel int `json:"riverLevel"`
	// OfficialLevel is the highest DMC or Department of Meteorology bulletin
	// naming the district: advisory 1, warning 2, red 3.
	OfficialLevel int `json:"officialLevel"`
	// Spilling counts the district's reservoirs spilling.
	Spilling int `json:"spilling"`
}

// Signal is one family's evidence.
type Signal struct {
	Family string  `json:"family"`
	Points int     `json:"points"`
	Value  float64 `json:"value"`
	// Unit is "mm" for rain, "level" for a source's own level, "count" for
	// reservoirs.
	Unit string `json:"unit"`
	// Window is the rain's window: 1h, 3h, 6h, 24h.
	Window string `json:"window,omitempty"`
}

// Assessment is a district's estimate.
type Assessment struct {
	District string `json:"district"`
	Level    int    `json:"level"`
	// Variant is "moderate", "high", "very_high" or "" at level 0.
	Variant  string   `json:"variant"`
	Score    int      `json:"score"`
	Families int      `json:"families"`
	Scenario string   `json:"scenario"`
	Signals  []Signal `json:"signals"`
}

// band scores a value against three rising thresholds.
func band(value float64, thresholds [3]float64) int {
	points := 0
	for _, threshold := range thresholds {
		if value >= threshold {
			points++
		}
	}
	return points
}

// Thresholds of the rain families, in mm. Measured rain is a lower bound, so
// these can be lower than the alert's own (alerts.RainBands): 15 mm in an hour
// is notable on its own here only alongside other evidence.
var (
	rainHour       = [3]float64{15, 30, 50}
	rainThree      = [3]float64{30, 60, 100}
	priorDay       = [3]float64{50, 100, 200}
	forecastSix    = [3]float64{30, 60, 100}
	forecastDay    = [3]float64{75, 150, 250}
	reservoirSpill = [3]float64{1, 3, 6}
)

// signals scores each family.
func signals(in Inputs) []Signal {
	out := []Signal{}
	add := func(signal Signal) {
		if signal.Points > 0 {
			out = append(out, signal)
		}
	}
	level := func(family string, value int) {
		add(Signal{Family: family, Points: min(value, 3), Value: float64(value), Unit: "level"})
	}
	level(FamilyOfficial, in.OfficialLevel)
	// A flash flood Google forecasts is already significant: likely 2.
	if in.FlashLevel > 0 {
		add(Signal{Family: FamilyGoogleFlash, Points: min(in.FlashLevel+1, 3), Value: float64(in.FlashLevel), Unit: "level"})
	}
	if rain := in.Rain; rain != nil {
		hour, three := band(rain.Rise1hMm, rainHour), band(rain.Rise3hMm, rainThree)
		if hour >= three {
			add(Signal{Family: FamilyRainNow, Points: hour, Value: rain.Rise1hMm, Unit: "mm", Window: "1h"})
		} else {
			add(Signal{Family: FamilyRainNow, Points: three, Value: rain.Rise3hMm, Unit: "mm", Window: "3h"})
		}
		add(Signal{Family: FamilyWetGround, Points: band(rain.PriorMm, priorDay), Value: rain.PriorMm, Unit: "mm", Window: "24h"})
	}
	level(FamilyRiver, in.RiverLevel)
	level(FamilyGoogleGauge, in.GaugeForecastLevel)
	if forecast := in.Forecast; forecast != nil {
		six, day := band(forecast.Next6hMaxMm, forecastSix), band(forecast.Next24hMeanMm, forecastDay)
		if six >= day {
			add(Signal{Family: FamilyRainForecast, Points: six, Value: forecast.Next6hMaxMm, Unit: "mm", Window: "6h"})
		} else {
			add(Signal{Family: FamilyRainForecast, Points: day, Value: forecast.Next24hMeanMm, Unit: "mm", Window: "24h"})
		}
	}
	add(Signal{Family: FamilyReservoir, Points: band(float64(in.Spilling), reservoirSpill), Value: float64(in.Spilling), Unit: "count"})
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Points != out[j].Points {
			return out[i].Points > out[j].Points
		}
		return slices.Index(familyOrder, out[i].Family) < slices.Index(familyOrder, out[j].Family)
	})
	return out
}

// Assess scores a district. The levels:
//
//   - moderate (flooding possible): two families with 3 points between them,
//     such as Google's flash-flood forecast and a shower of 15 mm in an hour;
//   - high (flooding likely): 4 points from two families, one of them
//     significant, such as a 30 mm hour under Google's forecast, a 50 mm hour
//     on wet ground, or 60 mm forecast in 6 hours on soaked ground;
//   - very high: 7 points from three families.
func Assess(in Inputs) Assessment {
	found := signals(in)
	assessment := Assessment{District: in.District, Signals: found, Families: len(found)}
	strongest := 0
	for _, signal := range found {
		assessment.Score += signal.Points
		strongest = max(strongest, signal.Points)
	}
	switch {
	case assessment.Families >= 3 && assessment.Score >= 7:
		assessment.Level = LevelVeryHigh
	case assessment.Families >= 2 && assessment.Score >= 4 && strongest >= 2:
		assessment.Level = LevelHigh
	case assessment.Families >= 2 && assessment.Score >= 3:
		assessment.Level = LevelModerate
	}
	assessment.Variant = Variants[assessment.Level]
	assessment.Scenario = scenario(found)
	return assessment
}

// scenario says what could happen from where the evidence lies: rain and flash
// forecasts point to flash flooding, rivers and reservoirs to rivers rising.
func scenario(found []Signal) string {
	flash, river := 0, 0
	for _, signal := range found {
		switch signal.Family {
		case FamilyRainNow, FamilyGoogleFlash, FamilyRainForecast:
			flash += signal.Points
		case FamilyRiver, FamilyGoogleGauge, FamilyReservoir:
			river += signal.Points
		}
	}
	switch {
	case flash > 0 && river > 0:
		return ScenarioBoth
	case flash > 0:
		return ScenarioFlash
	case river > 0:
		return ScenarioRiver
	}
	return ScenarioGeneral
}

// Top lists the families of the strongest signals, at most n.
func (assessment Assessment) Top(n int) []string {
	out := make([]string, 0, n)
	for _, signal := range assessment.Signals {
		if len(out) == n {
			break
		}
		out = append(out, signal.Family)
	}
	return out
}
