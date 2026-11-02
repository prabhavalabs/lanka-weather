// Package riverrisk estimates, for each stretch of river, whether it is at risk
// of overflowing, and tints the flood-forecast basins. It reads gauges, Google's
// forecasts and forecast rain from the tables their collectors fill, and is an
// estimate throughout: thresholds and official warnings are the Irrigation
// Department's and the DMC's (ADR 0010).
//
// Assess is a pure function, table-tested in assess_test.go. It is built from
// rules, one per kind of evidence, so a new input is a field on Inputs and a
// rule in the list: nothing else changes. See rules.
package riverrisk

import (
	"fmt"
	"sort"
	"time"
)

// Level is how at risk of overflowing a stretch is.
type Level string

const (
	// LevelUnknown means nothing current says either way. It is never a calm
	// state: a stretch with no gauge reads unknown, not safe.
	LevelUnknown Level = "unknown"
	LevelNormal  Level = "normal"
	LevelWatch   Level = "watch"
	LevelHigh    Level = "high"
	LevelSevere  Level = "severe"
)

// Rank orders levels from unknown (0) to severe (4).
func (level Level) Rank() int {
	switch level {
	case LevelNormal:
		return 1
	case LevelWatch:
		return 2
	case LevelHigh:
		return 3
	case LevelSevere:
		return 4
	}
	return 0
}

// AtRisk reports whether a level calls for attention.
func (level Level) AtRisk() bool { return level.Rank() >= LevelWatch.Rank() }

var byRank = []Level{LevelUnknown, LevelNormal, LevelWatch, LevelHigh, LevelSevere}

func raise(level Level, steps int) Level {
	return byRank[min(level.Rank()+steps, LevelSevere.Rank())]
}

// Gauge is the latest state of a river gauge. Thresholds are in the gauge's own
// unit and any of them may be missing: some stations publish none, and stations
// from other sources may publish fewer.
type Gauge struct {
	ID, Name, Unit     string
	Level, RatePerHour *float64
	Alert, Minor       *float64
	Major              *float64
	ObservedAt         *time.Time
	// Stale means the reading is too old to count as evidence.
	Stale bool
}

// Forecast is Google's latest forecast at a gauge on this stretch.
type Forecast struct {
	GaugeID string
	// Severity is no_flooding, above_normal, severe, extreme or unknown.
	Severity string
	// Trend is rise, fall, no_change or unknown.
	Trend string
	Stale bool
}

// Rain is forecast rain averaged over the catchment upstream of the stretch.
type Rain struct {
	Next24hMM, Next72hMM float64
	Stale                bool
}

// Inputs is everything known about one stretch.
type Inputs struct {
	Gauges    []Gauge
	Forecasts []Forecast
	Rain      *Rain
	// NextGauge names the first gauge below an ungauged stretch.
	NextGauge string
}

// Reason is one thing said about a stretch.
type Reason struct {
	Level Level `json:"level"`
	// Kind is gauge, forecast, rain or coverage.
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// Assessment is a stretch's estimated level and why.
type Assessment struct {
	Level   Level    `json:"level"`
	Reasons []Reason `json:"reasons"`
}

// Finding is what a rule concludes from one piece of evidence.
type Finding struct {
	Reason Reason
	// Direct says the river itself was measured or forecast. Only direct
	// evidence can call a stretch normal, because a dry forecast says nothing
	// about a river already high.
	Direct bool
	// Compounds says this adds to high water already measured, raising the
	// result one step when direct evidence is already at watch or above.
	Compounds bool
	// Cap limits how far a finding alone can take a stretch. Forecast rain is
	// not a measurement of the river, so it never reads severe by itself.
	Cap Level
}

// Rule turns one kind of evidence into findings.
type Rule func(Inputs) []Finding

// rules are the evidence an assessment weighs. To add an input (an upstream
// reservoir spilling, say): add a field to Inputs and a Rule here that returns
// a Finding with Compounds set, and capped as the evidence deserves. Nothing
// else changes; see TestANewRuleNeedsOnlyAFinding.
var rules = []Rule{gaugeRule, forecastRule, rainRule}

// Assess estimates a stretch's level from every rule.
func Assess(inputs Inputs) Assessment { return assessWith(rules, inputs) }

func assessWith(list []Rule, inputs Inputs) Assessment {
	var findings []Finding
	for _, rule := range list {
		findings = append(findings, rule(inputs)...)
	}
	direct, indirect := LevelUnknown, LevelUnknown
	haveDirect, compounding := false, false
	for _, finding := range findings {
		level := finding.Reason.Level
		if finding.Direct {
			if level != LevelUnknown {
				haveDirect = true
			}
			if level.Rank() > direct.Rank() {
				direct = level
			}
			continue
		}
		if finding.Cap != "" && level.Rank() > finding.Cap.Rank() {
			level = finding.Cap
		}
		if level.Rank() > indirect.Rank() {
			indirect = level
		}
		if finding.Compounds && level.AtRisk() {
			compounding = true
		}
	}
	result := direct
	if indirect.Rank() > result.Rank() {
		result = indirect
	}
	if direct.AtRisk() && compounding {
		result = raise(direct, 1)
	}
	reasons := make([]Reason, 0, len(findings)+1)
	for _, finding := range findings {
		reasons = append(reasons, finding.Reason)
	}
	if !haveDirect {
		text := "No gauge on this stretch, so its level is not known."
		if len(inputs.Gauges) > 0 || len(inputs.Forecasts) > 0 {
			text = "No current reading from its gauges, so its level is not known."
		} else if inputs.NextGauge != "" {
			text = fmt.Sprintf("No gauge on this stretch, so its level is not known. The next gauge downstream is %s.", inputs.NextGauge)
		}
		reasons = append(reasons, Reason{Level: LevelUnknown, Kind: "coverage", Text: text})
	}
	sort.SliceStable(reasons, func(i, j int) bool { return reasons[i].Level.Rank() > reasons[j].Level.Rank() })
	return Assessment{Level: result, Reasons: reasons}
}

const (
	// leadTime is how far ahead a rising river is projected. Six hours is long
	// enough to act on and short enough to trust: the rate comes from a window
	// of up to three hours, and extrapolating further than twice that says more
	// than the reading does.
	leadTime = 6 * time.Hour
	// nearFraction is how close to its alert level a river counts as near,
	// as a share of the department's own gap from alert to minor flood at that
	// station. A gauge's datum is arbitrary, so a share of the level itself
	// would mean nothing, and a fixed height would mean different things on
	// different rivers.
	nearFraction = 0.25
	// nearFallback replaces the gap when a station publishes no minor level: a
	// share of the alert level.
	nearFallback = 0.1
	// Catchment rain bands in millimetres, from the Department of Meteorology's
	// advisory language: "fairly heavy" above 50 mm, "heavy" above 100 mm. Those
	// describe a place; these are read against a whole catchment's average,
	// which is harder to reach, so a catchment at the band is at least as
	// serious. The 72-hour bands are twice the 24-hour ones: saturated ground
	// after days of rain, with less intensity needed.
	rainWatch24h = 50
	rainHigh24h  = 100
	rainWatch72h = 100
	rainHigh72h  = 200
)

// gaugeLevel reads a level against whichever thresholds a station has.
func gaugeLevel(gauge Gauge) (Level, bool) {
	level := *gauge.Level
	switch {
	case gauge.Major != nil && level >= *gauge.Major:
		return LevelSevere, true
	case gauge.Minor != nil && level >= *gauge.Minor:
		return LevelHigh, true
	case gauge.Alert != nil && level >= *gauge.Alert:
		return LevelWatch, true
	case gauge.Alert != nil:
		// Below the alert level: normal, as far as the department's levels say.
		return LevelNormal, true
	}
	// It has thresholds above alert only, and is below them; that does not say
	// it is below the first level that matters.
	return LevelUnknown, false
}

func format(value float64, unit string) string {
	if unit == "ft" {
		return fmt.Sprintf("%.1f %s", value, unit)
	}
	return fmt.Sprintf("%.2f %s", value, unit)
}

// nextThreshold is the lowest threshold above the level, with its name and the
// level reaching it would mean.
func nextThreshold(gauge Gauge) (float64, string, Level, bool) {
	level := *gauge.Level
	for _, candidate := range []struct {
		value *float64
		name  string
		then  Level
	}{{gauge.Alert, "alert", LevelWatch}, {gauge.Minor, "minor flood", LevelHigh}, {gauge.Major, "major flood", LevelSevere}} {
		if candidate.value != nil && level < *candidate.value {
			return *candidate.value, candidate.name, candidate.then, true
		}
	}
	return 0, "", LevelUnknown, false
}

func gaugeRule(inputs Inputs) []Finding {
	var findings []Finding
	for _, gauge := range inputs.Gauges {
		findings = append(findings, gaugeFinding(gauge))
	}
	return findings
}

func gaugeFinding(gauge Gauge) Finding {
	finding := func(level Level, text string) Finding {
		return Finding{Direct: true, Reason: Reason{Level: level, Kind: "gauge", Text: text}}
	}
	if gauge.Level == nil {
		return finding(LevelUnknown, gauge.Name+" has no recent reading.")
	}
	if gauge.Stale {
		return finding(LevelUnknown, gauge.Name+" last reported too long ago, so it is not used.")
	}
	if gauge.Alert == nil && gauge.Minor == nil && gauge.Major == nil {
		return finding(LevelUnknown, gauge.Name+" publishes no flood thresholds.")
	}
	level, known := gaugeLevel(gauge)
	if !known {
		return finding(LevelUnknown, gauge.Name+" publishes no alert level to compare with.")
	}
	reading := format(*gauge.Level, gauge.Unit)
	text := ""
	switch level {
	case LevelSevere:
		text = fmt.Sprintf("%s is at %s, above its major flood level of %s.", gauge.Name, reading, format(*gauge.Major, gauge.Unit))
	case LevelHigh:
		text = fmt.Sprintf("%s is at %s, above its minor flood level of %s.", gauge.Name, reading, format(*gauge.Minor, gauge.Unit))
	case LevelWatch:
		text = fmt.Sprintf("%s is at %s, above its alert level of %s.", gauge.Name, reading, format(*gauge.Alert, gauge.Unit))
	default:
		text = fmt.Sprintf("%s is at %s, below its alert level of %s.", gauge.Name, reading, format(*gauge.Alert, gauge.Unit))
	}
	target, name, then, more := nextThreshold(gauge)
	rising := gauge.RatePerHour != nil && *gauge.RatePerHour > 0
	falling := gauge.RatePerHour != nil && *gauge.RatePerHour < 0
	if more && rising {
		// It will reach the next threshold soon at this rate.
		if hours := (target - *gauge.Level) / *gauge.RatePerHour; hours <= leadTime.Hours() {
			level = then
			text += fmt.Sprintf(" Rising %s an hour, it would reach %s level in %s.",
				format(*gauge.RatePerHour, gauge.Unit), name, roundHours(hours))
			return finding(level, text)
		}
	}
	if level == LevelNormal && !falling && near(gauge) {
		level = LevelWatch
		text += fmt.Sprintf(" It is within %s of that level.", format(*gauge.Alert-*gauge.Level, gauge.Unit))
	}
	return finding(level, text)
}

// near reports whether a level below alert is close to it.
func near(gauge Gauge) bool {
	margin := *gauge.Alert - *gauge.Level
	if gauge.Minor != nil && *gauge.Minor > *gauge.Alert {
		return margin <= nearFraction*(*gauge.Minor-*gauge.Alert)
	}
	return margin <= nearFallback**gauge.Alert
}

func roundHours(hours float64) string {
	if hours < 1 {
		return "less than an hour"
	}
	rounded := int(hours + 0.5)
	if rounded == 1 {
		return "about an hour"
	}
	return fmt.Sprintf("about %d hours", rounded)
}

// forecastRule reads Google's forecast severity. Google's three levels follow
// its gauge models' warning, danger and extreme-danger thresholds, so they map
// onto the department's alert, minor and major flood. Trend matters only at the
// top: most gauges show "rise" on a dry day, so a rising calm forecast is news
// to no one. Stale or unknown forecasts count as no evidence.
func forecastRule(inputs Inputs) []Finding {
	var findings []Finding
	for _, forecast := range inputs.Forecasts {
		finding := func(level Level, text string) Finding {
			return Finding{Direct: true, Reason: Reason{Level: level, Kind: "forecast", Text: text}}
		}
		if forecast.Stale {
			findings = append(findings, finding(LevelUnknown, "A Google flood forecast here is out of date, so it is not used."))
			continue
		}
		switch forecast.Severity {
		case "no_flooding":
			findings = append(findings, finding(LevelNormal, "Google forecasts no flooding here."))
		case "above_normal":
			findings = append(findings, finding(LevelWatch, "Google forecasts the river above its warning level."))
		case "severe":
			if forecast.Trend == "rise" {
				findings = append(findings, finding(LevelSevere, "Google forecasts the river above its danger level and still rising."))
			} else {
				findings = append(findings, finding(LevelHigh, "Google forecasts the river above its danger level."))
			}
		case "extreme":
			findings = append(findings, finding(LevelSevere, "Google forecasts the river above its extreme danger level."))
		default:
			findings = append(findings, finding(LevelUnknown, "A Google flood forecast here gives no severity."))
		}
	}
	return findings
}

// rainRule reads forecast rain over the catchment. It can raise a stretch on its
// own, up to high, and adds to high water already measured. Without a forecast
// it says nothing rather than implying dry weather.
func rainRule(inputs Inputs) []Finding {
	rain := inputs.Rain
	if rain == nil {
		return nil
	}
	finding := func(level Level, text string) []Finding {
		return []Finding{{Compounds: true, Cap: LevelHigh, Reason: Reason{Level: level, Kind: "rain", Text: text}}}
	}
	if rain.Stale {
		return []Finding{{Reason: Reason{Level: LevelUnknown, Kind: "rain", Text: "The rain forecast is out of date, so it is not used."}}}
	}
	amounts := fmt.Sprintf("%.0f mm in the next 24 hours and %.0f mm in 72 hours", rain.Next24hMM, rain.Next72hMM)
	switch {
	case rain.Next24hMM >= rainHigh24h || rain.Next72hMM >= rainHigh72h:
		return finding(LevelHigh, fmt.Sprintf("Heavy rain is forecast over the catchment: %s on average.", amounts))
	case rain.Next24hMM >= rainWatch24h || rain.Next72hMM >= rainWatch72h:
		return finding(LevelWatch, fmt.Sprintf("Fairly heavy rain is forecast over the catchment: %s on average.", amounts))
	}
	return []Finding{{Reason: Reason{Level: LevelUnknown, Kind: "rain", Text: fmt.Sprintf("Forecast rain over the catchment is %s on average.", amounts)}}}
}
