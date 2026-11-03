//go:build integration

package push_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/push/webpush"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

var t0 = time.Date(2026, 11, 3, 4, 0, 0, 0, time.UTC)

// pushService records every request; a path's first segment picks the
// answer: ok, gone, busy (429) or flaky (500 twice, then 201).
type pushService struct {
	mu       sync.Mutex
	server   *httptest.Server
	requests map[string][]*http.Request
	flaky    int
}

func newPushService(t *testing.T) *pushService {
	t.Helper()
	fake := &pushService{requests: map[string][]*http.Request{}}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		fake.requests[r.URL.Path] = append(fake.requests[r.URL.Path], r)
		switch strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0] {
		case "gone":
			w.WriteHeader(http.StatusGone)
		case "busy":
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(http.StatusTooManyRequests)
		case "flaky":
			fake.flaky++
			if fake.flaky <= 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusCreated)
		}
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (fake *pushService) received(path string) []*http.Request {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.requests[path]
}

func subscription(t *testing.T, endpoint string) push.SubscriptionJSON {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := make([]byte, 16)
	_, _ = rand.Read(secret)
	var out push.SubscriptionJSON
	out.Endpoint = endpoint
	out.Keys.P256dh = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	out.Keys.Auth = base64.RawURLEncoding.EncodeToString(secret)
	return out
}

func insertDistrict(t *testing.T, pool *pgxpool.Pool, id, name, nameSi, nameTa string, west, south, east, north float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO regions (id, fid, level, name, name_si, name_ta, province_id, district_id, area_km2, centroid, label_point,
		                     west, south, east, north, geom, geom_simple, geom_3857, geom_simple_3857, source, source_version, source_time, fetched_at)
		SELECT $1, $2, 'district', $3, $4, $5, 'LK-1', $1, 700, ST_Centroid(g), ST_PointOnSurface(g), $6, $7, $8, $9,
		       g, g, ST_Transform(g, 3857), ST_Transform(g, 3857), 'test', 'v', now(), now()
		FROM (SELECT ST_Multi(ST_MakeEnvelope($6, $7, $8, $9, 4326)) AS g) s`,
		id, int64(len(id)*1000+int(id[3]-'0')*10+int(id[4]-'0')), name, nameSi, nameTa, west, south, east, north)
	if err != nil {
		t.Fatal(err)
	}
}

func open(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.Open(t, "push_subscriptions", "push_events", "push_alert_state", "regions")
	insertDistrict(t, pool, "LK-11", "Colombo", "කොළඹ", "கொழும்பு", 79.8, 6.8, 80.2, 7.0)
	insertDistrict(t, pool, "LK-91", "Ratnapura", "රත්නපුර", "இரத்தினபுரி", 80.2, 6.4, 80.8, 6.8)
	return pool
}

func allowAny(string) error { return nil }

func TestSubscriptionLifecycle(t *testing.T) {
	pool := open(t)
	fake := newPushService(t)
	public, private, _ := webpush.GenerateKeys()
	vapid, err := webpush.NewVAPID(public, private, "mailto:alerts@example.org")
	if err != nil {
		t.Fatal(err)
	}
	clock := t0
	service := &push.Service{Pool: pool, VAPID: vapid, Sender: &webpush.Client{VAPID: vapid, HTTP: fake.server.Client()},
		AllowEndpoint: allowAny, Now: func() time.Time { return clock }}
	ctx := context.Background()
	sub := subscription(t, fake.server.URL+"/ok/phone")
	preferences := push.Preferences{Topics: []push.Topic{push.TopicLandslide}, MinSeverity: push.SeverityWarning,
		Areas: push.Areas{Near: &push.Near{Lat: 6.93456, Lon: 79.85432, RadiusKm: 25}}}

	view, created, err := service.Subscribe(ctx, sub, preferences, "ta", "Mozilla/5.0 (Linux; Android 14) Chrome/130.0 Mobile")
	if err != nil || !created {
		t.Fatalf("subscribe: %v %v", created, err)
	}
	if view.Preferences.Areas.Near.Lat != 6.93 || view.Preferences.Areas.Near.Lon != 79.85 {
		t.Fatalf("the point is stored rounded: %+v", view.Preferences.Areas.Near)
	}
	row, err := store.New(pool).GetPushSubscriptionByEndpoint(ctx, sub.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if row.UserAgent != "chrome-android" || strings.Join(row.NearDistricts, ",") != "LK-11" {
		t.Fatalf("stored %q near %v", row.UserAgent, row.NearDistricts)
	}

	// Subscribing again updates; another secret is refused.
	if _, created, err := service.Subscribe(ctx, sub, preferences, "si", ""); err != nil || created {
		t.Fatalf("resubscribe: %v %v", created, err)
	}
	if _, err := service.Update(ctx, sub.Endpoint, "AAAAAAAAAAAAAAAAAAAAAA", preferences, "en", ""); !errors.Is(err, push.ErrNotOwner) {
		t.Fatalf("update with another secret: %v", err)
	}
	unknown := push.Preferences{Topics: []push.Topic{push.TopicRiver}, Areas: push.Areas{Districts: []string{"LK-52"}}}
	var validation *push.ValidationError
	if _, err := service.Update(ctx, sub.Endpoint, sub.Keys.Auth, unknown, "en", ""); !errors.As(err, &validation) {
		t.Fatalf("a district not in the regions table: %v", err)
	}

	// A test notice, then not another within the interval.
	if err := service.Test(ctx, sub.Endpoint, sub.Keys.Auth); err != nil {
		t.Fatal(err)
	}
	if got := fake.received("/ok/phone"); len(got) != 1 || got[0].Header.Get("Urgency") != "normal" || got[0].Header.Get("TTL") == "" ||
		got[0].Header.Get("Topic") != webpush.TopicFor("test") {
		t.Fatalf("test notice %v", got)
	}
	if err := service.Test(ctx, sub.Endpoint, sub.Keys.Auth); !errors.Is(err, push.ErrTooSoon) {
		t.Fatalf("second test: %v", err)
	}

	// The push service moves the subscription; preferences follow it.
	renewed := subscription(t, fake.server.URL+"/ok/phone-2")
	old := sub.Endpoint
	view, err = service.Renew(ctx, &old, renewed)
	if err != nil || view.Locale != "si" || view.Preferences.Areas.Near == nil {
		t.Fatalf("renew: %+v %v", view, err)
	}
	if _, err := service.Renew(ctx, nil, subscription(t, fake.server.URL+"/ok/stranger")); !errors.Is(err, push.ErrNotFound) {
		t.Fatalf("renewing an unknown subscription: %v", err)
	}
	if err := service.Delete(ctx, sub.Endpoint, sub.Keys.Auth); !errors.Is(err, push.ErrNotFound) {
		t.Fatalf("the old endpoint is gone: %v", err)
	}
	if err := service.Delete(ctx, renewed.Endpoint, renewed.Keys.Auth); err != nil {
		t.Fatal(err)
	}
	if count, _ := store.New(pool).CountPushSubscriptions(ctx); count != 0 {
		t.Fatalf("%d subscriptions left", count)
	}

	// Without keys, push is off, but leaving still works.
	off := &push.Service{Pool: pool, AllowEndpoint: allowAny}
	if _, _, err := off.Subscribe(ctx, sub, preferences, "en", ""); !errors.Is(err, push.ErrDisabled) {
		t.Fatalf("subscribe with push off: %v", err)
	}
}

func storeEvent(t *testing.T, pool *pgxpool.Pool, event push.Event) {
	t.Helper()
	params, err := push.InsertEventParams(event)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.New(pool).InsertPushEvent(context.Background(), params); err != nil {
		t.Fatal(err)
	}
}

func landslide(district string, level int, detected time.Time) push.Event {
	severity := map[int]push.Severity{2: push.SeverityWarning, 3: push.SeverityDanger}[level]
	variant := map[int]string{2: "2", 3: "3"}[level]
	return push.Event{Key: "landslide:" + district + ":" + variant, Kind: messages.KindLandslide, Variant: variant, Topic: push.TopicLandslide,
		Severity: severity, Subject: "landslide:" + district, Districts: []string{district}, Facts: map[string]string{"district": district},
		URL: "/?warnings=landslide&district=" + district, Tag: "landslide:district:" + district, DetectedAt: detected, ExpiresAt: detected.Add(12 * time.Hour)}
}

func TestDispatcherDeliversOnceAndCleansUp(t *testing.T) {
	pool := open(t)
	fake := newPushService(t)
	public, private, _ := webpush.GenerateKeys()
	vapid, _ := webpush.NewVAPID(public, private, "mailto:alerts@example.org")
	client := &webpush.Client{VAPID: vapid, HTTP: fake.server.Client()}
	clock := t0
	service := &push.Service{Pool: pool, VAPID: vapid, Sender: client, AllowEndpoint: allowAny, Now: func() time.Time { return clock }}
	ctx := context.Background()
	landslides := push.Preferences{Topics: []push.Topic{push.TopicLandslide}, MinSeverity: push.SeverityWarning}
	in := func(preferences push.Preferences, areas push.Areas) push.Preferences {
		preferences.Areas = areas
		return preferences
	}
	subscribe := func(path, locale string, preferences push.Preferences) push.SubscriptionJSON {
		sub := subscription(t, fake.server.URL+path)
		if _, _, err := service.Subscribe(ctx, sub, preferences, locale, ""); err != nil {
			t.Fatal(err)
		}
		return sub
	}
	subscribe("/ok/colombo", "si", in(landslides, push.Areas{Districts: []string{"LK-11"}}))
	subscribe("/gone/country", "en", in(landslides, push.Areas{Country: true}))
	subscribe("/ok/ratnapura", "en", in(landslides, push.Areas{Districts: []string{"LK-91"}}))
	busy := subscribe("/busy/colombo", "en", in(landslides, push.Areas{Districts: []string{"LK-11"}}))
	subscribe("/flaky/colombo", "ta", in(landslides, push.Areas{Districts: []string{"LK-11"}}))

	storeEvent(t, pool, landslide("LK-11", 3, t0.Add(time.Minute)))
	storeEvent(t, pool, landslide("LK-12", 2, t0.Add(time.Minute)))
	// Raised before anyone subscribed: never sent.
	storeEvent(t, pool, landslide("LK-13", 3, t0.Add(-time.Hour)))

	clock = t0.Add(2 * time.Minute)
	dispatcher := &push.Dispatcher{Pool: pool, Sender: client, Workers: 3, RetryDelay: time.Millisecond, Now: func() time.Time { return clock }}
	stats, err := dispatcher.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Notices != 4 || stats.Sent != 2 || stats.Gone != 1 || stats.Failed != 1 {
		t.Fatalf("stats %+v", stats)
	}
	colombo := fake.received("/ok/colombo")
	if len(colombo) != 1 {
		t.Fatalf("colombo received %d", len(colombo))
	}
	for header, want := range map[string]string{"TTL": "21600", "Urgency": "high", "Topic": webpush.TopicFor("landslide:district:LK-11"),
		"Content-Encoding": "aes128gcm"} {
		if got := colombo[0].Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if !strings.HasPrefix(colombo[0].Header.Get("Authorization"), "vapid t=") {
		t.Errorf("no VAPID authorization")
	}
	if got := fake.received("/ok/ratnapura"); len(got) != 0 {
		t.Fatalf("Ratnapura received a Colombo notice")
	}
	if got := fake.received("/flaky/colombo"); len(got) != 3 {
		t.Fatalf("a failing push service is retried twice: %d", len(got))
	}
	// The gone subscription is deleted with its deliveries.
	queries := store.New(pool)
	if _, err := queries.GetPushSubscriptionByEndpoint(ctx, fake.server.URL+"/gone/country"); err == nil {
		t.Fatal("a 410 subscription was kept")
	}
	// The busy one waits, and its notice was not marked sent.
	row, _ := queries.GetPushSubscriptionByEndpoint(ctx, busy.Endpoint)
	if row.RetryAfter == nil || row.RetryAfter.Before(clock.Add(2*time.Minute)) || row.ConsecutiveFailures != 1 {
		t.Fatalf("busy %+v", row)
	}

	// Nothing is sent twice; the busy subscription is still waiting.
	clock = clock.Add(time.Minute)
	stats, err = dispatcher.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Notices != 0 || stats.BackingOff != 1 || len(fake.received("/ok/colombo")) != 1 {
		t.Fatalf("second run %+v", stats)
	}
	// Once the wait is over, it gets the notice.
	clock = clock.Add(5 * time.Minute)
	if stats, err = dispatcher.Run(ctx); err != nil || stats.Notices != 1 || len(fake.received("/busy/colombo")) != 2 {
		t.Fatalf("after backoff %+v %v", stats, err)
	}
}

func TestDispatcherGathersAndCaps(t *testing.T) {
	pool := open(t)
	fake := newPushService(t)
	public, private, _ := webpush.GenerateKeys()
	vapid, _ := webpush.NewVAPID(public, private, "mailto:alerts@example.org")
	client := &webpush.Client{VAPID: vapid, HTTP: fake.server.Client()}
	clock := t0
	service := &push.Service{Pool: pool, VAPID: vapid, Sender: client, AllowEndpoint: allowAny, Now: func() time.Time { return clock }}
	ctx := context.Background()
	sub := subscription(t, fake.server.URL+"/ok/everything")
	if _, _, err := service.Subscribe(ctx, sub, push.Preferences{Topics: push.Topics, MinSeverity: push.SeverityWatch, Areas: push.Areas{Country: true}}, "en", ""); err != nil {
		t.Fatal(err)
	}
	for _, district := range []string{"LK-11", "LK-12", "LK-13"} {
		storeEvent(t, pool, landslide(district, 2, t0.Add(time.Minute)))
	}
	clock = t0.Add(2 * time.Minute)
	dispatcher := &push.Dispatcher{Pool: pool, Sender: client, HourlyCap: 1, Now: func() time.Time { return clock }}
	if stats, err := dispatcher.Run(ctx); err != nil || stats.Notices != 1 || stats.Sent != 1 {
		t.Fatalf("three landslide events make one notice: %+v %v", stats, err)
	}
	if got := fake.received("/ok/everything"); len(got) != 1 || got[0].Header.Get("Topic") != webpush.TopicFor("landslide:summary") {
		t.Fatalf("gathered %v", got)
	}
	// The cap is reached: a warning waits, a danger does not.
	storeEvent(t, pool, push.Event{Key: "river:a:2:1", Kind: messages.KindRiver, Variant: "minor_flood", Topic: push.TopicRiver, Severity: push.SeverityWarning,
		Subject: "river:a", National: true, Facts: map[string]string{"station": "A"}, URL: "/?station=a", Tag: "river:station:a",
		DetectedAt: clock, ExpiresAt: clock.Add(6 * time.Hour)})
	storeEvent(t, pool, landslide("LK-91", 3, clock))
	clock = clock.Add(time.Minute)
	stats, err := dispatcher.Run(ctx)
	if err != nil || stats.Sent != 1 || stats.Held != 1 {
		t.Fatalf("capped %+v %v", stats, err)
	}
	if got := fake.received("/ok/everything"); len(got) != 2 || got[1].Header.Get("Urgency") != "high" || got[1].Header.Get("Topic") != webpush.TopicFor("landslide:district:LK-91") {
		t.Fatalf("danger goes through the cap: %v", got)
	}
	// An hour later the held warning goes.
	clock = clock.Add(time.Hour)
	if stats, err := dispatcher.Run(ctx); err != nil || stats.Sent != 1 {
		t.Fatalf("after the hour %+v %v", stats, err)
	}
}
