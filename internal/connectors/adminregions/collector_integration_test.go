//go:build integration

package adminregions_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/adminregions"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/source"
)

// replay serves recorded files by the end of the requested URL and counts
// the requests.
type replay struct {
	files    map[string]string
	requests []string
}

func newReplay(boundarySHA string) *replay {
	return &replay{files: map[string]string{
		"/repos/nuuuwan/lk_admin_regions/commits":                             "testdata/boundary_commits.json",
		"/repos/nuuuwan/lk_census_2024/commits":                               "testdata/census_commits.json",
		"/data/Person-Sex/data.tsv":                                           "testdata/census.tsv",
		"/" + boundarySHA + "/data/geo/topojson/e4_medium/provinces.topojson": "testdata/provinces.topojson",
		"/" + boundarySHA + "/data/geo/topojson/e4_medium/districts.topojson": "testdata/districts.topojson",
		"/" + boundarySHA + "/data/geo/topojson/e4_medium/dsds.topojson":      "testdata/dsds.topojson",
		"/" + boundarySHA + "/data/geo/topojson/e4_medium/gnds.topojson":      "testdata/gnds.topojson",
	}}
}

func (r *replay) Fetch(_ context.Context, request source.Request) ([]byte, error) {
	r.requests = append(r.requests, request.URL)
	for suffix, file := range r.files {
		if strings.HasSuffix(request.URL, suffix) {
			return os.ReadFile(file)
		}
	}
	return nil, fmt.Errorf("no recording for %s", request.URL)
}

const fixtureSHA = "13e6784dce0ec813c3fd5db3886a730af82cf887"

// Minimums fit the fixtures: two provinces, districts and DS divisions, and
// three GN divisions.
var minimums = map[regions.Level]int{regions.LevelProvince: 2, regions.LevelDistrict: 2, regions.LevelDS: 2, regions.LevelGN: 3}

func openDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
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
	return pool
}

func TestCollectLoadsAndServesRegions(t *testing.T) {
	pool := openDatabase(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	fetcher := newReplay(fixtureSHA)
	collector := &adminregions.Collector{Pool: pool, Fetcher: fetcher, APIBase: "https://api.test", RawBase: "https://raw.test", Minimums: minimums}

	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 9 {
		t.Fatalf("stored %d regions, want 9", stored)
	}
	if got := fetcher.requests[0]; got != "https://api.test/repos/nuuuwan/lk_admin_regions/commits" {
		t.Fatalf("first request = %s", got)
	}

	var gnCount, populated int
	var source, version string
	var sourceTime, fetchedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE level = 'gn'), count(population), min(source), min(source_version),
	                                      min(source_time), min(fetched_at) FROM regions`).
		Scan(&gnCount, &populated, &source, &version, &sourceTime, &fetchedAt); err != nil {
		t.Fatal(err)
	}
	if gnCount != 3 || populated != 8 || source != regions.BoundarySource || version != fixtureSHA ||
		!sourceTime.Equal(time.Date(2026, 6, 23, 9, 8, 39, 0, time.UTC)) || !fetchedAt.Equal(now) {
		t.Fatalf("stored gn=%d populated=%d source=%s version=%s sourceTime=%v fetchedAt=%v",
			gnCount, populated, source, version, sourceTime, fetchedAt)
	}
	// The area missing from the source is computed: a 0.2° square at 7.3°N.
	var area float64
	if err := pool.QueryRow(ctx, `SELECT area_km2 FROM regions WHERE id = 'LK-2130120'`).Scan(&area); err != nil {
		t.Fatal(err)
	}
	if area < 480 || area > 495 {
		t.Fatalf("computed area = %v km², want about 487", area)
	}

	// An unchanged upstream version is not fetched again.
	requests := len(fetcher.requests)
	if stored, err := collector.Collect(ctx, now.Add(7*24*time.Hour)); err != nil || stored != 0 {
		t.Fatalf("unchanged run stored %d, %v", stored, err)
	}
	if len(fetcher.requests) != requests+2 {
		t.Fatalf("unchanged run made %d requests, want only the two version checks", len(fetcher.requests)-requests)
	}

	// A new version reloads, and regions no longer published are removed.
	if _, err := pool.Exec(ctx, `UPDATE source_versions SET version = 'older' WHERE source = $1`, regions.BoundarySource); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO regions (id, fid, level, name, province_id, area_km2, centroid, label_point, west, south, east, north,
		                     geom, geom_simple, geom_3857, geom_simple_3857, source, source_version, source_time, fetched_at)
		SELECT 'LK-1103099', 1103099, level, 'Removed upstream', province_id, area_km2, centroid, label_point,
		       west, south, east, north, geom, geom_simple, geom_3857, geom_simple_3857, source, source_version, source_time, fetched_at
		FROM regions WHERE id = 'LK-1103010'`); err != nil {
		t.Fatal(err)
	}
	if stored, err := collector.Collect(ctx, now.Add(14*24*time.Hour)); err != nil || stored != 9 {
		t.Fatalf("reload stored %d, %v", stored, err)
	}
	var removed int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM regions WHERE id = 'LK-1103099'`).Scan(&removed); err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatal("a region missing from the new version was kept")
	}

	service := &regions.Service{Pool: pool}
	t.Run("collection", func(t *testing.T) {
		document, err := service.Collection(ctx, regions.LevelProvince, "")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(document.Body, []byte(`"id":"LK-1"`)) || !bytes.Contains(document.Body, []byte(`"type":"MultiPolygon"`)) ||
			!bytes.Contains(document.Body, []byte(`"sourceVersion":"`+fixtureSHA+`"`)) || document.ETag == "" {
			t.Fatalf("collection = %s, etag %q", document.Body, document.ETag)
		}
		within, err := service.Collection(ctx, regions.LevelGN, "LK-1103")
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(within.Body, []byte(`"type":"Feature"`)) != 2 || bytes.Contains(within.Body, []byte("LK-2130120")) {
			t.Fatalf("GN divisions in LK-1103 = %s", within.Body)
		}
	})
	t.Run("detail", func(t *testing.T) {
		detail, err := service.Region(ctx, "LK-1103010", true)
		if err != nil {
			t.Fatal(err)
		}
		if len(detail.Hierarchy) != 3 || detail.Hierarchy[0].ID != "LK-1" || detail.Hierarchy[2].ID != "LK-1103" ||
			detail.GNNumber == nil || *detail.GNNumber != "520A" || detail.Population == nil || *detail.Population != 39710 {
			t.Fatalf("detail = %+v", detail)
		}
		if detail.BBox != [4]float64{79.9, 6.8, 80, 7} || len(detail.Geometry) == 0 {
			t.Fatalf("bbox = %v, geometry %s", detail.BBox, detail.Geometry)
		}
		district, err := service.Region(ctx, "LK-21", false)
		if err != nil {
			t.Fatal(err)
		}
		if district.ChildCount != 1 || district.ISOCode == nil || *district.ISOCode != "LK-21" || district.Geometry != nil {
			t.Fatalf("district = %+v", district)
		}
		if _, err := service.Region(ctx, "LK-9", false); err != regions.ErrRegionNotFound {
			t.Fatalf("unknown region error = %v", err)
		}
	})
	t.Run("lookup", func(t *testing.T) {
		lookup, err := service.Lookup(ctx, 79.95, 6.9)
		if err != nil {
			t.Fatal(err)
		}
		if lookup.Province == nil || lookup.Province.ID != "LK-1" || lookup.District.ID != "LK-11" ||
			lookup.DS.ID != "LK-1103" || lookup.GN == nil || lookup.GN.ID != "LK-1103010" {
			t.Fatalf("lookup = %+v", lookup)
		}
		sea, err := service.Lookup(ctx, 79.7, 6.9)
		if err != nil {
			t.Fatal(err)
		}
		if sea.Province != nil || sea.GN != nil {
			t.Fatalf("a point at sea found %+v", sea)
		}
	})
	t.Run("tiles", func(t *testing.T) {
		// z12 tile containing 79.85°E 6.9°N, and the z5 tile over Sri Lanka.
		detailed, err := service.Tile(ctx, 12, 2956, 1969)
		if err != nil {
			t.Fatal(err)
		}
		for _, layer := range []string{"province", "district", "ds", "gn", "LK-1103005", "provinceId"} {
			if !bytes.Contains(detailed.Body, []byte(layer)) {
				t.Errorf("z12 tile lacks %q", layer)
			}
		}
		overview, err := service.Tile(ctx, 5, 23, 15)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(overview.Body, []byte("province")) || bytes.Contains(overview.Body, []byte("LK-1103005")) {
			t.Fatal("z5 tile should hold provinces but not GN divisions")
		}
		empty, err := service.Tile(ctx, 5, 0, 0)
		if err != nil || len(empty.Body) != 0 {
			t.Fatalf("tile far from Sri Lanka = %d bytes, %v", len(empty.Body), err)
		}
	})
}
