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
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/layers"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
)

type fakeRivers struct {
	period time.Duration
	err    error
}

func (fake *fakeRivers) Layer(context.Context) (rivers.FeatureCollection, error) {
	if fake.err != nil {
		return rivers.FeatureCollection{}, fake.err
	}
	return rivers.FeatureCollection{Type: "FeatureCollection", Features: []rivers.Feature{{
		Type: "Feature", ID: "hanwella",
		Geometry:   rivers.Point{Type: "Point", Coordinates: [2]float64{80.08, 6.91}},
		Properties: rivers.StationProperties{ID: "hanwella", Status: rivers.StatusNormal},
	}}}, nil
}

func (fake *fakeRivers) StationHistory(_ context.Context, id string, period time.Duration) (rivers.History, error) {
	fake.period = period
	if id != "hanwella" {
		return rivers.History{}, rivers.ErrStationNotFound
	}
	return rivers.History{Station: rivers.StationProperties{ID: id}}, nil
}

func newServer(t *testing.T, deps Dependencies) *httptest.Server {
	t.Helper()
	deps.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(NewRouter(deps))
	t.Cleanup(server.Close)
	return server
}

func get(t *testing.T, server *httptest.Server, path string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(server.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, body
}

func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error == nil {
		t.Fatalf("expected an error body, got %s", body)
	}
	return envelope.Error.Code
}

func TestHealthAndReadiness(t *testing.T) {
	server := newServer(t, Dependencies{Ready: func(context.Context) error { return nil }})
	if resp, _ := get(t, server, "/healthz"); resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status = %d", resp.StatusCode)
	}
	if resp, _ := get(t, server, "/readyz"); resp.StatusCode != http.StatusOK {
		t.Fatalf("readyz status = %d", resp.StatusCode)
	}

	down := newServer(t, Dependencies{Ready: func(context.Context) error { return errors.New("no database") }})
	resp, body := get(t, down, "/readyz")
	if resp.StatusCode != http.StatusServiceUnavailable || errorCode(t, body) != "not_ready" {
		t.Fatalf("readyz when down = %d %s", resp.StatusCode, body)
	}
}

func TestListLayers(t *testing.T) {
	resp, body := get(t, newServer(t, Dependencies{}), "/v1/layers")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var envelope struct {
		Data []layers.Layer `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != len(layers.All()) {
		t.Fatalf("got %d layers, want %d", len(envelope.Data), len(layers.All()))
	}
	if resp.Header.Get("Cache-Control") == "" {
		t.Fatal("expected a Cache-Control header")
	}
}

func TestGetLayer(t *testing.T) {
	server := newServer(t, Dependencies{})
	if resp, _ := get(t, server, "/v1/layers/river-levels"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	resp, body := get(t, server, "/v1/layers/missing")
	if resp.StatusCode != http.StatusNotFound || errorCode(t, body) != "layer_not_found" {
		t.Fatalf("missing layer = %d %s", resp.StatusCode, body)
	}
}

func TestRiverLevels(t *testing.T) {
	resp, body := get(t, newServer(t, Dependencies{Rivers: &fakeRivers{}}), "/v1/rivers/levels")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/geo+json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	var collection rivers.FeatureCollection
	if err := json.Unmarshal(body, &collection); err != nil {
		t.Fatal(err)
	}
	if collection.Type != "FeatureCollection" || len(collection.Features) != 1 {
		t.Fatalf("collection = %+v", collection)
	}

	resp, body = get(t, newServer(t, Dependencies{Rivers: &fakeRivers{err: errors.New("boom")}}), "/v1/rivers/levels")
	if resp.StatusCode != http.StatusInternalServerError || errorCode(t, body) != "internal" {
		t.Fatalf("failure = %d %s", resp.StatusCode, body)
	}
}

func TestStationReadings(t *testing.T) {
	fake := &fakeRivers{}
	server := newServer(t, Dependencies{Rivers: fake})

	if resp, _ := get(t, server, "/v1/rivers/stations/hanwella/readings"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if fake.period != 72*time.Hour {
		t.Fatalf("default period = %v, want 72h", fake.period)
	}
	if resp, _ := get(t, server, "/v1/rivers/stations/hanwella/readings?hours=24"); resp.StatusCode != http.StatusOK || fake.period != 24*time.Hour {
		t.Fatalf("hours=24: status %d, period %v", resp.StatusCode, fake.period)
	}
	for _, bad := range []string{"0", "721", "abc"} {
		resp, body := get(t, server, "/v1/rivers/stations/hanwella/readings?hours="+bad)
		if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != "invalid_hours" {
			t.Fatalf("hours=%s: %d %s", bad, resp.StatusCode, body)
		}
	}
	resp, body := get(t, server, "/v1/rivers/stations/nowhere/readings")
	if resp.StatusCode != http.StatusNotFound || errorCode(t, body) != "station_not_found" {
		t.Fatalf("unknown station = %d %s", resp.StatusCode, body)
	}
}
