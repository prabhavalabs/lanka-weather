package overview

import (
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/environment"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

var now = time.Date(2026, 11, 2, 21, 0, 0, 0, time.UTC)

func fetched() *time.Time { at := now.Add(-time.Hour); return &at }

func number(value float64) *float64 { return &value }

func gaugeFeature(id string, status rivers.Status, stale bool) rivers.Feature {
	at := now.Add(-30 * time.Minute)
	return rivers.Feature{Properties: rivers.StationProperties{ID: id, Name: id, Basin: "Kelani Ganga", Unit: "m", WaterLevel: number(2), Status: status, Stale: stale, ObservedAt: &at}}
}

func gaugesOf(features ...rivers.Feature) *rivers.FeatureCollection {
	return &rivers.FeatureCollection{Features: features, Meta: rivers.LayerMeta{FetchedAt: fetched()}}
}

func spilling(id string, cusecs float64) reservoirs.Feature {
	return reservoirs.Feature{Properties: reservoirs.Properties{ID: id, Name: id, Status: "spilling", FetchedAt: now,
		Flow: &reservoirs.Flow{Kind: "spill", Cusecs: &cusecs}}}
}

func cycloneLayer(stale bool) *hazards.CycloneLayer {
	return &hazards.CycloneLayer{Meta: hazards.CycloneMeta{Status: hazards.CycloneActive, Stale: stale, FetchedAt: fetched(), Count: 1},
		Features: []hazards.CycloneFeature{{Properties: map[string]any{"kind": "position", "name": "MURASU-26", "eventId": "1001", "alertLevel": "Red",
			"lon": 80.5, "lat": 15.1, "distanceKm": 620.0, "windKmh": 157.0, "stale": false, "latestAt": now.Add(-3 * time.Hour)}}}}
}

// quiet is every feed answering, current, with nothing to report.
func quiet() Inputs {
	return Inputs{
		Warnings:    &warnings.Overview{InForce: []warnings.Bulletin{}, Meta: warnings.OverviewMeta{FetchedAt: fetched()}},
		Landslides:  &warnings.LandslideLayer{Features: []warnings.DivisionFeature{}, Meta: warnings.LandslideMeta{Status: warnings.StatusNone, Note: "Last report 31 Oct 2026; none currently active as far as we know.", FetchedAt: fetched()}},
		Risk:        &riverrisk.RiskDocument{Estimate: true, Note: "Estimate.", Meta: riverrisk.RiskMeta{GaugesCurrent: 2, GaugesTotal: 2, GaugesObservedAt: fetched()}},
		Gauges:      gaugesOf(gaugeFeature("a", rivers.StatusNormal, false), gaugeFeature("b", rivers.StatusNormal, false)),
		Reservoirs:  &reservoirs.FeatureCollection{Features: []reservoirs.Feature{{Properties: reservoirs.Properties{ID: "x", Name: "X", Status: "holding", FetchedAt: now}}}, Meta: reservoirs.Meta{FetchedAt: fetched()}},
		Cyclones:    &hazards.CycloneLayer{Meta: hazards.CycloneMeta{Status: hazards.CycloneNone, Note: "No active cyclone in the North Indian Ocean", FetchedAt: fetched()}},
		Earthquakes: &hazards.EarthquakeLayer{Meta: hazards.EarthquakeMeta{FetchedAt: fetched(), WindowDays: 30}},
		Weather:     &weather.RegionsResponse{ValidAt: fetched(), FetchedAt: fetched(), Regions: []weather.RegionWeather{{ID: "LK-11", Name: "Colombo", TemperatureC: 30, PrecipitationTodayMm: 4, WindSpeedMaxKmh: 20, WindGustsMaxKmh: 30}}},
		Rain:        &stations.RainCollection{Features: []stations.RainFeature{{Properties: stations.RainProperties{ID: "gauge:a", StationID: "a", Kind: stations.KindGauge, Name: "A", RainMm: 12, WindowEnd: now}}}},
		Air:         &AirWorst{AirPoint: environment.AirPoint{USAQI: 80, Category: environment.AQIModerate, FetchedAt: now}, Lon: 80, Lat: 7, Place: "Colombo District"},
		Sea:         &SeaHighest{SeaPoint: environment.SeaPoint{WaveHeightM: 1.4, State: "moderate", DistanceKm: 40, FetchedAt: now}, Lon: 80, Lat: 5.5},
	}
}

func section(t *testing.T, overview *Overview, id string) Section {
	t.Helper()
	for _, candidate := range overview.Sections {
		if candidate.ID == id {
			return candidate
		}
	}
	t.Fatalf("no section %s", id)
	return Section{}
}

func TestTheStatusSentence(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Inputs)
		want   string
	}{
		{"a quiet day still claims only what gauges show", func(*Inputs) {}, "No river gauge above its alert level."},
		{"the example of the brief", func(in *Inputs) {
			in.Cyclones = cycloneLayer(false)
			features := []reservoirs.Feature{}
			for range 10 {
				features = append(features, spilling("r", 100))
			}
			in.Reservoirs.Features = features
		}, "1 cyclone active north of Sri Lanka. 10 reservoirs spilling. No river gauge above its alert level."},
		{"gauges at alert replace the calm line", func(in *Inputs) {
			in.Gauges = gaugesOf(gaugeFeature("a", rivers.StatusAlert, false), gaugeFeature("b", rivers.StatusMajorFlood, false))
		}, "2 river gauges at or above alert level."},
		{"official warnings lead", func(in *Inputs) {
			in.Warnings.InForce = []warnings.Bulletin{{ID: "w", Title: "Warning for strong winds", Severity: "warning"}}
			in.Landslides.Features = []warnings.DivisionFeature{{Properties: warnings.DivisionAt{ID: "LK-2136", Name: "Doluwa", District: "Kandy", LevelWords: warnings.WordsFor(2)}}}
			in.Cyclones = cycloneLayer(false)
		}, "Landslide warnings shown for 1 DS division, up to Level 2 (Amber). 1 official warning or advisory in force. 1 cyclone active north of Sri Lanka. No river gauge above its alert level."},
		{"stale gauges make no calm statement", func(in *Inputs) {
			in.Gauges = gaugesOf(gaugeFeature("a", rivers.StatusNormal, true), gaugeFeature("b", rivers.StatusNormal, true), gaugeFeature("c", rivers.StatusNormal, true))
		}, "Out of date: river gauges."},
		{"a stale cyclone feed is not counted and is named", func(in *Inputs) {
			in.Cyclones = cycloneLayer(true)
		}, "No river gauge above its alert level. Out of date: cyclones."},
		{"stale reservoirs are not counted", func(in *Inputs) {
			in.Reservoirs.Features = []reservoirs.Feature{{Properties: reservoirs.Properties{ID: "r", Status: "spilling", Stale: true}}}
		}, "No river gauge above its alert level. Out of date: reservoirs."},
		{"a feed that cannot be read is named", func(in *Inputs) {
			in.Warnings = nil
			in.Gauges = nil
		}, "Could not read: official warnings, river gauges."},
		{"high water by estimate is labelled", func(in *Inputs) {
			in.Risk.Rivers = []riverrisk.RiverRisk{{ID: "kelani-ganga", Name: "Kelani Ganga", Named: true, Level: riverrisk.LevelHigh,
				Reaches: []riverrisk.ReachRisk{{FID: 1, Level: riverrisk.LevelHigh}}}}
		}, "Estimate: 1 river stretch at high or severe. No river gauge above its alert level."},
		{"a large earthquake in the last day", func(in *Inputs) {
			in.Earthquakes.Features = []hazards.EarthquakeFeature{{Properties: hazards.Earthquake{ID: "q", Magnitude: 7.2, Place: "Off Sumatra", Time: now.Add(-2 * time.Hour), DistanceKm: 1600}}}
		}, "M7.2 earthquake reported in the last day. No river gauge above its alert level."},
		{"nothing at all to say", func(in *Inputs) {
			*in = Inputs{}
		}, "Could not read: landslide levels, official warnings, river gauges, river estimates, reservoirs, cyclones, earthquakes, weather, gauge rainfall, air quality, sea conditions."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := quiet()
			c.change(&in)
			got := Build(in, now).Status
			if got != c.want {
				t.Fatalf("status:\n got %q\nwant %q", got, c.want)
			}
			lowered := strings.ToLower(got)
			for _, banned := range []string{"safe", "all clear", "no danger", "clear"} {
				if strings.Contains(lowered, banned) {
					t.Fatalf("the status must never say %q: %q", banned, got)
				}
			}
		})
	}
}

func TestNoSectionEverSpeaksOfSafety(t *testing.T) {
	overview := Build(quiet(), now)
	for _, part := range overview.Sections {
		for _, text := range []string{part.Summary, part.Note} {
			lowered := strings.ToLower(text)
			if strings.Contains(lowered, "safe") || strings.Contains(lowered, "all clear") {
				t.Fatalf("%s says %q", part.ID, text)
			}
		}
	}
}

func TestAFailedFeedIsUnavailableAndTheRestStillAnswer(t *testing.T) {
	in := quiet()
	in.Risk, in.Sea = nil, nil
	overview := Build(in, now)
	if strings.Join(overview.Unavailable, ",") != "river estimates,sea conditions" {
		t.Fatalf("unavailable = %v", overview.Unavailable)
	}
	if got := section(t, overview, SectionRivers); got.Available || !strings.Contains(got.Summary, "Could not be read") {
		t.Fatalf("rivers = %+v", got)
	}
	if got := section(t, overview, SectionGauges); !got.Available {
		t.Fatalf("gauges = %+v", got)
	}
}

func TestOfficialWarningsPassThroughUnchanged(t *testing.T) {
	in := quiet()
	colour := "Red"
	pdf := "https://dmc.gov.lk/a.pdf"
	in.Warnings.InForce = []warnings.Bulletin{{ID: "a", Type: "weather", TypeLabel: "Weather", Title: "Warning for STRONG winds and Rough Seas", Severity: "warning", Colour: &colour, PDFURL: pdf, IssuedAt: now.Add(-time.Hour)}}
	got := section(t, Build(in, now), SectionWarnings)
	if !got.Official || got.Estimate {
		t.Fatalf("section = %+v", got)
	}
	item := got.Items[0]
	if item.Title != "Warning for STRONG winds and Rough Seas" || item.Badge != "Warning: Red" || item.Tone != ToneSevere {
		t.Fatalf("item = %+v", item)
	}
	if item.Href == nil || *item.Href != pdf || item.Target.Kind != "warnings" || item.Target.ID != "weather" {
		t.Fatalf("item = %+v", item)
	}
	if item.At == nil {
		t.Fatal("every line shows its age")
	}
}

func TestAWarningThatMayBeWithdrawnSaysSo(t *testing.T) {
	in := quiet()
	in.Warnings.InForce = []warnings.Bulletin{{ID: "a", Type: "water_level", TypeLabel: "Flood", Title: "Flood Warning", Severity: "warning", MayBeWithdrawnBy: &warnings.NoticeRef{ID: "n"}}}
	got := section(t, Build(in, now), SectionWarnings).Items[0]
	if !strings.Contains(got.Detail, "may already have been withdrawn") {
		t.Fatalf("item = %+v", got)
	}
}

func TestLandslidesListTheWorstFirst(t *testing.T) {
	in := quiet()
	in.Landslides.Features = []warnings.DivisionFeature{
		{Properties: warnings.DivisionAt{ID: "LK-1", Name: "Yellow", District: "Kandy", LevelWords: warnings.WordsFor(1)}},
		{Properties: warnings.DivisionAt{ID: "LK-3", Name: "Red", District: "Badulla", LevelWords: warnings.WordsFor(3)}},
	}
	got := section(t, Build(in, now), SectionLandslides)
	if got.Items[0].ID != "LK-3" || got.Items[0].Tone != ToneSevere || got.Items[0].Badge != "Level 3 (Red)" || got.Items[0].Target.Kind != "landslide" {
		t.Fatalf("items = %+v", got.Items)
	}
	if got.Counts[0].Value != 1 || got.Counts[1].Value != 0 || got.Counts[2].Value != 1 {
		t.Fatalf("counts = %+v", got.Counts)
	}
}

func TestRiversAreEstimatesAndNameTheWorst(t *testing.T) {
	in := quiet()
	in.Risk.Rivers = []riverrisk.RiverRisk{
		{ID: "deduru-oya", Name: "Deduru Oya", Named: true, Level: riverrisk.LevelWatch, UplandKM: 2600,
			Reasons: []riverrisk.Reason{{Text: "Kimbulwana Oya reservoir is spilling 1,551 cusecs upstream."}},
			Reaches: []riverrisk.ReachRisk{{FID: 1, Level: riverrisk.LevelUnknown}, {FID: 2, Level: riverrisk.LevelWatch}}},
		{ID: "river-9", Name: "Unnamed", Named: false, Level: riverrisk.LevelSevere, Reaches: []riverrisk.ReachRisk{{FID: 3, Level: riverrisk.LevelSevere}}},
		{ID: "kala-oya", Name: "Kala Oya", Named: true, Level: riverrisk.LevelNormal, Reaches: []riverrisk.ReachRisk{{FID: 4, Level: riverrisk.LevelNormal}}},
	}
	got := section(t, Build(in, now), SectionRivers)
	if !got.Estimate || !strings.HasPrefix(got.Summary, "Estimate") {
		t.Fatalf("section = %+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "deduru-oya" || !got.Items[0].Estimate || *got.Items[0].Target.Reach != 2 {
		t.Fatalf("items = %+v", got.Items)
	}
	counts := map[string]int{}
	for _, count := range got.Counts {
		counts[count.Key] = count.Value
	}
	if counts["severe"] != 1 || counts["watch"] != 1 || counts["normal"] != 1 || counts["unknown"] != 1 {
		t.Fatalf("counts = %v", counts)
	}
}

func TestReservoirsListSpillsLargestFirstAndNameDownstreamFlow(t *testing.T) {
	in := quiet()
	downstream := 3225.0
	in.Reservoirs.Features = []reservoirs.Feature{
		spilling("small", 60), spilling("large", 1551),
		{Properties: reservoirs.Properties{ID: "nofigure", Name: "nofigure", Status: "spilling"}},
		{Properties: reservoirs.Properties{ID: "polgolla", Name: "Polgolla", Status: "releasing", Flow: &reservoirs.Flow{Kind: "downstream", Cusecs: &downstream}}},
		{Properties: reservoirs.Properties{ID: "old", Name: "old", Status: "spilling", Stale: true}},
	}
	got := section(t, Build(in, now), SectionReservoirs)
	order := []string{}
	for _, item := range got.Items {
		order = append(order, item.ID)
	}
	if strings.Join(order, " ") != "large small nofigure polgolla" {
		t.Fatalf("order = %v", order)
	}
	if got.Items[0].Detail != "1,551 cusecs" || !strings.Contains(got.Items[3].Detail, "can include power-station flow") {
		t.Fatalf("items = %+v", got.Items)
	}
	if got.Items[2].Detail != "no figure given" {
		t.Fatalf("no figure = %+v", got.Items[2])
	}
	if got.Note != reservoirNote {
		t.Fatalf("note = %q", got.Note)
	}
}

func TestEarthquakesKeepOnlyWhatStandsOut(t *testing.T) {
	in := quiet()
	quake := func(id string, magnitude, distance float64, age time.Duration, tsunami bool) hazards.EarthquakeFeature {
		return hazards.EarthquakeFeature{Properties: hazards.Earthquake{ID: id, Magnitude: magnitude, Place: id, DistanceKm: distance, Time: now.Add(-age), Tsunami: tsunami}}
	}
	day := 24 * time.Hour
	in.Earthquakes.Features = []hazards.EarthquakeFeature{
		quake("large-far", 6.1, 4000, day, false),
		quake("moderate-near", 5.1, 1200, day, false),
		quake("moderate-far", 5.4, 3000, day, false),
		quake("old", 7.5, 1000, 8*day, false),
		quake("small", 4.6, 2000, day, false),
		quake("close", 3.1, 200, day, false),
		quake("flagged", 4.6, 2500, day, true),
	}
	got := section(t, Build(in, now), SectionEarthquakes)
	ids := []string{}
	for _, item := range got.Items {
		ids = append(ids, item.ID)
	}
	if strings.Join(ids, " ") != "large-far moderate-near close" {
		t.Fatalf("items = %v", ids)
	}
}

func TestQuietSectionsAreCalmOnlyWhenFresh(t *testing.T) {
	in := quiet()
	in.Earthquakes.Meta.Stale = true
	in.Warnings.Meta.Stale = true
	overview := Build(in, now)
	if got := section(t, overview, SectionEarthquakes).Summary; !strings.Contains(got, "not been able to check") {
		t.Fatalf("earthquakes = %q", got)
	}
	if got := section(t, overview, SectionWarnings).Summary; !strings.Contains(got, "cannot say") {
		t.Fatalf("warnings = %q", got)
	}
	if strings.Join(overview.Stale, ",") != "official warnings,earthquakes" {
		t.Fatalf("stale = %v", overview.Stale)
	}
}

func TestWeatherNamesTheExtremesAndOpensTheirDistrict(t *testing.T) {
	in := quiet()
	in.Weather.Regions = []weather.RegionWeather{
		{ID: "LK-11", Name: "Colombo", TemperatureC: 31.4, PrecipitationTodayMm: 2, WindSpeedMaxKmh: 22, WindGustsMaxKmh: 40},
		{ID: "LK-23", Name: "Nuwara Eliya", TemperatureC: 16.2, PrecipitationTodayMm: 30, WindSpeedMaxKmh: 35, WindGustsMaxKmh: 60},
		{ID: "LK-31", Name: "Galle", TemperatureC: 28, PrecipitationTodayMm: 0, WindSpeedMaxKmh: 10, WindGustsMaxKmh: 15},
	}
	got := section(t, Build(in, now), SectionWeather)
	titles := map[string]Item{}
	for _, item := range got.Items {
		titles[item.ID] = item
	}
	if titles["hottest"].Title != "Hottest: Colombo" || titles["hottest"].Detail != "31.4 °C" || titles["hottest"].Target.Fill != "temperature" {
		t.Fatalf("hottest = %+v", titles["hottest"])
	}
	if titles["coolest"].Title != "Coolest: Nuwara Eliya" {
		t.Fatalf("coolest = %+v", titles["coolest"])
	}
	if titles["wettest-0"].Title != "Wettest today: Nuwara Eliya" || titles["wettest-0"].Target.Fill != "rain" {
		t.Fatalf("wettest = %+v", titles["wettest-0"])
	}
	if _, ok := titles["wettest-2"]; ok {
		t.Fatal("a dry district is not among the wettest")
	}
	if titles["windiest"].Title != "Strongest wind: Nuwara Eliya" || titles["windiest"].Detail != "35 km/h, gusts to 60 km/h" {
		t.Fatalf("windiest = %+v", titles["windiest"])
	}
	if !got.Estimate || !titles["hottest"].Estimate || titles["hottest"].At == nil {
		t.Fatalf("section = %+v", got)
	}
}

func TestDirectionWords(t *testing.T) {
	cases := []struct {
		lon, lat float64
		want     string
	}{{80.5, 15.1, "north"}, {88, 8, "east"}, {80.8, 2, "south"}, {72, 8, "west"}, {88, 15, "north-east"}, {72, 2, "south-west"}}
	for _, c := range cases {
		if got := direction(c.lon, c.lat); got != c.want {
			t.Errorf("(%v, %v) = %s, want %s", c.lon, c.lat, got, c.want)
		}
	}
}
