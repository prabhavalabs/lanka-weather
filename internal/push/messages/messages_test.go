package messages

import (
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

type names map[string][3]string

func (table names) District(id, locale string) string {
	entry, ok := table[id]
	if !ok {
		return id
	}
	switch locale {
	case "si":
		return entry[1]
	case "ta":
		return entry[2]
	}
	return entry[0]
}

var testNames = names{"LK-91": {"Ratnapura", "රත්නපුර", "இரத்தினபுரி"}, "LK-92": {"Kegalle", "කෑගල්ල", "கேகாலை"}}

// keysFor lists the templates a kind needs: one per variant, and a gathered
// form unless the kind is always single.
func keysFor(kind string) [][]string {
	out := [][]string{}
	for _, variant := range Variants[kind] {
		out = append(out, []string{kind + "." + variant})
		if !Single(kind) {
			out = append(out, []string{kind + "." + variant + ".many", kind + ".many"})
		}
	}
	return out
}

func TestEveryKindHasTextInEveryLanguage(t *testing.T) {
	for kind := range Variants {
		for _, alternatives := range keysFor(kind) {
			for _, locale := range Locales {
				found := false
				for _, key := range alternatives {
					if template, ok := Templates(locale)[key]; ok {
						found = true
						if strings.TrimSpace(template.Title) == "" || strings.TrimSpace(template.Body) == "" {
							t.Errorf("%s %s is empty", locale, key)
						}
					}
				}
				if !found {
					t.Errorf("%s has no template for %v", locale, alternatives)
				}
			}
		}
	}
}

func TestTranslationsKeepTheEnglishPlaceholdersAndKeys(t *testing.T) {
	english := Templates("en")
	for _, locale := range []string{"si", "ta"} {
		catalogue := Templates(locale)
		for key, source := range english {
			translated, ok := catalogue[key]
			if !ok {
				t.Errorf("%s lacks %s", locale, key)
				continue
			}
			for _, part := range []struct{ en, other string }{{source.Title, translated.Title}, {source.Body, translated.Body}} {
				want, got := Placeholders(part.en), Placeholders(part.other)
				slices.Sort(want)
				slices.Sort(got)
				if !slices.Equal(want, got) {
					t.Errorf("%s %s placeholders %v, English %v", locale, key, got, want)
				}
			}
		}
		for key := range catalogue {
			if _, ok := english[key]; !ok {
				t.Errorf("%s has %s, which English lacks", locale, key)
			}
		}
		for key := range Words("en") {
			if strings.TrimSpace(Words(locale)[key]) == "" {
				t.Errorf("%s lacks the word %s", locale, key)
			}
		}
	}
}

func TestEveryNoticeRendersCompletelyInEveryLanguage(t *testing.T) {
	facts := map[string]string{"district": "LK-91", "districts": "LK-91,LK-92", "station": "Ratnapura", "river": "Kalu Ganga",
		"site": "Putupaula", "name": "MONTHA", "title": "Flood warning for Kalu River", "place": "Sumatra, Indonesia",
		"magnitude": "6.8", "distance": "1450", "source": "irrigation", "level": "3", "issuedAt": "2026-11-06T14:45:58Z",
		"gauge": "DOM HEAD OFFICE", "mm": "67", "window": "1h"}
	for kind, variants := range Variants {
		for _, variant := range variants {
			for _, count := range []int{1, 3} {
				for _, locale := range Locales {
					title, body := Render(locale, Input{Kind: kind, Variant: variant, Count: count, Facts: facts}, testNames)
					for _, text := range []string{title, body} {
						if strings.ContainsAny(text, "{}") || strings.TrimSpace(text) == "" {
							t.Errorf("%s %s.%s ×%d: %q", locale, kind, variant, count, text)
						}
					}
					if utf8.RuneCountInString(title) > maxTitle || utf8.RuneCountInString(body) > maxBody {
						t.Errorf("%s %s.%s too long: %q / %q", locale, kind, variant, title, body)
					}
				}
			}
		}
	}
}

func TestRenderUsesTheSubscribersLanguageAndOfficialWords(t *testing.T) {
	title, body := Render("en", Input{Kind: KindLandslide, Variant: "3", Count: 1, Facts: map[string]string{"district": "LK-91", "level": "3"}}, testNames)
	if title != "Landslide Level 3 · Ratnapura" || !strings.HasPrefix(body, "NBRO: evacuate") {
		t.Fatalf("en: %q / %q", title, body)
	}
	title, _ = Render("si", Input{Kind: KindLandslide, Variant: "3", Count: 1, Facts: map[string]string{"district": "LK-91"}}, testNames)
	if !strings.Contains(title, "රත්නපුර") {
		t.Fatalf("si title %q", title)
	}
	title, _ = Render("ta", Input{Kind: KindLandslide, Variant: "2", Count: 3, Facts: map[string]string{"district": "LK-92"}}, testNames)
	if title != "மண்சரிவு நிலை 2 · 3 மாவட்டங்கள்" {
		t.Fatalf("ta title %q", title)
	}
	_, body = Render("xx", Input{Kind: KindRiver, Variant: "major_flood", Count: 1, Facts: map[string]string{"district": "LK-91", "station": "Ratnapura", "source": "community"}}, testNames)
	if !strings.Contains(body, "not official") {
		t.Fatalf("unknown language falls back to English with the source: %q", body)
	}
}

func TestForecastsAndHazardsAreNeverWordedAsOfficialWarnings(t *testing.T) {
	for _, locale := range Locales {
		for key, template := range Templates(locale) {
			kind := strings.Split(key, ".")[0]
			switch kind {
			case KindFloodForecast, KindFlashFlood:
				if !strings.Contains(template.Title+template.Body, map[string]string{"en": "estimate", "si": "ඇස්තමේන්තු", "ta": "மதிப்பீடு"}[locale]) {
					t.Errorf("%s %s does not say it is an estimate", locale, key)
				}
			case KindEarthquake:
				if key == "earthquake.plain" {
					continue
				}
				if !strings.Contains(template.Body, map[string]string{"en": "not a tsunami warning", "si": "සුනාමි අනතුරු ඇඟවීමක් නොවේ", "ta": "சுனாமி எச்சரிக்கை அல்ல"}[locale]) {
					t.Errorf("%s %s must say it is not a tsunami warning", locale, key)
				}
			}
			for _, banned := range []string{"safe", "all clear"} {
				if strings.Contains(strings.ToLower(template.Body), banned) {
					t.Errorf("%s %s says %q", locale, key, banned)
				}
			}
		}
	}
}

func TestGroupedNumbers(t *testing.T) {
	for value, want := range map[int]string{5: "5", 999: "999", 1000: "1,000", 1450: "1,450", 1234567: "1,234,567"} {
		if got := grouped(value); got != want {
			t.Errorf("grouped(%d) = %q", value, got)
		}
	}
}

func TestFlashFloodsSayWhenGoogleIssuedThem(t *testing.T) {
	// 20:15 in Sri Lanka on 6 November 2026, written at 00:15 the next day.
	now := time.Date(2026, 11, 6, 18, 45, 0, 0, time.UTC)
	facts := map[string]string{"district": "LK-91", "districts": "LK-91", "issuedAt": "2026-11-06T14:45:58Z"}
	_, body := Render("en", Input{Kind: KindFlashFlood, Variant: "likely", Count: 1, Facts: facts, Now: now}, testNames)
	if !strings.Contains(body, "Google issued it at 20:15 yesterday.") {
		t.Fatalf("body %q", body)
	}
	_, body = Render("en", Input{Kind: KindFlashFlood, Variant: "likely", Count: 1, Facts: facts, Now: now.Add(-3 * time.Hour)}, testNames)
	if !strings.Contains(body, "issued it at 20:15.") {
		t.Fatalf("same day %q", body)
	}
	_, body = Render("en", Input{Kind: KindFlashFlood, Variant: "likely", Count: 1, Facts: facts, Now: now.Add(48 * time.Hour)}, testNames)
	if !strings.Contains(body, "issued it at 6/11 20:15.") {
		t.Fatalf("older %q", body)
	}
	for _, locale := range []string{"si", "ta"} {
		if _, body := Render(locale, Input{Kind: KindFlashFlood, Variant: "likely", Count: 1, Facts: facts, Now: now}, testNames); !strings.Contains(body, "20:15") {
			t.Errorf("%s body %q", locale, body)
		}
	}
}
