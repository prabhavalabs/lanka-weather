//go:build integration

package weather_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/store"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// insertRegions adds rectangular regions to the regions table (migration
// 00002), which the admin_regions view (migration 00004) exposes.
const insertRegions = `
INSERT INTO regions (id, fid, level, name, parent_id, province_id, district_id, ds_id, area_km2, centroid, label_point,
                     west, south, east, north, geom, geom_simple, geom_3857, geom_simple_3857,
                     source, source_version, source_time, fetched_at)
SELECT v.id, v.fid, v.level, v.name, v.parent_id, v.province_id, v.district_id, v.ds_id, ST_Area(b.geom::geography) / 1e6,
       ST_Centroid(b.geom), ST_PointOnSurface(b.geom), v.west, v.south, v.east, v.north,
       b.geom, b.geom, ST_Transform(b.geom, 3857), ST_Transform(b.geom, 3857), 'test', 'test', now(), now()
FROM (VALUES
    ('LK-1', 1, 'province', 'Western', NULL, 'LK-1', NULL, NULL, 79.80, 6.40, 80.30, 7.20),
    ('LK-11', 11, 'district', 'Colombo', 'LK-1', 'LK-1', 'LK-11', NULL, 79.82, 6.70, 80.22, 7.00),
    ('LK-12', 12, 'district', 'Gampaha', 'LK-1', 'LK-1', 'LK-12', NULL, 79.82, 7.00, 80.22, 7.20),
    ('LK-1103', 1103, 'ds', 'Small division', 'LK-11', 'LK-1', 'LK-11', 'LK-1103', 79.85, 6.90, 79.90, 6.95),
    ('LK-1103005', 1103005, 'gn', 'Village', 'LK-1103', 'LK-1', 'LK-11', 'LK-1103', 79.86, 6.92, 79.87, 6.93)
) AS v(id, fid, level, name, parent_id, province_id, district_id, ds_id, west, south, east, north)
CROSS JOIN LATERAL (SELECT ST_Multi(ST_MakeEnvelope(v.west, v.south, v.east, v.north, 4326)) AS geom) AS b`

func TestPostGISRegionsReadsTheRegionsView(t *testing.T) {
	pool := dbtest.Open(t, "weather_grid_nodes", "regions")
	ctx := context.Background()
	weatherGrid := grid.SriLanka()

	source := &weather.PostGISRegions{Pool: pool, Grid: weatherGrid}
	if _, err := source.Regions(ctx, "district"); !errors.Is(err, weather.ErrRegionsUnavailable) {
		t.Fatalf("before regions are loaded = %v", err)
	}
	if _, err := pool.Exec(ctx, insertRegions); err != nil {
		t.Fatal(err)
	}
	// Regions without the grid's cells would all fall back to their point.
	if _, err := source.Regions(ctx, "district"); !errors.Is(err, weather.ErrNoData) {
		t.Fatalf("before grid cells are stored = %v", err)
	}

	half := weatherGrid.Step / 2
	params := make([]store.UpsertWeatherGridNodeParams, len(weatherGrid.Nodes))
	for index, node := range weatherGrid.Nodes {
		params[index] = store.UpsertWeatherGridNodeParams{Grid: weatherGrid.ID, Node: int32(node.Index), Lon: node.Lon, Lat: node.Lat,
			Land: node.Land, West: node.Lon - half, South: node.Lat - half, East: node.Lon + half, North: node.Lat + half,
			Source: "test", UpdatedAt: time.Now()}
	}
	store.New(pool).UpsertWeatherGridNode(ctx, params).Exec(func(_ int, err error) {
		if err != nil {
			t.Fatal(err)
		}
	})
	if err := source.Warm(ctx); err != nil {
		t.Fatal(err)
	}

	districts, err := source.Regions(ctx, "district")
	if err != nil {
		t.Fatal(err)
	}
	if len(districts) != 2 || districts[0].ID != "LK-11" || districts[0].Name != "Colombo" || districts[0].ProvinceID != "LK-1" {
		t.Fatalf("districts = %+v", districts)
	}
	colombo := districts[0]
	if len(colombo.Nodes) < 4 || len(colombo.Weights) != len(colombo.Nodes) {
		t.Fatalf("Colombo cells = %v %v", colombo.Nodes, colombo.Weights)
	}
	total := 0.0
	for _, weight := range colombo.Weights {
		if weight <= 0 || weight > 1+1e-9 {
			t.Fatalf("weight %v out of range", weight)
		}
		total += weight
	}
	// The box is 0.4° × 0.3°, three cells of 0.2° × 0.2°.
	if math.Abs(total-3) > 0.01 {
		t.Fatalf("cell shares add up to %v, want 3", total)
	}
	if !colombo.Within("LK-1") || colombo.Within("LK-12") {
		t.Fatal("Colombo is in the Western Province only")
	}

	// A DS division smaller than a cell, and every GN division, take the
	// value at a point inside them.
	for _, level := range []string{"ds", "gn"} {
		regions, err := source.Regions(ctx, level)
		if err != nil || len(regions) != 1 || len(regions[0].Nodes) != 0 || regions[0].Lon < 79.85 || regions[0].Lon > 79.9 {
			t.Fatalf("%s regions = %+v, %v", level, regions, err)
		}
		if !regions[0].Within("LK-11") || !regions[0].Within("LK-1") || regions[0].Within("LK-12") {
			t.Fatalf("%s ancestors = %+v", level, regions[0])
		}
	}
	if _, err := source.Regions(ctx, "village"); !errors.Is(err, weather.ErrUnknownLevel) {
		t.Fatalf("unknown level = %v", err)
	}

	// Another relation can be named, as long as it has the view's columns.
	missing := &weather.PostGISRegions{Pool: pool, Grid: weatherGrid, Relation: "no_such_relation"}
	if _, err := missing.Regions(ctx, "province"); !errors.Is(err, weather.ErrRegionsUnavailable) {
		t.Fatalf("missing relation = %v", err)
	}
}
