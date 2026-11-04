// Package dmcbulletins collects the Disaster Management Centre's bulletins
// through the indexes of nuuuwan/lk_dmc: landslide early-warning reports,
// river water level and flood warnings, weather reports and situation
// reports. The PDFs on dmc.gov.lk remain the official documents.
//
// Each index is a tab-separated list of the last hundred documents. Every
// document also has text extracted from its PDF. That text is mixed: Sinhala
// with broken glyphs, Tamil in a legacy font that extracts as Latin gibberish
// ("kz;rhpT"), tables run together, and often clean English paragraphs. The
// rules here keep only whole English sentences, unchanged, and everything else
// is shown as a title and a link to the PDF (ADR 0011).
package dmcbulletins

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SourceName identifies the source in docs/data-sources.md.
const SourceName = "dmc-reports"

// Attribution credits the issuers and the index.
const Attribution = "Disaster Management Centre, Sri Lanka, via nuuuwan/lk_dmc"

// Type is a family of bulletins.
type Type string

const (
	TypeLandslide  Type = "landslide"
	TypeWaterLevel Type = "water_level"
	TypeWeather    Type = "weather"
	TypeSituation  Type = "situation"
)

// Types lists the families in display order.
var Types = []Type{TypeLandslide, TypeWaterLevel, TypeWeather, TypeSituation}

// Severity says how clearly a bulletin is a warning.
type Severity string

const (
	SeverityWarning    Severity = "warning"
	SeverityAdvisory   Severity = "advisory"
	SeverityWithdrawal Severity = "withdrawal"
	SeverityRoutine    Severity = "routine"
)

// Text statuses of a bulletin's body.
const (
	TextPending     = "pending"
	TextClean       = "clean"
	TextUnusable    = "unusable"
	TextUnavailable = "unavailable"
)

// sriLanka is Sri Lanka Standard Time, UTC+05:30, with no daylight saving.
var sriLanka = time.FixedZone("Asia/Colombo", 5*60*60+30*60)

// Bulletin is one row of an index.
type Bulletin struct {
	DocID         string
	Type          Type
	Title         string
	Severity      Severity
	IssuedAt      time.Time
	PDFURL        string
	SourcePageURL string
	Language      string
}

var indexColumns = []string{"doc_type", "doc_id", "num", "date_str", "description", "url_metadata", "lang", "url_pdf", "time_str", "ut"}

// DecodeIndex reads one index (docs_last100.tsv). A row without a usable
// identifier, time or https link is skipped and counted, so one bad row does
// not lose the others. A file that is not an index at all is an error.
func DecodeIndex(payload []byte, kind Type) (bulletins []Bulletin, skipped int, err error) {
	reader := csv.NewReader(bytes.NewReader(payload))
	reader.Comma = '\t'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, 0, fmt.Errorf("read index header: %w", err)
	}
	column := make(map[string]int, len(header))
	for index, name := range header {
		column[strings.TrimSpace(name)] = index
	}
	for _, name := range indexColumns {
		if _, ok := column[name]; !ok {
			return nil, 0, fmt.Errorf("index has no %q column", name)
		}
	}
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return bulletins, skipped, nil
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read index: %w", err)
		}
		if len(record) != len(header) {
			skipped++
			continue
		}
		field := func(name string) string { return strings.TrimSpace(record[column[name]]) }
		bulletin, ok := decodeRow(field, kind)
		if !ok {
			skipped++
			continue
		}
		bulletins = append(bulletins, bulletin)
	}
}

func decodeRow(field func(string) string, kind Type) (Bulletin, bool) {
	id := field("doc_id")
	title := field("description")
	seconds, err := strconv.ParseFloat(field("ut"), 64)
	if id == "" || title == "" || err != nil || seconds <= 0 {
		return Bulletin{}, false
	}
	pdf, page := field("url_pdf"), field("url_metadata")
	if !httpLink(pdf) || !httpLink(page) {
		return Bulletin{}, false
	}
	return Bulletin{
		DocID: id, Type: kind, Title: title, Severity: Classify(kind, title, ""),
		IssuedAt: time.Unix(int64(seconds), 0).UTC(),
		PDFURL:   pdf, SourcePageURL: page, Language: language(title, field("lang")),
	}, true
}

// httpLink accepts only absolute web links, so nothing from an index can
// become a script link in the interface.
func httpLink(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

// language is what the title says, else the index's. The index says "en" even
// for the Tamil editions, whose titles say "(Tamil Language)", "- Tamil" or
// "Tamil".
func language(title, indexed string) string {
	lower := strings.ToLower(title)
	switch {
	case strings.Contains(lower, "tamil"):
		return "ta"
	case strings.Contains(lower, "sinhala"):
		return "si"
	case indexed != "":
		return indexed
	}
	return "en"
}

var (
	wordWarning    = regexp.MustCompile(`(?i)\bwarnings?\b`)
	wordAdvisory   = regexp.MustCompile(`(?i)\badvisory\b`)
	wordWithdrawal = regexp.MustCompile(`(?i)\bwithdraw(al|n)\b`)
)

// Classify reads how clearly a bulletin is a warning from its title and the
// colour it states. Landslide reports are always "routine" here: their
// seriousness is the level of each division, not the word in the title.
func Classify(kind Type, title, colour string) Severity {
	switch {
	case kind == TypeLandslide:
		return SeverityRoutine
	case wordWithdrawal.MatchString(title):
		return SeverityWithdrawal
	case wordWarning.MatchString(title), colour == "Red":
		return SeverityWarning
	case wordAdvisory.MatchString(title), colour == "Amber", colour == "Yellow":
		return SeverityAdvisory
	}
	return SeverityRoutine
}

// DecodeUpdatedAt reads when an index was last rebuilt from its summary.json;
// the file gives Sri Lanka time without saying so (a rebuild stamped 00:05:41
// matches a commit at 18:35:46 UTC).
func DecodeUpdatedAt(payload []byte) (time.Time, error) {
	var wire struct {
		TimeUpdated string `json:"time_updated"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil {
		return time.Time{}, fmt.Errorf("decode summary: %w", err)
	}
	at, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(wire.TimeUpdated), sriLanka)
	if err != nil {
		return time.Time{}, fmt.Errorf("summary time %q: %w", wire.TimeUpdated, err)
	}
	return at.UTC(), nil
}

// Extract is what can be relied on in a bulletin's extracted text.
type Extract struct {
	// WithdrawnBasins are the basins a withdrawal notice lists; nil when it is
	// not one or its list cannot be read.
	WithdrawnBasins []string
	// Body is whole English sentences copied unchanged, or "" when there are
	// none worth showing.
	Body       string
	Status     string
	Colour     string
	MessageID  string
	ValidFrom  *time.Time
	ValidUntil *time.Time
	// Districts are the codes of districts Body names by exact word.
	Districts []string
}

// minimumBody is the shortest excerpt worth showing, in bytes.
const minimumBody = 60

// ExtractText reads a bulletin's doc.txt. Landslide reports keep no body: their
// lines lose the level columns they belong to, so "Yatinuwara" alone says
// nothing; their levels come from the structured feed instead.
func ExtractText(raw string, kind Type, issuedAt time.Time) Extract {
	extract := Extract{Status: TextUnusable}
	extract.MessageID = messageID(raw)
	extract.Colour = statedColour(raw)
	extract.ValidFrom, extract.ValidUntil = validity(raw, issuedAt)
	if kind == TypeLandslide {
		return extract
	}
	if kind == TypeWaterLevel {
		extract.WithdrawnBasins = WithdrawnBasins(raw)
	}
	sentences := cleanParagraphs(raw)
	body := strings.Join(sentences, "\n\n")
	if len(body) < minimumBody {
		return extract
	}
	extract.Body = body
	extract.Status = TextClean
	extract.Districts = NamedDistricts(body)
	return extract
}

var (
	latinGibberish = regexp.MustCompile(`[A-Za-z];|[;^³œ’]\s?[a-z]`)
	wordPattern    = regexp.MustCompile(`[A-Za-z']+`)
	numberPattern  = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)
	// A Word run into a number is a table that lost its cells: "m15.00".
	runTogether = regexp.MustCompile(`[a-z]\d|\d[a-z]{3,}`)
)

// commonWords are short words English prose is full of and legacy-font
// gibberish and name lists are not.
var commonWords = map[string]bool{
	"the": true, "of": true, "in": true, "and": true, "to": true, "for": true, "at": true, "is": true, "are": true,
	"will": true, "be": true, "a": true, "on": true, "by": true, "from": true, "with": true, "this": true, "that": true,
	"may": true, "or": true, "can": true, "as": true, "it": true, "over": true, "these": true, "there": true,
	"have": true, "has": true, "been": true, "not": true, "their": true, "due": true, "during": true, "into": true,
}

// cleanParagraphs keeps the paragraphs that are whole English sentences, in
// order, each unchanged.
func cleanParagraphs(raw string) []string {
	var kept []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n\n") {
		paragraph = strings.TrimSpace(paragraph)
		if isCleanSentence(paragraph) {
			kept = append(kept, paragraph)
		}
	}
	return kept
}

// isCleanSentence reports whether a paragraph is plainly readable English
// prose: plain text, starting with a capital, ending a sentence, mostly words,
// with enough common words to be language rather than a list of names.
func isCleanSentence(paragraph string) bool {
	if paragraph == "" || strings.Contains(paragraph, "\n") || !isPlainText(paragraph) {
		return false
	}
	first := paragraph[0]
	if first < 'A' || first > 'Z' {
		return false
	}
	if last := paragraph[len(paragraph)-1]; last != '.' && last != '!' {
		return false
	}
	if latinGibberish.MatchString(paragraph) || runTogether.MatchString(paragraph) {
		return false
	}
	words := wordPattern.FindAllString(paragraph, -1)
	if len(words) < 8 {
		return false
	}
	common := 0
	for _, word := range words {
		if commonWords[strings.ToLower(word)] {
			common++
		}
	}
	if float64(common)/float64(len(words)) < 0.2 {
		return false
	}
	return len(numberPattern.FindAllString(paragraph, -1))*2 <= len(words)
}

func isPlainText(value string) bool {
	for _, character := range value {
		if character > 0x7e || (character < 0x20 && character != '\n') {
			return false
		}
	}
	return true
}

var (
	messagePattern = regexp.MustCompile(`\b(LEWM-\d{2}-\d{2}-\d{4}-\d{4}-\d+)\b|Bulletin No:\s*([0-9]+\s+[A-Z]+\s*/\d{2}/\d{2}/\d{2}/\d{2})`)
	// The colour a bulletin gives itself: "COLOR: Red" and "Colour – Amber".
	// Legends list every colour as "Color: -Red"; those do not match.
	colourPattern = regexp.MustCompile(`\bCOLOR:\s*(Red|Amber|Yellow|Green)\b|\bColour\s*[–-]\s*(Red|Amber|Yellow|Green)\b`)
)

func messageID(raw string) string {
	match := messagePattern.FindStringSubmatch(raw)
	if match == nil {
		return ""
	}
	if match[1] != "" {
		return match[1]
	}
	return strings.Join(strings.Fields(match[2]), " ")
}

func statedColour(raw string) string {
	match := colourPattern.FindStringSubmatch(raw)
	if match == nil {
		return ""
	}
	if match[1] != "" {
		return match[1]
	}
	return match[2]
}

var (
	// "Validity Period: From 16:00 hrs on 31.10.2026 To 16:00 hrs on 01.11.2026".
	periodPattern = regexp.MustCompile(`Validity Period\s*:\s*From\s+(\d{1,2}):(\d{2})\s*hrs\s+on\s+(\d{2})\.(\d{2})\.(\d{4})\s+To\s+(\d{1,2}):(\d{2})\s*hrs\s+on\s+(\d{2})\.(\d{2})\.(\d{4})`)
	// "valid for the next 24 hours".
	nextHoursPattern = regexp.MustCompile(`(?i)valid for the next (\d{1,3}) hours`)
	// "Valid till 9.30 a.m. on 23rd of October 2026", "valid for the period
	// until 5.30 a.m. on 27 October 2026", "until 11.30 p.m. 30 October 2026",
	// "until 12.00 noon on 22 October 2026".
	untilPattern = regexp.MustCompile(`(?i)valid (?:till|until|for the period until)\s+(\d{1,2})[.:](\d{2})\s*(a\.?m\.?|p\.?m\.?|noon)?\.?\s*(?:on\s+)?(\d{1,2})(?:st|nd|rd|th)?\s+(?:of\s+)?([A-Za-z]+),?\s+(\d{4})`)
)

// validity reads the period a bulletin states for itself, in English. Times
// are Sri Lanka time.
func validity(raw string, issuedAt time.Time) (from, until *time.Time) {
	if match := periodPattern.FindStringSubmatch(raw); match != nil {
		start := dmyTime(match[3], match[4], match[5], match[1], match[2])
		end := dmyTime(match[8], match[9], match[10], match[6], match[7])
		if start != nil && end != nil && end.After(*start) {
			return start, end
		}
	}
	if match := untilPattern.FindStringSubmatch(raw); match != nil {
		if end := clockTime(match); end != nil && end.After(issuedAt) {
			return nil, end
		}
	}
	if match := nextHoursPattern.FindStringSubmatch(raw); match != nil {
		hours, err := strconv.Atoi(match[1])
		if err == nil && hours > 0 && hours <= 240 {
			end := issuedAt.Add(time.Duration(hours) * time.Hour)
			return nil, &end
		}
	}
	return nil, nil
}

func dmyTime(day, month, year, hour, minute string) *time.Time {
	at, err := time.ParseInLocation("02.01.2006 15:04", day+"."+month+"."+year+" "+pad(hour)+":"+minute, sriLanka)
	if err != nil {
		return nil
	}
	at = at.UTC()
	return &at
}

func pad(hour string) string {
	if len(hour) == 1 {
		return "0" + hour
	}
	return hour
}

func clockTime(match []string) *time.Time {
	hour, err := strconv.Atoi(match[1])
	if err != nil || hour > 23 {
		return nil
	}
	minute, err := strconv.Atoi(match[2])
	if err != nil {
		return nil
	}
	day, err := strconv.Atoi(match[4])
	if err != nil {
		return nil
	}
	month, err := time.Parse("January", capitalised(match[5]))
	if err != nil {
		return nil
	}
	year, err := strconv.Atoi(match[6])
	if err != nil {
		return nil
	}
	switch marker := strings.ToLower(strings.ReplaceAll(match[3], ".", "")); marker {
	case "pm":
		if hour < 12 {
			hour += 12
		}
	case "am":
		if hour == 12 {
			hour = 0
		}
	}
	at := time.Date(year, month.Month(), day, hour, minute, 0, 0, sriLanka).UTC()
	return &at
}

func capitalised(word string) string {
	if word == "" {
		return word
	}
	return strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
}

// districts are the 25 districts' codes and the spellings bulletins use.
var districts = []struct {
	id    string
	names []string
}{
	{"LK-11", []string{"Colombo"}},
	{"LK-12", []string{"Gampaha"}},
	{"LK-13", []string{"Kalutara", "Kaluthara"}},
	{"LK-21", []string{"Kandy"}},
	{"LK-22", []string{"Matale"}},
	{"LK-23", []string{"Nuwara Eliya", "Nuwara-Eliya", "Nuwaraeliya"}},
	{"LK-31", []string{"Galle"}},
	{"LK-32", []string{"Matara"}},
	{"LK-33", []string{"Hambantota", "Hambanthota"}},
	{"LK-41", []string{"Jaffna"}},
	{"LK-42", []string{"Mannar"}},
	{"LK-43", []string{"Vavuniya"}},
	{"LK-44", []string{"Mullaitivu", "Mullaithivu"}},
	{"LK-45", []string{"Kilinochchi"}},
	{"LK-51", []string{"Batticaloa"}},
	{"LK-52", []string{"Ampara"}},
	{"LK-53", []string{"Trincomalee"}},
	{"LK-61", []string{"Kurunegala"}},
	{"LK-62", []string{"Puttalam"}},
	{"LK-71", []string{"Anuradhapura"}},
	{"LK-72", []string{"Polonnaruwa"}},
	{"LK-81", []string{"Badulla"}},
	{"LK-82", []string{"Monaragala", "Moneragala"}},
	{"LK-91", []string{"Ratnapura", "Rathnapura"}},
	{"LK-92", []string{"Kegalle"}},
}

var districtPatterns = func() map[string]*regexp.Regexp {
	patterns := make(map[string]*regexp.Regexp, len(districts))
	for _, district := range districts {
		quoted := make([]string, len(district.names))
		for index, name := range district.names {
			quoted[index] = regexp.QuoteMeta(name)
		}
		patterns[district.id] = regexp.MustCompile(`\b(?:` + strings.Join(quoted, "|") + `)\b`)
	}
	return patterns
}()

// NamedDistricts lists the codes of the districts a text names by exact word,
// in code order. A name is a name: "Mannar" in "via Mannar" counts, and it is
// labelled "Mentioned in bulletin" and nothing stronger.
func NamedDistricts(text string) []string {
	found := []string{}
	for _, district := range districts {
		if districtPatterns[district.id].MatchString(text) {
			found = append(found, district.id)
		}
	}
	return found
}

// DistrictName returns a district's name as our regions write it.
func DistrictName(id string) string {
	for _, district := range districts {
		if district.id == id {
			return district.names[0]
		}
	}
	return ""
}

var (
	// withdrawalPreamble is how the department introduces its list, in both
	// wordings seen (eight notices from 2025 to 2026): "... are hereby withdrawn,
	// as the water levels ...".
	withdrawalPreamble = regexp.MustCompile(`(?i)\bhereby\s+withdrawn\b`)
	// withdrawalEnd is the paragraph that follows the list.
	withdrawalEnd = regexp.MustCompile(`(?i)\bThe Hydrology and Disaster Management Division continues\b|\bEng\.\s`)
	numberedItem  = regexp.MustCompile(`(?:^|\s)(\d{1,2})\.\s+`)
)

// maxBasinName bounds one list item: anything longer is not a name read cleanly.
const maxBasinName = 60

// WithdrawnBasins reads the basins a flood warning withdrawal notice lists, in
// the issuer's spelling, without the warning details that can follow a name:
//
//  1. Kelani Ganga 2. Kalu Ganga 3. Gin Ganga
//  1. Kelani River Basin: Flood Warning - No. 03 issued on 03.08.2026 at 1:30 PM
//
// It returns nil unless the English text has the preamble and a list numbered
// from 1: some notices are only in Sinhala. Nothing is read from a list it cannot
// follow to its end, so a misread never withdraws the wrong warning.
func WithdrawnBasins(raw string) []string {
	text := strings.Join(strings.Fields(raw), " ")
	start := withdrawalPreamble.FindStringIndex(text)
	if start == nil {
		return nil
	}
	text = text[start[1]:]
	end := withdrawalEnd.FindStringIndex(text)
	if end == nil {
		return nil
	}
	text = text[:end[0]]
	marks := numberedItem.FindAllStringSubmatchIndex(text, -1)
	var items []string
	var from int
	next := 1
	for _, mark := range marks {
		number, err := strconv.Atoi(text[mark[2]:mark[3]])
		if err != nil || number != next {
			continue // a number within an item: "No. 03", "at 1.30"
		}
		if next > 1 {
			items = append(items, text[from:mark[0]])
		}
		from = mark[1]
		next++
	}
	if next == 1 {
		return nil
	}
	items = append(items, text[from:])
	basins := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		name, _, _ := strings.Cut(item, ":")
		name = strings.TrimSpace(name)
		if name == "" || len(name) > maxBasinName || !isPlainText(name) {
			return nil
		}
		if !seen[strings.ToLower(name)] {
			seen[strings.ToLower(name)] = true
			basins = append(basins, name)
		}
	}
	return basins
}
