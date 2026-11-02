//go:build integration

package gdacs_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/gdacs"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
)

type replay struct {
	list, geometry []byte
	geometryErr    error
	requests       []source.Request
}

func (r *replay) Fetch(_ context.Context, request source.Request) ([]byte, error) {
	r.requests = append(r.requests, request)
	if strings.HasSuffix(request.URL, "/polygons/getgeometry") {
		return r.geometry, r.geometryErr
	}
	return r.list, nil
}

func fixtures(t *testing.T) *replay {
	t.Helper()
	list, err := os.ReadFile("testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	geometry, err := os.ReadFile("testdata/geometry.json")
	if err != nil {
		t.Fatal(err)
	}
	return &replay{list: list, geometry: geometry}
}

func TestCollectServesAnActiveCycloneThenTheCalmState(t *testing.T) {
	pool := dbtest.Open(t, "cyclones", "source_versions")
	ctx := context.Background()
	fetcher := fixtures(t)
	collector := &gdacs.Collector{Pool: pool, Fetcher: fetcher}
	now := time.Date(2026, 11, 2, 19, 0, 0, 0, time.UTC)

	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 1 || len(fetcher.requests) != 2 {
		t.Fatalf("stored %d after %d requests", stored, len(fetcher.requests))
	}
	geometry := fetcher.requests[1].Query
	if geometry.Get("eventid") != "1001346" || geometry.Get("episodeid") != "7" {
		t.Fatalf("geometry query = %v", geometry)
	}

	service := &hazards.Service{Pool: pool, Now: func() time.Time { return now }, CyclonesStaleAfter: 6 * time.Hour}
	layer, err := service.Cyclones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if layer.Meta.Status != hazards.CycloneActive || layer.Meta.Count != 1 || layer.Meta.Stale {
		t.Fatalf("meta = %+v", layer.Meta)
	}
	position := layer.Features[0].Properties
	if position["kind"] != "position" || position["name"] != "MURASU-26" || position["alertLevel"] != "Red" || position["advisorySource"] != "JTWC" {
		t.Fatalf("position = %+v", position)
	}
	distance, _ := position["distanceKm"].(float64)
	if distance < 500 || distance > 600 {
		t.Fatalf("distance = %v", distance)
	}
	kinds := map[any]int{}
	for _, feature := range layer.Features {
		kinds[feature.Properties["kind"]]++
		if feature.Properties["eventId"] != "1001346" {
			t.Fatalf("feature = %+v", feature.Properties)
		}
	}
	if kinds["track"] == 0 || kinds["forecast-track"] == 0 || kinds["wind-buffer"] != 3 || kinds["track-point"] == 0 || kinds["uncertainty-cone"] != 1 {
		t.Fatalf("kinds = %v", kinds)
	}
	if _, err := json.Marshal(layer); err != nil {
		t.Fatal(err)
	}

	// A failed geometry fetch keeps the last geometry and reports the error.
	fetcher.geometryErr = errors.New("down")
	if _, err := collector.Collect(ctx, now.Add(time.Hour)); err == nil {
		t.Fatal("a failed geometry fetch must be reported")
	}
	if layer, err = service.Cyclones(ctx); err != nil || len(layer.Features) < 5 {
		t.Fatalf("features %d, err %v", len(layer.Features), err)
	}

	// The system ends: the calm state, with an age.
	calmAt := now.Add(2 * time.Hour)
	service.Now = func() time.Time { return calmAt }
	fetcher.geometryErr = nil
	fetcher.list = []byte(`{"type":"FeatureCollection","features":[]}`)
	if stored, err := collector.Collect(ctx, calmAt); err != nil || stored != 0 {
		t.Fatalf("stored %d, err %v", stored, err)
	}
	layer, err = service.Cyclones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if layer.Meta.Status != hazards.CycloneNone || layer.Meta.Note != "No active cyclone in the North Indian Ocean" || len(layer.Features) != 0 || layer.Meta.FetchedAt == nil {
		t.Fatalf("calm = %+v, %d features", layer.Meta, len(layer.Features))
	}

	// With the feed out of date, calm is not claimed.
	lateAt := calmAt.Add(7 * time.Hour)
	service.Now = func() time.Time { return lateAt }
	layer, err = service.Cyclones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if layer.Meta.Status != hazards.CycloneUnknown || strings.Contains(layer.Meta.Note, "No active") {
		t.Fatalf("stale = %+v", layer.Meta)
	}
}

func TestCyclonesBeforeAnyCollection(t *testing.T) {
	pool := dbtest.Open(t, "cyclones", "source_versions")
	layer, err := (&hazards.Service{Pool: pool, CyclonesStaleAfter: time.Hour}).Cyclones(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if layer.Meta.Status != hazards.CycloneWaiting || !layer.Meta.Stale || layer.Features == nil {
		t.Fatalf("layer = %+v", layer)
	}
}
