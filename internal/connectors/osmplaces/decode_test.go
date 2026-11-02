package osmplaces

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestDecodeReadsPlaces(t *testing.T) {
	payload, err := os.ReadFile("testdata/overpass.json")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	// Skipped: a node without a name, an unknown place kind, a town in
	// India and a way.
	if len(snapshot.Places) != 8 || snapshot.Skipped != 4 {
		t.Fatalf("places %d, skipped %d; want 8 and 4", len(snapshot.Places), snapshot.Skipped)
	}
	if !snapshot.SourceTime.Equal(time.Date(2026, 11, 2, 13, 47, 51, 0, time.UTC)) {
		t.Fatalf("source time = %v", snapshot.SourceTime)
	}
	byID := map[string]Place{}
	for _, place := range snapshot.Places {
		byID[place.ID] = place
	}

	kandy := byID["osm-node-2908772560"]
	if kandy.Name != "Kandy" || kandy.NameSI != "මහනුවර" || kandy.NameTA != "கண்டி" || kandy.Kind != "city" {
		t.Fatalf("Kandy = %+v", kandy)
	}
	if kandy.Population == nil || *kandy.Population != 125000 {
		t.Fatalf("Kandy population = %v", kandy.Population)
	}
	if strings.Join(kandy.AltNames, "|") != "Senkadagala|Maha Nuwara" {
		t.Fatalf("Kandy alternative names = %q", kandy.AltNames)
	}
	if colombo := byID["osm-node-50794342"]; colombo.Population == nil || *colombo.Population != 753000 {
		t.Fatalf("Colombo population = %v", colombo.Population)
	}

	// A name only in Sinhala is the display name and the Sinhala name.
	village := byID["osm-node-4455834909"]
	if village.Name != "මට්ටක්කුලිය" || village.NameSI != "මට්ටක්කුලිය" || len(village.AltNames) != 0 || village.Population != nil {
		t.Fatalf("Sinhala-only village = %+v", village)
	}
}

func TestDecodeRejectsIncompleteResponses(t *testing.T) {
	cases := []string{
		`{"osm3s": {"timestamp_osm_base": "2026-11-02T13:47:51Z"}, "remark": "runtime error: Query timed out", "elements": []}`,
		`{"elements": []}`,
		`<html>busy</html>`,
	}
	for _, payload := range cases {
		if _, err := Decode([]byte(payload)); err == nil {
			t.Errorf("Decode(%s) succeeded", payload)
		}
	}
}

func TestParsePopulation(t *testing.T) {
	cases := map[string]int32{"125000": 125000, "753,000": 753000, "~15000": 15000, " 1 234 ": 1234}
	for raw, want := range cases {
		if got := ParsePopulation(raw); got == nil || *got != want {
			t.Errorf("ParsePopulation(%q) = %v, want %d", raw, got, want)
		}
	}
	for _, bad := range []string{"", "about 7000", "5000-6000", "0", "-3", "99999999999"} {
		if got := ParsePopulation(bad); got != nil {
			t.Errorf("ParsePopulation(%q) = %d, want nil", bad, *got)
		}
	}
}

func TestQuerySelectsPlaceNodesInTheBox(t *testing.T) {
	for _, part := range []string{`[out:json]`, `node[place~"^(city|town|`, `isolated_dwelling)$"][name]`, `(5.85,79.5,9.9,81.95)`} {
		if !strings.Contains(Query, part) {
			t.Errorf("query lacks %q:\n%s", part, Query)
		}
	}
}
