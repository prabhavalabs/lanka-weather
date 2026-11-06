//go:build integration

package alerts_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/alerts"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/push/webpush"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

func reading(t *testing.T, pool *pgxpool.Pool, at time.Time, level float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO river_readings (station_id, observed_at, water_level, fetched_at)
		VALUES ('nagalagam', $1, $2, $1)`, at, level); err != nil {
		t.Fatal(err)
	}
}

func TestAGaugeCrossingMinorFloodReachesItsDistrict(t *testing.T) {
	pool := dbtest.Open(t, "push_subscriptions", "push_events", "push_alert_state", "regions", "river_stations", "source_runs")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO regions (id, fid, level, name, name_si, name_ta, province_id, district_id, area_km2, centroid, label_point,
		                     west, south, east, north, geom, geom_simple, geom_3857, geom_simple_3857, source, source_version, source_time, fetched_at)
		SELECT 'LK-11', 11, 'district', 'Colombo', 'කොළඹ', 'கொழும்பு', 'LK-1', 'LK-11', 700, ST_Centroid(g), ST_PointOnSurface(g),
		       79.8, 6.8, 80.2, 7.0, g, g, ST_Transform(g, 3857), ST_Transform(g, 3857), 'test', 'v', now(), now()
		FROM (SELECT ST_Multi(ST_MakeEnvelope(79.8, 6.8, 80.2, 7.0, 4326)) AS g) s`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO river_stations (id, name, basin, lon, lat, unit, alert_level, minor_flood_level, major_flood_level, source, updated_at)
		VALUES ('nagalagam', 'Nagalagam Street', 'Kelani Ganga', 79.879, 6.959, 'ft', 5, 7, 9, 'irrigation-gauges', now())`); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	received := []*http.Request{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		received = append(received, r)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	public, private, _ := webpush.GenerateKeys()
	vapid, _ := webpush.NewVAPID(public, private, "mailto:alerts@example.org")
	client := &webpush.Client{VAPID: vapid, HTTP: server.Client()}

	clock := time.Date(2026, 11, 3, 4, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	key, _ := ecdh.P256().GenerateKey(rand.Reader)
	secret := make([]byte, 16)
	_, _ = rand.Read(secret)
	var subscription push.SubscriptionJSON
	subscription.Endpoint = server.URL + "/colombo"
	subscription.Keys.P256dh = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	subscription.Keys.Auth = base64.RawURLEncoding.EncodeToString(secret)
	service := &push.Service{Pool: pool, VAPID: vapid, Sender: client, AllowEndpoint: func(string) error { return nil }, Now: now}
	if _, _, err := service.Subscribe(ctx, subscription, push.Preferences{Topics: []push.Topic{push.TopicRiver}, MinSeverity: push.SeverityWarning,
		Areas: push.Areas{Districts: []string{"LK-11"}}}, "en", ""); err != nil {
		t.Fatal(err)
	}

	engine := &alerts.Engine{Pool: pool, Rivers: &rivers.Service{Pool: pool, StaleAfter: 3 * time.Hour, Now: now},
		Dispatcher: &push.Dispatcher{Pool: pool, Sender: client, Now: now}}
	run := func() int {
		t.Helper()
		sent, err := engine.Collect(ctx, clock)
		if err != nil {
			t.Fatal(err)
		}
		return sent
	}

	// At alert level: a watch notice, below this subscriber's threshold.
	clock = clock.Add(15 * time.Minute)
	reading(t, pool, clock, 5.5)
	if sent := run(); sent != 0 {
		t.Fatalf("alert level sent %d", sent)
	}
	// Minor flood: one notice.
	clock = clock.Add(15 * time.Minute)
	reading(t, pool, clock, 7.2)
	if sent := run(); sent != 1 {
		t.Fatalf("minor flood sent %d", sent)
	}
	// Hovering: in and out of minor flood sends nothing more.
	for _, level := range []float64{6.9, 7.1, 6.95, 7.05} {
		clock = clock.Add(15 * time.Minute)
		reading(t, pool, clock, level)
		if sent := run(); sent != 0 {
			t.Fatalf("hovering at %.2f sent %d", level, sent)
		}
	}
	events, err := store.New(pool).ListActivePushEvents(ctx, clock)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[1].Kind != "river" || events[1].Variant != "minor_flood" || events[1].Districts[0] != "LK-11" ||
		events[1].Url != "/?station=nagalagam" {
		t.Fatalf("events %+v", events)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || received[0].Header.Get("Urgency") != "high" || received[0].Header.Get("Topic") != webpush.TopicFor("river:station:nagalagam") {
		t.Fatalf("received %v", received)
	}
}

// TestACloudburstUnderGooglesForecastRaisesRainAndRiskNotices runs the engine
// over a Colombo rain gauge and a stored flash-flood area: the gauge alerts
// heavy rain, and the two together raise the combined estimate to high.
func TestACloudburstUnderGooglesForecastRaisesRainAndRiskNotices(t *testing.T) {
	pool := dbtest.Open(t, "push_subscriptions", "push_events", "push_alert_state", "regions", "rain_gauge_readings", "rain_gauges",
		"flood_areas", "flood_gauges", "source_runs")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO regions (id, fid, level, name, name_si, name_ta, province_id, district_id, area_km2, centroid, label_point,
		                     west, south, east, north, geom, geom_simple, geom_3857, geom_simple_3857, source, source_version, source_time, fetched_at)
		SELECT 'LK-11', 11, 'district', 'Colombo', 'කොළඹ', 'கொழும்பு', 'LK-1', 'LK-11', 700, ST_Centroid(g), ST_PointOnSurface(g),
		       79.8, 6.8, 80.2, 7.0, g, g, ST_Transform(g, 3857), ST_Transform(g, 3857), 'test', 'v', now(), now()
		FROM (SELECT ST_Multi(ST_MakeEnvelope(79.8, 6.8, 80.2, 7.0, 4326)) AS g) s`); err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 11, 6, 13, 5, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	queries := store.New(pool)
	at := func(hour, minute int) time.Time { return time.Date(2026, 11, 6, hour, minute, 0, 0, time.UTC) }
	latest, latestMm := at(12, 50), 83.5
	if err := queries.UpsertRainGauge(ctx, store.UpsertRainGaugeParams{ID: "argcbo04", Device: "argcbo04", Name: "DOM HEAD OFFICE",
		Institution: "Met Department", Lon: 79.8717, Lat: 6.905, LatestObservedAt: &latest, LatestRainMm: &latestMm,
		Source: "hmis-rain-live", SourceTime: at(13, 3), FetchedAt: at(13, 3)}); err != nil {
		t.Fatal(err)
	}
	points := []store.InsertRainGaugeReadingParams{}
	for _, point := range []struct {
		at time.Time
		mm float64
	}{{at(11, 50), 16}, {at(12, 20), 26.5}, {at(12, 50), 83.5}} {
		points = append(points, store.InsertRainGaugeReadingParams{GaugeID: "argcbo04", ObservedAt: point.at, RainMm: point.mm,
			Source: "hmis-rain-live", SourceTime: at(13, 3), FetchedAt: at(13, 3)})
	}
	if err := queries.InsertRainGaugeReading(ctx, points).Close(); err != nil {
		t.Fatal(err)
	}
	issued, until := at(10, 0), at(10, 0).Add(24*time.Hour)
	if err := queries.InsertFloodArea(ctx, store.InsertFloodAreaParams{ID: "flash-colombo", Kind: "flash_likely", Source: "google-flood-forecasting",
		Severity: "likely", IssuedAt: &issued, ValidFrom: &issued, ValidUntil: &until, Properties: []byte(`{}`),
		Geojson:   `{"type":"Polygon","coordinates":[[[79.85,6.85],[79.95,6.85],[79.95,6.95],[79.85,6.95],[79.85,6.85]]]}`,
		FetchedAt: at(13, 0), FirstSeenAt: at(11, 0)}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"google-flood-forecasting", "google-flash-floods"} {
		id, err := queries.StartSourceRun(ctx, store.StartSourceRunParams{Source: name, StartedAt: at(13, 0)})
		if err != nil {
			t.Fatal(err)
		}
		finished := at(13, 0)
		if err := queries.FinishSourceRun(ctx, store.FinishSourceRunParams{ID: id, FinishedAt: &finished, Status: "ok"}); err != nil {
			t.Fatal(err)
		}
	}

	var mu sync.Mutex
	received := []*http.Request{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		received = append(received, r)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	public, private, _ := webpush.GenerateKeys()
	vapid, _ := webpush.NewVAPID(public, private, "mailto:alerts@example.org")
	client := &webpush.Client{VAPID: vapid, HTTP: server.Client()}
	key, _ := ecdh.P256().GenerateKey(rand.Reader)
	secret := make([]byte, 16)
	_, _ = rand.Read(secret)
	var subscription push.SubscriptionJSON
	subscription.Endpoint = server.URL + "/colombo"
	subscription.Keys.P256dh = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	subscription.Keys.Auth = base64.RawURLEncoding.EncodeToString(secret)
	service := &push.Service{Pool: pool, VAPID: vapid, Sender: client, AllowEndpoint: func(string) error { return nil }, Now: func() time.Time { return at(9, 0) }}
	if _, _, err := service.Subscribe(ctx, subscription, push.Preferences{Topics: []push.Topic{push.TopicHeavyRain, push.TopicFloodRisk},
		MinSeverity: push.SeverityWarning, Areas: push.Areas{Districts: []string{"LK-11"}}}, "en", ""); err != nil {
		t.Fatal(err)
	}

	engine := &alerts.Engine{Pool: pool,
		RainGauges: &raingauges.Service{Pool: pool, StaleAfter: 3 * time.Hour, Now: now},
		Floods:     &floods.Service{Pool: pool, Configured: true, StaleAfter: 30 * time.Hour, Now: now},
		Dispatcher: &push.Dispatcher{Pool: pool, Sender: client, Now: now}}
	sent, err := engine.Collect(ctx, clock)
	if err != nil {
		t.Fatal(err)
	}
	if sent != 2 {
		t.Fatalf("sent %d", sent)
	}
	events, err := queries.ListActivePushEvents(ctx, clock)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{}
	for _, event := range events {
		kinds[event.Kind] = event.Variant
	}
	if kinds["rain"] != "very_heavy" || kinds["flood_risk"] != "high" || kinds["flash_flood"] != "likely" {
		t.Fatalf("events %v", kinds)
	}
	// The next run, with nothing new, sends nothing.
	clock = clock.Add(30 * time.Second)
	if sent, err := engine.Collect(ctx, clock); err != nil || sent != 0 {
		t.Fatalf("second run sent %d, %v", sent, err)
	}
}
