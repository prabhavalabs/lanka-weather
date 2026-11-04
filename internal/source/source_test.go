package source

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchIdentifiesAndPostsForms(t *testing.T) {
	var got *http.Request
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		_ = r.ParseForm()
		form = r.PostForm
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := NewClient("https://example.org/contact")
	body, err := client.Fetch(context.Background(), Request{
		Method: http.MethodPost, URL: server.URL, Query: url.Values{"a": {"1"}},
		Form: url.Values{"data": {"node(1);out;"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok": true}` {
		t.Fatalf("body = %s", body)
	}
	if ua := got.Header.Get("User-Agent"); ua != "lanka-weather/0.1 (+https://example.org/contact)" {
		t.Fatalf("User-Agent = %q", ua)
	}
	if got.Method != http.MethodPost || got.URL.Query().Get("a") != "1" || form.Get("data") != "node(1);out;" {
		t.Fatalf("request = %s %s, form %v", got.Method, got.URL, form)
	}
	if got.Header.Get("Accept") != "application/json" {
		t.Fatalf("Accept = %q", got.Header.Get("Accept"))
	}
}

func TestFetchRejectsErrorsAndLargeBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/busy":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/slow":
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
		default:
			_, _ = w.Write([]byte(strings.Repeat("x", 100)))
		}
	}))
	defer server.Close()
	client := NewClient("")

	// A 429 also makes the client back off from the host, so it gets its own
	// client; TestDoBacksOffAfterTooManyRequests covers the back-off.
	if _, err := NewClient("").Fetch(context.Background(), Request{URL: server.URL + "/busy"}); err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("busy error = %v", err)
	}
	if _, err := client.Fetch(context.Background(), Request{URL: server.URL + "/big", MaxBytes: 50}); err == nil {
		t.Fatal("expected a size error")
	}
	if body, err := client.GetJSON(context.Background(), server.URL+"/big", nil); err != nil || len(body) != 100 {
		t.Fatalf("GetJSON = %d bytes, %v", len(body), err)
	}
	if _, err := client.Fetch(context.Background(), Request{URL: server.URL + "/slow", Timeout: 50 * time.Millisecond}); err == nil {
		t.Fatal("expected a timeout")
	}
}

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
