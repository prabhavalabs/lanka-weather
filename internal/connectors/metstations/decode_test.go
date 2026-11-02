package metstations

import (
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

func TestThreeHourlyTimesAreLocal(t *testing.T) {
	decoded, err := DecodeThreeHourly(read(t, "alert_data_3h.json"), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	// 08:30 in Colombo is 03:00 UTC, a WMO synoptic hour.
	want := time.Date(2026, 10, 29, 3, 0, 0, 0, time.UTC)
	if !decoded.Newest.Equal(want) {
		t.Fatalf("newest = %s, want %s", decoded.Newest, want)
	}
	byTime := map[time.Time]Reading3h{}
	for _, reading := range decoded.Readings {
		if reading.ObservedAt.Minute() != 0 || reading.ObservedAt.Hour()%3 != 0 {
			t.Errorf("%s at %s is not a synoptic hour", reading.StationID, reading.ObservedAt)
		}
		if reading.StationID == "colombo" {
			byTime[reading.ObservedAt] = reading
		}
	}
	morning := byTime[time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)]   // 05:30 local
	afternoon := byTime[time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)] // 14:30 local
	if morning.TempC == nil || afternoon.TempC == nil || *morning.TempC >= *afternoon.TempC {
		t.Fatalf("05:30 should be cooler than 14:30: %+v, %+v", morning, afternoon)
	}
	if *morning.TempC != 24.8 || *morning.RainMm != 2.4 || *morning.HumidityPct != 91 {
		t.Fatalf("colombo at 05:30 = temp %v rain %v rh %v", *morning.TempC, *morning.RainMm, *morning.HumidityPct)
	}
}

func TestThreeHourlyResolvesSpellingsAndStampedMinutes(t *testing.T) {
	decoded, err := DecodeThreeHourly(read(t, "alert_data_3h.json"), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Unknown) != 0 {
		t.Fatalf("unknown = %v", decoded.Unknown)
	}
	ids := map[string]bool{}
	for _, reading := range decoded.Readings {
		ids[reading.StationID] = true
	}
	for _, id := range []string{"colombo", "nuwara-eliya", "mullaitivu", "katugastota"} {
		if !ids[id] {
			t.Errorf("no readings for %s", id)
		}
	}
	// A reading stamped 17:3x belongs to the 17:30 report.
	at, ok := parseSlot("20260101", "1737")
	if !ok || !at.Equal(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("17:37 = %s, %v", at, ok)
	}
	for _, clock := range []string{"1200", "0930", "1729", "abcd", "17"} {
		if _, ok := parseSlot("20260101", clock); ok {
			t.Errorf("%s should not be a report slot", clock)
		}
	}
}

func TestThreeHourlyKeepsOnlyRecentAndDropsImpossibleValues(t *testing.T) {
	payload := []byte(`{"event":"weather_report_3h","event_data":{"Colombo":{"20261029":{
		"0830":{"rain_mm":-1,"temp_c":99,"rh":1.7},
		"0530":{"rain_mm":0,"temp_c":25,"rh":0.9}}}}}`)
	decoded, err := DecodeThreeHourly(payload, time.Date(2026, 10, 29, 2, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Readings) != 1 {
		t.Fatalf("readings = %+v", decoded.Readings)
	}
	reading := decoded.Readings[0]
	if reading.RainMm != nil || reading.TempC != nil || reading.HumidityPct != nil {
		t.Fatalf("impossible values kept: %+v", reading)
	}
	if !decoded.Newest.Equal(time.Date(2026, 10, 29, 3, 0, 0, 0, time.UTC)) {
		t.Fatalf("newest = %s", decoded.Newest)
	}
	if _, err := DecodeThreeHourly([]byte(`{"event":"other"}`), time.Time{}); err == nil {
		t.Fatal("expected an error for another kind of file")
	}
}

func TestDecodeDailyReadsARangeRequestsTail(t *testing.T) {
	decoded, err := DecodeDaily(read(t, "daily_tail.json"), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Readings) == 0 {
		t.Fatal("no rows")
	}
	last := decoded.Readings[len(decoded.Readings)-1]
	if last.StationID != "ayyiththimalai" || *last.RainMm != 63.5 {
		t.Fatalf("last = %+v", last)
	}
	// time_ut 1792693800 is 18:30 UTC on 22 October: midnight of the 23rd here.
	if got := last.Date.Format(time.DateOnly); got != "2026-10-23" {
		t.Fatalf("date = %s", got)
	}
	if !last.ObservedAt.Equal(time.Date(2026, 10, 23, 3, 0, 0, 0, time.UTC)) {
		t.Fatalf("observed = %s", last.ObservedAt)
	}
}

func TestDecodeDailyKeepsTheFirstOfADuplicatedStation(t *testing.T) {
	payload := []byte(`[{"id":"Hanwella","time_ut":1792693800.0,"rain_mm":76.0,"temp_min_c":null,"temp_max_c":null},
		{"id":"Hanwella","time_ut":1792693800.0,"rain_mm":69.2,"temp_min_c":null,"temp_max_c":null}]`)
	decoded, err := DecodeDaily(payload, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Readings) != 1 || decoded.Duplicates != 1 || *decoded.Readings[0].RainMm != 76 {
		t.Fatalf("decoded = %+v", decoded)
	}
	if _, err := DecodeDaily([]byte(`  "rain_mm": 3 }`), time.Time{}); err == nil {
		t.Fatal("a tail with no whole row is an error")
	}
}

func TestDecodePlacesLeavesOutBadPositions(t *testing.T) {
	places, err := DecodePlaces(read(t, "places.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"colombo", "negombo", "colombo-fort"} {
		if _, ok := places[id]; !ok {
			t.Errorf("%s should have a position", id)
		}
	}
	// Geocoded to (0, 0): Victoria, Laxapana, Eladuwa Estate, Ayyiththimalai.
	// To the centre of the country: Vogan, Port Authority, Melkopiwatta.
	for _, id := range []string{"victoria", "laxapana", "eladuwa-estate", "ayyiththimalai", "vogan", "port-authority", "melkopiwatta"} {
		if position, ok := places[id]; ok {
			t.Errorf("%s has an unusable position %+v", id, position)
		}
	}
}

func TestKnownStationsAreOnSriLankaAndNearTheirGeocodedTowns(t *testing.T) {
	stations := Known()
	if len(stations) != 24 {
		t.Fatalf("known stations = %d, want the 24 of the 3-hourly report", len(stations))
	}
	places, err := DecodePlaces(read(t, "places.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, station := range stations {
		if !onSriLanka(station.Lat, station.Lon) {
			t.Errorf("%s at %v, %v is not on Sri Lanka", station.ID, station.Lat, station.Lon)
		}
		if place, ok := places[station.ID]; ok {
			if distance := distanceKm(station.Lat, station.Lon, place.Lat, place.Lon); distance > 7 {
				t.Errorf("%s is %.1f km from its geocoded town", station.ID, distance)
			}
		}
	}
}

func TestIDsResolveTheFeedsSpellings(t *testing.T) {
	for name, want := range map[string]string{
		"Mullativu":         "mullaitivu",
		"Mullaitivu":        "mullaitivu",
		"Monaragala":        "moneragala",
		"Moneragala":        "moneragala",
		"Kandy":             "katugastota",
		"Maha Illuppallama": "maha-illuppallama",
		"Nuwara Eliya":      "nuwara-eliya",
		"Dambulla Tec.":     "dambulla-tec",
		"":                  "",
	} {
		if got := ID(name); got != want {
			t.Errorf("ID(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestResolveMatchesTableThenGeocodedAndNamesTheRest(t *testing.T) {
	readings := []ReadingDaily{
		{StationID: "colombo", Name: "Colombo"},
		{StationID: "negombo", Name: "Negombo"},
		{StationID: "victoria", Name: "Victoria"},
		{StationID: "negombo", Name: "Negombo"},
	}
	stations := Resolve(readings, map[string]Position{"negombo": {Lat: 7.2, Lon: 79.87}})
	if !stations.Matched["colombo"] || !stations.Matched["negombo"] || stations.Matched["victoria"] {
		t.Fatalf("matched = %v", stations.Matched)
	}
	if len(stations.Geocoded) != 1 || stations.Geocoded[0].PositionSource != PositionGeocoded {
		t.Fatalf("geocoded = %+v", stations.Geocoded)
	}
	if len(stations.Unmatched) != 1 || stations.Unmatched[0] != "Victoria" {
		t.Fatalf("unmatched = %v", stations.Unmatched)
	}
}
