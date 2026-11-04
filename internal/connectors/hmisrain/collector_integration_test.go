//go:build integration

package hmisrain_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/hmisrain"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

func TestCollectStoresGaugesAndServesThem(t *testing.T) {
	pool := dbtest.Open(t, "rain_gauges", "rain_gauge_readings", "source_versions")
	ctx := context.Background()
	payload, err := os.ReadFile("testdata/rain_live.json")
	if err != nil {
		t.Fatal(err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)
	// 2 h 20 after the first gauge's newest reading, 3 h 20 after the second's.
	now := time.Date(2026, 11, 2, 20, 40, 0, 0, time.UTC)
	collector := &hmisrain.Collector{Pool: pool, Client: source.NewClient("test"), Endpoint: server.URL, Minimum: 3}
	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 6 {
		t.Fatalf("stored %d readings", stored)
	}
	if again, err := collector.Collect(ctx, now.Add(30*time.Minute)); err != nil || again != 0 || requests != 2 {
		t.Fatalf("an unchanged file: %d readings, %v, %d requests", again, err, requests)
	}

	service := &raingauges.Service{Pool: pool, StaleAfter: 3 * time.Hour, Now: func() time.Time { return now }}
	layer, err := service.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if layer.Meta.Total != 5 || layer.Meta.Fresh != 1 || layer.Meta.Stale != 4 || layer.Meta.Attribution != hmisrain.Attribution {
		t.Fatalf("meta = %+v", layer.Meta)
	}
	byID := map[string]raingauges.Properties{}
	for _, feature := range layer.Features {
		byID[feature.ID] = feature.Properties
	}
	fresh := byID["argtest-07"]
	if fresh.Stale || fresh.Rain24hMm == nil || *fresh.Rain24hMm != 12.5 || fresh.Band != raingauges.BandModerate || fresh.Institution != "Met Department" {
		t.Fatalf("fresh = %+v", fresh)
	}
	// A stale zero must never read as dry.
	old := byID["test-oya-testgama"]
	if !old.Stale || old.Rain24hMm != nil || old.Band != raingauges.BandUnknown || old.LastRain24hMm == nil || *old.LastRain24hMm != 0 {
		t.Fatalf("old = %+v", old)
	}
	if silent := byID["r99"]; !silent.Stale || silent.ObservedAt != nil || silent.Rain24hMm != nil {
		t.Fatalf("a gauge that never reported = %+v", silent)
	}

	detail, err := service.Gauge(ctx, "argtest-07")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Series) != 3 || detail.Series[2].Rain24hMm != 12.5 {
		t.Fatalf("series = %+v", detail.Series)
	}
	if _, err := service.Gauge(ctx, "nothing"); err != raingauges.ErrNotFound {
		t.Fatalf("unknown gauge: %v", err)
	}

	near, err := service.Nearest(ctx, 81.5, 7.5)
	if err != nil || near == nil || near.ID != "argtest-07" || near.DistanceKm != 0 {
		t.Fatalf("nearest = %+v, %v", near, err)
	}
	if far, err := service.Nearest(ctx, 79.0, 9.8); err != nil || far != nil {
		t.Fatalf("nothing should be near the north-west sea: %+v, %v", far, err)
	}

	if err := collector.Prune(ctx, now.Add(8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if rows, _ := store.New(pool).RainGaugeReadingsSince(ctx, store.RainGaugeReadingsSinceParams{GaugeID: "argtest-07", ObservedAt: now.Add(-100 * time.Hour)}); len(rows) != 0 {
		t.Fatalf("old readings kept: %d", len(rows))
	}
	if err := collector.Prune(ctx, now.Add(31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if rows, _ := store.New(pool).ListRainGauges(ctx); len(rows) != 0 {
		t.Fatalf("a gauge the feed dropped is kept: %d", len(rows))
	}
}
