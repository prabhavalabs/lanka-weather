package alerts

import (
	"slices"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

// A flood warning whose text names no district is placed in its basin's
// districts, not sent to the whole country as "Sri Lanka" (November 2026).
func TestFloodWarningsTakeTheirBasinsDistricts(t *testing.T) {
	overview := &warnings.Overview{InForce: []warnings.Bulletin{
		{ID: "a", Type: "water_level", Title: "Flood Warning for Gin River Basin", Severity: "warning", IssuedAt: start, Current: true},
		{ID: "b", Type: "weather", Title: "Advisory for Thunderstorm", Severity: "advisory", IssuedAt: start, Current: true},
		{ID: "c", Type: "water_level", Title: "Flood warning for Nilwala River Basin", Severity: "warning", IssuedAt: start, Current: true,
			Districts: []warnings.DistrictRef{{ID: "LK-32"}}},
	}}
	basins := func(title string) []string {
		if title == "Flood Warning for Gin River Basin" {
			return []string{"LK-31", "LK-32"}
		}
		return nil
	}
	reading := WithBasinDistricts(Bulletins(overview, start), basins)
	byTitle := map[string]push.Event{}
	for _, observation := range reading.Observations {
		byTitle[observation.Event.Facts["title"]] = observation.Event
	}
	gin := byTitle["Flood Warning for Gin River Basin"]
	if gin.National || !slices.Equal(gin.Districts, []string{"LK-31", "LK-32"}) || gin.Facts["areaFrom"] != "basin" ||
		gin.Facts["districts"] != "LK-31,LK-32" || gin.Facts["issuedAt"] == "" {
		t.Fatalf("gin %+v", gin)
	}
	if storm := byTitle["Advisory for Thunderstorm"]; !storm.National || len(storm.Districts) != 0 {
		t.Fatalf("a weather advisory with no districts stays national: %+v", storm)
	}
	if nilwala := byTitle["Flood warning for Nilwala River Basin"]; nilwala.Facts["areaFrom"] != "" || nilwala.URL != "/?warnings=water_level&district=LK-32" {
		t.Fatalf("districts the text names are kept: %+v", nilwala)
	}
}

func TestLandslideWarningNamesItsDistrictsAtOnce(t *testing.T) {
	until := start.Add(24 * time.Hour)
	bulletin := &warnings.Bulletin{Type: "landslide", IssuedAt: start, ValidUntil: &until, TextStatus: "unusable",
		Districts: []warnings.DistrictRef{{ID: "LK-81"}, {ID: "LK-21"}}}
	reading := LandslideIssued(bulletin, start.Add(10*time.Minute))
	if !reading.Complete || reading.Kind != LandslideIssuedKind || len(reading.Observations) != 2 {
		t.Fatalf("reading %+v", reading)
	}
	first := reading.Observations[0]
	if first.Subject != "landslide-issued:LK-21" || first.Event.Kind != messages.KindLandslide || first.Event.Variant != "issued" ||
		first.Event.Topic != push.TopicLandslide || first.Event.Tag != "landslide:district:LK-21" {
		t.Fatalf("observation %+v", first)
	}
	transitions := Apply(reading, map[string]State{}, start.Add(10*time.Minute))
	if len(transitions) != 2 || transitions[0].Event == nil {
		t.Fatalf("first sight alerts: %+v", transitions)
	}
	if lapsed := LandslideIssued(bulletin, until.Add(time.Minute)); !lapsed.Complete || len(lapsed.Observations) != 0 {
		t.Fatalf("a lapsed report says nothing is in force: %+v", lapsed)
	}
	pending := *bulletin
	pending.TextStatus = "pending"
	if reading := LandslideIssued(&pending, start); reading.Complete {
		t.Fatal("a report not yet read changes nothing")
	}
	if reading := LandslideIssued(nil, start); reading.Complete {
		t.Fatal("no report changes nothing")
	}
}

func TestCompassAndNearestCoast(t *testing.T) {
	// A storm in the Bay of Bengal east of Trincomalee.
	lon, lat, km := grid.NearestCoast(87.5, 8.6)
	if km < 500 || km > 800 || lon < 81 || lat < 7 || lat > 10 {
		t.Fatalf("nearest coast %v %v %v", lon, lat, km)
	}
	if direction := grid.Compass(lon, lat, 87.5, 8.6); direction != "E" {
		t.Fatalf("direction %s", direction)
	}
	if grid.Compass(80, 7, 80, 12) != "N" || grid.Compass(80, 7, 78, 5) != "SW" {
		t.Fatal("compass points")
	}
}
