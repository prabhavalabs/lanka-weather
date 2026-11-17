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
// A title says what and where ("Landslide Level 3 · Ratnapura", "Flood
// warning · Galle, Matara"); a body says where in full, when the issuer issued
// or measured it, who said so and what to do. A test checks that every kind
// and variant has text in all three languages with the same placeholders.
package messages

import (
	"fmt"
	"regexp"
	"slices"
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
	KindRain          = "rain"
	KindFloodRisk     = "flood_risk"
)

// Variants lists each kind's variants. A notice for one event uses the
// template "<kind>.<variant>"; a notice gathering several uses
// "<kind>.<variant>.many", or "<kind>.many" when that is the same for every
// variant.
var Variants = map[string][]string{
	KindTest:          {"test"},
	KindBulletin:      {"advisory", "warning", "red"},
	KindLandslide:     {"issued", "1", "2", "3"},
	KindRiver:         {"alert", "minor_flood", "major_flood"},
	KindFloodForecast: {"warning", "danger", "extreme"},
	KindFlashFlood:    {"likely", "highly_likely"},
	KindCyclone:       {"track", "near", "close"},
	KindEarthquake:    {"plain", "major"},
	KindReservoir:     {"spill"},
	KindRain:          {"heavy", "very_heavy", "extreme"},
	KindFloodRisk:     {"moderate", "high", "very_high"},
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
	// magnitude, distance, source, issuedAt (RFC 3339, when the issuer issued
	// or measured it), for bulletins type and areaFrom, for landslides
	// divisions and divisionNames, for gauges waterLevel, threshold and unit,
	// for cyclones direction and nearDistrict, for rain the gauge, mm and
	// window (1h, 3h or 24h), and for the flood-risk estimate signals (family
	// codes, strongest first) and scenario.
	Facts map[string]string
	// Districts are every district the notice is about, in the order to name
	// them; empty takes them from the facts.
	Districts []string
	// Subjects name what a gathered notice gathers: gauges, reservoirs, sites.
	Subjects []string
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
	maxBody  = 300
	// maxWhere is how long a title's list of places may grow before the rest
	// are counted ("Galle, Matara +3").
	maxWhere = 34
	// maxSubjects are the gauges or reservoirs a gathered notice names.
	maxSubjects = 3
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
	codes := input.Districts
	if len(codes) == 0 {
		codes = splitCodes(facts["districts"])
	}
	if len(codes) == 0 && facts["district"] != "" {
		codes = []string{facts["district"]}
	}
	places := make([]string, 0, len(codes))
	for _, code := range codes {
		if name := district(code); name != "" && !slices.Contains(places, name) {
			places = append(places, name)
		}
	}
	out["district"] = word("country")
	switch {
	case facts["district"] != "":
		out["district"] = district(facts["district"])
	case len(places) > 0:
		out["district"] = places[0]
	}
	out["where"] = shortList(places, word("country"))
	out["places"] = word("country")
	if len(places) > 0 {
		out["places"] = joinList(places, word("list.and"))
	}
	out["subjects"] = subjectList(input.Subjects, facts)
	out["levelWord"] = word(input.Kind + "." + input.Variant)
	out["source"] = word("source." + facts["source"])
	out["likelihood"] = word(input.Kind + "." + input.Variant)
	out["level"] = facts["level"]
	count := max(input.Count, 1)
	out["count"] = strconv.Itoa(count)
	out["more"] = strconv.Itoa(count - 1)
	if distance, err := strconv.Atoi(facts["distance"]); err == nil {
		out["distance"] = grouped(distance)
	}
	if facts["window"] != "" {
		out["window"] = word("window." + facts["window"])
	}
	if codes := splitCodes(facts["signals"]); len(codes) > 0 {
		signals := make([]string, 0, len(codes))
		for _, code := range codes {
			signals = append(signals, word("signal."+code))
		}
		out["signals"] = joinList(signals, word("list.and"))
	}
	if facts["scenario"] != "" {
		out["scenario"] = word("scenario." + facts["scenario"])
	}
	out["issued"] = word("time.unknown")
	issuedAt := facts["issuedAt"]
	if issuedAt == "" {
		issuedAt = facts["observedAt"]
	}
	out["whenIssued"], out["whenRead"] = "", ""
	if issued, err := time.Parse(time.RFC3339, issuedAt); err == nil {
		out["issued"] = clock(locale, issued, input.Now, word)
		out["whenIssued"] = fill(word("when.issued"), map[string]string{"time": out["issued"]})
		out["whenRead"] = fill(word("when.read"), map[string]string{"time": out["issued"]})
	}

	// Composite lines, empty when their facts are missing.
	out["bulletinKind"] = word("kind.bulletin." + input.Variant)
	if kind, ok := words[locale]["kind.bulletin."+facts["type"]+"."+input.Variant]; ok {
		out["bulletinKind"] = kind
	}
	switch {
	case len(places) == 0:
		out["areaLine"] = word("area.unstated")
	case facts["areaFrom"] == "basin":
		out["areaLine"] = fill(word("area.basin"), map[string]string{"places": out["places"]})
	default:
		out["areaLine"] = fill(word("area.districts"), map[string]string{"places": out["places"]})
	}
	out["divisionsLine"] = ""
	if facts["divisions"] != "" {
		key := "divisions.count"
		if facts["divisionNames"] != "" {
			key = "divisions.named"
		}
		out["divisionsLine"] = fill(word(key), map[string]string{"count": facts["divisions"], "names": facts["divisionNames"]})
	}
	out["readingLine"] = ""
	if facts["waterLevel"] != "" && facts["threshold"] != "" {
		unit := facts["unit"]
		if unit == "" {
			unit = "m"
		}
		out["readingLine"] = fill(word("reading.line"), map[string]string{"level": trimNumber(facts["waterLevel"]),
			"threshold": trimNumber(facts["threshold"]), "unit": unit})
	}
	station := facts["station"]
	if station == "" {
		station = facts["site"]
	}
	switch {
	case station != "" && facts["river"] != "" && !strings.EqualFold(station, facts["river"]):
		out["stationPlace"] = fill(word("station.place"), map[string]string{"river": facts["river"], "station": station})
	case station != "":
		out["stationPlace"] = station
	default:
		out["stationPlace"] = facts["river"]
	}
	out["directionPlace"] = word("direction.none")
	if direction, ok := words[locale]["direction."+facts["direction"]]; ok && facts["direction"] != "" {
		out["directionPlace"] = direction
	}
	out["nearLine"] = ""
	if facts["nearDistrict"] != "" {
		out["nearLine"] = fill(word("near.line"), map[string]string{"district": district(facts["nearDistrict"])})
	}
	return out
}

// fill writes values into a text's {placeholders}; unknown ones are dropped.
func fill(text string, values map[string]string) string {
	return placeholder.ReplaceAllStringFunc(text, func(match string) string {
		return values[match[1:len(match)-1]]
	})
}

// trimNumber writes a reading without trailing zeros: 7.50 is 7.5, 7.00 is 7.
func trimNumber(value string) string {
	if !strings.Contains(value, ".") {
		return value
	}
	return strings.TrimSuffix(strings.TrimRight(value, "0"), ".")
}

// shortList names places for a title: as many as fit, then a count of the
// rest ("Galle, Matara +3"), or the fallback when there are none.
func shortList(places []string, fallback string) string {
	if len(places) == 0 {
		return fallback
	}
	text, shown := places[0], 1
	for shown < len(places) {
		next := text + ", " + places[shown]
		if utf8.RuneCountInString(next) > maxWhere {
			break
		}
		text, shown = next, shown+1
	}
	if shown < len(places) {
		text += " +" + strconv.Itoa(len(places)-shown)
	}
	return text
}

// subjectList names what a gathered notice gathers, at most three and a count
// of the rest, or the lead's own name.
func subjectList(subjects []string, facts map[string]string) string {
	if len(subjects) == 0 {
		for _, key := range []string{"station", "site", "name", "gauge"} {
			if facts[key] != "" {
				return facts[key]
			}
		}
		return ""
	}
	shown := subjects[:min(len(subjects), maxSubjects)]
	text := strings.Join(shown, ", ")
	if rest := len(subjects) - len(shown); rest > 0 {
		text += " +" + strconv.Itoa(rest)
	}
	return text
}

// Issuer names who a notice's information comes from, in a language.
func Issuer(locale, kind string, facts map[string]string) string {
	key := "issuer." + kind
	if kind == KindRiver && facts["source"] == "community" {
		key = "issuer.river.community"
	}
	if text, ok := words[locale][key]; ok {
		return text
	}
	if text, ok := words["en"][key]; ok {
		return text
	}
	return words[locale]["issuer.test"]
}

// Official reports whether a notice passes on a Sri Lankan authority's own
// warning or reading (DMC, NBRO, the Irrigation Department), rather than a
// forecast, an estimate or community data.
func Official(kind string, facts map[string]string) bool {
	switch kind {
	case KindBulletin, KindLandslide:
		return true
	case KindRiver:
		return facts["source"] == "irrigation"
	}
	return false
}

// joinList writes "a, b and c".
func joinList(items []string, and string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + and + " " + items[len(items)-1]
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
