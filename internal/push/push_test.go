package push

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/push/webpush"
)

var now = time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

func TestNormaliseCanonicalisesAndRounds(t *testing.T) {
	known := map[string]bool{"LK-91": true, "LK-92": true}
	got, err := Preferences{Topics: []Topic{TopicRiver, TopicOfficial, TopicRiver}, MinSeverity: SeverityWatch,
		Areas: Areas{Districts: []string{"lk-92", "LK-91", "LK-92"}, Near: &Near{Lat: 6.68279, Lon: 80.39951, RadiusKm: 10}}}.Normalise(known)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Topics, []Topic{TopicOfficial, TopicRiver}) || !slices.Equal(got.Areas.Districts, []string{"LK-91", "LK-92"}) {
		t.Fatalf("got %+v", got)
	}
	if got.Areas.Near.Lat != 6.68 || got.Areas.Near.Lon != 80.4 {
		t.Fatalf("point not rounded to about a kilometre: %+v", got.Areas.Near)
	}
}

func TestNormaliseRejects(t *testing.T) {
	known := map[string]bool{"LK-91": true}
	cases := map[string]Preferences{
		"no area":          {Topics: []Topic{TopicOfficial}},
		"unknown topic":    {Topics: []Topic{"gossip"}, Areas: Areas{Country: true}},
		"no topic":         {Areas: Areas{Country: true}},
		"unknown district": {Topics: []Topic{TopicOfficial}, Areas: Areas{Districts: []string{"LK-99"}}},
		"bad code":         {Topics: []Topic{TopicOfficial}, Areas: Areas{Districts: []string{"Colombo"}}},
		"far away":         {Topics: []Topic{TopicOfficial}, Areas: Areas{Near: &Near{Lat: 13, Lon: 80, RadiusKm: 10}}},
		"huge radius":      {Topics: []Topic{TopicOfficial}, Areas: Areas{Near: &Near{Lat: 7, Lon: 80, RadiusKm: 500}}},
		"bad severity":     {Topics: []Topic{TopicOfficial}, MinSeverity: "extreme", Areas: Areas{Country: true}},
	}
	for name, preferences := range cases {
		_, err := preferences.Normalise(known)
		var validation *ValidationError
		if !errors.As(err, &validation) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAllowedEndpoint(t *testing.T) {
	allowed := []string{
		"https://fcm.googleapis.com/fcm/send/abc:def",
		"https://updates.push.services.mozilla.com/wpush/v2/gAAAA",
		"https://web.push.apple.com/QH1x",
		"https://wns2-par02p.notify.windows.com/w/?token=AwYAAAB",
	}
	for _, endpoint := range allowed {
		if err := AllowedEndpoint(endpoint); err != nil {
			t.Errorf("%s: %v", endpoint, err)
		}
	}
	refused := []string{
		"http://fcm.googleapis.com/fcm/send/abc",
		"https://evil.example/fcm.googleapis.com",
		"https://fcm.googleapis.com.evil.example/x",
		"https://notify.windows.com/x",
		"https://user@fcm.googleapis.com/x",
		"https://fcm.googleapis.com:8443/x",
		"https://127.0.0.1/x",
		"https://fcm.googleapis.com/" + strings.Repeat("a", MaxEndpointLength),
	}
	for _, endpoint := range refused {
		if err := AllowedEndpoint(endpoint); err == nil {
			t.Errorf("%s accepted", endpoint)
		}
	}
}

func TestUserAgentFamily(t *testing.T) {
	cases := map[string]string{
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Mobile Safari/537.36":                       "chrome-android",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Mobile/15E148 Safari/604.1": "safari-ios",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36 Edg/130.0":                   "edge-windows",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:132.0) Gecko/20100101 Firefox/132.0":                                                    "firefox-macos",
		"": "other-other",
	}
	for ua, want := range cases {
		if got := UserAgentFamily(ua); got != want {
			t.Errorf("%q: %s, want %s", ua, got, want)
		}
	}
}

func lat(value float64) *float64 { return &value }

func subscriber(preferences Preferences) Subscriber {
	return Subscriber{ID: 1, Locale: "en", Preferences: preferences, CreatedAt: now.Add(-time.Hour)}
}

func TestWantsMatchesTopicSeverityAndArea(t *testing.T) {
	ratnapura := Subscriber{ID: 1, Locale: "en", CreatedAt: now.Add(-time.Hour), NearDistricts: []string{"LK-92"},
		Preferences: Preferences{Topics: []Topic{TopicLandslide, TopicRiver, TopicCyclone}, MinSeverity: SeverityWarning,
			Areas: Areas{Districts: []string{"LK-91"}, Near: &Near{Lat: 6.9, Lon: 79.86, RadiusKm: 10}}}}
	event := Event{Topic: TopicLandslide, Severity: SeverityDanger, Districts: []string{"LK-91"}, DetectedAt: now}
	if !ratnapura.Wants(event) {
		t.Error("chosen district")
	}
	near := event
	near.Districts = []string{"LK-92"}
	if !ratnapura.Wants(near) {
		t.Error("district within the near-me radius")
	}
	elsewhere := event
	elsewhere.Districts = []string{"LK-11"}
	if ratnapura.Wants(elsewhere) {
		t.Error("another district")
	}
	gauge := Event{Topic: TopicRiver, Severity: SeverityWarning, Districts: []string{"LK-11"}, Lat: lat(6.95), Lon: lat(79.88), DetectedAt: now}
	if !ratnapura.Wants(gauge) {
		t.Error("gauge within the radius")
	}
	weak := event
	weak.Severity = SeverityWatch
	if ratnapura.Wants(weak) {
		t.Error("below the minimum severity")
	}
	other := event
	other.Topic = TopicReservoir
	if ratnapura.Wants(other) {
		t.Error("topic not chosen")
	}
	national := Event{Topic: TopicCyclone, Severity: SeverityWarning, National: true, DetectedAt: now}
	if !ratnapura.Wants(national) {
		t.Error("national events reach every area")
	}
	old := event
	old.DetectedAt = now.Add(-2 * time.Hour)
	if ratnapura.Wants(old) {
		t.Error("events raised before subscribing are not sent")
	}
}

func event(kind, key string, severity Severity, detected time.Time) Event {
	return Event{Key: key, Kind: kind, Variant: "2", Topic: TopicLandslide, Severity: severity, Facts: map[string]string{"district": key},
		URL: "/?warnings=landslide&district=" + key, Tag: "landslide:district:" + key, DetectedAt: detected, ExpiresAt: detected.Add(12 * time.Hour)}
}

func TestPlanGathersOneNoticePerKind(t *testing.T) {
	events := []Event{
		event(messages.KindLandslide, "LK-91", SeverityWarning, now),
		event(messages.KindLandslide, "LK-92", SeverityWarning, now.Add(time.Minute)),
		event(messages.KindLandslide, "LK-21", SeverityWarning, now),
		{Key: "quake", Kind: messages.KindEarthquake, Variant: "plain", Severity: SeverityWatch, URL: "/?quake=x", Tag: "quake:x", DetectedAt: now},
	}
	send, held := Plan(events, 0, HourlyCap)
	if len(send) != 2 || len(held) != 0 {
		t.Fatalf("send %d held %d", len(send), len(held))
	}
	landslides := send[0]
	if landslides.Kind != messages.KindLandslide || landslides.Count != 3 || landslides.Tag != "landslide:summary" || landslides.URL != "/?warnings=landslide" {
		t.Fatalf("gathered notice %+v", landslides)
	}
	if landslides.Facts["district"] != "LK-92" {
		t.Fatalf("the newest of equal severity leads: %v", landslides.Facts)
	}
	title, _ := messages.Render("en", messages.Input{Kind: landslides.Kind, Variant: landslides.Variant, Count: landslides.Count, Facts: landslides.Facts}, nil)
	if title != "Landslide Level 2 · 3 districts" {
		t.Fatalf("title %q", title)
	}
	if send[1].Kind != messages.KindEarthquake || send[1].Count != 1 || send[1].Tag != "quake:x" {
		t.Fatalf("single notice keeps its own tag: %+v", send[1])
	}
}

func TestPlanCapsNoticesButNeverDanger(t *testing.T) {
	events := []Event{
		{Key: "a", Kind: messages.KindRiver, Variant: "minor_flood", Severity: SeverityWarning, DetectedAt: now},
		{Key: "b", Kind: messages.KindCyclone, Variant: "near", Severity: SeverityWarning, DetectedAt: now},
		{Key: "c", Kind: messages.KindLandslide, Variant: "3", Severity: SeverityDanger, DetectedAt: now},
	}
	send, held := Plan(events, HourlyCap-1, HourlyCap)
	if len(send) != 2 || send[0].Severity != SeverityDanger || send[1].Kind != messages.KindRiver {
		t.Fatalf("send %+v", send)
	}
	if len(held) != 1 || held[0].Key != "b" {
		t.Fatalf("held %+v", held)
	}
	send, held = Plan(events, HourlyCap+5, HourlyCap)
	if len(send) != 1 || send[0].Severity != SeverityDanger || len(held) != 2 {
		t.Fatalf("over the cap only danger goes: %+v / %+v", send, held)
	}
}

func TestMessageHeadersFollowSeverity(t *testing.T) {
	user := Subscriber{Locale: "si", Endpoint: "https://fcm.googleapis.com/fcm/send/x"}
	danger := Notice{Kind: messages.KindLandslide, Variant: "3", Severity: SeverityDanger, Count: 1,
		Facts: map[string]string{"district": "LK-91"}, URL: "/?warnings=landslide&district=LK-91", Tag: "landslide:district:LK-91",
		ExpiresAt: now.Add(12 * time.Hour)}
	names := DistrictNames{"LK-91": {"Ratnapura", "රත්නපුර", "இரத்தினபுரி"}}
	payload, message := danger.Message(user, names, now)
	if message.Urgency != webpush.UrgencyHigh || message.TTL != 6*time.Hour || message.Topic != webpush.TopicFor("landslide:district:LK-91") {
		t.Fatalf("danger message %+v", message)
	}
	if !payload.RequireInteraction || payload.Lang != "si" || payload.Severity != SeverityDanger || !strings.Contains(payload.Title, "රත්නපුර") {
		t.Fatalf("payload %+v", payload)
	}
	encoded, _ := json.Marshal(payload)
	var wire map[string]any
	_ = json.Unmarshal(encoded, &wire)
	for _, key := range []string{"title", "body", "url", "tag", "severity", "sentAt", "requireInteraction", "lang"} {
		if _, ok := wire[key]; !ok {
			t.Errorf("payload lacks %s: %s", key, encoded)
		}
	}
	watch := danger
	watch.Severity, watch.ExpiresAt = SeverityWatch, now.Add(30*time.Minute)
	payload, message = watch.Message(user, names, now)
	if message.Urgency != webpush.UrgencyNormal || message.TTL != 30*time.Minute || payload.RequireInteraction {
		t.Fatalf("watch message %+v %+v", message, payload)
	}
	encoded, _ = json.Marshal(payload)
	if strings.Contains(string(encoded), "requireInteraction") {
		t.Fatalf("requireInteraction is omitted when false: %s", encoded)
	}
}

func TestBackoffDoublesToAnHour(t *testing.T) {
	if backoff(1) != time.Minute || backoff(2) != 2*time.Minute || backoff(3) != 4*time.Minute || backoff(30) != time.Hour {
		t.Fatalf("backoff %v %v %v %v", backoff(1), backoff(2), backoff(3), backoff(30))
	}
}

func TestSameSecretIgnoresBase64Flavour(t *testing.T) {
	if !sameSecret("BTBZMqHH6r4Tts7J_aSIgg", "BTBZMqHH6r4Tts7J/aSIgg==") {
		t.Error("padding and alphabet")
	}
	if sameSecret("BTBZMqHH6r4Tts7J_aSIgg", "BTBZMqHH6r4Tts7J_aSIgh") {
		t.Error("different secret")
	}
}

func TestShortDistrictNameDropsTheCutWordDistrict(t *testing.T) {
	cases := map[string]string{
		"අනුරාධපුර දිස්ත්‍": "අනුරාධපුර",
		"කොළඹ දිස්ත්‍රික්ක": "කොළඹ",
		"ත්‍රිකුණාමළය දිස්": "ත්‍රිකුණාමළය",
		"அம்பாந்தோட்டை மாவ": "அம்பாந்தோட்டை",
		"கொழும்பு மாவட்டம்": "கொழும்பு",
		"Ratnapura": "Ratnapura",
	}
	for name, want := range cases {
		if got := ShortDistrictName(name); got != want {
			t.Errorf("%q: %q, want %q", name, got, want)
		}
	}
}
