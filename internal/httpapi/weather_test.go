package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/summary"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

type fakeWeather struct {
	noData      bool
	windAt      time.Time
	regionsErr  error
	pointCalled [2]float64
	hours, days int
}

func (fake *fakeWeather) CurrentGridJSON(context.Context) ([]byte, []byte, error) {
	if fake.noData {
		return nil, nil, weather.ErrNoData
	}
	plain := []byte(`{"grid":{"nx":15}}`)
	var zipped bytes.Buffer
	writer := gzip.NewWriter(&zipped)
	_, _ = writer.Write(plain)
	_ = writer.Close()
	return plain, zipped.Bytes(), nil
}

func (fake *fakeWeather) Point(_ context.Context, lon, lat float64, hours, days int) (*weather.Point, error) {
	fake.pointCalled, fake.hours, fake.days = [2]float64{lon, lat}, hours, days
	if lon < 79 {
		return nil, weather.ErrOutsideGrid
	}
	return &weather.Point{Location: weather.Location{Lat: lat, Lon: lon}}, nil
}

func (fake *fakeWeather) WindJSON(_ context.Context, at time.Time) ([]byte, []byte, error) {
	fake.windAt = at
	if at.Year() == 2030 {
		return nil, nil, weather.ErrNoWind
	}
	return []byte(`{"u":"AAA="}`), nil, nil
}

func (fake *fakeWeather) RegionWeather(_ context.Context, level string) (*weather.RegionsResponse, error) {
	if fake.regionsErr != nil {
		return nil, fake.regionsErr
	}
	return &weather.RegionsResponse{Level: level}, nil
}

type fakeFloods struct{}

func (fakeFloods) Layer(context.Context) (floods.FeatureCollection, error) {
	return floods.FeatureCollection{Type: "FeatureCollection", Features: []floods.Feature{}}, nil
}

type fakeSummary struct{}

func (fakeSummary) Summary(_ context.Context, lon, lat float64) (*summary.Summary, error) {
	return &summary.Summary{Location: weather.Location{Lat: lat, Lon: lon}, Alerts: []summary.Alert{}}, nil
}

func TestWeatherCurrentServesGzip(t *testing.T) {
	server := newServer(t, Dependencies{Weather: &fakeWeather{}})
	request, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/weather/current", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	// Go's transport would decode gzip itself; asking explicitly keeps it raw.
	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.Header.Get("Content-Encoding") != "gzip" || response.Header.Get("Cache-Control") != "public, max-age=300" {
		t.Fatalf("headers = %v", response.Header)
	}
	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(reader)
	if string(body) != `{"grid":{"nx":15}}` {
		t.Fatalf("body = %s", body)
	}

	request.Header.Del("Accept-Encoding")
	plain, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = plain.Body.Close() }()
	if plain.Header.Get("Content-Encoding") != "" {
		t.Fatal("a client that does not take gzip gets plain JSON")
	}

	resp, body := get(t, newServer(t, Dependencies{Weather: &fakeWeather{noData: true}}), "/v1/weather/current")
	if resp.StatusCode != http.StatusServiceUnavailable || errorCode(t, body) != "no_data" {
		t.Fatalf("no data = %d %s", resp.StatusCode, body)
	}
}

func TestWeatherPoint(t *testing.T) {
	fake := &fakeWeather{}
	server := newServer(t, Dependencies{Weather: fake})
	resp, body := get(t, server, "/v1/weather/point?lat=7.2906&lon=80.6337")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d %s", resp.StatusCode, body)
	}
	if fake.pointCalled != [2]float64{80.6337, 7.2906} || fake.hours != 72 || fake.days != 7 {
		t.Fatalf("called with %v, %d hours, %d days", fake.pointCalled, fake.hours, fake.days)
	}
	for _, bad := range []string{"", "?lat=7", "?lat=abc&lon=80", "?lat=91&lon=80", "?lat=7&lon=181"} {
		resp, body := get(t, server, "/v1/weather/point"+bad)
		if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != "invalid_coordinates" {
			t.Fatalf("%q = %d %s", bad, resp.StatusCode, body)
		}
	}
	resp, body = get(t, server, "/v1/weather/point?lat=7&lon=70")
	if resp.StatusCode != http.StatusNotFound || errorCode(t, body) != "outside_grid" {
		t.Fatalf("outside = %d %s", resp.StatusCode, body)
	}
}

func TestWeatherWind(t *testing.T) {
	fake := &fakeWeather{}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	server := newServer(t, Dependencies{Weather: fake, Now: func() time.Time { return now }})
	if resp, body := get(t, server, "/v1/weather/wind"); resp.StatusCode != http.StatusOK || !fake.windAt.Equal(now) || string(body) != `{"u":"AAA="}` {
		t.Fatalf("default = %d %s at %v", resp.StatusCode, body, fake.windAt)
	}
	if resp, _ := get(t, server, "/v1/weather/wind?t=2026-11-03T06:00:00Z"); resp.StatusCode != http.StatusOK ||
		!fake.windAt.Equal(time.Date(2026, 11, 3, 6, 0, 0, 0, time.UTC)) {
		t.Fatalf("t = %d at %v", resp.StatusCode, fake.windAt)
	}
	resp, body := get(t, server, "/v1/weather/wind?t=tomorrow")
	if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != "invalid_time" {
		t.Fatalf("bad t = %d %s", resp.StatusCode, body)
	}
	resp, body = get(t, server, "/v1/weather/wind?t=2030-01-01T00:00:00Z")
	if resp.StatusCode != http.StatusNotFound || errorCode(t, body) != "no_wind_field" {
		t.Fatalf("no field = %d %s", resp.StatusCode, body)
	}
}

func TestWeatherRegions(t *testing.T) {
	server := newServer(t, Dependencies{Weather: &fakeWeather{}})
	if resp, body := get(t, server, "/v1/weather/regions?level=district"); resp.StatusCode != http.StatusOK {
		t.Fatalf("district = %d %s", resp.StatusCode, body)
	}
	resp, body := get(t, server, "/v1/weather/regions?level=village")
	if resp.StatusCode != http.StatusBadRequest || errorCode(t, body) != "invalid_level" {
		t.Fatalf("bad level = %d %s", resp.StatusCode, body)
	}
	stub := newServer(t, Dependencies{Weather: &fakeWeather{regionsErr: weather.ErrRegionsUnavailable}})
	resp, body = get(t, stub, "/v1/weather/regions?level=gn")
	if resp.StatusCode != http.StatusServiceUnavailable || errorCode(t, body) != "not_available" {
		t.Fatalf("stub = %d %s", resp.StatusCode, body)
	}
	broken := newServer(t, Dependencies{Weather: &fakeWeather{regionsErr: errors.New("boom")}})
	if resp, body := get(t, broken, "/v1/weather/regions?level=ds"); resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("failure = %d %s", resp.StatusCode, body)
	}
}

func TestFloodForecastAndSummary(t *testing.T) {
	server := newServer(t, Dependencies{Floods: fakeFloods{}, Summary: fakeSummary{}})
	resp, body := get(t, server, "/v1/floods/forecast")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/geo+json; charset=utf-8" {
		t.Fatalf("floods = %d %s", resp.StatusCode, body)
	}
	resp, body = get(t, server, "/v1/summary?lat=6.9271&lon=79.8612")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("summary = %d %s", resp.StatusCode, body)
	}
	var envelope struct {
		Data summary.Summary `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Data.Location.Lat != 6.9271 {
		t.Fatalf("summary body = %s", body)
	}
	if resp, body := get(t, server, "/v1/summary?lat=x&lon=79"); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad summary = %d %s", resp.StatusCode, body)
	}
	unconfigured := newServer(t, Dependencies{})
	for _, path := range []string{"/v1/weather/current", "/v1/weather/point?lat=7&lon=80", "/v1/weather/wind", "/v1/weather/regions?level=gn", "/v1/floods/forecast", "/v1/summary?lat=7&lon=80"} {
		if resp, body := get(t, unconfigured, path); resp.StatusCode != http.StatusServiceUnavailable || errorCode(t, body) != "unavailable" {
			t.Fatalf("%s unconfigured = %d %s", path, resp.StatusCode, body)
		}
	}
}

func TestAcceptsGzip(t *testing.T) {
	cases := map[string]bool{"gzip": true, "br, gzip;q=0.8": true, "gzip;q=0": false, "identity": false, "": false, "GZIP": true}
	for header, want := range cases {
		if got := acceptsGzip(header); got != want {
			t.Errorf("acceptsGzip(%q) = %v", header, got)
		}
	}
}
