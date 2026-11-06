package push

import (
	"slices"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/push/webpush"
)

// HourlyCap is the most notices a subscription receives in an hour, except
// danger, which always goes. Events held back by the cap are gathered into
// the next notice while they are still relevant.
const HourlyCap = 4

// Notice is one push message to one subscriber, about one event or several of
// the same kind.
type Notice struct {
	Events   []Event
	Kind     string
	Variant  string
	Severity Severity
	Count    int
	Facts    map[string]string
	URL      string
	Tag      string
	// ExpiresAt is when the last of its events stops being relevant.
	ExpiresAt time.Time
}

// kindOrder breaks ties between notices of the same severity: official first.
var kindOrder = []string{messages.KindBulletin, messages.KindLandslide, messages.KindRiver, messages.KindCyclone,
	messages.KindEarthquake, messages.KindFloodForecast, messages.KindFlashFlood, messages.KindReservoir, messages.KindTest}

// summaryURLs open the list a gathered notice belongs to. Kinds not here use
// their leading event's link.
var summaryURLs = map[string]string{
	messages.KindBulletin:      "/?warnings=all",
	messages.KindLandslide:     "/?warnings=landslide",
	messages.KindRiver:         "/",
	messages.KindFloodForecast: "/",
	messages.KindFlashFlood:    "/",
	messages.KindReservoir:     "/?reservoirs=active",
}

// Plan gathers a subscriber's pending events into notices, one per kind, most
// severe first, and applies the hourly cap: sentLastHour notices other than
// danger have gone out in the last hour. It returns what to send now and what
// the cap held back.
func Plan(events []Event, sentLastHour, hourlyCap int) (send []Notice, held []Event) {
	groups := map[string][]Event{}
	for _, event := range events {
		groups[event.Kind] = append(groups[event.Kind], event)
	}
	notices := make([]Notice, 0, len(groups))
	for kind, group := range groups {
		notices = append(notices, gather(kind, group))
	}
	slices.SortFunc(notices, func(a, b Notice) int {
		if rank := b.Severity.Rank() - a.Severity.Rank(); rank != 0 {
			return rank
		}
		return indexOf(kindOrder, a.Kind) - indexOf(kindOrder, b.Kind)
	})
	remaining := hourlyCap - sentLastHour
	for _, notice := range notices {
		switch {
		case notice.Severity == SeverityDanger:
			send = append(send, notice)
		case remaining > 0:
			send = append(send, notice)
			remaining--
		default:
			held = append(held, notice.Events...)
		}
	}
	return send, held
}

func indexOf(list []string, value string) int {
	if index := slices.Index(list, value); index >= 0 {
		return index
	}
	return len(list)
}

// gather makes one notice of a kind's events: the most severe, then newest,
// leads and gives the notice its words.
func gather(kind string, events []Event) Notice {
	sorted := slices.Clone(events)
	slices.SortStableFunc(sorted, func(a, b Event) int {
		if rank := b.Severity.Rank() - a.Severity.Rank(); rank != 0 {
			return rank
		}
		if !a.DetectedAt.Equal(b.DetectedAt) {
			return b.DetectedAt.Compare(a.DetectedAt)
		}
		return strings.Compare(a.Key, b.Key)
	})
	lead := sorted[0]
	notice := Notice{Events: sorted, Kind: kind, Variant: lead.Variant, Severity: lead.Severity, Count: len(sorted),
		Facts: lead.Facts, URL: lead.URL, Tag: lead.Tag, ExpiresAt: lead.ExpiresAt}
	for _, event := range sorted {
		if event.ExpiresAt.After(notice.ExpiresAt) {
			notice.ExpiresAt = event.ExpiresAt
		}
	}
	if len(sorted) > 1 {
		notice.Tag = kind + ":summary"
		if url, ok := summaryURLs[kind]; ok {
			notice.URL = url
		}
	}
	return notice
}

// ttlFor is how long a push service may hold a notice for an offline device:
// hours, never days, and never past the events' relevance.
func ttlFor(severity Severity, expires, now time.Time) time.Duration {
	ttl := map[Severity]time.Duration{SeverityDanger: 6 * time.Hour, SeverityWarning: 4 * time.Hour,
		SeverityWatch: 2 * time.Hour, SeverityInfo: time.Hour}[severity]
	if left := expires.Sub(now); !expires.IsZero() && left < ttl {
		ttl = left
	}
	return max(ttl, 5*time.Minute)
}

// urgencyFor is RFC 8030's Urgency: high wakes a device in power saving.
func urgencyFor(severity Severity) string {
	if severity == SeverityWarning || severity == SeverityDanger {
		return webpush.UrgencyHigh
	}
	return webpush.UrgencyNormal
}

// Message builds a notice's payload in a subscriber's language and the
// headers it is sent with.
func (notice Notice) Message(subscriber Subscriber, names messages.Names, now time.Time) (Payload, webpush.Message) {
	title, body := messages.Render(subscriber.Locale, messages.Input{Kind: notice.Kind, Variant: notice.Variant,
		Count: notice.Count, Facts: notice.Facts, Now: now}, names)
	payload := Payload{Title: title, Body: body, URL: notice.URL, Tag: notice.Tag, Severity: notice.Severity,
		SentAt: now.UTC(), RequireInteraction: notice.Severity == SeverityDanger, Lang: subscriber.Locale}
	return payload, webpush.Message{Endpoint: subscriber.Endpoint, P256dh: subscriber.P256dh, Auth: subscriber.Auth,
		TTL: ttlFor(notice.Severity, notice.ExpiresAt, now), Urgency: urgencyFor(notice.Severity), Topic: webpush.TopicFor(notice.Tag)}
}

// TestNotice is the notice sent from the settings, to check delivery.
func TestNotice(now time.Time) Notice {
	return Notice{Kind: messages.KindTest, Variant: "test", Severity: SeverityInfo, Count: 1, Facts: map[string]string{},
		URL: "/", Tag: "test", ExpiresAt: now.Add(10 * time.Minute)}
}

// DistrictNames are districts' names in English, Sinhala and Tamil.
type DistrictNames map[string][3]string

// District names a district in a language, falling back to English.
func (names DistrictNames) District(id, locale string) string {
	entry, ok := names[id]
	if !ok {
		return id
	}
	index := map[string]int{"si": 1, "ta": 2}[locale]
	if entry[index] != "" {
		return entry[index]
	}
	return entry[0]
}
