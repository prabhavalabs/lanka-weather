package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/overview"
)

type fakeOverview struct {
	document *overview.Overview
	err      error
}

func (fake *fakeOverview) Overview(context.Context) (*overview.Overview, error) {
	return fake.document, fake.err
}

func TestOverviewReturnsTheDocumentInTheEnvelope(t *testing.T) {
	document := overview.Build(overview.Inputs{}, time.Date(2026, 11, 2, 21, 0, 0, 0, time.UTC))
	server := newServer(t, Dependencies{Overview: &fakeOverview{document: document}})
	response, body := get(t, server, "/v1/overview")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, body)
	}
	if response.Header.Get("Cache-Control") != "public, max-age=60" {
		t.Fatalf("cache = %q", response.Header.Get("Cache-Control"))
	}
	var envelope struct {
		Data struct {
			Status      string   `json:"status"`
			Unavailable []string `json:"unavailable"`
			Sections    []struct {
				ID        string          `json:"id"`
				Available bool            `json:"available"`
				Counts    json.RawMessage `json:"counts"`
				Items     json.RawMessage `json:"items"`
			} `json:"sections"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data.Sections) != 11 || envelope.Data.Sections[0].ID != overview.SectionLandslides {
		t.Fatalf("sections = %+v", envelope.Data.Sections)
	}
	for _, section := range envelope.Data.Sections {
		if section.Available {
			t.Fatalf("nothing was read: %+v", section)
		}
		if string(section.Counts) == "null" || string(section.Items) == "null" {
			t.Fatalf("%s lists are null: %s", section.ID, body)
		}
	}
	if len(envelope.Data.Unavailable) != 11 || envelope.Data.Status == "" {
		t.Fatalf("data = %+v", envelope.Data)
	}
}

func TestOverviewErrors(t *testing.T) {
	server := newServer(t, Dependencies{})
	if response, _ := get(t, server, "/v1/overview"); response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured = %d", response.StatusCode)
	}
	server = newServer(t, Dependencies{Overview: &fakeOverview{err: errors.New("database down")}})
	response, body := get(t, server, "/v1/overview")
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("failed = %d: %s", response.StatusCode, body)
	}
}
