package alerts

import (
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floodrisk"
	"github.com/prabhavalabs/lanka-weather/internal/push"
)

func TestFloodRiskAlertsADistrictOnItsEstimate(t *testing.T) {
	at := time.Date(2026, 11, 6, 8, 30, 0, 0, time.UTC)
	high := floodrisk.Assess(floodrisk.Inputs{District: "LK-11", FlashLevel: 1, Rain: &floodrisk.ObservedRain{Rise1hMm: 45, Rise3hMm: 45}})
	calm := floodrisk.Assess(floodrisk.Inputs{District: "LK-12"})
	reading := FloodRisk(map[string]floodrisk.Assessment{"LK-11": high, "LK-12": calm}, true, at)
	if !reading.Complete || len(reading.Observations) != 2 {
		t.Fatalf("reading %+v", reading)
	}
	colombo := reading.Observations[0]
	event := colombo.Event
	if colombo.Subject != "risk:district:LK-11" || colombo.Level != floodrisk.LevelHigh || event.Variant != "high" ||
		event.Severity != push.SeverityWarning || event.Topic != push.TopicFloodRisk || event.URL != "/?region=LK-11" ||
		event.Facts["signals"] != "google_flash,rain_now" || event.Facts["scenario"] != "flash" {
		t.Fatalf("observation %+v", colombo)
	}
	states := map[string]State{}
	alerted := 0
	for _, transition := range Apply(reading, states, at) {
		states[transition.State.Subject] = transition.State
		if transition.Event != nil {
			alerted++
		}
	}
	if alerted != 1 {
		t.Fatalf("alerted %d", alerted)
	}
	// A run missing a source says nothing of the calm districts, so nothing is
	// lowered by it.
	partial := FloodRisk(map[string]floodrisk.Assessment{"LK-11": calm}, false, at.Add(time.Minute))
	if partial.Complete || len(partial.Observations) != 0 {
		t.Fatalf("partial %+v", partial)
	}
}

func TestAModerateEstimateIsAWatch(t *testing.T) {
	moderate := floodrisk.Assess(floodrisk.Inputs{District: "LK-11", FlashLevel: 1, Rain: &floodrisk.ObservedRain{Rise1hMm: 16}})
	reading := FloodRisk(map[string]floodrisk.Assessment{"LK-11": moderate}, true, start)
	if reading.Observations[0].Event.Severity != push.SeverityWatch {
		t.Fatalf("%+v", reading.Observations[0])
	}
}
