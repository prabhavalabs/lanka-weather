package waterlevels

import (
	"math"
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

func TestDecodeReservoirs(t *testing.T) {
	snapshot, err := DecodeReservoirs(read(t, "reservoirs.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Reservoirs) != 9 || snapshot.Skipped != 0 {
		t.Fatalf("reservoirs %d, skipped %d", len(snapshot.Reservoirs), snapshot.Skipped)
	}
	if !snapshot.GeneratedAt.Equal(time.Date(2026, 11, 2, 19, 39, 21, 448e6, time.UTC)) {
		t.Fatalf("generated at %v", snapshot.GeneratedAt)
	}
	byID := map[string]Reservoir{}
	for _, reservoir := range snapshot.Reservoirs {
		byID[reservoir.ID] = reservoir
	}

	holding := byID["akathimuruppu"]
	if holding.Status != StatusHolding || *holding.StoragePercent != 44.6 || !*holding.StorageFresh ||
		holding.Flow != nil || *holding.District != "Mannar" || holding.River != nil || holding.Kind != "major" {
		t.Fatalf("akathimuruppu = %+v", holding)
	}

	// Spilling, with what left over the spillway and which agency said so.
	spilling := byID["denagama"]
	if spilling.Status != StatusSpilling || spilling.Flow == nil || spilling.Flow.Kind != "spill" ||
		*spilling.Flow.Cusecs != 17.22 || spilling.Flow.Agency != "Irrigation Department" ||
		spilling.Flow.Day.Format(time.DateOnly) != "2026-11-02" {
		t.Fatalf("denagama = %+v, flow %+v", spilling, spilling.Flow)
	}

	// A Mahaweli Authority downstream flow is carried as what it is.
	if flow := byID["polgolla"].Flow; flow == nil || flow.Kind != "downstream" || flow.Agency != "Mahaweli Authority" {
		t.Fatalf("polgolla flow = %+v", flow)
	}
	// Water through the outlets of a reservoir the source calls holding.
	if flow := byID["ambakolawewa"].Flow; byID["ambakolawewa"].Status != StatusHolding || flow == nil || flow.Kind != "outlets" {
		t.Fatalf("ambakolawewa = %+v", byID["ambakolawewa"])
	}

	// A status of nodata with an old figure keeps the figure as not fresh.
	old := byID["alikota-ara"]
	if old.Status != StatusNoData || *old.StoragePercent != 68.4 || *old.StorageFresh ||
		old.ReadingDay.Format(time.DateOnly) != "2026-10-29" {
		t.Fatalf("alikota-ara = %+v", old)
	}
	// Nothing is known: unknown stays null, never zero.
	empty := byID["kandiyapita-right-bank"]
	if empty.StoragePercent != nil || empty.StorageFresh != nil || empty.Agency != nil || empty.Status != StatusNoData {
		t.Fatalf("kandiyapita-right-bank = %+v", empty)
	}
	// Mahaweli reservoirs come with no district.
	if byID["victoria"].District != nil {
		t.Fatalf("victoria district = %v", *byID["victoria"].District)
	}
}

func TestDecodeReservoirsRejectsAndCopes(t *testing.T) {
	if _, err := DecodeReservoirs([]byte(`{`)); err == nil {
		t.Fatal("broken JSON was accepted")
	}
	if _, err := DecodeReservoirs([]byte(`{"reservoirs":[]}`)); err == nil {
		t.Fatal("a response without generatedAt was accepted")
	}
	snapshot, err := DecodeReservoirs([]byte(`{"generatedAt":"2026-11-02T19:39:21.448Z","reservoirs":[
		{"id":"","name":"No id"},
		{"id":"new","name":"New","type":"medium","status":"overflowing","storagePercent":12,"flow":{"kind":"spill","cusecs":-4,"source":"X","day":"soon"},"readingDay":"yesterday"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Skipped != 1 || len(snapshot.Reservoirs) != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	unknown := snapshot.Reservoirs[0]
	if unknown.Status != StatusNoData || unknown.ReadingDay != nil || unknown.Flow.Cusecs != nil || unknown.Flow.Day != nil {
		t.Fatalf("unknown values were not treated as unknown: %+v, flow %+v", unknown, unknown.Flow)
	}
}

func TestDecodeLevels(t *testing.T) {
	snapshot, err := DecodeLevels(read(t, "levels.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Reports) != 9 || snapshot.Skipped != 0 {
		t.Fatalf("reports %d, skipped %d", len(snapshot.Reports), snapshot.Skipped)
	}
	byName := map[string]Report{}
	for _, report := range snapshot.Reports {
		byName[report.Name] = report
	}

	// Marks in feet come back to the agency's figures in the published unit.
	pugoda := byName["Pugoda"]
	if pugoda.Unit != "m" || *pugoda.AlertLevel != 8 || *pugoda.MinorFloodLevel != 10 || *pugoda.MajorFloodLevel != 11.5 {
		t.Fatalf("pugoda marks = %v %v %v", *pugoda.AlertLevel, *pugoda.MinorFloodLevel, *pugoda.MajorFloodLevel)
	}
	if *pugoda.Level != 3.963 {
		t.Fatalf("pugoda level = %v", *pugoda.Level)
	}
	// Floating-point noise in a published level is removed.
	if level := *byName["Karawanella Bridge"].Level; level != 2.6 {
		t.Fatalf("karawanella level = %v", level)
	}
	// A station in feet stays in feet.
	street := byName["Nagalagam Street"]
	if street.Unit != "ft" || *street.AlertLevel != 4 || *street.MinorFloodLevel != 5 || *street.MajorFloodLevel != 7 || *street.Level != 1.6 {
		t.Fatalf("nagalagam = %+v", street)
	}
	// No published marks: no thresholds, a level all the same.
	kuru := byName["Kuru Ganga (Kuruvita)"]
	if kuru.AlertLevel != nil || kuru.Unit != "m" || *kuru.Level != 2.24 {
		t.Fatalf("kuru = %+v", kuru)
	}
	// An unknown unit, with a level of 0 and no marks.
	ampara := byName["Ampara"]
	if ampara.Unit != UnitUnknown || *ampara.Level != 0 || ampara.AlertLevel != nil || ampara.NextHour != nil {
		t.Fatalf("ampara = %+v", ampara)
	}
	// A gauge that has gone quiet still has a position, and no reading.
	quiet := byName["Calidonia"]
	if quiet.Level != nil || quiet.ObservedAt != nil || quiet.Unit != UnitUnknown || quiet.Lat == 0 {
		t.Fatalf("calidonia = %+v", quiet)
	}
	// The forecast is kept in feet as published, with its range and method.
	forecast := byName["Dunamale"].NextHour
	if forecast == nil || forecast.LevelFt != 5.92 || *forecast.LowFt != 5.79 || *forecast.HighFt != 6.06 || *forecast.Method != "model" {
		t.Fatalf("dunamale forecast = %+v", forecast)
	}
	// A trend estimate has no range.
	if forecast := byName["Wattala"].NextHour; forecast == nil || forecast.LowFt != nil || *forecast.Method != "trend" {
		t.Fatalf("wattala forecast = %+v", forecast)
	}
}

func TestMarksConvertsAndRejects(t *testing.T) {
	floats := func(value float64) *float64 { return &value }
	cases := []struct {
		name                string
		alert, minor, major *float64
		unit                string
		want                *[3]float64
	}{
		{"metres", floats(13.12), floats(19.69), floats(24.61), "m", &[3]float64{4, 6, 7.5}},
		{"feet are kept", floats(4), floats(5), floats(7), "ft", &[3]float64{4, 5, 7}},
		{"a missing mark", floats(13.12), nil, floats(24.61), "m", nil},
		{"zero", floats(0), floats(19.69), floats(24.61), "m", nil},
		{"out of order", floats(24.61), floats(19.69), floats(13.12), "m", nil},
		{"no unit to convert to", floats(13.12), floats(19.69), floats(24.61), UnitUnknown, nil},
		{"not a number", floats(math.NaN()), floats(19.69), floats(24.61), "m", nil},
	}
	for _, test := range cases {
		a, b, c := Marks(test.alert, test.minor, test.major, test.unit)
		if test.want == nil {
			if a != nil || b != nil || c != nil {
				t.Errorf("%s: got %v %v %v, want none", test.name, a, b, c)
			}
			continue
		}
		if a == nil || b == nil || c == nil || *a != test.want[0] || *b != test.want[1] || *c != test.want[2] {
			t.Errorf("%s: got %v %v %v, want %v", test.name, a, b, c, *test.want)
		}
	}
}

// Every Irrigation Department threshold in metres survives the source's trip
// through feet rounded to 0.01 ft, so converting back loses nothing.
func TestFromFeetRoundTrips(t *testing.T) {
	for metres := 0.1; metres < 25; metres += 0.05 {
		want := math.Round(metres*100) / 100
		feet := math.Round(want/metresPerFoot*100) / 100
		got, ok := FromFeet(feet, "m")
		if !ok || got != want {
			t.Fatalf("%v m through %v ft came back as %v", want, feet, got)
		}
	}
}

func TestStationIDAndAliases(t *testing.T) {
	cases := map[string]string{
		"Ratnapura":             "rathnapura",
		"Rathnapura":            "rathnapura",
		"Kuru Ganga (Kuruvita)": "kuru-ganga-kuruvita",
		"Nagalagam Street":      "nagalagam-street",
	}
	for name, want := range cases {
		if got := StationID(Report{Name: name}); got != want {
			t.Errorf("StationID(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestPlanLeavesOtherSourcesAlone(t *testing.T) {
	snapshot, err := DecodeLevels(read(t, "levels.json"))
	if err != nil {
		t.Fatal(err)
	}
	known := []KnownStation{
		{ID: "rathnapura", Source: "irrigation-gauges", Lon: 80.39, Lat: 6.68},
		{ID: "ampara", Source: "irrigation-gauges", Lon: 81.67, Lat: 7.29},
		{ID: "dunamale", Source: "irrigation-gauges", Lon: 80.08, Lat: 7.11},
		{ID: "nagalagam-street", Source: "irrigation-gauges", Lon: 79.87, Lat: 6.96},
		{ID: "pugoda", Source: GaugesSource, Lon: 80.1, Lat: 6.97},
	}
	plan := Plan(snapshot.Reports, known)
	extra := map[string]bool{}
	for _, station := range plan.Extra {
		extra[station.ID] = true
	}
	for _, want := range []string{"karawanella-bridge", "kuru-ganga-kuruvita", "wattala", "calidonia", "pugoda"} {
		if !extra[want] {
			t.Errorf("%s was not planned as an extra station: %v", want, extra)
		}
	}
	for _, owned := range []string{"rathnapura", "ratnapura", "ampara", "dunamale", "nagalagam-street"} {
		if extra[owned] {
			t.Errorf("%s belongs to the Irrigation Department and must not be planned", owned)
		}
	}
	// The forecast of a department station is kept under the department's id.
	forecasts := map[string]bool{}
	for _, forecast := range plan.Forecasts {
		forecasts[forecast.StationID] = true
	}
	for _, want := range []string{"rathnapura", "dunamale", "nagalagam-street", "wattala"} {
		if !forecasts[want] {
			t.Errorf("no forecast planned for %s: %v", want, forecasts)
		}
	}
	if forecasts["ratnapura"] {
		t.Error("a forecast was filed under the alias")
	}
}

func TestPlanDropsReportOnTopOfAnotherSourcesStation(t *testing.T) {
	reports := []Report{
		{Name: "Hanwella Town", Slug: "hanwella-town", Lon: 80.0801, Lat: 6.9101},
		{Name: "Far Away", Slug: "far-away", Lon: 80.5, Lat: 7.5},
	}
	plan := Plan(reports, []KnownStation{{ID: "hanwella", Source: "irrigation-gauges", Lon: 80.08, Lat: 6.91}})
	if len(plan.Duplicates) != 1 || plan.Duplicates[0] != "Hanwella Town" {
		t.Fatalf("duplicates = %v", plan.Duplicates)
	}
	if len(plan.Extra) != 1 || plan.Extra[0].ID != "far-away" {
		t.Fatalf("extra = %+v", plan.Extra)
	}
}

func TestSnapshotDayIsSriLankan(t *testing.T) {
	// 19:39 UTC on 2 November is already 3 November in Sri Lanka.
	got := SnapshotDay(time.Date(2026, 11, 2, 19, 39, 0, 0, time.UTC))
	if got.Format(time.DateOnly) != "2026-11-03" {
		t.Fatalf("snapshot day = %v", got)
	}
}
