// Package warnings serves official warnings: NBRO landslide levels per DS
// division and the Disaster Management Centre's bulletins. Words, levels and
// links are the issuers' own; this package decides only what counts as still
// in force, and says when it cannot tell.
package warnings

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/dmcbulletins"
)

// Landslide statuses. Only "none" may read as calm, and it never says "safe".
const (
	// StatusWaiting: nothing has been collected yet.
	StatusWaiting = "waiting"
	// StatusActive: the latest report is within its validity period.
	StatusActive = "active"
	// StatusLapsed: the validity period ended a short while ago. A new report
	// may be on its way, so the levels stay on the map, marked.
	StatusLapsed = "lapsed"
	// StatusPending: a newer report has been issued whose levels have not
	// been parsed yet.
	StatusPending = "pending"
	// StatusNone: the latest report's validity ended and the feed is up to date.
	StatusNone = "none"
	// StatusUnknown: the latest report's validity ended and the feed is out of
	// date, so nothing can be said about today.
	StatusUnknown = "unknown"
)

const (
	// assumedValidity applies when a report's text gives no period. Reports
	// state 24 hours (ADR 0011).
	assumedValidity = 24 * time.Hour
	// lapseGrace is how long levels stay shown after the validity period ends:
	// the next report is usually issued at the same time of day and reaches
	// this feed about five hours later.
	lapseGrace = 12 * time.Hour
	// assumedIssueHour applies to a report with no matching bulletin; NBRO
	// issues at 16:00 local time.
	assumedIssueHour = 16
	// defaultBulletinCurrency is how long a warning or advisory stating no
	// period counts as in force.
	defaultBulletinCurrency = 24 * time.Hour
)

// sriLanka is Sri Lanka Standard Time, UTC+05:30.
var sriLanka = time.FixedZone("Asia/Colombo", 5*60*60+30*60)

// LevelWords are NBRO's words for a level: as printed on its reports
// ("Level 1 (Yellow)") and what each asks people to do.
type LevelWords struct {
	Level  int    `json:"level"`
	Label  string `json:"label"`
	Colour string `json:"colour"`
	Action string `json:"action"`
}

// WordsFor returns NBRO's words for a level.
func WordsFor(level int) LevelWords {
	switch level {
	case 1:
		return LevelWords{Level: 1, Label: "Level 1 (Yellow)", Colour: "Yellow", Action: "Watch"}
	case 2:
		return LevelWords{Level: 2, Label: "Level 2 (Amber)", Colour: "Amber", Action: "Alert"}
	case 3:
		return LevelWords{Level: 3, Label: "Level 3 (Red)", Colour: "Red", Action: "Evacuate"}
	}
	return LevelWords{}
}

// ReportFacts is what is known about the latest report and the feed.
type ReportFacts struct {
	// Date is the report's local date, midnight UTC.
	Date time.Time
	// IssuedAt and ValidUntil come from the matching bulletin when one exists.
	IssuedAt   *time.Time
	ValidUntil *time.Time
	// CheckedAt is when the feed was last read, UpstreamAt when its upstream
	// last rebuilt; nil if never.
	CheckedAt  *time.Time
	UpstreamAt *time.Time
	// NewerIssuedAt is the issue time of a landslide bulletin newer than the
	// report, whose levels have not arrived.
	NewerIssuedAt *time.Time
}

// Assessment is the conclusion drawn from ReportFacts.
type Assessment struct {
	Status string
	// ShowLevels says whether the report's levels belong on the map.
	ShowLevels bool
	// Stale says the feed has not been checked recently enough to trust.
	Stale      bool
	IssuedAt   time.Time
	ValidUntil time.Time
	// ValidityStated is false when ValidUntil is assumed.
	ValidityStated bool
	Note           string
}

// Assess decides what can be said about landslide warnings now.
func Assess(facts ReportFacts, now time.Time, staleAfter time.Duration) Assessment {
	issued := time.Date(facts.Date.Year(), facts.Date.Month(), facts.Date.Day(), assumedIssueHour, 0, 0, 0, sriLanka).UTC()
	if facts.IssuedAt != nil {
		issued = *facts.IssuedAt
	}
	assessment := Assessment{IssuedAt: issued, ValidUntil: issued.Add(assumedValidity)}
	if facts.ValidUntil != nil {
		assessment.ValidUntil = *facts.ValidUntil
		assessment.ValidityStated = true
	}
	assessment.Stale = feedStale(facts, now, staleAfter)
	date := localDate(facts.Date)
	lastChecked := "never"
	if facts.CheckedAt != nil {
		lastChecked = localDateTime(*facts.CheckedAt)
	}

	switch {
	case !now.After(assessment.ValidUntil):
		assessment.Status, assessment.ShowLevels = StatusActive, true
		assessment.Note = fmt.Sprintf("Report of %s, valid until %s.", date, localDateTime(assessment.ValidUntil))
	case !now.After(assessment.ValidUntil.Add(lapseGrace)):
		assessment.Status, assessment.ShowLevels = StatusLapsed, true
		assessment.Note = fmt.Sprintf("Report of %s: its validity ended %s. A newer report may not have reached us yet, so its levels stay shown.",
			date, localDateTime(assessment.ValidUntil))
	case assessment.Stale:
		assessment.Status = StatusUnknown
		assessment.Note = fmt.Sprintf("Last report %s. We have not been able to check for newer reports since %s, so we cannot say whether a warning is active. Check with the DMC.",
			date, lastChecked)
	default:
		assessment.Status = StatusNone
		assessment.Note = fmt.Sprintf("Last report %s; none currently active as far as we know.", date)
	}

	if facts.NewerIssuedAt != nil && facts.NewerIssuedAt.After(issued) {
		assessment.Status = StatusPending
		assessment.Note = fmt.Sprintf("A report was issued %s, and its levels have not reached us yet. Read the official report.",
			localDateTime(*facts.NewerIssuedAt))
		return assessment
	}
	if assessment.Stale && assessment.ShowLevels {
		assessment.Note += fmt.Sprintf(" We have not been able to check for newer reports since %s.", lastChecked)
	}
	return assessment
}

func feedStale(facts ReportFacts, now time.Time, limit time.Duration) bool {
	return facts.CheckedAt == nil || now.Sub(*facts.CheckedAt) > limit ||
		facts.UpstreamAt == nil || now.Sub(*facts.UpstreamAt) > limit
}

func localDate(at time.Time) string {
	return at.Format("2 Jan 2006")
}

func localDateTime(at time.Time) string {
	return at.In(sriLanka).Format("2 Jan 2006 15:04")
}

// DateInSriLanka is a time's local date as midnight UTC, the form report dates
// take.
func DateInSriLanka(at time.Time) time.Time {
	local := at.In(sriLanka)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// Bulletin is an official bulletin as the API returns it.
type Bulletin struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	TypeLabel string     `json:"typeLabel"`
	Title     string     `json:"title"`
	Severity  string     `json:"severity"`
	Colour    *string    `json:"colour"`
	MessageID *string    `json:"messageId"`
	IssuedAt  time.Time  `json:"issuedAt"`
	ValidFrom *time.Time `json:"validFrom"`
	// ValidUntil is the period the bulletin states; null if it states none.
	ValidUntil *time.Time `json:"validUntil"`
	// Current says this warning or advisory is in force: within its stated
	// period, or within 24 hours of issue if it states none.
	Current       bool   `json:"current"`
	PDFURL        string `json:"pdfUrl"`
	SourcePageURL string `json:"sourcePageUrl"`
	Language      string `json:"language"`
	// Text is an excerpt copied unchanged from the bulletin's English
	// sentences, or null when no clean text is available.
	Text       *string `json:"text"`
	TextStatus string  `json:"textStatus"`
	// Districts are the districts the clean text names by exact word.
	Districts  []DistrictRef `json:"districts"`
	Source     string        `json:"source"`
	SourceTime time.Time     `json:"sourceTime"`
	FetchedAt  time.Time     `json:"fetchedAt"`
}

// DistrictRef names a district.
type DistrictRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TypeLabels are the display names of the bulletin types.
var TypeLabels = map[string]string{
	string(dmcbulletins.TypeLandslide):  "Landslide report",
	string(dmcbulletins.TypeWaterLevel): "Water level and flood",
	string(dmcbulletins.TypeWeather):    "Weather",
	string(dmcbulletins.TypeSituation):  "Situation report",
}

// IsCurrent reports whether a warning or advisory is in force. Withdrawals
// and routine bulletins never are. A withdrawal does not end earlier warnings
// here: it lists basins that cannot be read reliably, so an earlier warning
// runs its period and the withdrawal is shown beside it.
func IsCurrent(severity string, issuedAt time.Time, validUntil *time.Time, now time.Time) bool {
	if severity != string(dmcbulletins.SeverityWarning) && severity != string(dmcbulletins.SeverityAdvisory) {
		return false
	}
	until := issuedAt.Add(defaultBulletinCurrency)
	if validUntil != nil {
		until = *validUntil
	}
	return !now.After(until)
}

var titleNoise = regexp.MustCompile(`(?i)\(?\s*tamil(\s+language)?\s*\)?|-\s*no\.?\s*\d+|\bno\.?\s*\d+\b|\(extended warning\)|[-–]`)

// sameWarning reduces a title to what stays the same across a warning's Tamil
// edition and its numbered updates.
func sameWarning(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(titleNoise.ReplaceAllString(title, " "))), " ")
}

// InForce keeps the warnings and advisories in force, newest first, one per
// warning: the latest update, in English when there is one.
func InForce(bulletins []Bulletin) []Bulletin {
	sorted := append([]Bulletin(nil), bulletins...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].IssuedAt.After(sorted[j].IssuedAt) })
	index := map[string]int{}
	out := []Bulletin{}
	for _, bulletin := range sorted {
		if !bulletin.Current {
			continue
		}
		key := bulletin.Type + "|" + sameWarning(bulletin.Title)
		if at, seen := index[key]; seen {
			if out[at].Language == "ta" && bulletin.Language != "ta" {
				out[at] = bulletin
			}
			continue
		}
		index[key] = len(out)
		out = append(out, bulletin)
	}
	return out
}
