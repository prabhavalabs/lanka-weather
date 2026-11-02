package dmcbulletins

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestDecodeIndexReadsRowsAsPublished(t *testing.T) {
	bulletins, skipped, err := DecodeIndex(read(t, "index_weather_forecasts.tsv"), TypeWeather)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(bulletins) != 5 {
		t.Fatalf("%d bulletins, %d skipped", len(bulletins), skipped)
	}
	first := bulletins[1]
	if first.DocID != "2026-11-02-16-00-weather-report" || first.Title != "Weather Report" || first.Type != TypeWeather {
		t.Fatalf("first = %+v", first)
	}
	if bulletins[0].Language != "ta" {
		t.Fatalf("the Tamil edition reads %q", bulletins[0].Language)
	}
	// 16:00 in Sri Lanka is 10:30 UTC.
	if want := time.Date(2026, 11, 2, 10, 30, 0, 0, time.UTC); !first.IssuedAt.Equal(want) {
		t.Fatalf("issued at %s, want %s", first.IssuedAt, want)
	}
	if !strings.HasPrefix(first.PDFURL, "https://www.dmc.gov.lk/images/dmcreports/") {
		t.Fatalf("pdf = %q", first.PDFURL)
	}
	if first.Language != "en" || first.Severity != SeverityRoutine {
		t.Fatalf("language %q, severity %q", first.Language, first.Severity)
	}
}

func TestDecodeIndexKeepsOddLinksOut(t *testing.T) {
	header := "doc_type\tdoc_id\tnum\tdate_str\tdescription\turl_metadata\tlang\turl_pdf\ttime_str\tut\n"
	row := func(pdf string) string {
		return "t\tdoc\tn\t2026-11-02\tTitle\thttps://www.dmc.gov.lk/page\ten\t" + pdf + "\t16:00\t1793615400.0\n"
	}
	bulletins, skipped, err := DecodeIndex([]byte(header+row("javascript:alert(1)")+row("https://www.dmc.gov.lk/a.pdf")), TypeWeather)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 || len(bulletins) != 1 {
		t.Fatalf("%d bulletins, %d skipped", len(bulletins), skipped)
	}
	if _, _, err := DecodeIndex([]byte("not,an,index\n"), TypeWeather); err == nil {
		t.Fatal("a file that is not an index must be an error")
	}
}

func TestLanguageTrustsTheTitleOverTheIndex(t *testing.T) {
	cases := map[string]string{
		"Weather Report (Tamil Language)":                  "ta",
		"Warning for strong winds and rough seas - Tamil":  "ta",
		"Advisory for strong winds and rough seas (Tamil)": "ta",
		"Weather Report": "en",
	}
	for title, want := range cases {
		if got := language(title, "en"); got != want {
			t.Errorf("language(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		kind   Type
		title  string
		colour string
		want   Severity
	}{
		{TypeWaterLevel, "Flood warning for Kelani River", "", SeverityWarning},
		{TypeWaterLevel, "Withdrawal of Flood warnings", "", SeverityWithdrawal},
		{TypeWaterLevel, "Water level & Rainfall", "", SeverityRoutine},
		{TypeWeather, "Advisory for Severe Lightning", "", SeverityAdvisory},
		{TypeWeather, "Warning for strong winds and rough seas", "Red", SeverityWarning},
		// The title undersells it; the bulletin's own colour does not.
		{TypeWeather, "Weather Advisory", "Red", SeverityWarning},
		{TypeWeather, "Weather Report", "", SeverityRoutine},
		// Seriousness of a landslide report is in each division's level.
		{TypeLandslide, "Landslide Early Warning", "", SeverityRoutine},
	}
	for _, c := range cases {
		if got := Classify(c.kind, c.title, c.colour); got != c.want {
			t.Errorf("Classify(%q, %q) = %q, want %q", c.title, c.colour, got, c.want)
		}
	}
}

func TestDecodeUpdatedAtIsSriLankaTime(t *testing.T) {
	at, err := DecodeUpdatedAt(read(t, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 11, 2, 18, 35, 41, 0, time.UTC); !at.Equal(want) {
		t.Fatalf("updated at %s, want %s", at, want)
	}
}

var issued = time.Date(2026, 11, 2, 5, 0, 0, 0, time.UTC)

func TestExtractKeepsCleanEnglishOfAWeatherReport(t *testing.T) {
	extract := ExtractText(string(read(t, "2026-11-02-16-00-weather-report.txt")), TypeWeather, issued)
	if extract.Status != TextClean {
		t.Fatalf("status = %q", extract.Status)
	}
	want := "Showers or thundershowers will occur at several places in North-central, North-western, Central, Uva, Sabaragamuwa, Western and Southern provinces and in Trincomalee district. Fairly heavy falls above 75 mm are likely at some places in these areas."
	if !strings.Contains(extract.Body, want) {
		t.Fatalf("body lost a sentence:\n%s", extract.Body)
	}
	// The city table names Kandy and Colombo; only the forecast's words count.
	if strings.Contains(extract.Body, "Anuradhapura 32 24") {
		t.Fatalf("body kept the city table:\n%s", extract.Body)
	}
	if !slices.Equal(extract.Districts, []string{"LK-52", "LK-53"}) {
		t.Fatalf("districts = %v", extract.Districts)
	}
	for _, line := range strings.Split(extract.Body, "\n\n") {
		if !isPlainText(line) {
			t.Fatalf("not plain text: %q", line)
		}
	}
}

func TestExtractKeepsAWarningVerbatimWithItsColourAndPeriod(t *testing.T) {
	extract := ExtractText(string(read(t, "2026-11-02-10-30-warning-for-stron-22c40aa8.txt")), TypeWeather, issued)
	if extract.Status != TextClean || extract.Colour != "Red" {
		t.Fatalf("status %q, colour %q", extract.Status, extract.Colour)
	}
	if !strings.Contains(extract.Body, "The Severe Cyclone Murasu is very likely to move northwestwards") {
		t.Fatalf("body = %s", extract.Body)
	}
	if extract.ValidUntil == nil || !extract.ValidUntil.Equal(issued.Add(24*time.Hour)) {
		t.Fatalf("valid until %v", extract.ValidUntil)
	}
	if extract.MessageID != "07 MSW /26/11/02/01" {
		t.Fatalf("message id = %q", extract.MessageID)
	}
	// "via Kankasanthurai" and "from Mannar to Trincomalee" name places on the coast.
	if !slices.Contains(extract.Districts, "LK-42") || !slices.Contains(extract.Districts, "LK-53") {
		t.Fatalf("districts = %v", extract.Districts)
	}
}

func TestExtractTakesTheBulletinsOwnColourNotALegend(t *testing.T) {
	// The advisory prints "COLOR: Amber" for itself, and further down a legend
	// listing "Color: -Red Warning (Take action)" and the rest.
	extract := ExtractText(string(read(t, "2026-10-30-11-30-advisory-for-seve-914a1dbb.txt")), TypeWeather, time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC))
	if extract.Colour != "Amber" {
		t.Fatalf("colour = %q", extract.Colour)
	}
	if got := statedColour("Color: -Red Warning (Take action)\n\nColor: -Green Threat is over"); got != "" {
		t.Fatalf("a legend gave a colour: %q", got)
	}
	want := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC) // 11.30 p.m. in Sri Lanka
	if extract.ValidUntil == nil || !extract.ValidUntil.Equal(want) {
		t.Fatalf("valid until %v, want %v", extract.ValidUntil, want)
	}
	// Spelled "Hambanthota" in the bulletin.
	for _, id := range []string{"LK-33", "LK-51", "LK-52"} {
		if !slices.Contains(extract.Districts, id) {
			t.Fatalf("districts = %v, missing %s", extract.Districts, id)
		}
	}
}

func TestExtractRefusesTextThatIsNotClean(t *testing.T) {
	cases := map[string]Type{
		// Tables run together.
		"2026-11-02-18-00-water-level.txt":      TypeWaterLevel,
		"2026-10-31-10-00-situation-report.txt": TypeSituation,
		// A Tamil edition.
		"2026-11-01-15-00-warning-for-stron-4af02833.txt": TypeWeather,
		// Every line broken into its own paragraph.
		"2026-10-26-03-30-advisory-for-heavy-rain.txt": TypeWeather,
	}
	for name, kind := range cases {
		extract := ExtractText(string(read(t, name)), kind, issued)
		if extract.Status != TextUnusable || extract.Body != "" || len(extract.Districts) != 0 {
			t.Errorf("%s: status %q, body %q, districts %v", name, extract.Status, extract.Body, extract.Districts)
		}
	}
}

func TestExtractLandslideReportKeepsItsPeriodAndNoBody(t *testing.T) {
	extract := ExtractText(string(read(t, "2026-10-31-16-00-landslide-ew-report.txt")), TypeLandslide, issued)
	if extract.Body != "" || extract.Status != TextUnusable {
		t.Fatalf("status %q, body %q", extract.Status, extract.Body)
	}
	if extract.MessageID != "LEWM-31-10-2026-1600-38" {
		t.Fatalf("message id = %q", extract.MessageID)
	}
	// 16:00 on 31 October to 16:00 on 1 November, Sri Lanka time.
	from, until := time.Date(2026, 10, 31, 10, 30, 0, 0, time.UTC), time.Date(2026, 11, 1, 10, 30, 0, 0, time.UTC)
	if extract.ValidFrom == nil || !extract.ValidFrom.Equal(from) || extract.ValidUntil == nil || !extract.ValidUntil.Equal(until) {
		t.Fatalf("valid %v to %v", extract.ValidFrom, extract.ValidUntil)
	}
}

func TestValidityReadsTheStatedForms(t *testing.T) {
	at := func(year int, month time.Month, day, hour, minute int) time.Time {
		return time.Date(year, month, day, hour, minute, 0, 0, sriLanka).UTC()
	}
	start := at(2026, 8, 29, 10, 30)
	cases := map[string]time.Time{
		"Valid till 9.30 a.m. on 23rd of October 2026":                                              at(2026, 10, 23, 9, 30),
		"Valid till 11.00 am on 23rd of October 2026":                                               at(2026, 10, 23, 11, 0),
		"Issued at 10.30 a.m. 29 August 2026 Valid for the period until 10.30 a.m. 30 August 2026.": at(2026, 8, 30, 10, 30),
		"valid for the period until 5.30 a.m. on 27 October 2026":                                   at(2026, 10, 27, 5, 30),
		"valid for the period until 11.30 p.m. 30 October 2026":                                     at(2026, 10, 30, 23, 30),
		"valid for the next 12 hours.":                                                              start.Add(12 * time.Hour),
	}
	for text, want := range cases {
		_, until := validity(text, start)
		if until == nil || !until.Equal(want) {
			t.Errorf("validity(%q) = %v, want %v", text, until, want)
		}
	}
	if _, until := validity("Issued at 04.00 p.m. on 31 October 2026, valid for 01 November 2026.", start); until != nil {
		t.Errorf("an unreadable period must give none, got %v", until)
	}
}

func TestNamedDistrictsNeedsWholeWords(t *testing.T) {
	got := NamedDistricts("Heavy rain in Kandy and Nuwara-Eliya, with Moneragala too. Gallery Road, Mannarama and colombo do not count.")
	if !slices.Equal(got, []string{"LK-21", "LK-23", "LK-82"}) {
		t.Fatalf("districts = %v", got)
	}
}
