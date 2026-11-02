package landslides

import (
	"os"
	"testing"
)

func TestDecodeReadsReportsNewestFirst(t *testing.T) {
	payload, err := os.ReadFile("testdata/all.json")
	if err != nil {
		t.Fatal(err)
	}
	reports, skipped, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(reports) != 3 {
		t.Fatalf("%d reports, %d skipped", len(reports), skipped)
	}
	latest := reports[0]
	if latest.Date.Format("2006-01-02") != "2026-10-31" || len(latest.Levels) != 3 {
		t.Fatalf("latest = %+v", latest)
	}
	want := map[string]Level{
		"LK-2136": {DS: "LK-2136", District: "LK-21", Level: 1},
		"LK-9206": {DS: "LK-9206", District: "LK-92", Level: 1},
		"LK-9209": {DS: "LK-9209", District: "LK-92", Level: 1},
	}
	for _, level := range latest.Levels {
		if want[level.DS] != level {
			t.Errorf("level = %+v", level)
		}
	}
	// The oldest report in the feed has no levels at all.
	if oldest := reports[len(reports)-1]; len(oldest.Levels) != 0 {
		t.Fatalf("oldest = %+v", oldest)
	}
}

func TestDecodeSkipsWhatItCannotRead(t *testing.T) {
	payload := []byte(`[
	  {"date_id": "2026-10-31", "level_to_district_to_dsds": {
	    "1": {"LK-21": ["LK-2136", "not-a-code", "LK-21"]},
	    "2": {"LK-21": ["LK-2136"]},
	    "9": {"LK-21": ["LK-2139"]}
	  }},
	  {"date_id": "yesterday", "level_to_district_to_dsds": {}}
	]`)
	reports, skipped, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	// A bad code, a district code where a DS code belongs, level 9 and a bad date.
	if skipped != 4 || len(reports) != 1 || len(reports[0].Levels) != 1 {
		t.Fatalf("%d skipped, reports %+v", skipped, reports)
	}
	// Named at two levels, the division takes the higher.
	if reports[0].Levels[0].Level != 2 {
		t.Fatalf("level = %+v", reports[0].Levels[0])
	}
}

func TestDecodeRejectsEmptyOrBrokenFiles(t *testing.T) {
	for _, payload := range []string{`[]`, `{"oops":1}`, `not json`} {
		if _, _, err := Decode([]byte(payload)); err == nil {
			t.Errorf("Decode(%q) should fail", payload)
		}
	}
}
