package regions

import "testing"

func TestParseIDNormalisesSpellings(t *testing.T) {
	cases := []struct {
		raw   string
		id    string
		level Level
	}{
		{"LK-1", "LK-1", LevelProvince},
		{"LK11", "LK-11", LevelDistrict},
		{"lk-1127", "LK-1127", LevelDS},
		{" LK-1-1-27-005 ", "LK-1127005", LevelGN},
		{"LK1127005", "LK-1127005", LevelGN},
	}
	for _, c := range cases {
		id, level, ok := ParseID(c.raw)
		if !ok || id != c.id || level != c.level {
			t.Errorf("ParseID(%q) = %q, %q, %v; want %q, %q", c.raw, id, level, ok, c.id, c.level)
		}
	}
	for _, bad := range []string{"", "LK", "LK-", "LK-123", "LK-12345", "IN-11", "LK-1a", "LK-11270051"} {
		if _, _, ok := ParseID(bad); ok {
			t.Errorf("ParseID(%q) accepted", bad)
		}
	}
}

func TestHierarchyFromCodes(t *testing.T) {
	if got := ParentID("LK-1127005"); got != "LK-1127" {
		t.Fatalf("ParentID(gn) = %q", got)
	}
	if got := ParentID("LK1127"); got != "LK-11" {
		t.Fatalf("ParentID(ds) = %q", got)
	}
	if got := ParentID("LK-11"); got != "LK-1" {
		t.Fatalf("ParentID(district) = %q", got)
	}
	if got := ParentID("LK-1"); got != "" {
		t.Fatalf("ParentID(province) = %q", got)
	}
	province, district, ds := Ancestry("LK-1127005")
	if province != "LK-1" || district != "LK-11" || ds != "LK-1127" {
		t.Fatalf("Ancestry(gn) = %q %q %q", province, district, ds)
	}
	province, district, ds = Ancestry("LK-21")
	if province != "LK-2" || district != "LK-21" || ds != "" {
		t.Fatalf("Ancestry(district) = %q %q %q", province, district, ds)
	}
}

func TestFIDIsUniqueAcrossLevels(t *testing.T) {
	seen := map[int64]string{}
	for _, id := range []string{"LK-1", "LK-11", "LK-1101", "LK-1101001", "LK-2", "LK-21"} {
		fid, ok := FID(id)
		if !ok {
			t.Fatalf("FID(%q) failed", id)
		}
		if other, dup := seen[fid]; dup {
			t.Fatalf("FID(%q) = FID(%q) = %d", id, other, fid)
		}
		seen[fid] = id
	}
	if fid, _ := FID("LK-1127005"); fid != 1127005 {
		t.Fatalf("FID = %d", fid)
	}
}

func TestISOCodeCrosswalk(t *testing.T) {
	cases := map[string]string{
		"LK-1":       "LK-1",
		"LK-11":      "LK-11",
		"LK-41":      "LK-41", // Jaffna
		"LK-42":      "LK-43", // Mannar
		"LK-43":      "LK-44", // Vavuniya
		"LK-44":      "LK-45", // Mullaitivu
		"LK-45":      "LK-42", // Kilinochchi
		"LK-1127":    "",
		"LK-1127005": "",
	}
	for id, want := range cases {
		if got := ISOCode(id); got != want {
			t.Errorf("ISOCode(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestParseLevel(t *testing.T) {
	if level, ok := ParseLevel("ds"); !ok || level.Rank() != 3 {
		t.Fatalf("ParseLevel(ds) = %q %v", level, ok)
	}
	if _, ok := ParseLevel("village"); ok {
		t.Fatal("ParseLevel accepted village")
	}
}
