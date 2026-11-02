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

// The relation the integrator is expected to provide is documented in
// docs/api/weather-and-floods.md; this test builds one with that shape.
const relation = "test_weather_regions"

func TestPostGISRegionsJoinsGridCells(t *testing.T) {
	pool := dbtest.Open(t, "weather_grid_nodes")
	ctx := context.Background()
	weatherGrid := grid.SriLanka()
	half := weatherGrid.Step / 2
	params := make([]store.UpsertWeatherGridNodeParams, len(weatherGrid.Nodes))
	for index, node := range weatherGrid.Nodes {
		params[index] = store.UpsertWeatherGridNodeParams{Grid: weatherGrid.ID, Node: int32(node.Index), Lon: node.Lon, Lat: node.Lat,
			Land: node.Land, West: node.Lon - half, South: node.Lat - half, East: node.Lon + half, North: node.Lat + half,
			Source: "test", UpdatedAt: time.Now()}
	}
	results := store.New(pool).UpsertWeatherGridNode(ctx, params)
	results.Exec(func(_ int, err error) {
		if err != nil {
			t.Fatal(err)
		}
	})

	source := &weather.PostGISRegions{Pool: pool, Relation: relation, Grid: weatherGrid}
	if _, err := source.Regions(ctx, "district"); !errors.Is(err, weather.ErrRegionsUnavailable) {
		t.Fatalf("without the relation = %v", err)
	}
	if _, err := pool.Exec(ctx, `
DROP TABLE IF EXISTS `+relation+`;
CREATE TABLE `+relation+` (id text PRIMARY KEY, level text NOT NULL, name text NOT NULL, geom geometry(MultiPolygon, 4326) NOT NULL);
INSERT INTO `+relation+` VALUES
 ('LK-11', 'district', 'Colombo', ST_Multi(ST_MakeEnvelope(79.82, 6.70, 80.22, 7.00, 4326))),
 ('LK-1103', 'ds', 'Small division', ST_Multi(ST_MakeEnvelope(79.85, 6.90, 79.90, 6.95, 4326))),
 ('LK-1103-001', 'gn', 'Village', ST_Multi(ST_MakeEnvelope(79.86, 6.92, 79.87, 6.93, 4326)));`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS `+relation) })

	districts, err := source.Regions(ctx, "district")
	if err != nil {
		t.Fatal(err)
	}
	if len(districts) != 1 || len(districts[0].Nodes) < 4 || len(districts[0].Weights) != len(districts[0].Nodes) {
		t.Fatalf("districts = %+v", districts)
	}
	total := 0.0
	for _, weight := range districts[0].Weights {
		if weight <= 0 || weight > 1+1e-9 {
			t.Fatalf("weight %v out of range", weight)
		}
		total += weight
	}
	// The box is 0.4° × 0.3°, three cells of 0.2° × 0.2°.
	if math.Abs(total-3) > 0.01 {
		t.Fatalf("cell shares add up to %v, want 3", total)
	}
	for _, level := range []string{"ds", "gn"} {
		regions, err := source.Regions(ctx, level)
		if err != nil || len(regions) != 1 || len(regions[0].Nodes) != 0 || regions[0].Lon < 79.85 || regions[0].Lon > 79.9 {
			t.Fatalf("%s regions = %+v, %v", level, regions, err)
		}
	}
	if _, err := source.Regions(ctx, "village"); !errors.Is(err, weather.ErrUnknownLevel) {
		t.Fatalf("unknown level = %v", err)
	}
}
