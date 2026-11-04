//go:build integration

package osmplaces_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/adminregions"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/osmplaces"
	"github.com/prabhavalabs/lanka-weather/internal/places"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/source"
)

// replay serves the admin-region fixtures by URL suffix and the Overpass
// fixture for POSTs; the first Overpass endpoint is busy.
type replay struct {
	forms []url.Values
}

func (r *replay) Fetch(_ context.Context, request source.Request) ([]byte, error) {
	if request.Method == "POST" {
		r.forms = append(r.forms, request.Form)
		if strings.Contains(request.URL, "busy") {
			return nil, fmt.Errorf("unexpected status 504")
		}
		return os.ReadFile("testdata/overpass.json")
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
	return nil, fmt.Errorf("no recording for %s", request.URL)
}

func TestCollectPlacesAndSearch(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `TRUNCATE regions, places, search_names, source_versions, region_staging, place_staging`); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 11, 2, 14, 0, 0, 0, time.UTC)
	fetcher := &replay{}
	// Places loaded before any regions have no GN division yet; loading the
	// regions afterwards assigns them.
	collector := &osmplaces.Collector{Pool: pool, Fetcher: fetcher, Endpoints: []string{"https://busy.test", "https://ok.test"}, Minimum: 5}
	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 8 || len(fetcher.forms) != 2 || !strings.Contains(fetcher.forms[1].Get("data"), "node[place~") {
		t.Fatalf("stored %d places after %d requests", stored, len(fetcher.forms))
	}
	regionCollector := &adminregions.Collector{
		Pool: pool, Fetcher: fetcher, APIBase: "https://api.test", RawBase: "https://raw.test",
		Minimums: map[regions.Level]int{regions.LevelProvince: 2, regions.LevelDistrict: 2, regions.LevelDS: 2, regions.LevelGN: 3},
	}
	if _, err := regionCollector.Collect(ctx, now); err != nil {
		t.Fatal(err)
	}

	assigned := map[string]*string{}
	rows, err := pool.Query(ctx, `SELECT id, gn_id FROM places`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		var gn *string
		if err := rows.Scan(&id, &gn); err != nil {
			t.Fatal(err)
		}
		assigned[id] = gn
	}
	if gn := assigned["osm-node-2908772560"]; gn == nil || *gn != "LK-2130120" {
		t.Fatalf("Kandy's GN division = %v", gn)
	}
	// Just off the coast: the nearest GN division within 3 km.
	if gn := assigned["osm-node-7000000001"]; gn == nil || *gn != "LK-1103005" {
		t.Fatalf("coastal place's GN division = %v", gn)
	}
	if gn := assigned["osm-node-7000000002"]; gn != nil {
		t.Fatalf("a place far from every region was given %s", *gn)
	}

	// A second weekly run replaces the snapshot and stays idempotent.
	if _, err := collector.Collect(ctx, now.Add(7*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	var count int
	var sourceTime time.Time
	if err := pool.QueryRow(ctx, `SELECT count(*), min(source_time) FROM places`).Scan(&count, &sourceTime); err != nil {
		t.Fatal(err)
	}
	if count != 8 || !sourceTime.Equal(time.Date(2026, 11, 2, 13, 47, 51, 0, time.UTC)) {
		t.Fatalf("places after re-run = %d, source time %v", count, sourceTime)
	}

	service := &places.Service{Pool: pool}
	search := func(query string) []places.Match {
		t.Helper()
		result, err := service.Search(ctx, query, 10)
		if err != nil {
			t.Fatal(err)
		}
		return result.Results
	}

	// The city outranks the district and the hamlet of the same name.
	kandy := search("Kandy")
	if len(kandy) < 3 || kandy[0].ID != "osm-node-2908772560" || kandy[1].ID != "LK-21" {
		t.Fatalf("Kandy results = %+v", kandy)
	}
	city := kandy[0]
	if city.Type != "place" || city.Kind != "city" || city.BBox != nil || city.Population == nil ||
		len(city.Hierarchy) != 4 || city.Hierarchy[0].Name != "Central" || city.Hierarchy[3].ID != "LK-2130120" {
		t.Fatalf("Kandy city = %+v", city)
	}
	if district := kandy[1]; district.Type != "region" || district.BBox == nil || len(district.Hierarchy) != 1 {
		t.Fatalf("Kandy district = %+v", district)
	}

	for query, want := range map[string]string{
		"kandi":            "osm-node-2908772560", // a typo
		"KANDY":            "osm-node-2908772560",
		"senkadagala":      "osm-node-2908772560", // an alternative name
		"මහනුවර":           "osm-node-2908772560", // Sinhala
		"கண்டி":            "osm-node-2908772560", // Tamil
		"colombo district": "LK-11",
		"Mattakuliya":      "LK-1103010",
		"Colombo":          "osm-node-50794342",
		"gangawata":        "LK-2130",
	} {
		if results := search(query); len(results) == 0 || results[0].ID != want {
			t.Errorf("search %q = %+v, want %s first", query, results, want)
		}
	}
	if results := search("zzzzqqq"); len(results) != 0 {
		t.Fatalf("nonsense query found %+v", results)
	}
	if results := search("k"); len(results) == 0 {
		t.Fatal("a one-letter prefix found nothing")
	}
}
