//go:build integration

package environment_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/environment"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

var now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

// fakeAPI answers like the air-quality and marine APIs: one location per
// coordinate pair, in order, a list.
func fakeAPI(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		count := len(strings.Split(query.Get("latitude"), ","))
		var locations []string
		for range count {
			var fields []string
			for _, variable := range strings.Split(query.Get("current"), ",") {
				fields = append(fields, fmt.Sprintf("%q:60", variable))
			}
			locations = append(locations, fmt.Sprintf(`{"current":{"time":%d,%s}}`, now.Unix(), strings.Join(fields, ",")))
		}
		_, _ = w.Write([]byte("[" + strings.Join(locations, ",") + "]"))
	}))
}

func TestEnvironmentCollectorsStoreAndServe(t *testing.T) {
	pool := dbtest.Open(t, "weather_runs")
	ctx := context.Background()
	server := fakeAPI(t)
	defer server.Close()
	client := source.NewClient("test")

	service := &environment.Service{Pool: pool, AirStaleAfter: 9 * time.Hour, SeaStaleAfter: 9 * time.Hour, Now: func() time.Time { return now }}
	if _, _, err := service.AirJSON(ctx); err != weather.ErrNoData {
		t.Fatalf("before collection = %v", err)
	}
	if air, err := service.AirAt(ctx, 79.9, 6.9); air != nil || err != nil {
		t.Fatalf("air before collection = %+v, %v", air, err)
	}

	air := &openmeteo.EnvironmentCollector{Pool: pool, Client: client, Grid: grid.AirQuality(), Kind: openmeteo.KindAirQuality, Endpoint: server.URL, NoPacing: true}
	sea := &openmeteo.EnvironmentCollector{Pool: pool, Client: client, Grid: grid.Sea(), Kind: openmeteo.KindMarine, Endpoint: server.URL, NoPacing: true}
	for _, collector := range []*openmeteo.EnvironmentCollector{air, sea} {
		if _, err := collector.Collect(ctx, now); err != nil {
			t.Fatal(err)
		}
	}

	plain, gzipped, err := service.SeaJSON(ctx)
	if err != nil || len(gzipped) == 0 {
		t.Fatalf("sea json = %d bytes, %v", len(gzipped), err)
	}
	var body struct {
		Source string `json:"source"`
		Grid   struct {
			Nx int `json:"nx"`
		} `json:"grid"`
		Values map[string][]*float64 `json:"values"`
		Land   [][][2]float64        `json:"land"`
	}
	if err := json.Unmarshal(plain, &body); err != nil {
		t.Fatal(err)
	}
	if body.Source != openmeteo.SourceMarine || body.Grid.Nx != grid.SeaLattice.Nx || len(body.Land) == 0 || len(body.Values["waveHeightM"]) != grid.SeaLattice.Size() {
		t.Fatalf("body = %s...", plain[:200])
	}

	point, err := service.AirAt(ctx, 79.9, 6.9)
	if err != nil || point == nil || point.USAQI != 60 || point.Category != environment.AQIModerate || point.Stale || !point.Estimate || point.PM25 == nil {
		t.Fatalf("air point = %+v, %v", point, err)
	}
	coast, err := service.SeaAt(ctx, 79.86, 6.92)
	if err != nil || coast == nil || coast.WaveHeightM != 60 || coast.DistanceKm > 75 {
		t.Fatalf("coast = %+v, %v", coast, err)
	}
	inland, err := service.SeaAt(ctx, 80.77, 7.3)
	if err != nil || inland != nil {
		t.Fatalf("inland = %+v, %v", inland, err)
	}

	later := &environment.Service{Pool: pool, AirStaleAfter: 9 * time.Hour, SeaStaleAfter: 9 * time.Hour, Now: func() time.Time { return now.Add(10 * time.Hour) }}
	if old, err := later.AirAt(ctx, 79.9, 6.9); err != nil || old == nil || !old.Stale {
		t.Fatalf("old air = %+v, %v", old, err)
	}
}
