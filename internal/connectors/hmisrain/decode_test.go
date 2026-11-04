package hmisrain

import (
	"os"
	"testing"
	"time"
)

func TestDecode(t *testing.T) {
	payload, err := os.ReadFile("testdata/rain_live.json")
	if err != nil {
		t.Fatal(err)
	}
	feed, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Gauges) != 5 || feed.Skipped != 3 {
		t.Fatalf("gauges %d, skipped %d", len(feed.Gauges), feed.Skipped)
	}
	byID := map[string]Gauge{}
	for _, gauge := range feed.Gauges {
		byID[gauge.ID] = gauge
	}
	north := byID["argtest-07"]
	if north.Name != "ASC Test North" || north.Institution != "Met Department" || north.Device != "ARGTEST 07" {
		t.Fatalf("gauge = %+v", north)
	}
	if len(north.Points) != 4 || north.Points[0].RainMm != 0 {
		t.Fatalf("points should be oldest first: %+v", north.Points)
	}
	newest, ok := north.Newest()
	if !ok || newest.RainMm != 12.5 || !newest.At.Equal(time.Date(2026, 11, 2, 18, 20, 0, 0, time.UTC)) {
		t.Fatalf("newest = %+v, %v", newest, ok)
	}
	if byID["test-oya-testgama"].Device != "Test Oya_Testgama" {
		t.Fatalf("ids must be safe in a URL: %v", byID)
	}
	if got := byID["r99"]; got.Institution != NotStated || got.Name != "Agrarian Service Center, Alapilli" || len(got.Points) != 0 {
		t.Fatalf("gauge = %+v", got)
	}
	if got := byID["bad-values"]; len(got.Points) != 0 {
		t.Fatalf("negative, absurd and future readings must go: %+v", got.Points)
	}
}

func TestDecodeRejectsWhatIsNotTheFile(t *testing.T) {
	for _, payload := range []string{"<html>", `{"devices":{}}`} {
		if _, err := Decode([]byte(payload)); err == nil {
			t.Errorf("Decode(%q) should fail", payload)
		}
	}
}
