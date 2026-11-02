package googleflood

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestDecodeStatuses(t *testing.T) {
	statuses, next, skipped, err := DecodeStatuses(read(t, "statuses_lk.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 2 || skipped != 2 || next != "page-2" {
		t.Fatalf("statuses %d, skipped %d, next %q", len(statuses), skipped, next)
	}
	calm, severe := statuses[0], statuses[1]
	if calm.Severity != SeverityNoFlooding || calm.Trend != "rise" || calm.GaugeSource != "HYBAS" || !calm.QualityVerified {
		t.Fatalf("calm = %+v", calm)
	}
	if !calm.IssuedAt.Equal(time.Date(2026, 11, 2, 12, 23, 13, 138078000, time.UTC)) || calm.ForecastStart == nil ||
		!calm.ForecastEnd.Equal(time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("calm times = %v %v %v", calm.IssuedAt, calm.ForecastStart, calm.ForecastEnd)
	}
	if severe.Severity != SeveritySevere || severe.Trend != "unknown" || severe.InundationPolygons["high"] != "poly-high" {
		t.Fatalf("severe = %+v", severe)
	}
	if got := GaugeAreas(severe); len(got) != 3 || got[0].Kind != KindNotification || got[1].Kind != "inundation_high" || got[2].Kind != "inundation_low" {
		t.Fatalf("areas = %+v", got)
	}
	if len(GaugeAreas(calm)) != 0 {
		t.Fatal("a gauge at no flooding has no areas")
	}
	if statuses, _, _, err := DecodeStatuses([]byte("{}")); err != nil || len(statuses) != 0 {
		t.Fatal("an empty object is an empty page")
	}
	if _, _, _, err := DecodeStatuses([]byte("<html>")); err == nil {
		t.Fatal("HTML must fail")
	}
}

func TestSeverityRank(t *testing.T) {
	if !(SeverityExtreme.Rank() > SeveritySevere.Rank() && SeveritySevere.Rank() > SeverityAboveNormal.Rank() &&
		SeverityAboveNormal.Rank() > SeverityNoFlooding.Rank() && SeverityNoFlooding.Rank() > SeverityUnknown.Rank()) {
		t.Fatal("severities out of order")
	}
	if severity("ABOVE_NORMAL") != SeverityAboveNormal || severity("SOMETHING_NEW") != SeverityUnknown {
		t.Fatal("severity mapping")
	}
}

func TestDecodeModels(t *testing.T) {
	models, err := DecodeModels(read(t, "models.json"))
	if err != nil {
		t.Fatal(err)
	}
	model := models["hybas_4121601300"]
	if model.Unit != "m3/s" || model.WarningLevel == nil || *model.DangerLevel < 347 || *model.ExtremeLevel < 678 {
		t.Fatalf("model = %+v", model)
	}
	if other := models["hybas_4121156400"]; other.Unit != "m" || other.WarningLevel != nil {
		t.Fatalf("model without thresholds = %+v", other)
	}
}

func TestDecodeFlashFloodsKeepsSriLanka(t *testing.T) {
	areas, _, err := DecodeFlashFloods(read(t, "flash_floods.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 2 || areas[0].Kind != KindFlashHighly || areas[0].PolygonID != "flash-lk-highly" || areas[1].Kind != KindFlashLikely {
		t.Fatalf("areas = %+v", areas)
	}
	if areas[0].ValidUntil == nil || !areas[0].ValidUntil.Equal(time.Date(2026, 11, 3, 0, 41, 27, 0, time.UTC)) {
		t.Fatalf("valid until = %v", areas[0].ValidUntil)
	}
}

func TestDecodeSignificantEventsKeepsSriLanka(t *testing.T) {
	areas, _, err := DecodeSignificantEvents(read(t, "significant_events.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 || areas[0].PolygonID != "event-lk" || areas[0].Severity != "severe" ||
		areas[0].Properties["affectedPopulation"] != 120000.0 || areas[0].ValidUntil == nil {
		t.Fatalf("areas = %+v", areas)
	}
}

func TestParsePolygonKML(t *testing.T) {
	kml := `<kml><Document><Placemark><MultiGeometry>
	<Polygon><outerBoundaryIs><LinearRing><coordinates>80,7,0 81,7,0 81,8,0 80,8,0 80,7,0</coordinates></LinearRing></outerBoundaryIs>
	<innerBoundaryIs><LinearRing><coordinates>80.2,7.2 80.4,7.2 80.4,7.4 80.2,7.4</coordinates></LinearRing></innerBoundaryIs>
	<innerBoundaryIs><LinearRing><coordinates>80.2,7.2 80.2,7.2</coordinates></LinearRing></innerBoundaryIs></Polygon>
	<Polygon><outerBoundaryIs><LinearRing><coordinates>81,6 81,6</coordinates></LinearRing></outerBoundaryIs></Polygon>
	</MultiGeometry></Placemark></Document></kml>`
	raw, _ := json.Marshal(map[string]string{"polygonId": "area", "kml": kml})
	shape, err := ParsePolygon(raw)
	if err != nil {
		t.Fatal(err)
	}
	var geometry struct {
		Type        string
		Coordinates [][][][2]float64
	}
	if err := json.Unmarshal(shape, &geometry); err != nil {
		t.Fatal(err)
	}
	if geometry.Type != "MultiPolygon" || len(geometry.Coordinates) != 1 || len(geometry.Coordinates[0]) != 2 {
		t.Fatalf("shape = %s", shape)
	}
	hole := geometry.Coordinates[0][1]
	if hole[0] != hole[len(hole)-1] {
		t.Fatal("an unclosed ring must be closed")
	}
	geojson := []byte(`{"polygon":{"type":"Polygon","coordinates":[[[80,7],[81,7],[81,8],[80,7]]]}}`)
	if _, err := ParsePolygon(geojson); err != nil {
		t.Fatalf("GeoJSON polygon: %v", err)
	}
	for _, bad := range []string{`{}`, `{"kml":"<kml>"}`, `{"kml":"<Polygon><outerBoundaryIs><LinearRing><coordinates>bad</coordinates></LinearRing></outerBoundaryIs></Polygon>"}`,
		`{"type":"Polygon","coordinates":[[[200,7],[81,7],[81,8],[200,7]]]}`} {
		if _, err := ParsePolygon([]byte(bad)); err == nil {
			t.Errorf("%s must fail", bad)
		}
	}
}
