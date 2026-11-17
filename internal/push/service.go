package push

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/push/webpush"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Errors the API answers with.
var (
	// ErrDisabled means no VAPID keys are configured.
	ErrDisabled = errors.New("push alerts are not configured")
	// ErrNotFound means no subscription has that endpoint.
	ErrNotFound = errors.New("subscription not found")
	// ErrNotOwner means the auth secret does not match the subscription's.
	ErrNotOwner = errors.New("the auth secret does not match")
	// ErrTooSoon means a test notice was sent too recently.
	ErrTooSoon = errors.New("a test notice was sent recently")
	// ErrGone means the push service no longer knows the subscription.
	ErrGone = errors.New("the push service no longer accepts this subscription")
	// ErrDeliveryFailed means the push service refused or could not be reached.
	ErrDeliveryFailed = errors.New("the push service did not accept the notice")
)

// Sender sends one push message; *webpush.Client is one.
type Sender interface {
	Send(ctx context.Context, message webpush.Message) (webpush.Result, error)
}

// Service manages subscriptions for the API and sends test notices.
type Service struct {
	Pool *pgxpool.Pool
	// VAPID is the server's identity; nil switches push off.
	VAPID  *webpush.VAPID
	Sender Sender
	// AllowEndpoint checks an endpoint; nil is AllowedEndpoint. Tests use it to
	// reach a local push endpoint.
	AllowEndpoint func(endpoint string) error
	// TestInterval is the least time between test notices to one subscription.
	TestInterval time.Duration
	Now          func() time.Time

	districts districtCache
}

// SubscriptionJSON is the browser's PushSubscription.toJSON().
type SubscriptionJSON struct {
	Endpoint       string   `json:"endpoint"`
	ExpirationTime *float64 `json:"expirationTime"`
	Keys           struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// View is what the API returns about a subscription: its choices and health,
// never its endpoint or keys.
type View struct {
	Locale        string      `json:"locale"`
	Preferences   Preferences `json:"preferences"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
	LastSuccessAt *time.Time  `json:"lastSuccessAt"`
}

func viewOf(row store.PushSubscription) View {
	subscriber := SubscriberFrom(row)
	return View{Locale: row.Locale, Preferences: subscriber.Preferences, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		LastSuccessAt: row.LastSuccessAt}
}

// Enabled reports whether push alerts can be sent.
func (service *Service) Enabled() bool { return service != nil && service.VAPID != nil }

// PublicKey is the applicationServerKey to subscribe with, or "".
func (service *Service) PublicKey() string {
	if !service.Enabled() {
		return ""
	}
	return service.VAPID.PublicKey
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}

func (service *Service) checkSubscription(subscription SubscriptionJSON) error {
	allow := service.AllowEndpoint
	if allow == nil {
		allow = AllowedEndpoint
	}
	if err := allow(subscription.Endpoint); err != nil {
		return err
	}
	if _, _, err := webpush.SubscriptionKeys(subscription.Keys.P256dh, subscription.Keys.Auth); err != nil {
		return invalid("invalid_keys", err.Error())
	}
	return nil
}

// sameSecret compares auth secrets in constant time, whatever their base64.
func sameSecret(stored, given string) bool {
	normalise := func(value string) string {
		value = strings.TrimRight(strings.TrimSpace(value), "=")
		return strings.NewReplacer("+", "-", "/", "_").Replace(value)
	}
	a, b := normalise(stored), normalise(given)
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// prepared are preferences checked and the districts near the point found.
type prepared struct {
	preferences   Preferences
	nearDistricts []string
	topics        []string
}

func (service *Service) prepare(ctx context.Context, queries *store.Queries, preferences Preferences) (prepared, error) {
	known, err := service.districts.known(ctx, queries, service.now())
	if err != nil {
		return prepared{}, err
	}
	normal, err := preferences.Normalise(known)
	if err != nil {
		return prepared{}, err
	}
	out := prepared{preferences: normal, nearDistricts: []string{}}
	for _, topic := range normal.Topics {
		out.topics = append(out.topics, string(topic))
	}
	if near := normal.Areas.Near; near != nil {
		districts, err := queries.DistrictsWithin(ctx, store.DistrictsWithinParams{Lon: near.Lon, Lat: near.Lat, Meters: float64(near.RadiusKm) * 1000})
		if err != nil {
			return prepared{}, fmt.Errorf("find districts near a point: %w", err)
		}
		out.nearDistricts = districts
	}
	return out, nil
}

func nearColumns(near *Near) (lon, lat *float64, radius *int32) {
	if near == nil {
		return nil, nil, nil
	}
	km := int32(near.RadiusKm)
	return &near.Lon, &near.Lat, &km
}

// Subscribe stores a subscription with its preferences, or updates the
// preferences of one already stored with the same endpoint and auth secret.
// It reports whether the subscription is new.
func (service *Service) Subscribe(ctx context.Context, subscription SubscriptionJSON, preferences Preferences, locale, userAgent string) (View, bool, error) {
	if !service.Enabled() {
		return View{}, false, ErrDisabled
	}
	if err := service.checkSubscription(subscription); err != nil {
		return View{}, false, err
	}
	if !ValidLocale(locale) {
		return View{}, false, invalid("invalid_locale", "locale must be en, si or ta")
	}
	queries := store.New(service.Pool)
	ready, err := service.prepare(ctx, queries, preferences)
	if err != nil {
		return View{}, false, err
	}
	existing, err := queries.GetPushSubscriptionByEndpoint(ctx, subscription.Endpoint)
	if err == nil {
		if !sameSecret(existing.Auth, subscription.Keys.Auth) {
			return View{}, false, ErrNotOwner
		}
		row, err := service.update(ctx, queries, existing.ID, ready, locale, userAgent)
		return viewOf(row), false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return View{}, false, fmt.Errorf("read subscription: %w", err)
	}
	lon, lat, radius := nearColumns(ready.preferences.Areas.Near)
	row, err := queries.InsertPushSubscription(ctx, store.InsertPushSubscriptionParams{
		Endpoint: subscription.Endpoint, P256dh: subscription.Keys.P256dh, Auth: subscription.Keys.Auth, Locale: locale,
		UserAgent: UserAgentFamily(userAgent), Topics: ready.topics, MinSeverity: string(ready.preferences.MinSeverity),
		WholeCountry: ready.preferences.Areas.Country, Districts: ready.preferences.Areas.Districts,
		NearLon: lon, NearLat: lat, NearRadiusKm: radius, NearDistricts: ready.nearDistricts, Now: service.now(),
	})
	if err != nil {
		return View{}, false, fmt.Errorf("store subscription: %w", err)
	}
	return viewOf(row), true, nil
}

func (service *Service) update(ctx context.Context, queries *store.Queries, id int64, ready prepared, locale, userAgent string) (store.PushSubscription, error) {
	lon, lat, radius := nearColumns(ready.preferences.Areas.Near)
	row, err := queries.UpdatePushPreferences(ctx, store.UpdatePushPreferencesParams{
		ID: id, Locale: locale, UserAgent: UserAgentFamily(userAgent), Topics: ready.topics,
		MinSeverity: string(ready.preferences.MinSeverity), WholeCountry: ready.preferences.Areas.Country,
		Districts: ready.preferences.Areas.Districts, NearLon: lon, NearLat: lat, NearRadiusKm: radius,
		NearDistricts: ready.nearDistricts, Now: service.now(),
	})
	if err != nil {
		return store.PushSubscription{}, fmt.Errorf("update subscription: %w", err)
	}
	return row, nil
}

// owned reads a subscription by endpoint, checking the auth secret.
func (service *Service) owned(ctx context.Context, queries *store.Queries, endpoint, auth string) (store.PushSubscription, error) {
	if endpoint == "" || len(endpoint) > MaxEndpointLength || auth == "" {
		return store.PushSubscription{}, invalid("invalid_subscription", "endpoint and auth are required")
	}
	row, err := queries.GetPushSubscriptionByEndpoint(ctx, endpoint)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.PushSubscription{}, ErrNotFound
	}
	if err != nil {
		return store.PushSubscription{}, fmt.Errorf("read subscription: %w", err)
	}
	if !sameSecret(row.Auth, auth) {
		return store.PushSubscription{}, ErrNotOwner
	}
	return row, nil
}

// Update changes a subscription's preferences and language.
func (service *Service) Update(ctx context.Context, endpoint, auth string, preferences Preferences, locale, userAgent string) (View, error) {
	if !service.Enabled() {
		return View{}, ErrDisabled
	}
	if !ValidLocale(locale) {
		return View{}, invalid("invalid_locale", "locale must be en, si or ta")
	}
	queries := store.New(service.Pool)
	row, err := service.owned(ctx, queries, endpoint, auth)
	if err != nil {
		return View{}, err
	}
	ready, err := service.prepare(ctx, queries, preferences)
	if err != nil {
		return View{}, err
	}
	updated, err := service.update(ctx, queries, row.ID, ready, locale, userAgent)
	return viewOf(updated), err
}

// Delete removes a subscription and everything sent to it. It works when push
// is switched off too, so people can always leave.
func (service *Service) Delete(ctx context.Context, endpoint, auth string) error {
	queries := store.New(service.Pool)
	row, err := service.owned(ctx, queries, endpoint, auth)
	if err != nil {
		return err
	}
	if _, err := queries.DeletePushSubscription(ctx, row.ID); err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	return nil
}

// Renew moves a subscription to the new endpoint and keys a push service
// issued (the service worker's pushsubscriptionchange), keeping its
// preferences. Knowing the old endpoint, a secret URL, is the proof of
// ownership the event allows. With no old endpoint, or one we do not know, a
// subscription already stored under the new endpoint has its keys refreshed;
// otherwise ErrNotFound, and the app subscribes afresh when next opened.
func (service *Service) Renew(ctx context.Context, oldEndpoint *string, subscription SubscriptionJSON) (View, error) {
	if !service.Enabled() {
		return View{}, ErrDisabled
	}
	if err := service.checkSubscription(subscription); err != nil {
		return View{}, err
	}
	queries := store.New(service.Pool)
	renew := func(id int64) (View, error) {
		row, err := queries.RenewPushSubscription(ctx, store.RenewPushSubscriptionParams{ID: id, Endpoint: subscription.Endpoint,
			P256dh: subscription.Keys.P256dh, Auth: subscription.Keys.Auth, Now: service.now()})
		if err != nil {
			return View{}, fmt.Errorf("renew subscription: %w", err)
		}
		return viewOf(row), nil
	}
	current, err := queries.GetPushSubscriptionByEndpoint(ctx, subscription.Endpoint)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return View{}, fmt.Errorf("read subscription: %w", err)
	}
	hasCurrent := err == nil
	if oldEndpoint != nil && *oldEndpoint != "" && *oldEndpoint != subscription.Endpoint && len(*oldEndpoint) <= MaxEndpointLength {
		old, err := queries.GetPushSubscriptionByEndpoint(ctx, *oldEndpoint)
		if err == nil {
			if hasCurrent {
				// Both exist: keep the one with the person's latest choices.
				if _, err := queries.DeletePushSubscription(ctx, current.ID); err != nil {
					return View{}, fmt.Errorf("delete duplicate subscription: %w", err)
				}
			}
			return renew(old.ID)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return View{}, fmt.Errorf("read old subscription: %w", err)
		}
	}
	if hasCurrent {
		return renew(current.ID)
	}
	return View{}, ErrNotFound
}

// Test sends a test notice to a subscription, at most once per TestInterval.
func (service *Service) Test(ctx context.Context, endpoint, auth string) error {
	if !service.Enabled() || service.Sender == nil {
		return ErrDisabled
	}
	queries := store.New(service.Pool)
	row, err := service.owned(ctx, queries, endpoint, auth)
	if err != nil {
		return err
	}
	now := service.now()
	interval := service.TestInterval
	if interval <= 0 {
		interval = time.Minute
	}
	claimed, err := queries.ClaimPushTest(ctx, store.ClaimPushTestParams{ID: row.ID, Now: now, NotBefore: now.Add(-interval)})
	if err != nil {
		return fmt.Errorf("claim test: %w", err)
	}
	if claimed == 0 {
		return ErrTooSoon
	}
	payload, message := TestNotice(now).Message(SubscriberFrom(row), nil, now, 0)
	if message.Payload, err = json.Marshal(payload); err != nil {
		return err
	}
	result, err := service.Sender.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeliveryFailed, err)
	}
	switch {
	case result.Accepted():
		return queries.RecordPushSuccess(ctx, store.RecordPushSuccessParams{ID: row.ID, At: now})
	case result.Gone:
		if _, err := queries.DeletePushSubscription(ctx, row.ID); err != nil {
			return fmt.Errorf("delete gone subscription: %w", err)
		}
		return ErrGone
	}
	return fmt.Errorf("%w: status %d", ErrDeliveryFailed, result.Status)
}
