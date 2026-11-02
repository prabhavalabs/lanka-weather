package weather

import (
	"encoding/json"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
)

// testSnapshot builds a snapshot on a 3 × 3 lattice of 1° around 80° E,
// 7° N, with the corner (0, 2) masked. Temperatures rise 1 °C per degree
// east and 2 °C per degree north.
func testSnapshot(now time.Time) *Snapshot {
	lattice := grid.Lattice{Lon0: 80, Lat0: 7, Step: 1, Nx: 3, Ny: 3}
	size := lattice.Size()
	nan := float32(math.NaN())
	field := func(steps int, step time.Duration, from time.Time, value func(index, step int) float32) *Field {
		values := make([]float32, steps*size)
		for s := range steps {
			for index := range size {
				values[s*size+index] = value(index, s)
				if index == lattice.Index(0, 2) {
					values[s*size+index] = nan
				}
			}
		}
		return &Field{ValidFrom: from, Step: step, Steps: steps, Values: values}
	}
	temperature := func(index, step int) float32 {
		i, j := index%3, index/3
		return float32(20+i+2*j) + float32(step)/10
	}
	constant := func(value float32) func(int, int) float32 { return func(int, int) float32 { return value } }
	currentAt := now.Add(-20 * time.Minute)
	current := &Run{ID: 1, Source: openmeteo.SourceName, Model: openmeteo.Model, SourceTime: currentAt, FetchedAt: currentAt.Add(time.Minute),
		Fields: map[string]*Field{
			"temperature_2m":     field(1, 0, currentAt, temperature),
			"wind_speed_10m":     field(1, 0, currentAt, constant(10)),
			"wind_direction_10m": field(1, 0, currentAt, func(index, _ int) float32 { return []float32{350, 10}[index%2] }),
			"weather_code":       field(1, 0, currentAt, func(index, _ int) float32 { return float32(index) }),
			"precipitation":      field(1, 0, currentAt, constant(0.4)),
			"wind_gusts_10m":     field(1, 0, currentAt, func(index, _ int) float32 { return float32(30 + index) }),
		}}
	hourlyFrom := now.Truncate(24 * time.Hour).Add(-24 * time.Hour)
	dayStart := openmeteo.LocalDayStart(now)
	forecast := &Run{ID: 2, Source: openmeteo.SourceName, Model: openmeteo.Model, SourceTime: hourlyFrom, FetchedAt: now.Add(-time.Hour),
		Fields: map[string]*Field{
			openmeteo.HourlyPrefix + "temperature_2m":            field(216, time.Hour, hourlyFrom, temperature),
			openmeteo.HourlyPrefix + "precipitation_probability": field(216, time.Hour, hourlyFrom, func(_, step int) float32 { return float32(step % 100) }),
			openmeteo.HourlyPrefix + "uv_index":                  field(216, time.Hour, hourlyFrom, constant(7)),
			openmeteo.DailyPrefix + "temperature_2m_max":         field(7, 24*time.Hour, dayStart, func(index, step int) float32 { return 30 + float32(index) + float32(step) }),
			openmeteo.DailyPrefix + "temperature_2m_min":         field(7, 24*time.Hour, dayStart, func(index, _ int) float32 { return 20 + float32(index) }),
			openmeteo.DailyPrefix + "precipitation_sum":          field(7, 24*time.Hour, dayStart, constant(12)),
		}}
	return &Snapshot{Lattice: lattice, GridID: "test", Current: current, Forecast: forecast}
}

var testNow = time.Date(2026, 11, 2, 9, 40, 0, 0, time.UTC)

func TestCurrentGridShape(t *testing.T) {
	snapshot := testSnapshot(testNow)
	response, err := BuildCurrentGrid(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if response.Grid.Nx != 3 || response.Grid.Ny != 3 || len(response.Values["temperatureC"]) != 9 {
		t.Fatalf("grid = %+v", response.Grid)
	}
	body, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Values map[string][]*float64 `json:"values"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	temperatures := decoded.Values["temperatureC"]
	if temperatures[6] != nil || temperatures[0] == nil || *temperatures[0] != 20 || *temperatures[8] != 26 {
		t.Fatalf("temperatures = %v", temperatures)
	}
	// UV index and precipitation probability come from the forecast hour
	// nearest the current time (09:20 rounds to 09:00, step 33).
	if uv := decoded.Values["uvIndexMax"]; uv != nil {
		t.Fatal("daily values must not appear in the current grid")
	}
	if probability := decoded.Values["precipitationProbabilityPct"]; probability == nil || *probability[0] != 33 {
		t.Fatalf("precipitation probability = %v", probability)
	}
	if _, ok := decoded.Values["pressureHpa"]; ok {
		t.Fatal("a variable the run lacks must be left out")
	}
}

func TestPointInterpolation(t *testing.T) {
	snapshot := testSnapshot(testNow)
	point, err := BuildPoint(snapshot, 80.5, 7.25, testNow, PointOptions{Hours: 72, Days: 7, CurrentStaleAfter: 3 * time.Hour, ForecastStaleAfter: 12 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	// 20 + 0.5 + 2 × 0.25 = 21.
	if got := float64(point.Current["temperatureC"].(Value)); math.Abs(got-21) > 1e-4 {
		t.Fatalf("temperature = %v, want 21", got)
	}
	// Directions of 350° and 10° average to north, not south.
	if got := float64(point.Current["windDirectionDeg"].(Value)); got != 0 && got != 360 {
		t.Fatalf("direction = %v, want 0", got)
	}
	// The weather code is the nearest node's: (0, 0) has the largest weight.
	if got := point.Current["weatherCode"].(Value); got != 0 && got != 1 {
		t.Fatalf("weather code = %v", got)
	}
	if point.Meta.CurrentStale || point.Meta.ForecastStale {
		t.Fatal("fresh data must not be stale")
	}

	times := point.Hourly["times"].([]time.Time)
	if len(times) != 72 || !times[0].Equal(time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("hourly from %v, %d hours", times[0], len(times))
	}
	if got := float64(point.Hourly["temperatureC"].(Values)[1]); math.Abs(got-(21+3.4)) > 1e-4 {
		t.Fatalf("hourly temperature at step 34 = %v", got)
	}
	dates := point.Daily["dates"].([]string)
	if len(dates) != 7 || dates[0] != "2026-11-02" {
		t.Fatalf("dates = %v", dates)
	}
	if got := float64(point.Daily["precipitationSumMm"].(Values)[0]); got != 12 {
		t.Fatalf("daily precipitation = %v", got)
	}

	// Beside the masked corner the present corners carry the value.
	near, err := BuildPoint(snapshot, 80.1, 8.9, testNow, PointOptions{Hours: 1, Days: 1})
	if err != nil || math.IsNaN(float64(near.Current["temperatureC"].(Value))) {
		t.Fatalf("point beside a masked node = %v, %v", near, err)
	}
	if _, err := BuildPoint(snapshot, 79.5, 7.5, testNow, PointOptions{}); err != ErrOutsideGrid {
		t.Fatalf("outside the lattice = %v", err)
	}
}

func TestPointMarksStaleData(t *testing.T) {
	snapshot := testSnapshot(testNow)
	later := testNow.Add(5 * time.Hour)
	point, err := BuildPoint(snapshot, 80.5, 7.5, later, PointOptions{Hours: 1, Days: 1, CurrentStaleAfter: 3 * time.Hour, ForecastStaleAfter: 12 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if !point.Meta.CurrentStale || point.Meta.ForecastStale {
		t.Fatalf("stale = %v %v", point.Meta.CurrentStale, point.Meta.ForecastStale)
	}
}

func TestValueEncoding(t *testing.T) {
	body, _ := json.Marshal(Values{1.24, Value(math.NaN()), -3, 0.05})
	if string(body) != "[1.2,null,-3,0.1]" {
		t.Fatalf("values = %s", body)
	}
}

func TestAggregateRegions(t *testing.T) {
	snapshot := testSnapshot(testNow)
	regions := []Region{
		{ID: "LK-1", Name: "Large", Nodes: []int{0, 1, 3, 4}, Weights: []float64{1, 1, 1, 0.05}},
		{ID: "LK-11", Name: "Small", Lon: 80.5, Lat: 7.25},
		{ID: "LK-99", Name: "Far away", Lon: 70, Lat: 7},
	}
	response := AggregateRegions(snapshot, "district", regions, testNow)
	large, small, far := response.Regions[0], response.Regions[1], response.Regions[2]
	if large.Method != "cells" || large.Cells != 4 {
		t.Fatalf("large = %+v", large)
	}
	// Mean of 20, 21, 22 and 23 × 0.05, by share: (63 + 1.15) / 3.05.
	if math.Abs(float64(large.TemperatureC)-(63+1.15)/3.05) > 1e-4 {
		t.Fatalf("mean temperature = %v", large.TemperatureC)
	}
	// Node 4 is a sliver: it counts towards the mean, not the extremes.
	if large.TemperatureMaxC != 33 || large.TemperatureMinC != 20 || large.WindGustsMaxKmh != 33 {
		t.Fatalf("extremes = %v %v %v", large.TemperatureMinC, large.TemperatureMaxC, large.WindGustsMaxKmh)
	}
	if small.Method != "point" || math.Abs(float64(small.TemperatureC)-21) > 1e-4 || small.PrecipitationTodayMm != 12 {
		t.Fatalf("small = %+v", small)
	}
	if !math.IsNaN(float64(far.TemperatureC)) {
		t.Fatalf("a region outside the grid has no values: %+v", far)
	}
	if response.ValidAt == nil || response.ForecastFetchedAt == nil {
		t.Fatal("times missing")
	}
}

func TestWithinParent(t *testing.T) {
	snapshot := testSnapshot(testNow)
	regions := []Region{
		{ID: "LK-1103005", Name: "Fort", ProvinceID: "LK-1", DistrictID: "LK-11", DSID: "LK-1103", Lon: 79.85, Lat: 6.93},
		{ID: "LK-1106010", Name: "Kotte", ProvinceID: "LK-1", DistrictID: "LK-11", DSID: "LK-1106", Lon: 79.9, Lat: 6.89},
		{ID: "LK-2103001", Name: "Kandy", ProvinceID: "LK-2", DistrictID: "LK-21", DSID: "LK-2103", Lon: 80.63, Lat: 7.29},
	}
	all := AggregateRegions(snapshot, "gn", regions, testNow)
	cases := map[string][]string{
		"LK-1":    {"LK-1103005", "LK-1106010"},
		"LK-11":   {"LK-1103005", "LK-1106010"},
		"LK-1103": {"LK-1103005"},
		"LK-21":   {"LK-2103001"},
		"LK-3":    {},
	}
	for parent, want := range cases {
		limited := WithinParent(all, parent)
		got := []string{}
		for _, region := range limited.Regions {
			got = append(got, region.ID)
		}
		if limited.Parent != parent || !slices.Equal(got, want) {
			t.Errorf("within %s = %v (parent %q), want %v", parent, got, limited.Parent, want)
		}
	}
	if len(all.Regions) != 3 || all.Parent != "" {
		t.Fatal("limiting must not change the cached aggregate")
	}
}

func TestValidTime(t *testing.T) {
	cases := map[string]string{
		"2026-11-02T10:29:00Z": "2026-11-02T09:00:00Z",
		"2026-11-02T10:31:00Z": "2026-11-02T12:00:00Z",
		"2026-11-02T22:45:00Z": "2026-11-03T00:00:00Z",
	}
	for in, want := range cases {
		at, _ := time.Parse(time.RFC3339, in)
		if got := ValidTime(at).Format(time.RFC3339); got != want {
			t.Errorf("ValidTime(%s) = %s, want %s", in, got, want)
		}
	}
}
