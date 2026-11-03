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
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/push/webpush"
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
