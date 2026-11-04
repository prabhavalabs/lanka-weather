package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/layers"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

type fakeWarnings struct {
	err    error
	filter warnings.Filter
}

func (fake *fakeWarnings) Landslides(context.Context) (warnings.LandslideLayer, error) {
	return warnings.LandslideLayer{Type: "FeatureCollection", Features: []warnings.DivisionFeature{},
		Meta: warnings.LandslideMeta{Status: warnings.StatusNone, Note: "Last report 31 Oct 2026; none currently active as far as we know."}}, fake.err
}

func (fake *fakeWarnings) Division(_ context.Context, id string) (*warnings.Division, error) {
	if id == "LK-9999" {
		return nil, warnings.ErrDivisionNotFound
	}
	return &warnings.Division{ID: id, Name: "Yatinuwara", History: []warnings.HistoryEntry{}}, fake.err
}

func (fake *fakeWarnings) Overview(_ context.Context, filter warnings.Filter) (*warnings.Overview, error) {
	fake.filter = filter
	return &warnings.Overview{Bulletins: []warnings.Bulletin{{ID: "doc", Title: "Flood warning for Kelani River"}},
		Banner: []warnings.BannerItem{}, ActiveCount: 1}, fake.err
}

type fakeHazards struct{ err error }

func (fake fakeHazards) Earthquakes(context.Context) (hazards.EarthquakeLayer, error) {
	return hazards.EarthquakeLayer{Type: "FeatureCollection", Features: []hazards.EarthquakeFeature{}}, fake.err
}

func (fake fakeHazards) Cyclones(context.Context) (hazards.CycloneLayer, error) {
	return hazards.CycloneLayer{Type: "FeatureCollection", Features: []hazards.CycloneFeature{},
		Meta: hazards.CycloneMeta{Status: hazards.CycloneNone, Note: "No active cyclone in the North Indian Ocean"}}, fake.err
}

func hazardServer(t *testing.T, warn *fakeWarnings, hazard Hazards) *httptest.Server {
	t.Helper()
	deps := Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Hazards: hazard}
	if warn != nil {
		deps.Warnings = warn
	}
	server := httptest.NewServer(NewRouter(deps))
	t.Cleanup(server.Close)
	return server
}

func fetch(t *testing.T, url string) (int, http.Header, string) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, response.Header, string(body)
}

func TestWarningsOverviewPassesFiltersOn(t *testing.T) {
	warn := &fakeWarnings{}
	server := hazardServer(t, warn, fakeHazards{})
	status, header, body := fetch(t, server.URL+"/v1/warnings?type=water_level&district=lk21&limit=5")
	if status != http.StatusOK || !strings.Contains(body, "Flood warning for Kelani River") {
		t.Fatalf("status %d, body %s", status, body)
	}
	if !strings.Contains(header.Get("Cache-Control"), "max-age=60") {
		t.Fatalf("cache-control = %q", header.Get("Cache-Control"))
	}
	if warn.filter.Type != "water_level" || warn.filter.District != "LK-21" || warn.filter.Limit != 5 {
		t.Fatalf("filter = %+v", warn.filter)
	}
}

func TestWarningsOverviewRejectsBadParameters(t *testing.T) {
	server := hazardServer(t, &fakeWarnings{}, fakeHazards{})
	cases := map[string]string{
		"?type=rumour":      "invalid_type",
		"?district=LK-2136": "invalid_district",
		"?district=kandy":   "invalid_district",
		"?limit=0":          "invalid_limit",
		"?limit=201":        "invalid_limit",
	}
	for query, code := range cases {
		status, _, body := fetch(t, server.URL+"/v1/warnings"+query)
		if status != http.StatusBadRequest || !strings.Contains(body, code) {
			t.Errorf("%s: status %d, body %s", query, status, body)
		}
	}
}

func TestLandslideLayerIsBareGeoJSON(t *testing.T) {
	server := hazardServer(t, &fakeWarnings{}, fakeHazards{})
	status, header, body := fetch(t, server.URL+"/v1/warnings/landslides")
	if status != http.StatusOK || !strings.HasPrefix(header.Get("Content-Type"), "application/geo+json") {
		t.Fatalf("status %d, content type %q", status, header.Get("Content-Type"))
	}
	var layer warnings.LandslideLayer
	if err := json.Unmarshal([]byte(body), &layer); err != nil {
		t.Fatal(err)
	}
	if layer.Type != "FeatureCollection" || !strings.Contains(layer.Meta.Note, "as far as we know") {
		t.Fatalf("layer = %+v", layer)
	}
}

func TestLandslideDivision(t *testing.T) {
	server := hazardServer(t, &fakeWarnings{}, fakeHazards{})
	if status, _, body := fetch(t, server.URL+"/v1/warnings/landslides/LK-2136"); status != http.StatusOK || !strings.Contains(body, "Yatinuwara") {
		t.Fatalf("status %d, body %s", status, body)
	}
	if status, _, body := fetch(t, server.URL+"/v1/warnings/landslides/LK-9999"); status != http.StatusNotFound || !strings.Contains(body, "division_not_found") {
		t.Fatalf("status %d, body %s", status, body)
	}
}

func TestHazardLayersAreBareGeoJSON(t *testing.T) {
	server := hazardServer(t, &fakeWarnings{}, fakeHazards{})
	for _, path := range []string{"/v1/hazards/earthquakes", "/v1/hazards/cyclones"} {
		status, header, body := fetch(t, server.URL+path)
		if status != http.StatusOK || !strings.HasPrefix(header.Get("Content-Type"), "application/geo+json") || !strings.Contains(body, "FeatureCollection") {
			t.Fatalf("%s: status %d, type %q, body %s", path, status, header.Get("Content-Type"), body)
		}
	}
	_, _, body := fetch(t, server.URL+"/v1/hazards/cyclones")
	if !strings.Contains(body, "No active cyclone in the North Indian Ocean") {
		t.Fatalf("body = %s", body)
	}
}

func TestFailuresAreInternalErrorsWithoutDetail(t *testing.T) {
	broken := errors.New("connection refused to the database at secret-host")
	server := hazardServer(t, &fakeWarnings{err: broken}, fakeHazards{err: broken})
	for _, path := range []string{"/v1/warnings", "/v1/warnings/landslides", "/v1/warnings/landslides/LK-2136", "/v1/hazards/earthquakes", "/v1/hazards/cyclones"} {
		status, _, body := fetch(t, server.URL+path)
		if status != http.StatusInternalServerError || strings.Contains(body, "secret-host") {
			t.Errorf("%s: status %d, body %s", path, status, body)
		}
	}
}

func TestUnconfiguredServicesAnswerUnavailable(t *testing.T) {
	server := hazardServer(t, nil, nil)
	for _, path := range []string{"/v1/warnings", "/v1/warnings/landslides", "/v1/warnings/landslides/LK-2136", "/v1/hazards/earthquakes", "/v1/hazards/cyclones"} {
		if status, _, body := fetch(t, server.URL+path); status != http.StatusServiceUnavailable || !strings.Contains(body, "unavailable") {
			t.Errorf("%s: status %d, body %s", path, status, body)
		}
	}
}

func TestHazardLayersAreAvailableInTheCatalogue(t *testing.T) {
	want := map[string]string{
		"landslide-warnings": "/v1/warnings/landslides",
		"official-warnings":  "/v1/warnings",
		"earthquakes":        "/v1/hazards/earthquakes",
		"cyclones":           "/v1/hazards/cyclones",
	}
	for id, path := range want {
		layer, ok := layers.ByID(id)
		if !ok || layer.Status != layers.StatusAvailable || layer.DataPath != path {
			t.Errorf("%s = %+v", id, layer)
		}
	}
}
