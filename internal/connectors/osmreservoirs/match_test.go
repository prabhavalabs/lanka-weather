package osmreservoirs

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestDecode(t *testing.T) {
	payload, err := os.ReadFile("testdata/overpass.json")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.SourceTime.Equal(time.Date(2026, 11, 2, 19, 46, 36, 0, time.UTC)) {
		t.Fatalf("source time = %v", snapshot.SourceTime)
	}
	if snapshot.Elements < 5 {
		t.Fatalf("elements = %d", snapshot.Elements)
	}
	kinds := map[string]string{}
	shapes := map[string]string{}
	for _, candidate := range snapshot.Candidates {
		kinds[candidate.Name] = candidate.Kind
		shapes[candidate.Name] = candidate.Lines
	}
	if kinds["Rantembe Dam"] != KindDam || kinds["Polgolla Barrage"] != KindDam {
		t.Fatalf("dams were not recognised: %v", kinds)
	}
	if kinds["Dhoby Tank"] != KindWater || !strings.Contains(shapes["Dhoby Tank"], `"LineString"`) {
		t.Fatalf("Dhoby Tank = %q, %s", kinds["Dhoby Tank"], shapes["Dhoby Tank"])
	}
	// A lagoon is never a reservoir, whatever it is called.
	if _, found := kinds["Andarakala Lagoon"]; found {
		t.Fatalf("a lagoon became a candidate: %v", kinds)
	}
	relations := 0
	for _, candidate := range snapshot.Candidates {
		if strings.HasPrefix(candidate.OSMID, "relation/") {
			relations++
			if !strings.Contains(candidate.Lines, `"MultiLineString"`) {
				t.Fatalf("relation %s has shape %s", candidate.OSMID, candidate.Lines)
			}
		}
	}
	if relations == 0 {
		t.Fatal("fixture has no relation")
	}
}

func TestDecodeRejectsIncompleteAnswers(t *testing.T) {
	if _, err := Decode([]byte(`{"remark":"runtime error: Query timed out","osm3s":{"timestamp_osm_base":"2026-11-02T19:46:36Z"},"elements":[]}`)); err == nil {
		t.Fatal("a timed-out query was accepted")
	}
	if _, err := Decode([]byte(`{"elements":[]}`)); err == nil {
		t.Fatal("a response without a timestamp was accepted")
	}
}

func TestKeyFoldsSpellingsNotMeanings(t *testing.T) {
	same := [][2]string{
		{"Kala Wewa", "Kalawewa tank"},
		{"Thissa Wewa", "Tissa Wewa"},
		{"Basavakkulama", "Basawakkulama"},
		{"Pavatkulam", "Pavat Kulam"},
		{"Yudhaganawa", "Yudaganawa_Wewa"},
		{"Ambewela", "Ambewela Rerservoir"},
		{"Giants Tank", "Giant's Tank"},
		{"Dayaraba", "dayaraaba lake"},
		{"Kalugaloya", "Kalugaloya Reservoir"},
	}
	for _, pair := range same {
		if Key(pair[0]) != Key(pair[1]) {
			t.Errorf("Key(%q) = %q and Key(%q) = %q should match", pair[0], Key(pair[0]), pair[1], Key(pair[1]))
		}
	}
	different := [][2]string{
		{"Debarawewa", "Debara Ara"},
		{"Kala Wewa", "Kala Oya"},
		{"Maha Aluthgamara", "Maha Wewa"},
		{"Minneriya", "Giritale"},
	}
	for _, pair := range different {
		if Key(pair[0]) == Key(pair[1]) {
			t.Errorf("Key(%q) and Key(%q) are both %q", pair[0], pair[1], Key(pair[0]))
		}
	}
	for _, generic := range []string{"Wewa", "Tank", "Reservoir", "வவு"} {
		if key := Key(generic); key != "" {
			t.Errorf("Key(%q) = %q, want none", generic, key)
		}
	}
}

func district(name string) *string { return &name }

func water(id, name string, area float64, lon, lat float64, districts ...string) Measured {
	return Measured{OSMID: id, Name: name, Kind: KindWater, Names: []string{name}, Lon: lon, Lat: lat, AreaHa: area, Districts: districts, NearDistricts: districts}
}

func TestMatchNeverConfusesSameNamedTanksInOtherDistricts(t *testing.T) {
	results := Match([]Target{
		{ID: "thissa-wewa", Name: "Thissa Wewa", District: district("Anuradhapura"), Kind: "major"},
		{ID: "tissa-wewa", Name: "Tissa Wewa", District: district("Hambantota"), Kind: "major"},
		{ID: "jayawewa", Name: "Jayawewa", District: district("Kurunegala"), Kind: "major"},
	}, []Measured{
		water("way/1", "Tissa Wewa", 247, 80.379, 8.335, "Anuradhapura"),
		water("way/2", "Tissa Wewa", 160, 81.285, 6.291, "Hambantota"),
		water("way/3", "Jaya Wewa", 50, 79.886, 8.172, "Puttalam"),
	})
	if results[0].Candidate == nil || results[0].Candidate.OSMID != "way/1" {
		t.Fatalf("Thissa Wewa = %+v", results[0])
	}
	if results[1].Candidate == nil || results[1].Candidate.OSMID != "way/2" {
		t.Fatalf("Tissa Wewa = %+v", results[1])
	}
	if results[2].Candidate != nil || results[2].Reason != ReasonWrongDistrict {
		t.Fatalf("a tank in another district was taken: %+v", results[2])
	}
}

func TestMatchIsConservative(t *testing.T) {
	cases := []struct {
		name       string
		target     Target
		candidates []Measured
		wantID     string
		wantReason string
		wantNear   bool
	}{
		{
			name:       "nothing with the name",
			target:     Target{ID: "x", Name: "Ellewela", District: district("Matara"), Kind: "major"},
			candidates: []Measured{water("way/1", "Kekanadura", 50, 80, 6, "Matara")},
			wantReason: ReasonNoNameMatch,
		},
		{
			name:       "no district known",
			target:     Target{ID: "unlisted-mahaweli", Name: "Dambu", Kind: "mahaweli"},
			candidates: []Measured{water("way/1", "Dambu Wewa", 50, 80, 8, "Matale")},
			wantReason: ReasonNoDistrict,
		},
		{
			name:       "two tanks of a similar size far apart",
			target:     Target{ID: "x", Name: "Ihala Wewa", District: district("Kurunegala"), Kind: "major"},
			candidates: []Measured{water("way/1", "Ihala Wewa", 50, 80.1, 7.5, "Kurunegala"), water("way/2", "Ihala Wewa", 40, 80.4, 7.9, "Kurunegala")},
			wantReason: ReasonAmbiguous,
		},
		{
			name:       "a much larger tank wins over a pond",
			target:     Target{ID: "x", Name: "Ihala Wewa", District: district("Kurunegala"), Kind: "major"},
			candidates: []Measured{water("way/1", "Ihala Wewa", 500, 80.1, 7.5, "Kurunegala"), water("way/2", "Ihala Wewa", 12, 80.4, 7.9, "Kurunegala")},
			wantID:     "way/1",
		},
		{
			name:       "two parts of one water body",
			target:     Target{ID: "x", Name: "Beira", District: district("Colombo"), Kind: "major"},
			candidates: []Measured{water("way/1", "Beira Lake", 50, 79.85, 6.92, "Colombo"), water("way/2", "Beira Lake", 45, 79.86, 6.93, "Colombo")},
			wantID:     "way/1",
		},
		{
			name:       "too small to be the scheme",
			target:     Target{ID: "x", Name: "Kande Ela", District: district("Nuwara Eliya"), Kind: "major"},
			candidates: []Measured{water("way/1", "Kande Ela Pond", 1, 80.79, 6.91, "Nuwara Eliya"), water("way/2", "Kande Ela", 1, 80.79, 6.92, "Nuwara Eliya")},
			wantReason: ReasonTooSmall,
		},
		{
			name:   "a dam marks a reservoir with no named water",
			target: Target{ID: "rantambe", Name: "Rantambe", Kind: "mahaweli"},
			candidates: []Measured{{OSMID: "way/9", Name: "Rantembe Dam", Kind: KindDam, Names: []string{"Rantembe Dam"},
				Districts: []string{"Kandy"}, NearDistricts: []string{"Kandy"}}},
			wantID: "way/9",
		},
		{
			name:   "water wins over a dam",
			target: Target{ID: "rantambe", Name: "Rantambe", Kind: "mahaweli"},
			candidates: []Measured{{OSMID: "way/9", Name: "Rantembe Dam", Kind: KindDam, Names: []string{"Rantembe Dam"}, Districts: []string{"Kandy"}},
				water("way/1", "Rantambe Reservoir", 200, 80.9, 7.2, "Kandy")},
			wantID: "way/1",
		},
		{
			name:   "just over the line, and the only one in the island",
			target: Target{ID: "x", Name: "Mau Ara", District: district("Hambantota"), Kind: "major"},
			candidates: []Measured{{OSMID: "way/1", Name: "Mau Ara", Kind: KindWater, Names: []string{"Mau Ara"}, AreaHa: 800,
				Districts: []string{"Monaragala"}, NearDistricts: []string{"Hambantota", "Monaragala"}}},
			wantID:   "way/1",
			wantNear: true,
		},
		{
			name:   "just over the line, but the name is not unique",
			target: Target{ID: "x", Name: "Mau Ara", District: district("Hambantota"), Kind: "major"},
			candidates: []Measured{
				{OSMID: "way/1", Name: "Mau Ara", Kind: KindWater, Names: []string{"Mau Ara"}, AreaHa: 800, Districts: []string{"Monaragala"}, NearDistricts: []string{"Hambantota"}},
				water("way/2", "Mau Ara", 500, 80.3, 8.1, "Anuradhapura"),
			},
			wantReason: ReasonWrongDistrict,
		},
		{
			name:       "the source's spelling of a district",
			target:     Target{ID: "x", Name: "Hambegamuwa", District: district("Moneragala"), Kind: "major"},
			candidates: []Measured{water("way/1", "Hambegamuwa Tank", 186, 80.95, 6.54, "Monaragala")},
			wantID:     "way/1",
		},
	}
	for _, test := range cases {
		results := Match([]Target{test.target}, test.candidates)
		got := results[0]
		switch {
		case test.wantID == "" && got.Candidate != nil:
			t.Errorf("%s: mapped to %s, want %s", test.name, got.Candidate.OSMID, test.wantReason)
		case test.wantID != "" && (got.Candidate == nil || got.Candidate.OSMID != test.wantID):
			t.Errorf("%s: got %+v, want %s", test.name, got, test.wantID)
		case test.wantID == "" && got.Reason != test.wantReason:
			t.Errorf("%s: reason %s, want %s", test.name, got.Reason, test.wantReason)
		case got.Near != test.wantNear:
			t.Errorf("%s: near = %v", test.name, got.Near)
		}
	}
}

// Every reservoir the Mahaweli Authority reports has a district to check
// against, so none is left to a name alone.
func TestDistrictsCoverMahaweliReservoirs(t *testing.T) {
	for _, id := range []string{"victoria", "kotmale", "randenigala", "rantambe", "maduru-oya", "udawalawe", "maussakele", "castlereagh", "kala-wewa"} {
		if len(Districts[id]) == 0 {
			t.Errorf("%s has no district", id)
		}
	}
}
