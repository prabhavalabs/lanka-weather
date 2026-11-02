//go:build integration

package osmreservoirs_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/adminregions"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/osmreservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/source"
)

// Water bodies drawn inside the fixture's Kandy and Colombo districts, and a
// same-named one in neither.
const overpass = `{"osm3s":{"timestamp_osm_base":"2026-11-02T19:46:36Z"},"elements":[
 {"type":"relation","id":3553743,"tags":{"natural":"water","water":"reservoir","name":"Victoaria Resevoir"},
  "members":[
   {"type":"way","role":"outer","geometry":[{"lat":7.27,"lon":80.60},{"lat":7.27,"lon":80.65},{"lat":7.31,"lon":80.65}]},
   {"type":"way","role":"outer","geometry":[{"lat":7.31,"lon":80.65},{"lat":7.31,"lon":80.60},{"lat":7.27,"lon":80.60}]}
  ]},
 {"type":"way","id":232724598,"tags":{"man_made":"dam","name":"Victoria Dam"},
  "geometry":[{"lat":7.24,"lon":80.70},{"lat":7.25,"lon":80.71}]},
 {"type":"way","id":3,"tags":{"man_made":"dam","name":"Polgolla Barrage"},
  "geometry":[{"lat":7.32,"lon":80.64},{"lat":7.33,"lon":80.65}]},
 {"type":"way","id":4,"tags":{"natural":"water","name":"Akathimuruppu Tank"},
  "geometry":[{"lat":9.0,"lon":80.0},{"lat":9.0,"lon":80.1},{"lat":9.1,"lon":80.1},{"lat":9.0,"lon":80.0}]}
]}`

type replay struct{ busy *int }

func (r replay) Fetch(_ context.Context, request source.Request) ([]byte, error) {
	if request.Method == "POST" {
		if !strings.Contains(request.Form.Get("data"), `"man_made"="dam"`) {
			return nil, errors.New("unexpected query")
		}
		return []byte(overpass), nil
	}
	if strings.Contains(request.URL, "reservoirs") {
		return os.ReadFile("../waterlevels/testdata/reservoirs.json")
	}
	files := map[string]string{
		"lk_admin_regions/commits":     "boundary_commits.json",
		"lk_census_2024/commits":       "census_commits.json",
		"Person-Sex/data.tsv":          "census.tsv",
		"e4_medium/provinces.topojson": "provinces.topojson",
		"e4_medium/districts.topojson": "districts.topojson",
		"e4_medium/dsds.topojson":      "dsds.topojson",
		"e4_medium/gnds.topojson":      "gnds.topojson",
	}
	for suffix, file := range files {
		if strings.HasSuffix(request.URL, suffix) {
			return os.ReadFile("../adminregions/testdata/" + file)
		}
	}
	return nil, errors.New("no recording for " + request.URL)
}

func TestMapReservoirsIntegration(t *testing.T) {
	databaseURL := os.Getenv("LANKAWEATHER_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LANKAWEATHER_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE reservoirs, reservoir_readings, regions, places, search_names, source_versions, region_staging, place_staging CASCADE`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 2, 19, 45, 0, 0, time.UTC)
	fetcher := replay{}

	collector := &osmreservoirs.Collector{Pool: pool, Fetcher: fetcher, Endpoints: []string{"https://ok.test"}, Minimum: 1, MinimumMatched: 1}
	// Nothing to map yet: the weekly query is not spent, and the run retries.
	if _, err := collector.Collect(ctx, now); !errors.Is(err, osmreservoirs.ErrNoReservoirs) {
		t.Fatalf("collect before the feed: %v", err)
	}

	regionCollector := &adminregions.Collector{
		Pool: pool, Fetcher: fetcher, APIBase: "https://api.test", RawBase: "https://raw.test",
		Minimums: map[regions.Level]int{regions.LevelProvince: 2, regions.LevelDistrict: 2, regions.LevelDS: 2, regions.LevelGN: 3},
	}
	if _, err := regionCollector.Collect(ctx, now); err != nil {
		t.Fatal(err)
	}
	feed := &waterlevels.ReservoirCollector{Pool: pool, Fetcher: fetcher, Endpoint: "https://x.test/reservoirs", Minimum: 1}
	if _, err := feed.Collect(ctx, now); err != nil {
		t.Fatal(err)
	}

	mapped, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if mapped != 2 {
		t.Fatalf("mapped %d, want Victoria and Polgolla", mapped)
	}

	service := &reservoirs.Service{Pool: pool, StaleAfter: 48 * time.Hour, Now: func() time.Time { return now }}
	layer, err := service.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]reservoirs.Feature{}
	for _, feature := range layer.Features {
		byID[feature.ID] = feature
	}
	// The polygon is stitched from two ways, and wins over the dam.
	victoria := byID["victoria"]
	if victoria.Geometry == nil || *victoria.Properties.OSMKind != "water" || *victoria.Properties.OSMName != "Victoaria Resevoir" {
		t.Fatalf("victoria = %+v", victoria.Properties)
	}
	// A reservoir known only by its barrage is a marker with no polygon.
	if polgolla := byID["polgolla"]; polgolla.Geometry == nil || *polgolla.Properties.OSMKind != "dam" {
		t.Fatalf("polgolla = %+v", polgolla)
	}
	// Akathimuruppu's name matches, but in a place outside its district.
	if tank := byID["akathimuruppu"]; tank.Geometry != nil || tank.Properties.Mapped {
		t.Fatalf("akathimuruppu was mapped outside Mannar: %+v", tank.Geometry)
	}
	if layer.Meta.Counts.Mapped != 2 || layer.Meta.MapTime == nil {
		t.Fatalf("meta = %+v", layer.Meta)
	}
	areas, err := service.Areas(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas.Features) != 1 || areas.Features[0].ID != "victoria" || !strings.Contains(string(areas.Features[0].Geometry), "Polygon") {
		t.Fatalf("areas = %+v", areas)
	}

	// A run that matches too little keeps the last good map.
	strict := &osmreservoirs.Collector{Pool: pool, Fetcher: fetcher, Endpoints: []string{"https://ok.test"}, Minimum: 1, MinimumMatched: 50}
	if _, err := strict.Collect(ctx, now.Add(7*24*time.Hour)); err == nil {
		t.Fatal("a poor match was accepted")
	}
	after, err := service.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.Meta.Counts.Mapped != 2 {
		t.Fatalf("the last map was lost: %+v", after.Meta.Counts)
	}
	// The staging table is left empty.
	var staged int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM reservoir_candidate_staging`).Scan(&staged); err != nil {
		t.Fatal(err)
	}
	if staged != 0 {
		t.Fatalf("staged rows left = %d", staged)
	}
}
