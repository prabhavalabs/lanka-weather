package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadStreamsToFileAndSendsUserAgent(t *testing.T) {
	var agent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agent = r.Header.Get("User-Agent")
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte(strings.Repeat("x", 1024)))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "archive.zip")
	client := NewClient("test@example.org")
	header, written, err := client.Download(context.Background(), Request{URL: server.URL, MaxBytes: 2048, Timeout: time.Minute}, path)
	if err != nil {
		t.Fatal(err)
	}
	if written != 1024 || header.Get("ETag") != `"v1"` {
		t.Fatalf("written %d, header %v", written, header)
	}
	if agent != client.UserAgent() {
		t.Fatalf("agent %q", agent)
	}
	if info, err := os.Stat(path); err != nil || info.Size() != 1024 {
		t.Fatalf("file %v, %v", info, err)
	}
}

func TestDownloadFailsAboveTheLimitAndLeavesNoFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 4096)))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "archive.zip")
	_, _, err := NewClient("").Download(context.Background(), Request{URL: server.URL, MaxBytes: 1024, Timeout: time.Minute}, path)
	if err == nil {
		t.Fatal("want an error above the limit")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("the partial file should be removed, stat: %v", statErr)
	}
}

func TestDownloadRequiresLimits(t *testing.T) {
	if _, _, err := NewClient("").Download(context.Background(), Request{URL: "http://localhost"}, "x"); err == nil {
		t.Fatal("want an error without a limit")
	}
}

func TestDownloadReportsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	_, _, err := NewClient("").Download(context.Background(), Request{URL: server.URL, MaxBytes: 10, Timeout: time.Minute}, filepath.Join(t.TempDir(), "x"))
	if !IsStatus(err, http.StatusNotFound) {
		t.Fatalf("got %v", err)
	}
}
