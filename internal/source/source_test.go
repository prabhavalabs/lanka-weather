package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

	if _, err := client.Fetch(context.Background(), Request{URL: server.URL + "/busy"}); err == nil || !strings.Contains(err.Error(), "429") {
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
