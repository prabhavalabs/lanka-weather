package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/places"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
)

type fakeRegions struct {
	level    regions.Level
	ancestor string
	geometry bool
	lon, lat float64
	tile     [3]int
	empty    bool
	err      error
}

func (fake *fakeRegions) Collection(_ context.Context, level regions.Level, ancestor string) (regions.Document, error) {
	fake.level, fake.ancestor = level, ancestor
	return regions.Document{Body: []byte(`{"type":"FeatureCollection","features":[]}`), ETag: `W/"v1-` + string(level) + `"`}, fake.err
}

func (fake *fakeRegions) Region(_ context.Context, id string, geometry bool) (regions.Detail, error) {
	fake.geometry = geometry
	if id != "LK-21" {
		return regions.Detail{}, regions.ErrRegionNotFound
	}
	return regions.Detail{ID: id, Level: regions.LevelDistrict, Name: "Kandy"}, fake.err
}

func (fake *fakeRegions) Lookup(_ context.Context, lon, lat float64) (regions.Lookup, error) {
	fake.lon, fake.lat = lon, lat
	return regions.Lookup{Lat: lat, Lon: lon, Province: &regions.Ref{ID: "LK-1", Level: regions.LevelProvince, Name: "Western"}}, fake.err
}

func (fake *fakeRegions) Tile(_ context.Context, z, x, y int) (regions.Document, error) {
	fake.tile = [3]int{z, x, y}
	if fake.empty {
		return regions.Document{ETag: `W/"v1"`}, fake.err
	}
	return regions.Document{Body: []byte{0x1a, 0x02}, ETag: `W/"v1"`}, fake.err
}

type fakePlaces struct {
	query string
	limit int
}

func (fake *fakePlaces) Search(_ context.Context, query string, limit int) (places.Result, error) {
	fake.query, fake.limit = query, limit
	return places.Result{Query: query, Results: []places.Match{{ID: "LK-21", Type: "region", Kind: "district", Name: "Kandy"}}}, nil
}

func TestRegionCollection(t *testing.T) {
	fake := &fakeRegions{}
	server := newServer(t, Dependencies{Regions: fake})

	resp, body := get(t, server, "/v1/regions?level=district")
	if resp.StatusCode != http.StatusOK || fake.level != regions.LevelDistrict || fake.ancestor != "" {
		t.Fatalf("district: %d %s, level %q", resp.StatusCode, body, fake.level)
	}
	if resp.Header.Get("Content-Type") != "application/geo+json; charset=utf-8" ||
		resp.Header.Get("ETag") != `W/"v1-district"` || resp.Header.Get("Cache-Control") == "" {
		t.Fatalf("headers = %v", resp.Header)
	}

	if resp, _ := get(t, server, "/v1/regions?level=gn&parent=LK2103"); resp.StatusCode != http.StatusOK || fake.ancestor != "LK-2103" {
		t.Fatalf("gn within a DS division: %d, ancestor %q", resp.StatusCode, fake.ancestor)
	}

	for path, code := range map[string]string{
		"/v1/regions":                          "invalid_level",
		"/v1/regions?level=village":            "invalid_level",
		"/v1/regions?level=gn":                 "parent_required",
		"/v1/regions?level=gn&parent=LK-2":     "invalid_parent",
		"/v1/regions?level=ds&parent=LK-2103":  "invalid_parent",
		"/v1/regions?level=district&parent=xx": "invalid_parent",
	} {
		resp, body := get(t, server, path)
		if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != code {
			t.Errorf("%s = %d %s, want 400 %s", path, resp.StatusCode, body, code)
		}
	}
}

func TestRegionCollectionRevalidates(t *testing.T) {
	server := newServer(t, Dependencies{Regions: &fakeRegions{}})
	request, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/regions?level=province", nil)
	request.Header.Set("If-None-Match", `"x", W/"v1-province"`)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", resp.StatusCode)
	}
}

func TestRegionDetail(t *testing.T) {
	fake := &fakeRegions{}
	server := newServer(t, Dependencies{Regions: fake})

	resp, body := get(t, server, "/v1/regions/lk21?geometry=true")
	if resp.StatusCode != http.StatusOK || !fake.geometry {
		t.Fatalf("detail: %d %s, geometry %v", resp.StatusCode, body, fake.geometry)
	}
	var envelope struct {
		Data regions.Detail `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Data.Name != "Kandy" {
		t.Fatalf("detail body = %s (%v)", body, err)
	}
	for path, want := range map[string]struct {
		status int
		code   string
	}{
		"/v1/regions/LK-22":                {http.StatusNotFound, "region_not_found"},
		"/v1/regions/Kandy":                {http.StatusBadRequest, "invalid_region_id"},
		"/v1/regions/LK-21?geometry=maybe": {http.StatusBadRequest, "invalid_geometry"},
	} {
		resp, body := get(t, server, path)
		if resp.StatusCode != want.status || errorCode(t, body) != want.code {
			t.Errorf("%s = %d %s, want %d %s", path, resp.StatusCode, body, want.status, want.code)
		}
	}
}

func TestRegionLookup(t *testing.T) {
	fake := &fakeRegions{}
	server := newServer(t, Dependencies{Regions: fake})

	resp, body := get(t, server, "/v1/regions/lookup?lat=6.9271&lon=79.8612")
	if resp.StatusCode != http.StatusOK || fake.lat != 6.9271 || fake.lon != 79.8612 {
		t.Fatalf("lookup: %d %s, point %v %v", resp.StatusCode, body, fake.lat, fake.lon)
	}
	for path, code := range map[string]string{
		"/v1/regions/lookup":                   "invalid_point",
		"/v1/regions/lookup?lat=abc&lon=80":    "invalid_point",
		"/v1/regions/lookup?lat=NaN&lon=80":    "invalid_point",
		"/v1/regions/lookup?lat=51.5&lon=-0.1": "out_of_range",
	} {
		resp, body := get(t, server, path)
		if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != code {
			t.Errorf("%s = %d %s, want 400 %s", path, resp.StatusCode, body, code)
		}
	}
}

func TestRegionTiles(t *testing.T) {
	fake := &fakeRegions{}
	server := newServer(t, Dependencies{Regions: fake})

	resp, body := get(t, server, "/v1/tiles/regions/12/2956/1968.pbf")
	if resp.StatusCode != http.StatusOK || fake.tile != [3]int{12, 2956, 1968} || len(body) != 2 {
		t.Fatalf("tile: %d, %v, %d bytes", resp.StatusCode, fake.tile, len(body))
	}
	if resp.Header.Get("Content-Type") != "application/vnd.mapbox-vector-tile" || resp.Header.Get("ETag") == "" {
		t.Fatalf("headers = %v", resp.Header)
	}

	fake.empty = true
	if resp, _ := get(t, server, "/v1/tiles/regions/3/0/0.pbf"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("empty tile status = %d, want 204", resp.StatusCode)
	}
	for _, path := range []string{"/v1/tiles/regions/15/0/0.pbf", "/v1/tiles/regions/2/4/0.pbf", "/v1/tiles/regions/2/0/-1.pbf", "/v1/tiles/regions/a/0/0.pbf"} {
		resp, body := get(t, server, path)
		if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != "invalid_tile" {
			t.Errorf("%s = %d %s, want 400 invalid_tile", path, resp.StatusCode, body)
		}
	}

	failing := newServer(t, Dependencies{Regions: &fakeRegions{err: errors.New("boom")}})
	if resp, body := get(t, failing, "/v1/tiles/regions/1/0/0.pbf"); resp.StatusCode != http.StatusInternalServerError || errorCode(t, body) != "internal" {
		t.Fatalf("failure = %d %s", resp.StatusCode, body)
	}
}

func TestPlaceSearch(t *testing.T) {
	fake := &fakePlaces{}
	server := newServer(t, Dependencies{Places: fake})

	resp, body := get(t, server, "/v1/places/search?q=++Kandy++Lake+")
	if resp.StatusCode != http.StatusOK || fake.query != "Kandy Lake" || fake.limit != places.DefaultLimit {
		t.Fatalf("search: %d %s, query %q, limit %d", resp.StatusCode, body, fake.query, fake.limit)
	}
	if resp, _ := get(t, server, "/v1/places/search?q=galle&limit=50"); resp.StatusCode != http.StatusOK || fake.limit != 50 {
		t.Fatalf("limit=50: %d, %d", resp.StatusCode, fake.limit)
	}
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	for path, code := range map[string]string{
		"/v1/places/search":                    "invalid_query",
		"/v1/places/search?q=+++":              "invalid_query",
		"/v1/places/search?q=" + string(long):  "invalid_query",
		"/v1/places/search?q=galle&limit=0":    "invalid_limit",
		"/v1/places/search?q=galle&limit=51":   "invalid_limit",
		"/v1/places/search?q=galle&limit=many": "invalid_limit",
	} {
		resp, body := get(t, server, path)
		if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != code {
			t.Errorf("%s = %d %s, want 400 %s", path, resp.StatusCode, body, code)
		}
	}
}

func TestGeographyUnavailableWithoutServices(t *testing.T) {
	server := newServer(t, Dependencies{})
	for _, path := range []string{"/v1/regions?level=province", "/v1/regions/LK-1", "/v1/regions/lookup?lat=7&lon=80", "/v1/tiles/regions/0/0/0.pbf", "/v1/places/search?q=kandy"} {
		if resp, _ := get(t, server, path); resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("%s = %d, want 503", path, resp.StatusCode)
		}
	}
}
