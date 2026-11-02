package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/rainfall"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

type fakeStations struct{ days int }

func (fake *fakeStations) Layer(context.Context) (stations.FeatureCollection, error) {
	return stations.FeatureCollection{Type: "FeatureCollection", Features: []stations.Feature{}}, nil
}

func (fake *fakeStations) Station(_ context.Context, id string, days int) (*stations.Detail, error) {
	fake.days = days
	if id != "colombo" {
		return nil, stations.ErrStationNotFound
	}
	return &stations.Detail{}, nil
}

func (fake *fakeStations) ObservedRainfall(context.Context) (*stations.RainCollection, error) {
	return &stations.RainCollection{Type: "FeatureCollection", Features: []stations.RainFeature{}}, nil
}

type fakeRain struct{ empty bool }

func (fake *fakeRain) Steps(context.Context) (*rainfall.Steps, error) {
	if fake.empty {
		return &rainfall.Steps{Steps: []rainfall.Step{}}, nil
	}
	return &rainfall.Steps{Steps: []rainfall.Step{{Time: "20261102T1330Z"}}}, nil
}

func (fake *fakeRain) Tile(_ context.Context, timeID string, z, x, y int) ([]byte, error) {
	if timeID != "20261102T1330Z" || z != 6 || x != 46 || y != 30 {
		return nil, rainfall.ErrTileNotFound
	}
	return []byte("png"), nil
}

type fakeEnvironment struct{ err error }

func (fake *fakeEnvironment) AirJSON(context.Context) ([]byte, []byte, error) {
	return []byte(`{"values":{"usAqi":[1]}}`), nil, fake.err
}

func (fake *fakeEnvironment) SeaJSON(context.Context) ([]byte, []byte, error) {
	return []byte(`{"values":{"waveHeightM":[1]}}`), nil, fake.err
}

func observedRouter(deps Dependencies) http.Handler {
	deps.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(deps)
}

func observedGet(handler http.Handler, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func observedErrorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil || envelope.Error == nil {
		t.Fatalf("no error envelope in %s", recorder.Body)
	}
	return envelope.Error.Code
}

func TestStationEndpoints(t *testing.T) {
	fake := &fakeStations{}
	router := observedRouter(Dependencies{Stations: fake})
	for _, path := range []string{"/v1/stations", "/v1/rain/observed"} {
		response := observedGet(router, path)
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/geo+json; charset=utf-8" {
			t.Errorf("%s: %d %s", path, response.Code, response.Header().Get("Content-Type"))
		}
	}
	if response := observedGet(router, "/v1/stations/colombo"); response.Code != http.StatusOK || fake.days != 7 {
		t.Errorf("station: %d, days %d", response.Code, fake.days)
	}
	if response := observedGet(router, "/v1/stations/colombo?days=30"); response.Code != http.StatusOK || fake.days != 30 {
		t.Errorf("30 days: %d, days %d", response.Code, fake.days)
	}
	response := observedGet(router, "/v1/stations/nowhere")
	if response.Code != http.StatusNotFound || observedErrorCode(t, response) != "station_not_found" {
		t.Errorf("unknown station: %d %s", response.Code, response.Body)
	}
	for _, days := range []string{"0", "46", "x"} {
		response := observedGet(router, "/v1/stations/colombo?days="+days)
		if response.Code != http.StatusBadRequest || observedErrorCode(t, response) != "invalid_days" {
			t.Errorf("days=%s: %d", days, response.Code)
		}
	}
}

func TestRainEndpoints(t *testing.T) {
	router := observedRouter(Dependencies{Rain: &fakeRain{}})
	if response := observedGet(router, "/v1/rain/steps"); response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "public, max-age=120" {
		t.Errorf("steps: %d %v", response.Code, response.Header())
	}
	tile := observedGet(router, "/v1/rain/tiles/20261102T1330Z/6/46/30.png")
	if tile.Code != http.StatusOK || tile.Header().Get("Content-Type") != "image/png" || tile.Body.String() != "png" {
		t.Errorf("tile: %d %v", tile.Code, tile.Header())
	}
	if tile.Header().Get("Cache-Control") != "public, max-age=604800, immutable" {
		t.Errorf("tile cache = %s", tile.Header().Get("Cache-Control"))
	}
	if missing := observedGet(router, "/v1/rain/tiles/20261102T1330Z/6/1/1.png"); missing.Code != http.StatusNotFound || observedErrorCode(t, missing) != "tile_not_found" {
		t.Errorf("missing tile: %d", missing.Code)
	}
	if bad := observedGet(router, "/v1/rain/tiles/20261102T1330Z/six/1/1.png"); bad.Code != http.StatusBadRequest {
		t.Errorf("bad tile: %d", bad.Code)
	}
	empty := observedRouter(Dependencies{Rain: &fakeRain{empty: true}})
	if response := observedGet(empty, "/v1/rain/steps"); response.Code != http.StatusServiceUnavailable || observedErrorCode(t, response) != "no_data" {
		t.Errorf("no steps: %d %s", response.Code, response.Body)
	}
}

func TestEnvironmentEndpoints(t *testing.T) {
	router := observedRouter(Dependencies{Environment: &fakeEnvironment{}})
	for path, want := range map[string]string{"/v1/air/current": "usAqi", "/v1/sea/current": "waveHeightM"} {
		response := observedGet(router, path)
		var body struct {
			Values map[string][]float64 `json:"values"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || response.Code != http.StatusOK || body.Values[want] == nil {
			t.Errorf("%s: %d %s", path, response.Code, response.Body)
		}
	}
	waiting := observedRouter(Dependencies{Environment: &fakeEnvironment{err: weather.ErrNoData}})
	if response := observedGet(waiting, "/v1/air/current"); response.Code != http.StatusServiceUnavailable || observedErrorCode(t, response) != "no_data" {
		t.Errorf("no data: %d", response.Code)
	}
	failing := observedRouter(Dependencies{Environment: &fakeEnvironment{err: errors.New("boom")}})
	if response := observedGet(failing, "/v1/sea/current"); response.Code != http.StatusInternalServerError {
		t.Errorf("failure: %d", response.Code)
	}
}

func TestObservedEndpointsAnswerWhenNotConfigured(t *testing.T) {
	router := observedRouter(Dependencies{})
	for _, path := range []string{"/v1/stations", "/v1/stations/x", "/v1/rain/observed", "/v1/rain/steps", "/v1/rain/tiles/x/6/1/1.png", "/v1/air/current", "/v1/sea/current"} {
		if response := observedGet(router, path); response.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: %d", path, response.Code)
		}
	}
}
