package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/layers"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
)

type fakeReservoirs struct {
	days int
	err  error
}

func (fake *fakeReservoirs) Layer(context.Context) (reservoirs.FeatureCollection, error) {
	if fake.err != nil {
		return reservoirs.FeatureCollection{}, fake.err
	}
	storage := 44.6
	return reservoirs.FeatureCollection{Type: "FeatureCollection", Features: []reservoirs.Feature{
		{Type: "Feature", ID: "minneriya", Geometry: &reservoirs.Point{Type: "Point", Coordinates: [2]float64{80.86, 8.02}},
			Properties: reservoirs.Properties{ID: "minneriya", Status: "releasing", StoragePercent: &storage, Mapped: true}},
		{Type: "Feature", ID: "ellewela", Properties: reservoirs.Properties{ID: "ellewela", Status: "spilling"}},
	}}, nil
}

func (fake *fakeReservoirs) Areas(context.Context) (reservoirs.Areas, error) {
	return reservoirs.Areas{Type: "FeatureCollection", Features: []reservoirs.AreaFeature{}}, fake.err
}

func (fake *fakeReservoirs) History(_ context.Context, id string, days int) (reservoirs.History, error) {
	fake.days = days
	if fake.err != nil {
		return reservoirs.History{}, fake.err
	}
	if id != "minneriya" {
		return reservoirs.History{}, reservoirs.ErrNotFound
	}
	return reservoirs.History{Reservoir: reservoirs.Properties{ID: id}, Note: reservoirs.Note, Readings: []reservoirs.Reading{}}, nil
}

func TestReservoirLayer(t *testing.T) {
	server := newServer(t, Dependencies{Reservoirs: &fakeReservoirs{}})
	resp, body := get(t, server, "/v1/reservoirs")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/geo+json; charset=utf-8" {
		t.Fatalf("status %d, type %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var collection struct {
		Type     string `json:"type"`
		Features []struct {
			ID       string           `json:"id"`
			Geometry *json.RawMessage `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &collection); err != nil {
		t.Fatal(err)
	}
	if collection.Type != "FeatureCollection" || len(collection.Features) != 2 {
		t.Fatalf("collection = %s", body)
	}
	// A reservoir with no position has a null geometry and is still listed.
	if collection.Features[0].Geometry == nil || collection.Features[1].Geometry != nil {
		t.Fatalf("geometries = %s", body)
	}
}

func TestReservoirAreas(t *testing.T) {
	server := newServer(t, Dependencies{Reservoirs: &fakeReservoirs{}})
	if resp, _ := get(t, server, "/v1/reservoirs/areas"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestReservoirReadings(t *testing.T) {
	fake := &fakeReservoirs{}
	server := newServer(t, Dependencies{Reservoirs: fake})

	if resp, _ := get(t, server, "/v1/reservoirs/minneriya/readings"); resp.StatusCode != http.StatusOK || fake.days != defaultReservoirDays {
		t.Fatalf("status %d, days %d", resp.StatusCode, fake.days)
	}
	if resp, _ := get(t, server, "/v1/reservoirs/minneriya/readings?days=10"); resp.StatusCode != http.StatusOK || fake.days != 10 {
		t.Fatalf("status %d, days %d", resp.StatusCode, fake.days)
	}
	for _, bad := range []string{"0", "401", "x", "1.5"} {
		resp, body := get(t, server, "/v1/reservoirs/minneriya/readings?days="+bad)
		if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != "invalid_days" {
			t.Errorf("days=%s: status %d, body %s", bad, resp.StatusCode, body)
		}
	}
	resp, body := get(t, server, "/v1/reservoirs/nowhere/readings")
	if resp.StatusCode != http.StatusNotFound || errorCode(t, body) != "reservoir_not_found" {
		t.Fatalf("status %d, body %s", resp.StatusCode, body)
	}
}

func TestReservoirsUnavailableAndFailing(t *testing.T) {
	unconfigured := newServer(t, Dependencies{})
	for _, path := range []string{"/v1/reservoirs", "/v1/reservoirs/areas", "/v1/reservoirs/x/readings"} {
		resp, body := get(t, unconfigured, path)
		if resp.StatusCode != http.StatusServiceUnavailable || errorCode(t, body) != "unavailable" {
			t.Errorf("%s: status %d, body %s", path, resp.StatusCode, body)
		}
	}
	failing := newServer(t, Dependencies{Reservoirs: &fakeReservoirs{err: errors.New("database is down")}})
	for _, path := range []string{"/v1/reservoirs", "/v1/reservoirs/areas", "/v1/reservoirs/minneriya/readings"} {
		resp, body := get(t, failing, path)
		if resp.StatusCode != http.StatusInternalServerError || errorCode(t, body) != "internal" {
			t.Errorf("%s: status %d, body %s", path, resp.StatusCode, body)
		}
	}
}

func TestReservoirsLayerIsAvailable(t *testing.T) {
	layer, ok := layers.ByID("reservoirs")
	if !ok || layer.Status != layers.StatusAvailable || layer.DataPath != "/v1/reservoirs" {
		t.Fatalf("layer = %+v", layer)
	}
	server := newServer(t, Dependencies{})
	if resp, _ := get(t, server, "/v1/layers/reservoirs"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
