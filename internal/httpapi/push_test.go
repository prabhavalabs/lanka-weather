package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/push"
)

type fakePush struct {
	enabled     bool
	err         error
	locale      string
	userAgent   string
	preferences push.Preferences
	oldEndpoint *string
	endpoint    string
	auth        string
}

func (fake *fakePush) Enabled() bool     { return fake.enabled }
func (fake *fakePush) PublicKey() string { return "BPublicKey" }

func (fake *fakePush) Subscribe(_ context.Context, subscription push.SubscriptionJSON, preferences push.Preferences, locale, userAgent string) (push.View, bool, error) {
	fake.endpoint, fake.auth, fake.preferences, fake.locale, fake.userAgent = subscription.Endpoint, subscription.Keys.Auth, preferences, locale, userAgent
	return push.View{Locale: locale, Preferences: preferences}, true, fake.err
}

func (fake *fakePush) Update(_ context.Context, endpoint, auth string, preferences push.Preferences, locale, _ string) (push.View, error) {
	fake.endpoint, fake.auth, fake.preferences, fake.locale = endpoint, auth, preferences, locale
	return push.View{Locale: locale, Preferences: preferences}, fake.err
}

func (fake *fakePush) Delete(_ context.Context, endpoint, auth string) error {
	fake.endpoint, fake.auth = endpoint, auth
	return fake.err
}

func (fake *fakePush) Renew(_ context.Context, oldEndpoint *string, subscription push.SubscriptionJSON) (push.View, error) {
	fake.oldEndpoint, fake.endpoint = oldEndpoint, subscription.Endpoint
	return push.View{}, fake.err
}

func (fake *fakePush) Inbox(_ context.Context, endpoint, auth, locale string) ([]push.InboxNotice, error) {
	fake.endpoint, fake.auth, fake.locale = endpoint, auth, locale
	if fake.err != nil {
		return nil, fake.err
	}
	return []push.InboxNotice{{ID: "7", Kind: "river", Severity: push.SeverityWarning, Title: "Minor flood · Kalu Ganga at Ratnapura",
		Areas: []push.Area{{ID: "LK-91", Name: "Ratnapura"}}, Events: []push.InboxEvent{}}}, nil
}

func (fake *fakePush) Test(_ context.Context, endpoint, auth string) error {
	fake.endpoint, fake.auth = endpoint, auth
	return fake.err
}

func pushServer(t *testing.T, fake Push) *httptest.Server {
	t.Helper()
	deps := Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if fake != nil {
		deps.Push = fake
	}
	server := httptest.NewServer(NewRouter(deps))
	t.Cleanup(server.Close)
	return server
}

func send(t *testing.T, server *httptest.Server, method, path, body string) (*http.Response, Envelope) {
	t.Helper()
	request, _ := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14) Chrome/130.0 Mobile")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope Envelope
	_ = json.NewDecoder(response.Body).Decode(&envelope)
	return response, envelope
}

func TestPushKeySaysWhetherPushIsOn(t *testing.T) {
	for _, test := range []struct {
		fake *fakePush
		want string
	}{{nil, `{"enabled":false}`}, {&fakePush{enabled: false}, `{"enabled":false}`}, {&fakePush{enabled: true}, `{"enabled":true,"publicKey":"BPublicKey"}`}} {
		var deps Push
		if test.fake != nil {
			deps = test.fake
		}
		server := pushServer(t, deps)
		response, envelope := send(t, server, http.MethodGet, "/api/push/key", "")
		data, _ := json.Marshal(envelope.Data)
		if response.StatusCode != http.StatusOK || string(data) != test.want || response.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("%d %s %q", response.StatusCode, data, response.Header.Get("Cache-Control"))
		}
	}
}

const subscribeBody = `{"subscription":{"endpoint":"https://fcm.googleapis.com/fcm/send/x","expirationTime":null,
	"keys":{"p256dh":"BK","auth":"AU"}},"preferences":{"topics":["official","landslide"],"minSeverity":"warning",
	"areas":{"country":false,"districts":["LK-91"],"near":{"lat":6.7,"lon":80.4,"radiusKm":10}}},"locale":"si"}`

func TestPushSubscribeCreates(t *testing.T) {
	fake := &fakePush{enabled: true}
	response, envelope := send(t, pushServer(t, fake), http.MethodPost, "/api/push/subscriptions", subscribeBody)
	if response.StatusCode != http.StatusCreated || envelope.Error != nil {
		t.Fatalf("%d %+v", response.StatusCode, envelope.Error)
	}
	if fake.locale != "si" || fake.auth != "AU" || fake.preferences.Areas.Near.RadiusKm != 10 || fake.preferences.Areas.Districts[0] != "LK-91" ||
		!strings.Contains(fake.userAgent, "Android") {
		t.Fatalf("passed %+v", fake)
	}
}

func TestPushErrorsMapToStatuses(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{&push.ValidationError{Code: "invalid_areas", Message: "choose"}, http.StatusBadRequest, "invalid_areas"},
		{push.ErrDisabled, http.StatusServiceUnavailable, "push_disabled"},
		{push.ErrNotFound, http.StatusNotFound, "subscription_not_found"},
		{push.ErrNotOwner, http.StatusForbidden, "not_owner"},
		{push.ErrTooSoon, http.StatusTooManyRequests, "rate_limited"},
		{push.ErrGone, http.StatusGone, "subscription_gone"},
		{push.ErrDeliveryFailed, http.StatusBadGateway, "delivery_failed"},
	}
	for _, test := range cases {
		server := pushServer(t, &fakePush{enabled: true, err: test.err})
		response, envelope := send(t, server, http.MethodPut, "/api/push/subscriptions", `{"endpoint":"e","auth":"a","locale":"en"}`)
		if response.StatusCode != test.status || envelope.Error == nil || envelope.Error.Code != test.code {
			t.Errorf("%v: %d %+v", test.err, response.StatusCode, envelope.Error)
		}
	}
}

func TestPushRoutesReadTheirBodies(t *testing.T) {
	fake := &fakePush{enabled: true}
	server := pushServer(t, fake)
	if response, _ := send(t, server, http.MethodDelete, "/api/push/subscriptions", `{"endpoint":"https://fcm.googleapis.com/fcm/send/x","auth":"AU"}`); response.StatusCode != http.StatusOK ||
		fake.endpoint != "https://fcm.googleapis.com/fcm/send/x" || fake.auth != "AU" {
		t.Fatalf("delete %d %+v", response.StatusCode, fake)
	}
	if response, _ := send(t, server, http.MethodPost, "/api/push/subscriptions/renew",
		`{"oldEndpoint":"https://fcm.googleapis.com/fcm/send/old","subscription":{"endpoint":"https://fcm.googleapis.com/fcm/send/new","keys":{"p256dh":"x","auth":"y"}}}`); response.StatusCode != http.StatusOK ||
		fake.oldEndpoint == nil || *fake.oldEndpoint != "https://fcm.googleapis.com/fcm/send/old" || fake.endpoint != "https://fcm.googleapis.com/fcm/send/new" {
		t.Fatalf("renew %d %+v", response.StatusCode, fake)
	}
	if response, _ := send(t, server, http.MethodPost, "/api/push/subscriptions/renew", `{"oldEndpoint":null,"subscription":{"endpoint":"n","keys":{}}}`); response.StatusCode != http.StatusOK || fake.oldEndpoint != nil {
		t.Fatalf("renew without an old endpoint %d", response.StatusCode)
	}
	if response, _ := send(t, server, http.MethodPost, "/api/push/test", `{"endpoint":"e","auth":"a"}`); response.StatusCode != http.StatusAccepted {
		t.Fatalf("test %d", response.StatusCode)
	}
	if response, envelope := send(t, server, http.MethodPost, "/api/push/subscriptions", `not json`); response.StatusCode != http.StatusBadRequest || envelope.Error.Code != "invalid_json" {
		t.Fatalf("bad json %d", response.StatusCode)
	}
	big := `{"locale":"` + strings.Repeat("x", maxPushBody) + `"}`
	if response, envelope := send(t, server, http.MethodPost, "/api/push/subscriptions", big); response.StatusCode != http.StatusRequestEntityTooLarge || envelope.Error.Code != "body_too_large" {
		t.Fatalf("big body %d", response.StatusCode)
	}
}

func TestPushWithoutAServiceSaysDisabled(t *testing.T) {
	response, envelope := send(t, pushServer(t, nil), http.MethodPost, "/api/push/subscriptions", subscribeBody)
	if response.StatusCode != http.StatusServiceUnavailable || envelope.Error.Code != "push_disabled" {
		t.Fatalf("%d %+v", response.StatusCode, envelope.Error)
	}
}

func TestPushTestsAreRateLimitedPerClient(t *testing.T) {
	server := pushServer(t, &fakePush{enabled: true})
	for index := range pushTestsPerTenMinutes {
		if response, _ := send(t, server, http.MethodPost, "/api/push/test", `{"endpoint":"e","auth":"a"}`); response.StatusCode != http.StatusAccepted {
			t.Fatalf("test %d: %d", index, response.StatusCode)
		}
	}
	response, envelope := send(t, server, http.MethodPost, "/api/push/test", `{"endpoint":"e","auth":"a"}`)
	if response.StatusCode != http.StatusTooManyRequests || envelope.Error.Code != "rate_limited" || response.Header.Get("Retry-After") == "" {
		t.Fatalf("over the limit %d %+v", response.StatusCode, envelope.Error)
	}
}

func TestLimiterWindows(t *testing.T) {
	clock := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	limit := newLimiter(2, time.Minute)
	limit.now = func() time.Time { return clock }
	if ok, _ := limit.allow("a"); !ok {
		t.Fatal("first")
	}
	limit.allow("a")
	if ok, wait := limit.allow("a"); ok || wait != time.Minute {
		t.Fatalf("third: %v %v", ok, wait)
	}
	if ok, _ := limit.allow("b"); !ok {
		t.Fatal("clients are counted apart")
	}
	clock = clock.Add(time.Minute)
	if ok, _ := limit.allow("a"); !ok {
		t.Fatal("a new window")
	}
}

func TestPushInboxReadsTheCallersNotices(t *testing.T) {
	fake := &fakePush{enabled: true}
	server := pushServer(t, fake)
	response, envelope := send(t, server, http.MethodPost, "/api/push/inbox", `{"endpoint":"https://fcm.googleapis.com/fcm/send/x","auth":"AU","locale":"si"}`)
	data, _ := json.Marshal(envelope.Data)
	if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-store" ||
		!strings.Contains(string(data), `"id":"7"`) || !strings.Contains(string(data), `"name":"Ratnapura"`) {
		t.Fatalf("%d %s", response.StatusCode, data)
	}
	if fake.endpoint != "https://fcm.googleapis.com/fcm/send/x" || fake.auth != "AU" || fake.locale != "si" {
		t.Fatalf("passed %q %q %q", fake.endpoint, fake.auth, fake.locale)
	}
	fake.err = push.ErrNotOwner
	if response, _ := send(t, server, http.MethodPost, "/api/push/inbox", `{"endpoint":"e","auth":"wrong","locale":"en"}`); response.StatusCode != http.StatusForbidden {
		t.Fatalf("a wrong secret is refused: %d", response.StatusCode)
	}
}
