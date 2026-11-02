package irrigation

import (
	"os"
	"testing"
	"time"
)

func loadPage(t *testing.T) Page {
	t.Helper()
	payload, err := os.ReadFile("testdata/gauges_page.json")
	if err != nil {
		t.Fatal(err)
	}
	page, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func TestDecodeRecordedPage(t *testing.T) {
	page := loadPage(t)
	if len(page.Readings) != 9 {
		t.Fatalf("readings = %d, want 9", len(page.Readings))
	}
	if len(page.Stations) != 5 {
		t.Fatalf("stations = %d, want 5", len(page.Stations))
	}
	if page.More || page.Skipped != 0 {
		t.Fatalf("More = %v, Skipped = %d", page.More, page.Skipped)
	}

	stations := map[string]Station{}
	for _, station := range page.Stations {
		stations[station.ID] = station
	}

	colombo := stations["nagalagam-street"]
	if colombo.Unit != "ft" || colombo.Basin != "Kelani Ganga" {
		t.Fatalf("nagalagam-street = %+v", colombo)
	}
	if colombo.AlertLevel == nil || *colombo.AlertLevel != 4 || *colombo.MinorFloodLevel != 5 || *colombo.MajorFloodLevel != 7 {
		t.Fatalf("nagalagam-street thresholds = %v %v %v", colombo.AlertLevel, colombo.MinorFloodLevel, colombo.MajorFloodLevel)
	}

	if ampara := stations["ampara"]; ampara.AlertLevel != nil {
		t.Fatal("ampara publishes no thresholds; they must decode as missing")
	}
	if ratnapura := stations["rathnapura"]; ratnapura.Unit != "m" {
		t.Fatalf("rathnapura unit = %q, want m", ratnapura.Unit)
	}
}

func TestDecodeKeepsLatestStationDescription(t *testing.T) {
	page := loadPage(t)
	for _, station := range page.Stations {
		for _, reading := range page.Readings {
			if reading.StationID == station.ID && reading.ObservedAt.After(station.ObservedAt) {
				t.Fatalf("%s described by %v but has a later reading at %v", station.ID, station.ObservedAt, reading.ObservedAt)
			}
		}
	}
}

func TestDecodeServiceError(t *testing.T) {
	if _, err := Decode([]byte(`{"error":{"code":400,"message":"Invalid query"}}`)); err == nil {
		t.Fatal("expected an error")
	}
}

func TestDecodeSkipsUnusableFeatures(t *testing.T) {
	payload := []byte(`{"exceededTransferLimit":true,"features":[
		{"attributes":{"basin":"Kelani Ganga","gauge":"","water_level":1,"CreationDate":1793620568040}},
		{"attributes":{"basin":"Kelani Ganga","gauge":"Hanwella","water_level":null,"CreationDate":1793620568040}},
		{"attributes":{"basin":"Kelani Ganga","gauge":"Hanwella","water_level":2.5,"rain_fall":-1,"CreationDate":1793620568040,
		  "alertpull":7.5,"minorpull":6.5,"majorpull":9.5},"geometry":{"x":0,"y":0}}
	]}`)
	page, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !page.More || page.Skipped != 2 || len(page.Readings) != 1 {
		t.Fatalf("page = %+v", page)
	}
	if page.Readings[0].RainfallMM != nil {
		t.Fatal("negative rainfall must decode as missing")
	}
	if len(page.Stations) != 0 {
		t.Fatal("a position outside Sri Lanka must not create a station")
	}
	if got := page.Readings[0].ObservedAt; !got.Equal(time.UnixMilli(1793620568040)) {
		t.Fatalf("observed at %v", got)
	}
}

func TestThresholdsRejectInvertedOrZero(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	if a, _, _ := thresholds(f(0), f(0), f(0)); a != nil {
		t.Fatal("zero thresholds must be missing")
	}
	if a, _, _ := thresholds(f(5), f(4), f(6)); a != nil {
		t.Fatal("inverted thresholds must be missing")
	}
	if a, b, c := thresholds(f(3), f(4), f(5)); a == nil || *b != 4 || *c != 5 {
		t.Fatal("valid thresholds must be kept")
	}
}

func TestStationID(t *testing.T) {
	cases := map[string]string{
		"Kalawellawa (Millakanda)": "kalawellawa-millakanda",
		"Nagalagam Street":         "nagalagam-street",
		"  Yaka Wewa ":             "yaka-wewa",
	}
	for in, want := range cases {
		if got := StationID(in); got != want {
			t.Errorf("StationID(%q) = %q, want %q", in, got, want)
		}
	}
}
