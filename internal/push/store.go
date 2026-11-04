package push

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// SubscriberFrom reads a stored subscription.
func SubscriberFrom(row store.PushSubscription) Subscriber {
	topics := make([]Topic, 0, len(row.Topics))
	for _, topic := range row.Topics {
		topics = append(topics, Topic(topic))
	}
	preferences := Preferences{Topics: topics, MinSeverity: Severity(row.MinSeverity),
		Areas: Areas{Country: row.WholeCountry, Districts: append([]string{}, row.Districts...)}}
	if row.NearLat != nil && row.NearLon != nil && row.NearRadiusKm != nil {
		preferences.Areas.Near = &Near{Lat: *row.NearLat, Lon: *row.NearLon, RadiusKm: int(*row.NearRadiusKm)}
	}
	return Subscriber{ID: row.ID, Endpoint: row.Endpoint, P256dh: row.P256dh, Auth: row.Auth, Locale: row.Locale,
		Preferences: preferences, NearDistricts: row.NearDistricts, CreatedAt: row.CreatedAt, RetryAfter: row.RetryAfter}
}

// EventFrom reads a stored event.
func EventFrom(row store.PushEvent) (Event, error) {
	facts := map[string]string{}
	if err := json.Unmarshal(row.Facts, &facts); err != nil {
		return Event{}, fmt.Errorf("read facts of %s: %w", row.DedupeKey, err)
	}
	return Event{Key: row.DedupeKey, Kind: row.Kind, Variant: row.Variant, Topic: Topic(row.Topic), Severity: Severity(row.Severity),
		Subject: row.Subject, National: row.National, Districts: row.Districts, Lon: row.Lon, Lat: row.Lat, Facts: facts,
		URL: row.Url, Tag: row.Tag, DetectedAt: row.DetectedAt, ExpiresAt: row.ExpiresAt}, nil
}

// InsertEventParams writes an event for InsertPushEvent.
func InsertEventParams(event Event) (store.InsertPushEventParams, error) {
	facts, err := json.Marshal(event.Facts)
	if err != nil {
		return store.InsertPushEventParams{}, err
	}
	districts := event.Districts
	if districts == nil {
		districts = []string{}
	}
	return store.InsertPushEventParams{DedupeKey: event.Key, Kind: event.Kind, Variant: event.Variant, Topic: string(event.Topic),
		Severity: string(event.Severity), Subject: event.Subject, National: event.National, Districts: districts,
		Lon: event.Lon, Lat: event.Lat, Facts: facts, Url: event.URL, Tag: event.Tag,
		DetectedAt: event.DetectedAt, ExpiresAt: event.ExpiresAt}, nil
}

// districtCache keeps the districts' codes and names, which change weekly at
// most.
type districtCache struct {
	mu      sync.Mutex
	names   DistrictNames
	readAt  time.Time
	refresh time.Duration
}

func (cache *districtCache) load(ctx context.Context, queries *store.Queries, now time.Time) (DistrictNames, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	refresh := cache.refresh
	if refresh == 0 {
		refresh = time.Hour
	}
	if cache.names != nil && now.Sub(cache.readAt) < refresh {
		return cache.names, nil
	}
	rows, err := queries.ListDistrictNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("read district names: %w", err)
	}
	names := make(DistrictNames, len(rows))
	for _, row := range rows {
		entry := [3]string{row.Name}
		if row.NameSi != nil {
			entry[1] = ShortDistrictName(*row.NameSi)
		}
		if row.NameTa != nil {
			entry[2] = ShortDistrictName(*row.NameTa)
		}
		names[row.ID] = entry
	}
	cache.names, cache.readAt = names, now
	return names, nil
}

// districtWord matches the word "district" and what follows it in Sinhala
// (දිස්ත්‍රික්කය) and Tamil (மாவட்டம்). The regions source cuts these names at
// a fixed length, so the word is often cut short: "අනුරාධපුර දිස්ත්‍".
var districtWord = regexp.MustCompile(`\s+(දිස්|மாவ)\S*$`)

// ShortDistrictName drops the trailing word "district" from a Sinhala or Tamil
// district name: a notice's title says "Landslide Level 3 · රත්නපුර".
func ShortDistrictName(name string) string {
	if short := strings.TrimSpace(districtWord.ReplaceAllString(name, "")); short != "" {
		return short
	}
	return name
}

// known is the set of district codes, or nil before regions are loaded.
func (cache *districtCache) known(ctx context.Context, queries *store.Queries, now time.Time) (map[string]bool, error) {
	names, err := cache.load(ctx, queries, now)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		// Regions not loaded yet: codes are checked by form only.
		return nil, nil
	}
	known := make(map[string]bool, len(names))
	for id := range names {
		known[id] = true
	}
	return known, nil
}
