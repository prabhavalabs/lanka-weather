package waterlevels

import (
	"os"
	"testing"
)

func TestDecodeCanals(t *testing.T) {
	payload, err := os.ReadFile("testdata/canals.json")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := DecodeCanals(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Gauges) != 3 || snapshot.Skipped != 1 {
		t.Fatalf("gauges %d, skipped %d", len(snapshot.Gauges), snapshot.Skipped)
	}
	if snapshot.Credit != "Canal levels: CUrW / SLLDC (pub.curwsl.org)" || snapshot.URL != "http://pub.curwsl.org/" {
		t.Fatalf("credit = %q, %q", snapshot.Credit, snapshot.URL)
	}
	if snapshot.SeaLevelM == nil || *snapshot.SeaLevelM != 0.141 || len(snapshot.Tide) != 1 {
		t.Fatalf("sea %v, tide %+v", snapshot.SeaLevelM, snapshot.Tide)
	}
	first := snapshot.Gauges[0]
	if first.ID != "test-bridge" || first.State != "normal" || *first.LevelM != 0.414 || len(first.Last24h) != 1 {
		t.Fatalf("a null reading should be left out: %+v", first)
	}
	if higher := snapshot.Gauges[1]; higher.State != "higher" {
		t.Fatalf("the source's state is kept: %+v", higher)
	}
	if silent := snapshot.Gauges[2]; silent.LevelM != nil || silent.ObservedAt != nil || silent.State != "offline" {
		t.Fatalf("a missing figure is empty, never zero: %+v", silent)
	}
	if _, err := DecodeCanals([]byte(`{"gauges":[]}`)); err == nil {
		t.Fatal("no generatedAt must fail")
	}
}
