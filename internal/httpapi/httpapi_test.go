package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prabhavalabs/open-weather/internal/layers"
)

func get(t *testing.T, path string) (*http.Response, []byte) {
	t.Helper()
	server := httptest.NewServer(NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(server.Close)
	resp, err := http.Get(server.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, body
}

func TestHealthz(t *testing.T) {
	resp, _ := get(t, "/healthz")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestListLayers(t *testing.T) {
	resp, body := get(t, "/v1/layers")
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
	if resp, _ := get(t, "/v1/layers/river-levels"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	resp, body := get(t, "/v1/layers/missing")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error == nil || envelope.Error.Code != "layer_not_found" {
		t.Fatalf("unexpected error body: %s", body)
	}
}
