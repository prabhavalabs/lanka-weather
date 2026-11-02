//go:build integration

package googleflood_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const testKey = "test-key-not-real"

// fakeGoogle answers the Flood Forecasting endpoints from the recorded
// fixtures and checks every request carries the key in its header only.
type fakeGoogle struct {
	t      *testing.T
	mutex  sync.Mutex
	bodies map[string][]string
}

func (fake *fakeGoogle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Goog-Api-Key") != testKey || strings.Contains(r.URL.RawQuery, testKey) {
		http.Error(w, `{"error":{"code":403}}`, http.StatusForbidden)
		return
	}
	body := make([]byte, r.ContentLength)
	if r.ContentLength > 0 {
		_, _ = r.Body.Read(body)
	}
	fake.mutex.Lock()
	fake.bodies[r.URL.Path] = append(fake.bodies[r.URL.Path], string(body))
	fake.mutex.Unlock()
	read := func(name string) {
		payload, err := os.ReadFile("testdata/" + name)
		if err != nil {
			fake.t.Error(err)
		}
		_, _ = w.Write(payload)
	}
	switch {
	case r.URL.Path == "/v1/floodStatus:searchLatestFloodStatusByArea":
		if strings.Contains(string(body), "page-2") {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		read("statuses_lk.json")
	case r.URL.Path == "/v1/gaugeModels:batchGet":
		read("models.json")
	case r.URL.Path == "/v1/flashFloods:search":
		read("flash_floods.json")
	case r.URL.Path == "/v1/significantEvents:search":
		read("significant_events.json")
	case strings.HasPrefix(r.URL.Path, "/v1/serializedPolygons/"):
		if strings.HasSuffix(r.URL.Path, "poly-low") {
			http.Error(w, `{"error":{"code":404}}`, http.StatusNotFound)
			return
		}
		kml := `<kml><Polygon><outerBoundaryIs><LinearRing><coordinates>81.0,8.1 81.4,8.1 81.4,8.4 81.0,8.4 81.0,8.1</coordinates></LinearRing></outerBoundaryIs></Polygon></kml>`
		_ = json.NewEncoder(w).Encode(map[string]string{"kml": kml})
	default:
		http.NotFound(w, r)
	}
}

func TestCollectStoresSriLankaForecasts(t *testing.T) {
	pool := dbtest.Open(t, "flood_gauges", "flood_areas", "source_runs")
	ctx := context.Background()
	fake := &fakeGoogle{t: t, bodies: map[string][]string{}}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	now := time.Date(2026, 11, 2, 13, 0, 0, 0, time.UTC)

	collector := &googleflood.Collector{Pool: pool, Client: source.NewClient("test"), APIKey: testKey, BaseURL: server.URL}
	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	// Two gauges; the severe gauge's notification and high inundation
	// polygons, two Sri Lankan flash-flood polygons and one event (the low
	// inundation polygon fails to load).
	if stored != 2+5 {
		t.Fatalf("stored %d", stored)
	}
	statusBodies := fake.bodies["/v1/floodStatus:searchLatestFloodStatusByArea"]
	if len(statusBodies) != 2 || !strings.Contains(statusBodies[0], `"regionCode":"LK"`) || strings.Contains(statusBodies[0], "includeNonQualityVerified") {
		t.Fatalf("status requests = %v", statusBodies)
	}

	service := &floods.Service{Pool: pool, Configured: true, StaleAfter: 30 * time.Hour, Now: func() time.Time { return now }}
	layer, err := service.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if layer.Meta.Gauges != 2 || layer.Meta.Areas != 5 || len(layer.Features) != 7 {
		t.Fatalf("layer meta = %+v, %d features", layer.Meta, len(layer.Features))
	}
	gauges, err := service.Gauges(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var severe floods.Gauge
	for _, gauge := range gauges {
		if gauge.GaugeID == "hybas_4121156400" {
			severe = gauge
		}
	}
	if severe.Severity != "severe" || severe.Thresholds.Unit != "m" || severe.ForecastEnd == nil {
		t.Fatalf("severe gauge = %+v", severe)
	}
	areas, err := service.AreasAt(ctx, 81.2, 8.25)
	if err != nil || len(areas) != 5 {
		t.Fatalf("areas at the gauge = %+v, %v", areas, err)
	}
	if areas, _ := service.AreasAt(ctx, 79.86, 6.93); len(areas) != 0 {
		t.Fatalf("Colombo is in no flood area: %+v", areas)
	}

	// Nothing older than a day is served or kept.
	later := now.Add(25 * time.Hour)
	service.Now = func() time.Time { return later }
	if gauges, _ := service.Gauges(ctx); len(gauges) != 0 {
		t.Fatalf("day-old gauges served: %d", len(gauges))
	}
	if err := collector.Prune(ctx, later); err != nil {
		t.Fatal(err)
	}
	if rows, _ := store.New(pool).ListFloodGauges(ctx); len(rows) != 0 {
		t.Fatalf("day-old gauges kept: %d", len(rows))
	}

	if _, err := (&googleflood.Collector{Pool: pool, Client: source.NewClient("test"), BaseURL: server.URL}).Collect(ctx, now); err == nil {
		t.Fatal("collecting without a key must fail")
	}
}
