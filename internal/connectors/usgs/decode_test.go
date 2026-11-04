package usgs

import (
	"os"
	"testing"
	"time"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestDecodeSeparatesEventsDeletionsAndNonEvents(t *testing.T) {
	events, deleted, skipped, err := Decode(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 || len(deleted) != 1 || deleted[0] != "fixture-deleted" {
		t.Fatalf("%d events, deleted %v", len(events), deleted)
	}
	// An explosion, and an event with no magnitude.
	if skipped != 2 {
		t.Fatalf("skipped = %d", skipped)
	}
	first := events[0]
	if first.ID != "us7000tty7" || first.Magnitude != 5.1 || first.Lon != 91.6593 || first.Lat != 13.3785 || first.DepthKm != 10 {
		t.Fatalf("first = %+v", first)
	}
	if first.Place != "218 km NNW of Bamboo Flat, India" || first.URL == "" || first.Tsunami {
		t.Fatalf("first = %+v", first)
	}
	if want := time.UnixMilli(1793263782448).UTC(); !first.OccurredAt.Equal(want) {
		t.Fatalf("occurred %s, want %s", first.OccurredAt, want)
	}
	flagged := events[3]
	if flagged.ID != "fixture-tsunami" || !flagged.Tsunami || flagged.Alert != "yellow" {
		t.Fatalf("flagged = %+v", flagged)
	}
}

func TestDecodeRejectsWhatIsNotACollection(t *testing.T) {
	if _, _, _, err := Decode([]byte(`{"error":"bad"}`)); err == nil {
		t.Fatal("expected an error")
	}
}

func TestMergeKeepsTheLatestRevisionAndHonoursDeletion(t *testing.T) {
	early, late := time.Unix(100, 0), time.Unix(200, 0)
	merged := Merge([]string{"gone"},
		[]Event{{ID: "a", Magnitude: 4.5, UpdatedAt: early}, {ID: "gone"}},
		[]Event{{ID: "a", Magnitude: 4.9, UpdatedAt: late}, {ID: "b", UpdatedAt: late}},
	)
	if len(merged) != 2 || merged[0].Magnitude != 4.9 || merged[1].ID != "b" {
		t.Fatalf("merged = %+v", merged)
	}
}
