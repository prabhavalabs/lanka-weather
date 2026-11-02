package hydrorivers

import (
	"os"
	"testing"
)

func TestDecodeSegmentsReadsARecordedPage(t *testing.T) {
	payload, err := os.ReadFile("testdata/rivers_sample.geojson")
	if err != nil {
		t.Fatal(err)
	}
	segments, skipped, err := DecodeSegments(payload)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || len(segments) < 2 {
		t.Fatalf("%d segments, %d skipped", len(segments), skipped)
	}
	var mouth *Segment
	for i := range segments {
		if segments[i].ID == 41405515 {
			mouth = &segments[i]
		}
	}
	if mouth == nil {
		t.Fatal("the Kelani's mouth is missing")
	}
	if mouth.NextDown != 0 || mouth.MainRiv != 41405515 || mouth.UplandKM2 < 2000 || mouth.HybasL12 == 0 {
		t.Fatalf("mouth = %+v", mouth)
	}
	// Lines run downstream: the Kelani reaches the sea at about 79.87° E.
	last := mouth.Line[len(mouth.Line)-1]
	if last[0] > 79.9 || last[0] < 79.8 {
		t.Fatalf("the line ends at %v", last)
	}
}

func TestDecodeSegmentsHandlesBothLineForms(t *testing.T) {
	payload := []byte(`{"type":"FeatureCollection","features":[
	  {"properties":{"HYRIV_ID":1,"NEXT_DOWN":0,"MAIN_RIV":1,"HYBAS_L12":4120050170.0},
	   "geometry":{"type":"LineString","coordinates":[[80,7],[80.1,7]]}},
	  {"properties":{"HYRIV_ID":2,"NEXT_DOWN":1,"MAIN_RIV":1},
	   "geometry":{"type":"MultiLineString","coordinates":[[[80,7.2],[80,7.1]],[[80,7.1],[80,7]]]}},
	  {"properties":{"HYRIV_ID":3},"geometry":{"type":"Point","coordinates":[80,7]}},
	  {"properties":{"HYRIV_ID":0},"geometry":{"type":"LineString","coordinates":[[80,7],[80.1,7]]}}
	]}`)
	segments, skipped, err := DecodeSegments(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 2 || skipped != 2 {
		t.Fatalf("%d segments, %d skipped", len(segments), skipped)
	}
	if segments[0].HybasL12 != 4120050170 {
		t.Fatalf("hybas = %d", segments[0].HybasL12)
	}
	if len(segments[1].Line) != 3 {
		t.Fatalf("joined line = %v, want the shared point once", segments[1].Line)
	}
}

func TestDecodeSegmentsRejectsAnotherDocument(t *testing.T) {
	if _, _, err := DecodeSegments([]byte(`{"message":"rate limited"}`)); err == nil {
		t.Fatal("want an error")
	}
}

func TestDecodeNamesDropsNamesThatSayNothing(t *testing.T) {
	payload, err := os.ReadFile("testdata/names.json")
	if err != nil {
		t.Fatal(err)
	}
	names, err := DecodeNames(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 {
		t.Fatalf("names = %v", names)
	}
	if names[41405515] != "Kelani Ganga" || names[41405096] != "Kelani Ganga" {
		t.Fatalf("names = %v", names)
	}
	if _, ok := names[41402815]; ok {
		t.Fatal("a blank name should be dropped")
	}
	if _, ok := names[41406108]; ok {
		t.Fatal("(Unnamed) should be dropped")
	}
}
