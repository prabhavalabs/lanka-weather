package irrigationgis

import (
	"os"
	"strings"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/geo"
)

func page(t *testing.T, name string) []Feature {
	t.Helper()
	payload, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	features, _, err := DecodePage(payload)
	if err != nil {
		t.Fatal(err)
	}
	return features
}

func square(west, south, size float64) []geo.Ring {
	return []geo.Ring{{{west, south}, {west, south + size}, {west + size, south + size}, {west + size, south}, {west, south}}}
}

func feature(fid int, name string, rings []geo.Ring) Feature {
	return Feature{Attributes: map[string]any{"FID": float64(fid), "Name": name}, Rings: rings}
}

func TestDecodePageReadsARecordedPage(t *testing.T) {
	features := page(t, "river_basins_page.json")
	if len(features) == 0 {
		t.Fatal("no features")
	}
	if name := features[0].Text("Name"); name == "" {
		t.Fatalf("attributes = %v", features[0].Attributes)
	}
	if len(features[0].Rings) == 0 {
		t.Fatal("no rings")
	}
}

func TestDecodePageSurfacesServiceErrorsAndMorePages(t *testing.T) {
	if _, _, err := DecodePage([]byte(`{"error":{"code":400,"message":"Invalid query"}}`)); err == nil ||
		!strings.Contains(err.Error(), "Invalid query") {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := DecodePage([]byte(`{"message":"not arcgis"}`)); err == nil {
		t.Fatal("want an error for another document")
	}
	_, more, err := DecodePage([]byte(`{"features":[],"exceededTransferLimit":true}`))
	if err != nil || !more {
		t.Fatalf("more = %v, err = %v", more, err)
	}
}

func TestCleanBasins(t *testing.T) {
	big := square(80, 7, 0.5)
	basins := CleanBasins([]Feature{
		feature(1, "Kelani Ganga", big),
		feature(2, "Kelani Ganga", square(80.6, 7, 0.001)), // a scrap: merged
		feature(3, "Placemark", square(81, 7, 0.001)),      // a sliver: dropped
		feature(4, "Placemark", square(81, 8, 0.2)),        // large: a coastal area
		feature(5, "Pali Aru", square(80, 9, 0.2)),         // two rivers, one name
		feature(6, "Pali Aru", square(80.5, 9.2, 0.1)),
		feature(7, "Tiny Oya", square(81.5, 8, 0.001)), // wholly a scrap: lost
		feature(8, "Telwatta Ganaga", square(80, 6, 0.1)),
	})
	byID := map[string]Basin{}
	for _, basin := range basins {
		byID[basin.ID] = basin
	}
	if len(basins) != 5 {
		t.Fatalf("%d basins: %v", len(basins), keys(byID))
	}
	kelani, ok := byID["kelani-ganga"]
	if !ok || kelani.FID != 1 || len(kelani.Shape) != 2 || !kelani.Named {
		t.Fatalf("Kelani = %+v", kelani)
	}
	coastal, ok := byID["coastal-4"]
	if !ok || coastal.Named || coastal.Name != CoastalName {
		t.Fatalf("coastal = %+v", coastal)
	}
	if _, ok := byID["pali-aru"]; !ok {
		t.Fatalf("basins: %v", keys(byID))
	}
	if _, ok := byID["pali-aru-6"]; !ok {
		t.Fatalf("the second Pali Aru should stay apart: %v", keys(byID))
	}
	if _, ok := byID["telwatta-ganga"]; !ok {
		t.Fatalf("spelling should be fixed: %v", keys(byID))
	}
}

func keys(basins map[string]Basin) []string {
	out := []string{}
	for id := range basins {
		out = append(out, id)
	}
	return out
}

func TestCleanBasinsOnARecordedPageDropsPlaceholders(t *testing.T) {
	for _, basin := range CleanBasins(page(t, "river_basins_page.json")) {
		if basin.Name == basinPlaceholder {
			t.Fatalf("placeholder kept: %+v", basin.ID)
		}
	}
}

func TestParseEvent(t *testing.T) {
	cases := []struct {
		source, name, river, magnitude string
		year                           int
	}{
		{"Kelani Ganga Flood 1989 Critical", "Kelani Ganga flood, 1989 (critical)", "Kelani Ganga", "critical", 1989},
		{"Kelani Ganag Flood 2016 Major", "Kelani Ganga flood, 2016 (major)", "Kelani Ganga", "major", 2016},
		{"Kelani Ganga Flood 2012 Minor", "Kelani Ganga flood, 2012 (minor)", "Kelani Ganga", "minor", 2012},
		{"Maha Oya (102) Flood 2016", "Maha Oya flood, 2016", "Maha Oya", "", 2016},
		{"Mundani Aru Flood 2014", "Mundeni Aru flood, 2014", "Mundeni Aru", "", 2014},
		{"Attanagala Oya Flood Inundation 2025 Nov", "Attanagalu Oya flood, 2025", "Attanagalu Oya", "", 2025},
		{"Mahawali Flood 2025 Nov", "Mahaweli Ganga flood, 2025", "Mahaweli Ganga", "", 2025},
		{"  Yan Oya Flood 2016 ", "Yan Oya flood, 2016", "Yan Oya", "", 2016},
	}
	for _, c := range cases {
		event, ok := ParseEvent(c.source)
		if !ok {
			t.Fatalf("%q was not parsed", c.source)
		}
		if event.Name != c.name || event.River != c.river || event.Magnitude != c.magnitude || event.Year != c.year {
			t.Fatalf("%q = %+v", c.source, event)
		}
		if event.ID == "" || event.FID == 0 {
			t.Fatalf("%q has no identity: %+v", c.source, event)
		}
	}
}

func TestParseEventRefusesAValueWithoutAYear(t *testing.T) {
	for _, source := range []string{"", "Flood Inundation Area", "Kelani Ganga Flood"} {
		if _, ok := ParseEvent(source); ok {
			t.Fatalf("%q should not be an event", source)
		}
	}
}

func TestEventIdentitiesAreStableAndDistinct(t *testing.T) {
	a, _ := ParseEvent("Kelani Ganga Flood 2016 Major")
	b, _ := ParseEvent("Kelani Ganag Flood 2016 Major")
	c, _ := ParseEvent("Kelani Ganga Flood 2012 Minor")
	if a.ID != b.ID || a.FID != b.FID {
		t.Fatalf("spelling should not change identity: %+v, %+v", a, b)
	}
	if a.ID == c.ID || a.FID == c.FID {
		t.Fatalf("different events share an identity: %+v, %+v", a, c)
	}
	if puttalam := PuttalamEvent(); puttalam.Date == nil || puttalam.Year != 2025 || puttalam.FID == 0 {
		t.Fatalf("Puttalam = %+v", puttalam)
	}
}

func TestDecodeStreamsDropsTheUnnamedAndWhatIsNotARiver(t *testing.T) {
	streams := DecodeStreams([]Feature{
		{Attributes: map[string]any{"FID": 1.0, "NAME": " "}, Rings: square(80, 7, 0.1)},
		{Attributes: map[string]any{"FID": 2.0, "NAME": "MAHAWELI GANGA"}, Rings: square(80, 7, 0.1)},
		{Attributes: map[string]any{"FID": 3.0, "NAME": "Flood Outlet"}, Rings: square(80, 7, 0.1)},
		{Attributes: map[string]any{"FID": 4.0, "NAME": "Kuda Oya"}, Rings: square(80, 7, 0.1)},
	})
	if len(streams) != 2 || streams[0].Name != "Mahaweli Ganga" || streams[1].Name != "Kuda Oya" {
		t.Fatalf("streams = %+v", streams)
	}
}

func TestDecodeStationsReadsARecordedPage(t *testing.T) {
	stations := DecodeStations(page(t, "hydrostations_page.json"))
	found := false
	for _, station := range stations {
		if station.Name == "Holombuwa" {
			found = true
			if station.Tributary != "Gurugoda Oya" || station.Basin != "Kelani Ganga" {
				t.Fatalf("Holombuwa = %+v", station)
			}
		}
		if station.Basin != strings.TrimSpace(station.Basin) {
			t.Fatalf("basin not trimmed: %q", station.Basin)
		}
	}
	if !found {
		t.Fatalf("stations = %+v", stations)
	}
}

func TestDecodeServiceLayersAndLastEdit(t *testing.T) {
	layers, err := DecodeServiceLayers([]byte(`{"layers":[{"id":2,"name":"Flood Inundation Area"},{"id":3,"name":"River_Basin"}]}`))
	if err != nil || len(layers) != 2 || layers[0].ID != 2 || layers[0].Name != "Flood Inundation Area" {
		t.Fatalf("layers = %+v, err = %v", layers, err)
	}
	if _, err := DecodeServiceLayers([]byte(`{}`)); err == nil {
		t.Fatal("want an error for a service with no layers")
	}
	if got := DecodeLastEdit([]byte(`{"editingInfo":{"lastEditDate":1662458967537}}`)); got != 1662458967537 {
		t.Fatalf("last edit = %d", got)
	}
	if got := DecodeLastEdit([]byte(`{}`)); got != 0 {
		t.Fatalf("last edit = %d", got)
	}
}

func TestExtentShapeDropsRingsTooSmallToMatter(t *testing.T) {
	big := square(80, 7, 0.01)[0]    // about 1.2 km²
	cell := square(81, 7, 0.0001)[0] // about 120 m²: one raster cell of water
	hole := geo.Ring{{80.002, 7.002}, {80.008, 7.002}, {80.008, 7.008}, {80.002, 7.008}, {80.002, 7.002}}
	speck := geo.Ring{{80.0005, 7.0005}, {80.0006, 7.0005}, {80.0006, 7.0006}, {80.0005, 7.0006}, {80.0005, 7.0005}}
	shape := ExtentShape([]geo.Ring{big, cell, hole, speck})
	if len(shape) != 1 {
		t.Fatalf("polygons = %d, want the big one", len(shape))
	}
	if len(shape[0]) != 2 {
		t.Fatalf("rings = %d, want the big one with its large hole", len(shape[0]))
	}
}
