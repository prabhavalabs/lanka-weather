package alerts

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floodrisk"
	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
)

// HoldRisk is how long a district must stay below its alerted estimate before
// that level alerts again.
const HoldRisk = 6 * time.Hour

// RiskSubject is a district's flood-risk subject.
func RiskSubject(district string) string { return "risk:district:" + district }

// riskSeverity keeps the estimate below danger: moderate is watch, high and
// very high warning. Danger is for official red warnings and measured major
// floods, which have alerts of their own.
func riskSeverity(level int) push.Severity {
	if level >= floodrisk.LevelHigh {
		return push.SeverityWarning
	}
	return push.SeverityWatch
}

// FloodRisk reads the combined estimate (package floodrisk) by district. It is
// complete only when every source was read this run, so a source that failed
// cannot lower an estimate.
func FloodRisk(assessments map[string]floodrisk.Assessment, complete bool, now time.Time) Reading {
	reading := Reading{Kind: messages.KindFloodRisk, HoldFor: HoldRisk, Complete: complete}
	districts := make([]string, 0, len(assessments))
	for district := range assessments {
		districts = append(districts, district)
	}
	sort.Strings(districts)
	for _, district := range districts {
		assessment := assessments[district]
		if assessment.Level == floodrisk.LevelNone && !complete {
			continue
		}
		event := push.Event{Kind: messages.KindFloodRisk, Variant: assessment.Variant, Topic: push.TopicFloodRisk,
			Severity: riskSeverity(assessment.Level), Districts: []string{district},
			Facts: map[string]string{"district": district, "signals": strings.Join(assessment.Top(3), ","),
				"scenario": assessment.Scenario, "score": strconv.Itoa(assessment.Score)},
			URL: "/?region=" + district, Tag: "risk:district:" + district, ExpiresAt: now.Add(6 * time.Hour)}
		reading.Observations = append(reading.Observations, Observation{Subject: RiskSubject(district), Level: assessment.Level, Event: event})
	}
	return reading
}
