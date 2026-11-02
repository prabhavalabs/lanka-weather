//go:build integration

package openmeteo_test

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// now is 15:30 local on 2 November 2026.
var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

// fakeOpenMeteo answers like the forecast API: one location per coordinate
// pair, in order, with the temperature 30 - (lat - 6) and 30 % humidity
// more on land. It counts requests.
func fakeOpenMeteo(t *testing.T) (*httptest.Server, *atomic.Int32) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		query := r.URL.Query()
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "lanka-weather/") {
			http.Error(w, `{"error":true,"reason":"no user agent"}`, http.StatusBadRequest)
			return
		}
		lats, lons := strings.Split(query.Get("latitude"), ","), strings.Split(query.Get("longitude"), ",")
		if len(lats) != len(lons) {
			http.Error(w, `{"error":true,"reason":"mismatch"}`, http.StatusBadRequest)
			return
		}
		land := query.Get("cell_selection") == "land"
		var locations []map[string]any
		for index := range lats {
			lat, _ := strconv.ParseFloat(lats[index], 64)
			temperature := 30 - (lat - 6)
			humidity := 60.0
			if land {
				humidity = 90
			}
			location := map[string]any{"latitude": lat, "longitude": lons[index], "elevation": 10 * index}
			if query.Has("current") {
				current := map[string]any{"time": now.Add(-15 * time.Minute).Unix(), "interval": 900}
				for _, variable := range strings.Split(query.Get("current"), ",") {
					current[variable] = 1.0
				}
				current["temperature_2m"], current["relative_humidity_2m"] = temperature, humidity
				current["wind_direction_10m"], current["wind_speed_10m"] = 90, 18
				location["current"] = current
			} else {
				first := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
				hours := 24 * 9
				hourly := map[string]any{}
				times := make([]int64, hours)
				for step := range times {
					times[step] = first.Add(time.Duration(step) * time.Hour).Unix()
				}
				hourly["time"] = times
				for _, variable := range strings.Split(query.Get("hourly"), ",") {
					values := make([]any, hours)
					for step := range values {
						values[step] = 2.0
						if variable == "temperature_2m" {
							values[step] = temperature + float64(step%24)/10
						}
						if variable == "uv_index" && step == hours-1 {
							values[step] = nil
						}
					}
					hourly[variable] = values
				}
				location["hourly"] = hourly
			}
			locations = append(locations, location)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(locations)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func TestCollectStoresAndServesTheGrid(t *testing.T) {
	pool := dbtest.Open(t, "weather_runs", "weather_grid_nodes", "source_runs")
	ctx := context.Background()
	server, requests := fakeOpenMeteo(t)
	client := source.NewClient("https://example.org/test")
	weatherGrid := grid.SriLanka()
	collect := func(kind openmeteo.Kind, at time.Time) {
		t.Helper()
		collector := &openmeteo.Collector{Pool: pool, Client: client, Grid: weatherGrid, Kind: kind, Endpoint: server.URL, NoPacing: true}
		stored, err := collector.Collect(ctx, at)
		if err != nil {
			t.Fatal(err)
		}
		if stored == 0 {
			t.Fatal("nothing stored")
		}
	}
	collect(openmeteo.KindCurrent, now)
	collect(openmeteo.KindForecast, now)
	if got, want := int(requests.Load()), 2*len(openmeteo.Chunks(weatherGrid.Nodes, 50)); got != want {
		t.Fatalf("requests = %d, want %d", got, want)
	}
	var nodes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM weather_grid_nodes WHERE grid = $1 AND ST_Area(cell) > 0.039`, weatherGrid.ID).Scan(&nodes); err != nil {
		t.Fatal(err)
	}
	if nodes != len(weatherGrid.Nodes) {
		t.Fatalf("stored %d nodes with cells, want %d", nodes, len(weatherGrid.Nodes))
	}

	service := &weather.Service{Pool: pool, CurrentStaleAfter: 3 * time.Hour, ForecastStaleAfter: 12 * time.Hour, Now: func() time.Time { return now }}
	point, err := service.Point(ctx, 80.6337, 7.2906, 72, 7)
	if err != nil {
		t.Fatal(err)
	}
	temperature := float64(point.Current["temperatureC"].(weather.Value))
	if math.Abs(temperature-(30-(7.2906-6))) > 0.01 {
		t.Fatalf("Kandy temperature = %v", temperature)
	}
	if direction := point.Current["windDirectionDeg"].(weather.Value); direction != 90 {
		t.Fatalf("wind direction = %v", direction)
	}
	if dew := float64(point.Current["dewPointC"].(weather.Value)); math.IsNaN(dew) || dew >= temperature {
		t.Fatalf("dew point = %v", dew)
	}
	times := point.Hourly["times"].([]time.Time)
	if len(times) != 72 || !times[0].Equal(now) {
		t.Fatalf("hourly from %v (%d)", times[0], len(times))
	}
	if dates := point.Daily["dates"].([]string); len(dates) != 7 || dates[0] != "2026-11-02" {
		t.Fatalf("dates = %v", dates)
	}
	if sum := point.Daily["precipitationSumMm"].(weather.Values)[0]; sum != 48 {
		t.Fatalf("daily precipitation = %v, want 24 × 2 mm", sum)
	}

	plain, gzipped, err := service.CurrentGridJSON(ctx)
	if err != nil || len(gzipped) == 0 || len(gzipped) >= len(plain) {
		t.Fatalf("grid = %d bytes, %d gzipped, %v", len(plain), len(gzipped), err)
	}
	var currentGrid weather.CurrentGrid
	if err := json.Unmarshal(plain, &currentGrid); err != nil {
		t.Fatal(err)
	}
	if currentGrid.Grid.Nx != weatherGrid.Nx || len(currentGrid.Values["temperatureC"]) != weatherGrid.Size() {
		t.Fatalf("grid = %+v", currentGrid.Grid)
	}

	// A second current run replaces the first for readers; pruning a day
	// later keeps only the newest run of each kind.
	collect(openmeteo.KindCurrent, now.Add(time.Hour))
	collector := &openmeteo.Collector{Pool: pool, Client: client, Grid: weatherGrid, Kind: openmeteo.KindCurrent}
	if err := collector.Prune(ctx, now.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	ids, err := store.New(pool).LatestWeatherRunIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var runs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM weather_runs`).Scan(&runs); err != nil {
		t.Fatal(err)
	}
	if runs != 2 || ids.CurrentID == 0 || ids.ForecastID == 0 {
		t.Fatalf("after pruning: %d runs, ids %+v", runs, ids)
	}
}
