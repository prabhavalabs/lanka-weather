package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/prabhavalabs/lanka-weather/internal/push"
)

// Push manages Web Push subscriptions (ADR 0018).
type Push interface {
	Enabled() bool
	PublicKey() string
	Subscribe(ctx context.Context, subscription push.SubscriptionJSON, preferences push.Preferences, locale, userAgent string) (push.View, bool, error)
	Update(ctx context.Context, endpoint, auth string, preferences push.Preferences, locale, userAgent string) (push.View, error)
	Delete(ctx context.Context, endpoint, auth string) error
	Renew(ctx context.Context, oldEndpoint *string, subscription push.SubscriptionJSON) (push.View, error)
	Test(ctx context.Context, endpoint, auth string) error
}

const (
	// maxPushBody caps a push request's body: a subscription with its
	// preferences is well under 2 kB.
	maxPushBody = 8 << 10
	// Requests per client: any push route, and test notices.
	pushRequestsPerMinute  = 30
	pushTestsPerTenMinutes = 5
)

// pushRoutes mounts /api/push. Every response is private and uncached; the
// client is known only to the rate limiter, in memory, and never logged.
func (api *api) pushRoutes(r chi.Router) {
	requests := newLimiter(pushRequestsPerMinute, time.Minute)
	tests := newLimiter(pushTestsPerTenMinutes, 10*time.Minute)
	r.Route("/api/push", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				if ok, wait := requests.allow(clientKey(r)); !ok {
					tooMany(w, wait)
					return
				}
				r.Body = http.MaxBytesReader(w, r.Body, maxPushBody)
				next.ServeHTTP(w, r)
			})
		})
		r.Get("/key", api.pushKey)
		r.Post("/subscriptions", api.pushSubscribe)
		r.Put("/subscriptions", api.pushUpdate)
		r.Delete("/subscriptions", api.pushDelete)
		r.Post("/subscriptions/renew", api.pushRenew)
		r.With(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if ok, wait := tests.allow(clientKey(r)); !ok {
					tooMany(w, wait)
					return
				}
				next.ServeHTTP(w, r)
			})
		}).Post("/test", api.pushTest)
	})
}

func tooMany(w http.ResponseWriter, wait time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(max(1, int(wait.Seconds()+0.5))))
	writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests; try again shortly")
}

// pushKeyResponse says whether alerts can be turned on, and with which key.
type pushKeyResponse struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"publicKey,omitempty"`
}

func (api *api) pushKey(w http.ResponseWriter, _ *http.Request) {
	response := pushKeyResponse{}
	if api.deps.Push != nil && api.deps.Push.Enabled() {
		response = pushKeyResponse{Enabled: true, PublicKey: api.deps.Push.PublicKey()}
	}
	writeJSON(w, http.StatusOK, Envelope{Data: response})
}

type subscribeRequest struct {
	Subscription push.SubscriptionJSON `json:"subscription"`
	Preferences  push.Preferences      `json:"preferences"`
	Locale       string                `json:"locale"`
}

type ownedRequest struct {
	Endpoint    string           `json:"endpoint"`
	Auth        string           `json:"auth"`
	Preferences push.Preferences `json:"preferences"`
	Locale      string           `json:"locale"`
}

type renewRequest struct {
	OldEndpoint  *string               `json:"oldEndpoint"`
	Subscription push.SubscriptionJSON `json:"subscription"`
}

// readBody decodes a JSON body, answering for it when it cannot.
func readBody(w http.ResponseWriter, r *http.Request, into any) bool {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(into); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "the request body is too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "the request body must be JSON")
		return false
	}
	return true
}

func (api *api) pushAvailable(w http.ResponseWriter) bool {
	if api.deps.Push == nil {
		writeError(w, http.StatusServiceUnavailable, "push_disabled", "push alerts are not configured on this server")
		return false
	}
	return true
}

// pushError answers for a push service error.
func (api *api) pushError(w http.ResponseWriter, err error) {
	var validation *push.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(w, http.StatusBadRequest, validation.Code, validation.Message)
	case errors.Is(err, push.ErrDisabled):
		writeError(w, http.StatusServiceUnavailable, "push_disabled", "push alerts are not configured on this server")
	case errors.Is(err, push.ErrNotFound):
		writeError(w, http.StatusNotFound, "subscription_not_found", "no subscription has this endpoint")
	case errors.Is(err, push.ErrNotOwner):
		writeError(w, http.StatusForbidden, "not_owner", "the auth secret does not match this subscription")
	case errors.Is(err, push.ErrTooSoon):
		tooMany(w, time.Minute)
	case errors.Is(err, push.ErrGone):
		writeError(w, http.StatusGone, "subscription_gone", "the push service no longer accepts this subscription; subscribe again")
	case errors.Is(err, push.ErrDeliveryFailed):
		api.deps.Logger.Warn("push test not delivered", "error", err)
		writeError(w, http.StatusBadGateway, "delivery_failed", "the push service did not accept the notice")
	default:
		api.internalError(w, "push", err)
	}
}

func (api *api) pushSubscribe(w http.ResponseWriter, r *http.Request) {
	if !api.pushAvailable(w) {
		return
	}
	var request subscribeRequest
	if !readBody(w, r, &request) {
		return
	}
	view, created, err := api.deps.Push.Subscribe(r.Context(), request.Subscription, request.Preferences, request.Locale, r.UserAgent())
	if err != nil {
		api.pushError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, Envelope{Data: view})
}

func (api *api) pushUpdate(w http.ResponseWriter, r *http.Request) {
	if !api.pushAvailable(w) {
		return
	}
	var request ownedRequest
	if !readBody(w, r, &request) {
		return
	}
	view, err := api.deps.Push.Update(r.Context(), request.Endpoint, request.Auth, request.Preferences, request.Locale, r.UserAgent())
	if err != nil {
		api.pushError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{Data: view})
}

func (api *api) pushDelete(w http.ResponseWriter, r *http.Request) {
	if !api.pushAvailable(w) {
		return
	}
	var request ownedRequest
	if !readBody(w, r, &request) {
		return
	}
	if err := api.deps.Push.Delete(r.Context(), request.Endpoint, request.Auth); err != nil {
		api.pushError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{Data: map[string]bool{"deleted": true}})
}

func (api *api) pushRenew(w http.ResponseWriter, r *http.Request) {
	if !api.pushAvailable(w) {
		return
	}
	var request renewRequest
	if !readBody(w, r, &request) {
		return
	}
	view, err := api.deps.Push.Renew(r.Context(), request.OldEndpoint, request.Subscription)
	if err != nil {
		api.pushError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{Data: view})
}

func (api *api) pushTest(w http.ResponseWriter, r *http.Request) {
	if !api.pushAvailable(w) {
		return
	}
	var request ownedRequest
	if !readBody(w, r, &request) {
		return
	}
	if err := api.deps.Push.Test(r.Context(), request.Endpoint, request.Auth); err != nil {
		api.pushError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, Envelope{Data: map[string]bool{"sent": true}})
}

// clientKey identifies a client for rate limiting: its address as RealIP
// leaves it, without the port.
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// limiter allows a number of requests per client in a fixed window. It is in
// memory, per process: enough to stop a loop or a script, not a distributed
// attack, which belongs at the proxy.
type limiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	clients   map[string]*windowCount
	lastSweep time.Time
	now       func() time.Time
}

type windowCount struct {
	start time.Time
	count int
}

func newLimiter(limit int, window time.Duration) *limiter {
	return &limiter{limit: limit, window: window, clients: map[string]*windowCount{}, now: time.Now}
}

// allow counts a request and reports whether it is within the limit, and if
// not how long until the window ends.
func (limiter *limiter) allow(key string) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	now := limiter.now()
	if now.Sub(limiter.lastSweep) > limiter.window {
		for client, entry := range limiter.clients {
			if now.Sub(entry.start) >= limiter.window {
				delete(limiter.clients, client)
			}
		}
		limiter.lastSweep = now
	}
	entry, ok := limiter.clients[key]
	if !ok || now.Sub(entry.start) >= limiter.window {
		limiter.clients[key] = &windowCount{start: now, count: 1}
		return true, 0
	}
	if entry.count >= limiter.limit {
		return false, entry.start.Add(limiter.window).Sub(now)
	}
	entry.count++
	return true, 0
}
