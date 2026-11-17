package push

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Inbox limits: what the app's list of notices can show, as long as events
// are kept (alerts.Retention).
const (
	InboxWindow     = 30 * 24 * time.Hour
	InboxMaxNotices = 100
	inboxMaxRows    = 600
)

// Area is a district named in the subscriber's language.
type Area struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InboxEvent is one event a notice gathered, written on its own.
type InboxEvent struct {
	Key        string     `json:"key"`
	Kind       string     `json:"kind"`
	Severity   Severity   `json:"severity"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	URL        string     `json:"url"`
	DetectedAt time.Time  `json:"detectedAt"`
	IssuedAt   *time.Time `json:"issuedAt,omitempty"`
	Areas      []Area     `json:"areas"`
}

// InboxNotice is a notice as it was sent, with what the app's detail view
// shows: who it comes from, whether that is official, every district and
// each event.
type InboxNotice struct {
	ID       string       `json:"id"`
	SentAt   time.Time    `json:"sentAt"`
	Kind     string       `json:"kind"`
	Severity Severity     `json:"severity"`
	Title    string       `json:"title"`
	Body     string       `json:"body"`
	URL      string       `json:"url"`
	Official bool         `json:"official"`
	Source   string       `json:"source"`
	National bool         `json:"national"`
	Areas    []Area       `json:"areas"`
	Events   []InboxEvent `json:"events"`
}

// Inbox lists the notices sent to a subscription in the last 30 days, newest
// first, written in a language, as the app's list of notices shows them. The
// auth secret proves the subscription is the caller's.
func (service *Service) Inbox(ctx context.Context, endpoint, auth, locale string) ([]InboxNotice, error) {
	if !ValidLocale(locale) {
		return nil, invalid("invalid_locale", "locale must be en, si or ta")
	}
	queries := store.New(service.Pool)
	row, err := service.owned(ctx, queries, endpoint, auth)
	if err != nil {
		return nil, err
	}
	now := service.now()
	rows, err := queries.PushInbox(ctx, store.PushInboxParams{SubscriptionID: row.ID, Since: now.Add(-InboxWindow), MaxRows: inboxMaxRows})
	if err != nil {
		return nil, fmt.Errorf("read inbox: %w", err)
	}
	names, err := service.districts.load(ctx, queries, now)
	if err != nil {
		return nil, err
	}
	return InboxFrom(rows, locale, names)
}

// InboxFrom groups delivery rows into notices and writes them.
func InboxFrom(rows []store.PushInboxRow, locale string, names DistrictNames) ([]InboxNotice, error) {
	type sent struct {
		id       int64
		at       time.Time
		severity Severity
		events   []Event
	}
	order := []int64{}
	byID := map[int64]*sent{}
	for _, row := range rows {
		facts := map[string]string{}
		if err := json.Unmarshal(row.Facts, &facts); err != nil {
			return nil, fmt.Errorf("read facts of %s: %w", row.DedupeKey, err)
		}
		event := Event{Key: row.DedupeKey, Kind: row.Kind, Variant: row.Variant, Topic: Topic(row.Topic), Severity: Severity(row.Severity),
			Subject: row.Subject, National: row.National, Districts: row.Districts, Lon: row.Lon, Lat: row.Lat, Facts: facts,
			URL: row.Url, Tag: row.Tag, DetectedAt: row.DetectedAt, ExpiresAt: row.ExpiresAt}
		entry, ok := byID[row.NoticeID]
		if !ok {
			if len(order) == InboxMaxNotices {
				continue
			}
			entry = &sent{id: row.NoticeID, at: row.SentAt, severity: Severity(row.NoticeSeverity)}
			byID[row.NoticeID] = entry
			order = append(order, row.NoticeID)
		}
		entry.events = append(entry.events, event)
	}
	areas := func(codes []string) []Area {
		out := make([]Area, 0, len(codes))
		for _, code := range codes {
			out = append(out, Area{ID: code, Name: names.District(code, locale)})
		}
		return out
	}
	notices := make([]InboxNotice, 0, len(order))
	for _, id := range order {
		entry := byID[id]
		notice := gather(entry.events[0].Kind, entry.events)
		title, body := notice.Render(locale, names, entry.at)
		severity := entry.severity
		if severity.Rank() == 0 {
			severity = notice.Severity
		}
		national := false
		events := make([]InboxEvent, 0, len(notice.Events))
		for _, event := range notice.Events {
			national = national || event.National
			single := Notice{Kind: event.Kind, Variant: event.Variant, Count: 1, Facts: event.Facts, Districts: event.Districts}
			eventTitle, eventBody := single.Render(locale, names, entry.at)
			item := InboxEvent{Key: event.Key, Kind: event.Kind, Severity: event.Severity, Title: eventTitle, Body: eventBody,
				URL: event.URL, DetectedAt: event.DetectedAt, Areas: areas(event.Districts)}
			for _, key := range []string{"issuedAt", "observedAt"} {
				if at, err := time.Parse(time.RFC3339, event.Facts[key]); err == nil {
					item.IssuedAt = &at
					break
				}
			}
			events = append(events, item)
		}
		notices = append(notices, InboxNotice{
			ID: fmt.Sprint(id), SentAt: entry.at, Kind: notice.Kind, Severity: severity, Title: title, Body: body, URL: notice.URL,
			Official: messages.Official(notice.Kind, notice.Facts), Source: messages.Issuer(locale, notice.Kind, notice.Facts),
			National: national && len(notice.Districts) == 0, Areas: areas(notice.Districts), Events: events,
		})
	}
	return notices, nil
}
