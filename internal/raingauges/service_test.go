package raingauges

import (
	"testing"
	"time"
)

func TestBandOf(t *testing.T) {
	cases := map[float64]string{0: BandNone, 0.5: BandLight, 9.9: BandLight, 10: BandModerate, 49.9: BandModerate,
		50: BandHeavy, 99.9: BandHeavy, 100: BandVeryHeavy, 300: BandVeryHeavy}
	for mm, want := range cases {
		if got := BandOf(mm); got != want {
			t.Errorf("BandOf(%v) = %s, want %s", mm, got, want)
		}
	}
}

func feature(id string, lon, lat float64, rain *float64, stale bool) Feature {
	return Feature{Type: "Feature", ID: id, Geometry: Point{Type: "Point", Coordinates: [2]float64{lon, lat}},
		Properties: Properties{ID: id, Name: id, Rain24hMm: rain, Stale: stale, FetchedAt: time.Now()}}
}

func mm(value float64) *float64 { return &value }

func TestNearestPrefersACurrentGauge(t *testing.T) {
	layer := FeatureCollection{Features: []Feature{
		feature("old-beside-you", 80.0, 7.0, nil, true),
		feature("current-further", 80.1, 7.0, mm(3), false),
		feature("beyond-reach", 81.0, 7.0, mm(30), false),
	}}
	if got := NearestIn(layer, 80.0, 7.0); got == nil || got.ID != "current-further" {
		t.Fatalf("nearest = %+v", got)
	}
	layer.Features = layer.Features[:1]
	if got := NearestIn(layer, 80.0, 7.0); got == nil || got.ID != "old-beside-you" || !got.Stale {
		t.Fatalf("with nothing current, the old one should say so: %+v", got)
	}
	if got := NearestIn(layer, 82.0, 9.0); got != nil {
		t.Fatalf("nothing is within reach: %+v", got)
	}
}

func TestWettestLeavesOutDryAndOutOfDateGauges(t *testing.T) {
	layer := FeatureCollection{Features: []Feature{
		feature("dry", 80, 7, mm(0), false),
		feature("old", 80, 7, nil, true),
		feature("wet", 80, 7, mm(12), false),
		feature("wetter", 80, 7, mm(120), false),
	}}
	got := Wettest(layer, 5)
	if len(got) != 2 || got[0].ID != "wetter" || got[1].ID != "wet" {
		t.Fatalf("wettest = %+v", got)
	}
	if len(Wettest(layer, 1)) != 1 {
		t.Fatal("the limit should hold")
	}
}
