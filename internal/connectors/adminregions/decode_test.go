package adminregions

import (
	"os"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/regions"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestDecodeLevelReadsRegions(t *testing.T) {
	decoded, skipped, err := DecodeLevel(regions.LevelGN, read(t, "gnds.topojson"))
	if err != nil {
		t.Fatal(err)
	}
	// Skipped: a shape with only a humdata id, a district code in the GN
	// file, and a feature without geometry.
	if len(decoded) != 3 || skipped != 3 {
		t.Fatalf("decoded %d, skipped %d; want 3 and 3", len(decoded), skipped)
	}
	byID := map[string]Region{}
	for _, region := range decoded {
		byID[region.ID] = region
	}

	west := byID["LK-1103005"]
	if west.Name != "Sammanthranapura" || west.NameSI != "සම්මන්ත්‍රණපුර" || west.NameTA != "" {
		t.Fatalf("names = %q %q %q", west.Name, west.NameSI, west.NameTA)
	}
	if west.FID != 1103005 || west.ParentID != "LK-1103" || west.ProvinceID != "LK-1" ||
		west.DistrictID != "LK-11" || west.DSID != "LK-1103" || west.ISOCode != "" {
		t.Fatalf("hierarchy = %+v", west)
	}
	if west.GNNumber != "" || west.AreaKM2 != 242.5 || west.CenterLon != 79.85 || west.CenterLat != 6.9 {
		t.Fatalf("attributes = number %q, area %v, centre %v %v", west.GNNumber, west.AreaKM2, west.CenterLon, west.CenterLat)
	}
	if box := west.Shape.Bounds(); box.West != 79.8 || box.North != 7.0 {
		t.Fatalf("bounds = %+v", box)
	}
	if byID["LK-1103010"].GNNumber != "520A" {
		t.Fatalf("GN number = %q", byID["LK-1103010"].GNNumber)
	}

	// A region without a published centre or area gets the middle of its box
	// and leaves the area to the database.
	kandy := byID["LK-2130120"]
	if kandy.AreaKM2 != 0 || kandy.CenterLon < 80.649 || kandy.CenterLon > 80.651 || kandy.CenterLat < 7.299 || kandy.CenterLat > 7.301 {
		t.Fatalf("fallback centre = %v %v, area %v", kandy.CenterLon, kandy.CenterLat, kandy.AreaKM2)
	}
	if kandy.NameSI != "මහනුවර" || kandy.NameTA != "மகாநுவர" {
		t.Fatalf("other names = %q %q", kandy.NameSI, kandy.NameTA)
	}
}

func TestDecodeLevelProvinces(t *testing.T) {
	decoded, _, err := DecodeLevel(regions.LevelProvince, read(t, "provinces.topojson"))
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 || decoded[0].ID != "LK-1" || decoded[0].ParentID != "" || decoded[0].ISOCode != "LK-1" {
		t.Fatalf("provinces = %+v", decoded)
	}
	if decoded[1].NameSI != "මධ්‍යම පළාත" || decoded[1].NameTA != "மத்திய மாகாணம்" {
		t.Fatalf("names = %q %q", decoded[1].NameSI, decoded[1].NameTA)
	}
}

func TestDecodeLevelRejectsOtherFiles(t *testing.T) {
	if _, _, err := DecodeLevel(regions.LevelGN, []byte(`{"type": "FeatureCollection"}`)); err == nil {
		t.Fatal("expected an error for GeoJSON")
	}
	if _, _, err := DecodeLevel(regions.LevelGN, []byte(`{"type": "Topology", "objects": {}, "arcs": []}`)); err == nil {
		t.Fatal("expected an error for a topology without objects")
	}
}

func TestSplitOtherNames(t *testing.T) {
	cases := []struct{ raw, si, ta string }{
		{"கொழும்பு,කොළඹ", "කොළඹ", "கொழும்பு"},
		{"සම්මන්ත්‍රණපුර", "සම්මන්ත්‍රණපුර", ""},
		{"Acres  20,40,60,අක්කර විස්ස,අක්කර", "අක්කර විස්ස, අක්කර", ""},
		{"Akkaraipattu,8/1,අක්කරපත්තු 8/1", "අක්කරපත්තු 8/1", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		si, ta := SplitOtherNames(c.raw)
		if si != c.si || ta != c.ta {
			t.Errorf("SplitOtherNames(%q) = %q, %q; want %q, %q", c.raw, si, ta, c.si, c.ta)
		}
	}
}

func TestDecodeCensus(t *testing.T) {
	population, err := DecodeCensus(read(t, "census.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	// Country and local-authority rows are not regions; a row with an
	// unreadable count is skipped.
	if len(population) != 8 {
		t.Fatalf("regions = %d, want 8: %v", len(population), population)
	}
	if got := population["LK-21"]; got.Total != 1461895 || got.Year != 2024 {
		t.Fatalf("Kandy = %+v", got)
	}
	if _, ok := population["LK-2130120"]; ok {
		t.Fatal("a row with an unreadable count was kept")
	}
	if _, err := DecodeCensus([]byte("id\tname\n")); err == nil {
		t.Fatal("expected an error for a table without the needed columns")
	}
}

func TestDecodeCommits(t *testing.T) {
	commit, err := DecodeCommits(read(t, "boundary_commits.json"))
	if err != nil {
		t.Fatal(err)
	}
	if commit.SHA != "13e6784dce0ec813c3fd5db3886a730af82cf887" || !commit.Time.Equal(time.Date(2026, 6, 23, 9, 8, 39, 0, time.UTC)) {
		t.Fatalf("commit = %+v", commit)
	}
	for _, bad := range []string{`[]`, `{"message": "API rate limit exceeded"}`, `[{"sha": "abc"}]`} {
		if _, err := DecodeCommits([]byte(bad)); err == nil {
			t.Errorf("DecodeCommits(%s) succeeded", bad)
		}
	}
}

func TestDropOrphans(t *testing.T) {
	all := []Region{{ID: "LK-1"}, {ID: "LK-11", ParentID: "LK-1"}}
	for i := range 200 {
		all = append(all, Region{ID: "LK-1101" + string(rune('0'+i%10)), ParentID: "LK-11"})
	}
	all = append(all, Region{ID: "LK-1199", ParentID: "LK-19"})
	kept, err := dropOrphans(all)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != len(all)-1 {
		t.Fatalf("kept %d of %d", len(kept), len(all))
	}
	if _, err := dropOrphans([]Region{{ID: "LK-11", ParentID: "LK-1"}}); err == nil {
		t.Fatal("expected an error when most regions are orphans")
	}
}
