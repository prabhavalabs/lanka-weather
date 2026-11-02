//go:build integration

package metstations_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/metstations"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
)

// now is a day after the recorded 3-hourly file ends, and six days after the
// recorded daily file ends.
var now = time.Date(2026, 10, 29, 5, 0, 0, 0, time.UTC)

func recorded(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

type replay struct {
	t        *testing.T
	notFresh bool
	requests []source.Request
}

func (fake *replay) Do(_ context.Context, request source.Request) (source.Response, error) {
	fake.requests = append(fake.requests, request)
	switch request.URL {
	case metstations.EndpointThreeHourly:
		if fake.notFresh && request.Header.Get("If-None-Match") == `"v1"` {
			return source.Response{Status: http.StatusNotModified}, nil
		}
		return source.Response{Status: 200, Header: http.Header{"Etag": {`"v1"`}}, Body: recorded(fake.t, "alert_data_3h.json")}, nil
	case metstations.EndpointDaily:
		return source.Response{Status: http.StatusPartialContent, Body: recorded(fake.t, "daily_tail.json")}, nil
	case metstations.EndpointPlaces:
		return source.Response{Status: 200, Body: recorded(fake.t, "places.json")}, nil
	}
	fake.t.Fatalf("unexpected request %s", request.URL)
	return source.Response{}, nil
}

func TestCollectorsStoreAndServeRecordedFiles(t *testing.T) {
	pool := dbtest.Open(t, "met_readings_3h", "met_readings_daily", "met_stations", "source_versions")
	ctx := context.Background()
	fake := &replay{t: t}

	three := &metstations.ThreeHourlyCollector{Pool: pool, Client: fake}
	stored, err := three.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored == 0 {
		t.Fatal("no 3-hourly readings stored")
	}
	// A file that has not changed costs one conditional request and stores nothing.
	fake.notFresh = true
	if again, err := three.Collect(ctx, now.Add(time.Hour)); err != nil || again != 0 {
		t.Fatalf("unchanged file: stored %d, %v", again, err)
	}
	if got := fake.requests[len(fake.requests)-1].Header.Get("If-None-Match"); got != `"v1"` {
		t.Fatalf("conditional header = %q", got)
	}

	daily := &metstations.DailyCollector{Pool: pool, Client: fake}
	rows, err := daily.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if rows == 0 {
		t.Fatal("no daily readings stored")
	}
	if got := fake.requests[len(fake.requests)-2].Header.Get("Range"); got == "" {
		t.Fatal("the daily file should be read by range")
	}

	service := &stations.Service{Pool: pool, StaleAfter: 6 * time.Hour, Now: func() time.Time { return now }}
	layer, err := service.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]stations.Feature{}
	for _, feature := range layer.Features {
		byID[feature.ID] = feature
	}
	colombo := byID["colombo"].Properties
	if colombo.Reading != stations.ReadingThreeHourly || colombo.Stale || colombo.TemperatureC == nil || *colombo.TemperatureC != 28.1 {
		t.Fatalf("colombo = %+v", colombo)
	}
	if byID["colombo"].Geometry.Coordinates != [2]float64{79.867, 6.9} || colombo.PositionApproximate {
		t.Fatalf("colombo position = %+v", byID["colombo"].Geometry)
	}
	coldest := byID["nuwara-eliya"].Properties
	if coldest.TemperatureC == nil || *coldest.TemperatureC >= *colombo.TemperatureC {
		t.Fatalf("Nuwara Eliya = %+v", coldest)
	}
	if layer.Meta.ThreeHourly.Stale || !layer.Meta.Observed {
		t.Fatalf("meta = %+v", layer.Meta)
	}
	// The daily tail's Negombo has a geocoded position, flagged as approximate;
	// a name geocoded to nowhere (Ayyiththimalai) is not stored.
	if negombo, ok := byID["negombo"]; ok && !negombo.Properties.PositionApproximate {
		t.Fatalf("negombo = %+v", negombo)
	}
	if _, ok := byID["ayyiththimalai"]; ok {
		t.Fatal("a station with no position must not be stored")
	}

	// Four days on, nothing is current: the last reading stays, marked stale.
	later := &stations.Service{Pool: pool, StaleAfter: 6 * time.Hour, Now: func() time.Time { return now.Add(4 * 24 * time.Hour) }}
	old, err := later.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !old.Meta.Stale {
		t.Fatalf("meta = %+v", old.Meta)
	}
	for _, feature := range old.Features {
		if feature.ID == "colombo" && (!feature.Properties.Stale || feature.Properties.TemperatureC == nil) {
			t.Fatalf("colombo later = %+v", feature.Properties)
		}
	}

	detail, err := later.Station(ctx, "colombo", 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.ThreeHourly) == 0 || !detail.Station.Properties.Stale {
		t.Fatalf("detail = %+v", detail)
	}
	if _, err := later.Station(ctx, "nowhere", 7); err != stations.ErrStationNotFound {
		t.Fatalf("unknown station error = %v", err)
	}

	nearby, err := later.Nearest(ctx, 79.86, 6.92)
	if err != nil || nearby == nil || nearby.ID != "colombo" {
		t.Fatalf("nearest = %+v, %v", nearby, err)
	}
	rain, err := service.ObservedRainfall(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rain.Type != "FeatureCollection" || rain.Features == nil || !rain.Meta.Observed {
		t.Fatalf("rain = %+v", rain.Meta)
	}
}
