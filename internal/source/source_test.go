package source

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoSendsUserAgentAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "lanka-weather/0.1 (+https://example.org)" || r.Header.Get("X-Test") != "yes" {
			http.Error(w, "missing headers", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	client := NewClient("https://example.org")
	response, err := client.Do(context.Background(), Request{URL: server.URL, Header: http.Header{"X-Test": {"yes"}}})
	if err != nil || response.Status != http.StatusOK {
		t.Fatalf("Do = %+v, %v", response, err)
	}
}

func TestDoBacksOffAfterTooManyRequests(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	client := NewClient("")
	client.now = func() time.Time { return now }

	_, err := client.Do(context.Background(), Request{URL: server.URL})
	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.Status != http.StatusTooManyRequests || statusErr.RetryAfter != 2*time.Minute {
		t.Fatalf("first error = %v", err)
	}
	_, err = client.Do(context.Background(), Request{URL: server.URL})
	var backoff *BackoffError
	if !errors.As(err, &backoff) || hits.Load() != 1 {
		t.Fatalf("second request should not reach the server: err %v, hits %d", err, hits.Load())
	}
	now = now.Add(2 * time.Minute)
	if _, err := client.Do(context.Background(), Request{URL: server.URL}); !IsStatus(err, http.StatusTooManyRequests) || hits.Load() != 2 {
		t.Fatalf("after the delay the server is asked again: err %v, hits %d", err, hits.Load())
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	if got := retryAfter("30", now); got != 30*time.Second {
		t.Fatalf("seconds = %v", got)
	}
	if got := retryAfter(now.Add(time.Minute).Format(http.TimeFormat), now); got != time.Minute {
		t.Fatalf("date = %v", got)
	}
	if got := retryAfter("soon", now); got != 0 {
		t.Fatalf("invalid = %v", got)
	}
}
