package warnings

import (
	"strings"
	"testing"
	"time"
)

var (
	reportDate = time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	issued     = time.Date(2026, 10, 31, 10, 30, 0, 0, time.UTC) // 16:00 in Sri Lanka
	until      = time.Date(2026, 11, 1, 10, 30, 0, 0, time.UTC)
)

const staleLimit = 12 * time.Hour

func facts(now time.Time, checkedAgo, upstreamAgo time.Duration) ReportFacts {
	checked, upstream := now.Add(-checkedAgo), now.Add(-upstreamAgo)
	return ReportFacts{Date: reportDate, IssuedAt: &issued, ValidUntil: &until, CheckedAt: &checked, UpstreamAt: &upstream}
}

func TestAssessWithinValidityShowsLevels(t *testing.T) {
	now := until.Add(-time.Hour)
	got := Assess(facts(now, time.Hour, time.Hour), now, staleLimit)
	if got.Status != StatusActive || !got.ShowLevels || got.Stale || !got.ValidityStated {
		t.Fatalf("got %+v", got)
	}
}

func TestAssessJustAfterValidityKeepsLevelsAndSaysSo(t *testing.T) {
	now := until.Add(2 * time.Hour)
	got := Assess(facts(now, time.Hour, time.Hour), now, staleLimit)
	if got.Status != StatusLapsed || !got.ShowLevels || !strings.Contains(got.Note, "validity ended") {
		t.Fatalf("got %+v", got)
	}
}

func TestAssessLongAfterValiditySaysNoneAsFarAsWeKnow(t *testing.T) {
	now := until.Add(33 * time.Hour)
	got := Assess(facts(now, time.Hour, time.Hour), now, staleLimit)
	if got.Status != StatusNone || got.ShowLevels {
		t.Fatalf("got %+v", got)
	}
	if got.Note != "Last report 31 Oct 2026; none currently active as far as we know." {
		t.Fatalf("note = %q", got.Note)
	}
}

func TestAssessNeverImpliesCalmWhenTheFeedIsStale(t *testing.T) {
	now := until.Add(33 * time.Hour)
	cases := map[string]ReportFacts{
		"our fetches stopped":    facts(now, 20*time.Hour, time.Hour),
		"the upstream stopped":   facts(now, time.Hour, 20*time.Hour),
		"upstream never checked": {Date: reportDate, IssuedAt: &issued, ValidUntil: &until, CheckedAt: &now},
	}
	for name, input := range cases {
		got := Assess(input, now, staleLimit)
		if got.Status != StatusUnknown || !got.Stale || strings.Contains(got.Note, "as far as we know") {
			t.Errorf("%s: got %+v", name, got)
		}
		if !strings.Contains(got.Note, "cannot say") {
			t.Errorf("%s: note = %q", name, got.Note)
		}
	}
}

func TestAssessStaleFeedStillShowsAReportInForceAndSaysSo(t *testing.T) {
	now := until.Add(-time.Hour)
	got := Assess(facts(now, 20*time.Hour, time.Hour), now, staleLimit)
	if got.Status != StatusActive || !got.ShowLevels || !got.Stale || !strings.Contains(got.Note, "not been able to check") {
		t.Fatalf("got %+v", got)
	}
}

func TestAssessPendingWhenANewerReportHasNoLevelsYet(t *testing.T) {
	now := until.Add(5 * time.Hour)
	input := facts(now, time.Hour, time.Hour)
	newer := until
	input.NewerIssuedAt = &newer
	got := Assess(input, now, staleLimit)
	if got.Status != StatusPending || !strings.Contains(got.Note, "levels have not reached us") {
		t.Fatalf("got %+v", got)
	}
}

func TestAssessAssumesADayWithoutAStatedPeriod(t *testing.T) {
	now := time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC)
	checked := now
	got := Assess(ReportFacts{Date: reportDate, CheckedAt: &checked, UpstreamAt: &checked}, now, staleLimit)
	if got.ValidityStated || !got.IssuedAt.Equal(issued) || !got.ValidUntil.Equal(until) || got.Status != StatusActive {
		t.Fatalf("got %+v", got)
	}
}

func TestWordsForUsesNBROsWords(t *testing.T) {
	want := map[int]string{1: "Level 1 (Yellow)", 2: "Level 2 (Amber)", 3: "Level 3 (Red)"}
	actions := map[int]string{1: "Watch", 2: "Alert", 3: "Evacuate"}
	for level, label := range want {
		words := WordsFor(level)
		if words.Label != label || words.Action != actions[level] {
			t.Errorf("WordsFor(%d) = %+v", level, words)
		}
	}
}

func TestIsCurrent(t *testing.T) {
	at := time.Date(2026, 11, 2, 5, 0, 0, 0, time.UTC)
	stated := at.Add(6 * time.Hour)
	cases := []struct {
		name     string
		severity string
		until    *time.Time
		now      time.Time
		want     bool
	}{
		{"warning within its period", "warning", &stated, at.Add(time.Hour), true},
		{"warning past its period", "warning", &stated, at.Add(7 * time.Hour), false},
		{"advisory with no period, within a day", "advisory", nil, at.Add(23 * time.Hour), true},
		{"advisory with no period, after a day", "advisory", nil, at.Add(25 * time.Hour), false},
		{"routine report", "routine", nil, at, false},
		{"withdrawal", "withdrawal", nil, at, false},
	}
	for _, c := range cases {
		if got := IsCurrent(c.severity, at, c.until, c.now); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestInForceKeepsOnePerWarning(t *testing.T) {
	at := time.Date(2026, 11, 2, 5, 0, 0, 0, time.UTC)
	bulletin := func(id, title, language string, ago time.Duration) Bulletin {
		return Bulletin{ID: id, Type: "weather", Title: title, Language: language, IssuedAt: at.Add(-ago), Current: true}
	}
	list := []Bulletin{
		bulletin("tamil-new", "Warning for strong winds and rough seas (Tamil Language)", "ta", 0),
		bulletin("english", "Warning for strong winds and rough seas", "en", time.Hour),
		bulletin("older", "Warning for Strong Winds and Rough Seas", "en", 5*time.Hour),
		bulletin("flood-3", "Flood warning for Kalu River Basin - No 03 - (Extended warning)", "en", 2*time.Hour),
		bulletin("flood-1", "Flood warning for Kalu River Basin - No 01", "en", 9*time.Hour),
		bulletin("gin", "Flood Warning for Gin River Basin", "en", 3*time.Hour),
		{ID: "ended", Type: "weather", Title: "Advisory", Current: false},
	}
	got := InForce(list)
	ids := make([]string, len(got))
	for index, item := range got {
		ids[index] = item.ID
	}
	want := "english,flood-3,gin"
	if strings.Join(ids, ",") != want {
		t.Fatalf("in force = %v, want %s", ids, want)
	}
}

func TestBannerRaisedOnlyByLevelsTwoAndThreeAndByWarnings(t *testing.T) {
	date := "2026-10-31"
	division := func(id string, level int) DivisionFeature {
		return DivisionFeature{ID: id, Properties: DivisionAt{ID: id, LevelWords: WordsFor(level)}}
	}
	calm := loaded{show: true, meta: LandslideMeta{ReportDate: &date}, divisions: []DivisionFeature{division("LK-2136", 1)}}
	advisory := Bulletin{ID: "a", Title: "Advisory for Severe Lightning", Severity: "advisory", Current: true}
	if items := banner(calm, []Bulletin{advisory}); len(items) != 0 {
		t.Fatalf("level 1 and an advisory raised the banner: %+v", items)
	}

	serious := loaded{show: true, meta: LandslideMeta{ReportDate: &date}, divisions: []DivisionFeature{division("LK-2136", 3), division("LK-9206", 2), division("LK-9209", 2)}}
	warning := Bulletin{ID: "w", Title: "Flood warning for Kelani River", Severity: "warning", Current: true}
	items := banner(serious, []Bulletin{warning, advisory})
	if len(items) != 2 || items[0].Kind != "landslide" || items[0].Level != 3 || items[1].Title != "Flood warning for Kelani River" {
		t.Fatalf("items = %+v", items)
	}
	if items[0].Detail != "Level 3 (Red): 1 DS division; Level 2 (Amber): 2" {
		t.Fatalf("detail = %q", items[0].Detail)
	}

	// A report past showing raises nothing, whatever it said.
	serious.show = false
	if items := banner(serious, nil); len(items) != 0 {
		t.Fatalf("items = %+v", items)
	}
}

func TestHighlightsSeparateStructuredLevelsFromMentions(t *testing.T) {
	state := loaded{show: true, divisions: []DivisionFeature{
		{ID: "LK-2136", Properties: DivisionAt{DistrictID: "LK-21", District: "Kandy", LevelWords: WordsFor(1)}},
		{ID: "LK-2139", Properties: DivisionAt{DistrictID: "LK-21", District: "Kandy", LevelWords: WordsFor(2)}},
	}}
	inForce := []Bulletin{{ID: "w", Title: "Warning", Districts: []DistrictRef{{ID: "LK-21", Name: "Kandy"}, {ID: "LK-42", Name: "Mannar"}}}}
	got := highlights(state, inForce)
	if len(got) != 2 || got[0].ID != "LK-21" || got[0].LandslideLevel != 2 || len(got[0].Mentioned) != 1 {
		t.Fatalf("got %+v", got)
	}
	if got[1].ID != "LK-42" || got[1].LandslideLevel != 0 || got[1].Mentioned[0].ID != "w" {
		t.Fatalf("got %+v", got)
	}
}
