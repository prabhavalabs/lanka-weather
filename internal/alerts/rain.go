package alerts

import (
	"sort"
	"strconv"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
)

// HoldRain is how long a district must stay below its alerted rain level
// before that level alerts again.
const HoldRain = 6 * time.Hour

// RainFreshFor is how old a gauge's newest reading may be to count. The feed
// is refreshed about every 30 minutes; three missed refreshes is out of date.
const RainFreshFor = 90 * time.Minute

// RainBand is one level's thresholds: rain measured in the last hour, the last
// three hours or the rolling 24 hours, any of which reaches the level. The
// hourly and three-hourly figures are lower bounds (raingauges.RiseOver).
type RainBand struct {
	Level            int
	Variant          string
	Hour, Three, Day float64
}

// RainBands are the levels, highest first. The 24-hour figures follow the
// Department of Meteorology's heavy (100 mm) and very heavy (150 mm) rain; the
// short ones are the bursts that flood towns before any river rises. On 6
// November 2026 Colombo's head-office gauge measured at least 67.5 mm in the
// hour to 18:20, six hours before Google's flash-flood forecast reached us.
var RainBands = []RainBand{
	{Level: 3, Variant: "extreme", Hour: 75, Three: 150, Day: 250},
	{Level: 2, Variant: "very_heavy", Hour: 50, Three: 100, Day: 150},
	{Level: 1, Variant: "heavy", Hour: 30, Three: 50, Day: 100},
}

// RainMeasure is the figure that set a gauge's level: the shortest window that
// reached it.
type RainMeasure struct {
	Level   int
	Variant string
	// Window is "1h", "3h" or "24h", and Mm the rain measured over it.
	Window string
	Mm     float64
}

// RainLevel is a gauge's level from what it measured.
func RainLevel(gauge raingauges.Intensity) RainMeasure {
	for _, band := range RainBands {
		switch {
		case gauge.Rise1hMm >= band.Hour:
			return RainMeasure{Level: band.Level, Variant: band.Variant, Window: "1h", Mm: gauge.Rise1hMm}
		case gauge.Rise3hMm >= band.Three:
			return RainMeasure{Level: band.Level, Variant: band.Variant, Window: "3h", Mm: gauge.Rise3hMm}
		case gauge.Rain24hMm >= band.Day:
			return RainMeasure{Level: band.Level, Variant: band.Variant, Window: "24h", Mm: gauge.Rain24hMm}
		}
	}
	return RainMeasure{}
}

// rainSeverity keeps measured rain at warning: the gauges are instruments, but
// what they mean for a place is our reading of them, not an official warning.
func rainSeverity(level int) push.Severity {
	if level >= 2 {
		return push.SeverityWarning
	}
	return push.SeverityWatch
}

// RainSubject is a district's heavy-rain subject.
func RainSubject(district string) string { return "rain:district:" + district }

// HeavyRain reads the live rain gauges by district: each district at the level
// of its wettest current gauge. A district whose gauges are all out of date is
// held; a suspect gauge is left out.
func HeavyRain(intensities raingauges.Intensities, districtOf DistrictOf, now time.Time) Reading {
	fresh := intensities.FetchedAt != nil && now.Sub(*intensities.FetchedAt) <= RainFreshFor
	reading := Reading{Kind: messages.KindRain, HoldFor: HoldRain, Complete: fresh}
	type lead struct {
		gauge   raingauges.Intensity
		measure RainMeasure
		current bool
	}
	leads := map[string]*lead{}
	for _, gauge := range intensities.Gauges {
		if gauge.Suspect {
			continue
		}
		district := districtOf(gauge.Lon, gauge.Lat)
		if district == "" {
			continue
		}
		current := !gauge.Stale && now.Sub(gauge.ObservedAt) <= RainFreshFor
		entry, ok := leads[district]
		if !ok {
			entry = &lead{}
			leads[district] = entry
		}
		if !current {
			continue
		}
		measure := RainLevel(gauge)
		if !entry.current || measure.Level > entry.measure.Level ||
			(measure.Level == entry.measure.Level && measure.Mm > entry.measure.Mm) {
			entry.gauge, entry.measure, entry.current = gauge, measure, true
		}
	}
	districts := make([]string, 0, len(leads))
	for district := range leads {
		districts = append(districts, district)
	}
	sort.Strings(districts)
	for _, district := range districts {
		entry := leads[district]
		subject := RainSubject(district)
		if !entry.current {
			reading.Observations = append(reading.Observations, Observation{Subject: subject, Hold: true})
			continue
		}
		measure := entry.measure
		event := push.Event{Kind: messages.KindRain, Variant: measure.Variant, Topic: push.TopicHeavyRain, Severity: rainSeverity(measure.Level),
			Districts: []string{district},
			Facts: map[string]string{"district": district, "gauge": entry.gauge.Name, "window": measure.Window,
				"mm": strconv.Itoa(int(measure.Mm)), "observedAt": entry.gauge.ObservedAt.UTC().Format(time.RFC3339),
				"issuedAt": entry.gauge.ObservedAt.UTC().Format(time.RFC3339)},
			URL: "/?raingauge=" + entry.gauge.ID, Tag: "rain:district:" + district, ExpiresAt: now.Add(6 * time.Hour)}
		event.Lon, event.Lat = point(entry.gauge.Lon, entry.gauge.Lat)
		reading.Observations = append(reading.Observations, Observation{Subject: subject, Level: measure.Level, Event: event})
	}
	return reading
}
