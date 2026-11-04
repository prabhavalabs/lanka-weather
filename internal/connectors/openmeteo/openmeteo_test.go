package openmeteo

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestDecodeCurrentList(t *testing.T) {
	locations, err := Decode(KindCurrent, read(t, "current_two_locations.json"), 2)
	if err != nil {
		t.Fatal(err)
	}
	kandy, galle := locations[0], locations[1]
	if !kandy.CurrentTime.Equal(time.Date(2026, 11, 2, 13, 45, 0, 0, time.UTC)) {
		t.Fatalf("time = %v", kandy.CurrentTime)
	}
	if kandy.Current["temperature_2m"] != 22.7 || *kandy.Elevation != 545 || kandy.Current["weather_code"] != 51 {
		t.Fatalf("kandy = %+v", kandy)
	}
	if !math.IsNaN(float64(galle.Current["pressure_msl"])) {
		t.Fatal("null must decode as NaN")
	}
	if _, err := Decode(KindCurrent, read(t, "current_two_locations.json"), 3); err == nil {
		t.Fatal("a response with fewer locations than asked must fail")
	}
}

func TestDecodeForecastObject(t *testing.T) {
	locations, err := Decode(KindForecast, read(t, "forecast_one_location.json"), 1)
	if err != nil {
		t.Fatal(err)
	}
	location := locations[0]
	if len(location.Times) != 3 || location.Times[1].Sub(location.Times[0]) != time.Hour {
		t.Fatalf("times = %v", location.Times)
	}
	if got := location.Hourly["weather_code"][2]; got != 61 {
		t.Fatalf("weather code = %v", got)
	}
	if !math.IsNaN(float64(location.Hourly["precipitation_probability"][2])) {
		t.Fatal("null must decode as NaN")
	}
}

func TestDecodeErrors(t *testing.T) {
	cases := map[string]string{
		"api error":     `{"error":true,"reason":"Latitude must be in range of -90 to 90°."}`,
		"not json":      `<html>`,
		"short series":  `{"hourly":{"time":[1,2],"temperature_2m":[1]}}`,
		"no current":    `{"latitude":1}`,
		"string number": `{"current":{"time":1793627100,"temperature_2m":"warm"}}`,
	}
	for name, payload := range cases {
		kind := KindForecast
		if name == "no current" || name == "string number" {
			kind = KindCurrent
		}
		if _, err := Decode(kind, []byte(payload), 1); err == nil {
			t.Errorf("%s: expected an error", name)
		} else if name == "api error" && !strings.Contains(err.Error(), "Latitude") {
			t.Errorf("api error should carry the reason: %v", err)
		}
	}
}

func TestChunksSeparateLandAndSea(t *testing.T) {
	nodes := grid.SriLanka().Nodes
	chunks := Chunks(nodes, 50)
	seen := map[int]bool{}
	for _, chunk := range chunks {
		if len(chunk.Positions) == 0 || len(chunk.Positions) > 50 {
			t.Fatalf("chunk of %d nodes", len(chunk.Positions))
		}
		for _, position := range chunk.Positions {
			if nodes[position].Land != chunk.Land || seen[position] {
				t.Fatalf("node %d misplaced", position)
			}
			seen[position] = true
		}
	}
	if len(seen) != len(nodes) {
		t.Fatalf("chunks cover %d of %d nodes", len(seen), len(nodes))
	}
	query := Query(KindForecast, nodes, chunks[0])
	if query.Get("cell_selection") != "land" || query.Get("past_days") != "1" || query.Get("forecast_days") != "8" ||
		strings.Count(query.Get("latitude"), ",") != len(chunks[0].Positions)-1 || query.Get("timeformat") != "unixtime" {
		t.Fatalf("query = %v", query)
	}
	if strings.Count(query.Get("hourly"), ",") != len(HourlyVariables)-1 || query.Has("current") {
		t.Fatalf("forecast query asks for %q", query.Get("hourly"))
	}
	if current := Query(KindCurrent, nodes, chunks[len(chunks)-1]); current.Get("cell_selection") != "sea" || current.Has("hourly") {
		t.Fatalf("current query = %v", current)
	}
}

// The default grid and cadences must stay inside Open-Meteo's free tier
// (10,000 weighted calls a day) with room for retries and a second run.
func TestDefaultsStayWithinTheFreeTier(t *testing.T) {
	nodes := len(grid.SriLanka().Nodes)
	if got := Weight(KindCurrent, 1); got != 1 {
		t.Fatalf("a current node weighs %v calls; keep current variables at ten or fewer", got)
	}
	if got := Weight(KindForecast, 1); got > 1.25 {
		t.Fatalf("a forecast node weighs %v calls", got)
	}
	daily := DailyCalls(nodes, CurrentInterval, ForecastInterval)
	if daily > 7500 {
		t.Fatalf("defaults cost %.0f weighted calls a day; budget is 7,500 of 10,000", daily)
	}
	if hourly := Weight(KindCurrent, nodes) + Weight(KindForecast, nodes); hourly > 5000 {
		t.Fatalf("an hour with both runs costs %.0f calls", hourly)
	}
	t.Logf("%d nodes: %.0f weighted calls a day", nodes, daily)
}

func TestCurrentFieldsDeriveDewPointAndDaylight(t *testing.T) {
	locations, err := Decode(KindCurrent, read(t, "current_two_locations.json"), 2)
	if err != nil {
		t.Fatal(err)
	}
	nodes := []grid.Node{{Index: 0, Lon: 80.6, Lat: 7.2}, {Index: 1, Lon: 80.0, Lat: 6.0}}
	at, fields, err := CurrentFields(nodes, locations)
	if err != nil {
		t.Fatal(err)
	}
	if at.Hour() != 13 || len(fields) != len(CurrentVariables)+2 {
		t.Fatalf("at %v, %d fields", at, len(fields))
	}
	byName := map[string]Field{}
	for _, field := range fields {
		byName[field.Variable] = field
	}
	if dew := byName[VariableDewPoint].Values[0]; math.Abs(float64(dew)-22.5) > 0.2 {
		t.Fatalf("dew point at 22.7 °C and 99 %% = %v", dew)
	}
	// 13:45 UTC is 19:15 local, after sunset.
	if day := byName[VariableIsDay].Values; day[0] != 0 || day[1] != 0 {
		t.Fatalf("is_day = %v, want night", day)
	}
}

// hourlyLocation builds a node's forecast of `hours` hours from `first`,
// with every variable set by fill.
func hourlyLocation(first time.Time, hours int, fill func(variable string, at time.Time) float32) Location {
	location := Location{Hourly: map[string][]float32{}}
	for step := range hours {
		location.Times = append(location.Times, first.Add(time.Duration(step)*time.Hour))
	}
	for _, variable := range HourlyVariables {
		series := make([]float32, hours)
		for step, at := range location.Times {
			series[step] = fill(variable, at)
		}
		location.Hourly[variable] = series
	}
	return location
}

func TestForecastFieldsAndDailySummary(t *testing.T) {
	first := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) // yesterday 00Z
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)  // 17:30 local on 2 November
	fill := func(variable string, at time.Time) float32 {
		local := at.Add(LocalOffset)
		switch variable {
		case "temperature_2m":
			return float32(20 + local.Hour()/2) // 20 at midnight to 31 at 23:00
		case "precipitation":
			return 1
		case "wind_direction_10m":
			if local.Hour()%2 == 0 {
				return 350
			}
			return 20
		case "weather_code":
			if local.Hour() == 15 {
				return 95
			}
			return 3
		}
		return 10
	}
	nodes := []grid.Node{{Index: 0, Lon: 80.6, Lat: 7.3}, {Index: 1, Lon: 79.9, Lat: 6.9}}
	locations := []Location{hourlyLocation(first, 216, fill), hourlyLocation(first, 216, fill)}
	// The second node misses most of its last local day.
	for step := 216 - 40; step < 216; step++ {
		locations[1].Hourly["temperature_2m"][step] = nan
	}
	validFrom, fields, err := ForecastFields(nodes, locations, now)
	if err != nil {
		t.Fatal(err)
	}
	if !validFrom.Equal(first) {
		t.Fatalf("valid from %v", validFrom)
	}
	byName := map[string]Field{}
	for _, field := range fields {
		byName[field.Variable] = field
	}
	hourly := byName[HourlyPrefix+"temperature_2m"]
	if hourly.Steps != 216 || len(hourly.Values) != 216*2 || hourly.Step != time.Hour {
		t.Fatalf("hourly temperature = %d steps, %d values", hourly.Steps, len(hourly.Values))
	}
	if _, ok := byName[HourlyPrefix+VariableDewPoint]; !ok {
		t.Fatal("hourly dew point missing")
	}

	daily := byName[DailyPrefix+"temperature_2m_max"]
	wantStart := time.Date(2026, 11, 1, 18, 30, 0, 0, time.UTC) // local midnight starting 2 November
	if daily.Steps != DailyDays || !daily.ValidFrom.Equal(wantStart) || daily.Step != 24*time.Hour {
		t.Fatalf("daily = %d steps from %v", daily.Steps, daily.ValidFrom)
	}
	value := func(name string, day, node int) float32 { return byName[DailyPrefix+name].Values[day*2+node] }
	if value("temperature_2m_max", 0, 0) != 31 || value("temperature_2m_min", 0, 0) != 20 {
		t.Fatalf("day 1 range = %v to %v", value("temperature_2m_min", 0, 0), value("temperature_2m_max", 0, 0))
	}
	if value("precipitation_sum", 3, 1) != 24 {
		t.Fatalf("precipitation sum = %v, want 24 one-millimetre hours", value("precipitation_sum", 3, 1))
	}
	if value("weather_code", 0, 0) != 95 {
		t.Fatalf("daily weather code = %v, want the day's highest", value("weather_code", 0, 0))
	}
	if direction := value("wind_direction_10m_dominant", 0, 0); direction != 5 {
		t.Fatalf("dominant direction of 350° and 20° = %v, want 5°", direction)
	}
	if !math.IsNaN(float64(value("temperature_2m_max", 6, 1))) || math.IsNaN(float64(value("temperature_2m_max", 6, 0))) {
		t.Fatal("a day with too few hours has no summary; a whole one has")
	}
}

func TestLocalDayStart(t *testing.T) {
	cases := map[string]string{
		"2026-11-02T12:00:00Z": "2026-11-01T18:30:00Z",
		"2026-11-02T18:29:00Z": "2026-11-01T18:30:00Z",
		"2026-11-02T18:30:00Z": "2026-11-02T18:30:00Z",
	}
	for in, want := range cases {
		at, _ := time.Parse(time.RFC3339, in)
		if got := LocalDayStart(at).Format(time.RFC3339); got != want {
			t.Errorf("LocalDayStart(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestEnvironmentBudgetStaysWellInsideTheFreeTier(t *testing.T) {
	air, sea := grid.AirQuality(), grid.Sea()
	extra := EnvironmentDailyCalls(len(air.Nodes), len(sea.Nodes), EnvironmentInterval)
	weather := DailyCalls(len(grid.SriLanka().Nodes), CurrentInterval, ForecastInterval)
	if extra > 1300 {
		t.Fatalf("air quality and sea cost %.0f calls a day", extra)
	}
	if total := weather + extra; total > 8500 {
		t.Fatalf("all Open-Meteo collecting costs %.0f calls a day, against a free tier of 10,000", total)
	}
}

func TestDecodeRecordedAirQuality(t *testing.T) {
	payload, err := os.ReadFile("testdata/air_quality_two_locations.json")
	if err != nil {
		t.Fatal(err)
	}
	locations, err := DecodeEnvironment(KindAirQuality, payload, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, location := range locations {
		if location.Values["us_aqi"] <= 0 || location.Values["pm2_5"] <= 0 || location.Time.IsZero() {
			t.Fatalf("location = %+v", location)
		}
	}
	at, fields, err := EnvironmentFields(KindAirQuality, locations)
	if err != nil || at.IsZero() || len(fields) != 2 || fields[0].Variable != "us_aqi" {
		t.Fatalf("fields = %+v at %s, %v", fields, at, err)
	}
}

func TestDecodeRecordedMarine(t *testing.T) {
	payload, err := os.ReadFile("testdata/marine_two_locations.json")
	if err != nil {
		t.Fatal(err)
	}
	locations, err := DecodeEnvironment(KindMarine, payload, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, location := range locations {
		if height := location.Values["wave_height"]; height <= 0 || height > 20 {
			t.Fatalf("wave height = %v", height)
		}
		if _, ok := location.Values["swell_wave_period"]; !ok {
			t.Fatalf("location = %+v", location)
		}
	}
}

func TestEnvironmentFailsOnAWholeMissingVariableOrWrongCount(t *testing.T) {
	missing := `{"current":{"time":1793000000,"us_aqi":null,"pm2_5":null}}`
	locations, err := DecodeEnvironment(KindAirQuality, []byte(missing), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(float64(locations[0].Values["us_aqi"])) {
		t.Fatalf("null should be NaN: %+v", locations[0])
	}
	if _, _, err := EnvironmentFields(KindAirQuality, locations); err == nil {
		t.Fatal("a run with no values at all must not replace a good one")
	}
	if _, err := DecodeEnvironment(KindMarine, []byte(`[{"current":{"time":1793000000}}]`), 2); err == nil {
		t.Fatal("expected an error for the wrong number of locations")
	}
	if _, err := DecodeEnvironment(KindMarine, []byte(`{"error":true,"reason":"bad"}`), 1); err == nil {
		t.Fatal("expected the API's error")
	}
}

func TestEnvironmentQueryAsksTheSeaForSeaCells(t *testing.T) {
	sea := grid.Sea()
	query := EnvironmentQuery(KindMarine, sea.Nodes, []int{0, 1})
	if query.Get("cell_selection") != "sea" || strings.Count(query.Get("latitude"), ",") != 1 {
		t.Fatalf("query = %v", query)
	}
	if !strings.Contains(query.Get("current"), "swell_wave_height") {
		t.Fatalf("current = %s", query.Get("current"))
	}
	air := EnvironmentQuery(KindAirQuality, sea.Nodes, []int{0})
	if air.Get("cell_selection") != "" || air.Get("current") != "us_aqi,pm2_5" {
		t.Fatalf("air query = %v", air)
	}
	chunks := environmentChunks(120, 50)
	if len(chunks) != 3 || len(chunks[2]) != 20 || chunks[2][19] != 119 {
		t.Fatalf("chunks = %v", chunks)
	}
}
