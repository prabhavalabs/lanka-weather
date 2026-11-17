package dmcbulletins

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestSiteDocIDMatchesTheIndex(t *testing.T) {
	for _, test := range []struct{ date, clock, title, want string }{
		// Identifiers the index gave these bulletins (production, November 2026).
		{"2026-11-11", "22:00", "Flood Warning for Kuda Ganga SubBasin Of Kalu River - NO 01 (2026.11.11)", "2026-11-11-22-00-flood-warning-for-ff46bb6a"},
		{"2026-11-02", "18:00", "Water Level", "2026-11-02-18-00-water-level"},
		{"2026-10-31", "16:00", "Landslide EW Report", "2026-10-31-16-00-landslide-ew-report"},
	} {
		if got := SiteDocID(test.date, test.clock, test.title); got != test.want {
			t.Errorf("%q: %q, want %q", test.title, got, test.want)
		}
	}
}

func TestDecodeSitePage(t *testing.T) {
	page, err := os.ReadFile("testdata/site_water_level.html")
	if err != nil {
		t.Fatal(err)
	}
	pageURL := SitePageURL("https://www.dmc.gov.lk", TypeWaterLevel)
	bulletins, err := DecodeSitePage(page, TypeWaterLevel, "https://www.dmc.gov.lk", pageURL)
	if err != nil {
		t.Fatal(err)
	}
	if len(bulletins) != 5 {
		t.Fatalf("four reports and a warning, not the image: %d", len(bulletins))
	}
	warning := bulletins[4]
	if warning.DocID != "2026-11-11-22-00-flood-warning-for-ff46bb6a" || warning.Severity != SeverityWarning ||
		!warning.IssuedAt.Equal(time.Date(2026, 11, 11, 16, 30, 0, 0, time.UTC)) ||
		warning.PDFURL != "https://www.dmc.gov.lk/images/dmcreports/Flood_Warning_for_Kuda_Ganga_2026__1794424000.pdf" ||
		warning.SourcePageURL != pageURL || warning.Language != "en" {
		t.Fatalf("warning %+v", warning)
	}
	if bulletins[0].Severity != SeverityRoutine || !strings.HasPrefix(bulletins[0].PDFURL, "https://www.dmc.gov.lk/images/") {
		t.Fatalf("report %+v", bulletins[0])
	}
	if _, err := DecodeSitePage([]byte("<html>maintenance</html>"), TypeWaterLevel, "https://www.dmc.gov.lk", pageURL); err == nil {
		t.Fatal("a page without the table is an error")
	}
}

func TestJoinPDFBlocksAndLandslideDistricts(t *testing.T) {
	raw := "Landslide Early Warning\nMessage ID: LEWM-14-11-2026-1600-09\n\n\fLocations for Potential Risk\n District  Level 1 (Yellow)\n" +
		"Badulla\n Divisional Secretariat\n\nKandy  Divisional\nSecretariat\n\nNuwara Eliya Divisional\n\nNote: shifted one level up\n\n" +
		"Areas and places which need special attention near Colombo\n"
	joined := JoinPDFBlocks(raw)
	if strings.Contains(joined, "\f") || !strings.Contains(joined, "Kandy Divisional Secretariat") {
		t.Fatalf("joined %q", joined)
	}
	if got := LandslideDistricts(joined); !slices.Equal(got, []string{"LK-21", "LK-23", "LK-81"}) {
		t.Fatalf("districts %v (Colombo is after the table)", got)
	}
	extract := ExtractText(joined, TypeLandslide, time.Date(2026, 11, 14, 10, 30, 0, 0, time.UTC))
	if extract.MessageID != "LEWM-14-11-2026-1600-09" || len(extract.Districts) != 3 || extract.Body != "" {
		t.Fatalf("extract %+v", extract)
	}
}
