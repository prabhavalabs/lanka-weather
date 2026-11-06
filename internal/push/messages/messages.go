// Package messages writes push notices in English, Sinhala and Tamil.
//
// Every notice is rendered on the server in the subscriber's language from a
// template per event kind and variant (a landslide level, a flood level). The
// wording follows the issuers: NBRO's levels (Watch, Alert, Evacuate), the
// Irrigation Department's alert, minor flood and major flood levels, and
// Flood Hub's Warning, Danger and Extreme for Google's forecasts, which are
// always called forecasts or estimates. Official titles are passed through in
// the issuer's own English words. Names of places come from the regions
// table in each language; names of gauges and rivers are English only.
//
// A title stays short ("Landslide Level 3 · Ratnapura"); a body says what to
// do and who said so. A test checks that every kind and variant has text in
// all three languages with the same placeholders.
package messages

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Locales are the languages notices are written in.
var Locales = []string{"en", "si", "ta"}

// Kinds of notice. Each matches an event kind of the alert engine.
const (
	KindTest          = "test"
	KindBulletin      = "bulletin"
	KindLandslide     = "landslide"
	KindRiver         = "river"
	KindFloodForecast = "flood_forecast"
	KindFlashFlood    = "flash_flood"
	KindCyclone       = "cyclone"
	KindEarthquake    = "earthquake"
	KindReservoir     = "reservoir"
)

// Variants lists each kind's variants. A notice for one event uses the
// template "<kind>.<variant>"; a notice gathering several uses
// "<kind>.<variant>.many", or "<kind>.many" when that is the same for every
// variant.
var Variants = map[string][]string{
	KindTest:          {"test"},
	KindBulletin:      {"advisory", "warning", "red"},
	KindLandslide:     {"1", "2", "3"},
	KindRiver:         {"alert", "minor_flood", "major_flood"},
	KindFloodForecast: {"warning", "danger", "extreme"},
	KindFlashFlood:    {"likely", "highly_likely"},
	KindCyclone:       {"track", "near", "close"},
	KindEarthquake:    {"plain", "major"},
	KindReservoir:     {"spill"},
}

// Single reports whether a kind is never gathered (a test is one notice).
func Single(kind string) bool { return kind == KindTest }

// Template is a notice's title and body with {placeholders}.
type Template struct {
	Title string
	Body  string
}

// Names gives a district's name in a language, falling back to English.
type Names interface {
	District(id, locale string) string
}

// Input is what a notice says.
type Input struct {
	Kind    string
	Variant string
	// Count is how many events the notice gathers; 1 for a single event.
	Count int
	// Facts are the leading event's values: district (a code), districts
	// (codes, comma separated), station, river, site, name, title, place,
	// magnitude, distance, source and issuedAt (RFC 3339).
	Facts map[string]string
	// Now is when the notice is written, for times relative to it: an issue
	// time today is written as the hour, yesterday's says so.
	Now time.Time
}

// localZone is Sri Lanka's time, which has no daylight saving.
var localZone = time.FixedZone("Asia/Colombo", 5*3600+30*60)

// clock writes a time in Sri Lanka as "20:15", "20:15 yesterday" or, if older,
// "6/11 20:15" (day and month).
func clock(locale string, at, now time.Time, word func(string) string) string {
	local := at.In(localZone)
	text := local.Format("15:04")
	if now.IsZero() {
		return text
	}
	today := now.In(localZone)
	switch day := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, localZone) }; {
	case day(local).Equal(day(today)):
		return text
	case day(local).Equal(day(today).AddDate(0, 0, -1)):
		return text + " " + word("time.yesterday")
	}
	return fmt.Sprintf("%d/%d %s", local.Day(), int(local.Month()), text)
}

const (
	maxTitle = 70
	maxBody  = 240
)

var placeholder = regexp.MustCompile(`\{(\w+)\}`)

// Render writes a notice in a language (English if it is not one we have).
func Render(locale string, input Input, names Names) (title, body string) {
	catalogue, ok := templates[locale]
	if !ok {
		locale, catalogue = "en", templates["en"]
	}
	template := lookup(catalogue, input)
	values := values(locale, input, names)
	fill := func(text string) string {
		return placeholder.ReplaceAllStringFunc(text, func(match string) string {
			if value, ok := values[match[1:len(match)-1]]; ok {
				return value
			}
			return ""
		})
	}
	return truncate(fill(template.Title), maxTitle), truncate(fill(template.Body), maxBody)
}

func lookup(catalogue map[string]Template, input Input) Template {
	keys := []string{input.Kind + "." + input.Variant}
	if input.Count > 1 && !Single(input.Kind) {
		keys = []string{input.Kind + "." + input.Variant + ".many", input.Kind + ".many"}
	}
	for _, key := range keys {
		if template, ok := catalogue[key]; ok {
			return template
		}
	}
	return catalogue["fallback"]
}

// values are the placeholders' text in a language.
func values(locale string, input Input, names Names) map[string]string {
	facts := input.Facts
	out := map[string]string{}
	for key, value := range facts {
		out[key] = value
	}
	word := func(key string) string {
		if text, ok := words[locale][key]; ok {
			return text
		}
		return words["en"][key]
	}
	district := func(id string) string {
		if names == nil || id == "" {
			return id
		}
		return names.District(id, locale)
	}
	out["district"] = district(facts["district"])
	if facts["district"] == "" {
		out["district"] = word("country")
	}
	place := word("country")
	if codes := splitCodes(facts["districts"]); len(codes) > 0 {
		place = district(codes[0])
		if len(codes) > 1 {
			place += " +" + strconv.Itoa(len(codes)-1)
		}
	}
	out["place"] = place
	out["levelWord"] = word(input.Kind + "." + input.Variant)
	out["severityWord"] = word(input.Kind + "." + input.Variant)
	out["source"] = word("source." + facts["source"])
	out["likelihood"] = word(input.Kind + "." + input.Variant)
	out["level"] = facts["level"]
	count := max(input.Count, 1)
	out["count"] = strconv.Itoa(count)
	out["more"] = strconv.Itoa(count - 1)
	if distance, err := strconv.Atoi(facts["distance"]); err == nil {
		out["distance"] = grouped(distance)
	}
	out["issued"] = word("time.unknown")
	if issued, err := time.Parse(time.RFC3339, facts["issuedAt"]); err == nil {
		out["issued"] = clock(locale, issued, input.Now, word)
	}
	return out
}

func splitCodes(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.Split(value, ",")
}

// grouped writes a whole number with thousands separators: 1,200.
func grouped(value int) string {
	text := strconv.Itoa(value)
	if value < 0 {
		return text
	}
	var out strings.Builder
	for index, digit := range text {
		if index > 0 && (len(text)-index)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(digit)
	}
	return out.String()
}

// truncate cuts text to a number of characters, with an ellipsis.
func truncate(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}

// Magnitude writes a magnitude to one decimal, as USGS does.
func Magnitude(value float64) string { return fmt.Sprintf("%.1f", value) }

// Placeholders lists a template's placeholders, for tests.
func Placeholders(text string) []string {
	found := map[string]bool{}
	for _, match := range placeholder.FindAllStringSubmatch(text, -1) {
		found[match[1]] = true
	}
	out := make([]string, 0, len(found))
	for name := range found {
		out = append(out, name)
	}
	return out
}

// Templates returns a language's templates, for tests.
func Templates(locale string) map[string]Template { return templates[locale] }

// Words returns a language's words, for tests.
func Words(locale string) map[string]string { return words[locale] }
