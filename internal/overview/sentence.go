package overview

import (
	"fmt"
	"strings"
	"time"
)

// recentQuake is how recent a large earthquake must be to reach the status.
const recentQuake = 24 * time.Hour

// Sentence builds the status at the top from fixed rules, most serious first:
//
//   - official levels and warnings;
//   - cyclones;
//   - gauges at or above their alert level, and estimated high water;
//   - a large recent earthquake or a USGS tsunami flag;
//   - reservoirs spilling;
//   - the one calm thing it will say, that no river gauge is above its alert
//     level, only while the gauges themselves are current;
//   - which feeds are out of date or could not be read.
//
// It never says "safe" or "all clear". A stale or unread feed is left out of what
// it counts and named at the end, so its silence is never taken for good news.
func Sentence(sections []Section, now time.Time) string {
	byID := make(map[string]Section, len(sections))
	for _, section := range sections {
		byID[section.ID] = section
	}
	var parts []string
	add := func(format string, values ...any) { parts = append(parts, fmt.Sprintf(format, values...)) }

	if section := byID[SectionLandslides]; section.Available {
		shown, highest := 0, ""
		for _, count := range section.Counts {
			if count.Value > 0 && highest == "" {
				highest = count.Label
			}
			shown += count.Value
		}
		if shown > 0 {
			add("Landslide warnings shown for %s, up to %s.", plural(shown, "DS division", "DS divisions"), highest)
		}
	}
	if section := byID[SectionWarnings]; section.Available && len(section.Items) > 0 {
		add("%s in force.", plural(len(section.Items), "official warning or advisory", "official warnings and advisories"))
	}
	if section := byID[SectionCyclones]; section.Available && !section.Stale {
		fresh := []string{}
		for _, item := range section.Items {
			if !item.Stale {
				fresh = append(fresh, item.ID)
			}
		}
		switch {
		case len(fresh) == 1:
			add("1 cyclone active %s of Sri Lanka.", directionWord(section, fresh[0]))
		case len(fresh) > 1:
			add("%d cyclones active in the North Indian Ocean.", len(fresh))
		}
	}
	gauges := byID[SectionGauges]
	if gauges.Available && !gauges.Stale && len(gauges.Items) > 0 {
		add("%s at or above alert level.", plural(len(gauges.Items), "river gauge", "river gauges"))
	}
	if section := byID[SectionRivers]; section.Available && !section.Stale {
		high := 0
		for _, count := range section.Counts {
			if count.Key == "severe" || count.Key == "high" {
				high += count.Value
			}
		}
		if high > 0 {
			add("Estimate: %s at high or severe.", plural(high, "river stretch", "river stretches"))
		}
	}
	if section := byID[SectionEarthquakes]; section.Available && !section.Stale {
		for _, item := range section.Items {
			if item.At == nil || now.Sub(*item.At) > recentQuake {
				continue
			}
			if item.Tone != ToneNeutral || strings.HasPrefix(item.Title, "M6") || strings.HasPrefix(item.Title, "M7") ||
				strings.HasPrefix(item.Title, "M8") || strings.HasPrefix(item.Title, "M9") {
				add("%s reported in the last day.", item.Title)
				break
			}
		}
	}
	if section := byID[SectionReservoirs]; section.Available && !section.Stale {
		for _, count := range section.Counts {
			if count.Key == "spilling" && count.Value > 0 {
				add("%s spilling.", plural(count.Value, "reservoir", "reservoirs"))
			}
		}
	}
	if gauges.Available && !gauges.Stale && len(gauges.Items) == 0 {
		add("No river gauge above its alert level.")
	}

	var stale, unread []string
	for _, section := range sections {
		switch {
		case !section.Available:
			unread = append(unread, feedNames[section.ID])
		case section.Stale:
			stale = append(stale, feedNames[section.ID])
		}
	}
	if len(stale) > 0 {
		add("Out of date: %s.", strings.Join(stale, ", "))
	}
	if len(unread) > 0 {
		add("Could not read: %s.", strings.Join(unread, ", "))
	}
	if len(parts) == 0 {
		return "Nothing to report from the feeds below. Follow official warnings."
	}
	return strings.Join(parts, " ")
}

// directionWord reads back the direction a cyclone's line gives.
func directionWord(section Section, id string) string {
	for _, item := range section.Items {
		if item.ID != id {
			continue
		}
		if _, after, ok := strings.Cut(item.Detail, " km "); ok {
			if word, _, ok := strings.Cut(after, " of Sri Lanka"); ok {
				return word
			}
		}
	}
	return "near"
}
