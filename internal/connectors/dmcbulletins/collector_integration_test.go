//go:build integration

package dmcbulletins_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/dmcbulletins"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

// replay serves recorded files by the end of the URL; anything else is a 404,
// as for a document whose text has not been generated yet.
type replay struct {
	t        *testing.T
	requests []string
}

var indexFiles = map[string]string{
	"lk_dmc_landslide_warnings/docs_last100.tsv":                   "index_landslide_warnings.tsv",
	"lk_dmc_river_water_level_and_flood_warnings/docs_last100.tsv": "index_river_water_level_and_flood_warnings.tsv",
	"lk_dmc_weather_forecasts/docs_last100.tsv":                    "index_weather_forecasts.tsv",
	"lk_dmc_situation_reports/docs_last100.tsv":                    "index_situation_reports.tsv",
}

func (r *replay) Fetch(_ context.Context, request source.Request) ([]byte, error) {
	r.requests = append(r.requests, request.URL)
	file := ""
	for suffix, name := range indexFiles {
		if strings.HasSuffix(request.URL, suffix) {
			file = name
		}
	}
	switch {
	case strings.HasSuffix(request.URL, "/summary.json"):
		file = "summary.json"
	case strings.HasSuffix(request.URL, "/doc.txt"):
		parts := strings.Split(request.URL, "/")
		file = parts[len(parts)-2] + ".txt"
	}
	payload, err := os.ReadFile("testdata/" + file)
	if err != nil || file == "" {
		return nil, &source.StatusError{Status: http.StatusNotFound}
	}
	return payload, nil
}

var tables = []string{"official_bulletins", "landslide_levels", "landslide_reports", "source_versions"}

func TestCollectStoresBulletinsAndServesThem(t *testing.T) {
	pool := dbtest.Open(t, tables...)
	ctx := context.Background()
	fetcher := &replay{t: t}
	collector := &dmcbulletins.Collector{Pool: pool, Fetcher: fetcher}
	// Late on 2 November: the 10:30 strong-winds warning is in force.
	now := time.Date(2026, 11, 2, 19, 0, 0, 0, time.UTC)

	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 11 {
		t.Fatalf("stored %d bulletins, want 11", stored)
	}

	service := &warnings.Service{Pool: pool, Now: func() time.Time { return now },
		LandslideStaleAfter: 12 * time.Hour, BulletinStaleAfter: 6 * time.Hour}
	overview, err := service.Overview(ctx, warnings.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if overview.Meta.Stale || overview.Meta.FetchedAt == nil || len(overview.Bulletins) == 0 {
		t.Fatalf("meta = %+v, %d bulletins", overview.Meta, len(overview.Bulletins))
	}
	for index := 1; index < len(overview.Bulletins); index++ {
		if overview.Bulletins[index].IssuedAt.After(overview.Bulletins[index-1].IssuedAt) {
			t.Fatal("bulletins are not newest first")
		}
	}
	byID := map[string]warnings.Bulletin{}
	for _, bulletin := range overview.Bulletins {
		byID[bulletin.ID] = bulletin
	}

	warning := byID["2026-11-02-10-30-warning-for-stron-22c40aa8"]
	if warning.Severity != "warning" || warning.Colour == nil || *warning.Colour != "Red" || !warning.Current || warning.Text == nil {
		t.Fatalf("warning = %+v", warning)
	}
	if !strings.Contains(*warning.Text, "The Severe Cyclone Murasu is very likely to move northwestwards") {
		t.Fatalf("text = %s", *warning.Text)
	}
	if !strings.HasPrefix(warning.PDFURL, "https://www.dmc.gov.lk/images/dmcreports/") || warning.Source != "dmc-reports" {
		t.Fatalf("warning = %+v", warning)
	}

	// Tables are not shown as text; the row keeps its title and link.
	water := byID["2026-11-02-18-00-water-level"]
	if water.Severity != "routine" || water.Text != nil || water.TextStatus != "unusable" {
		t.Fatalf("water level = %+v", water)
	}

	if len(overview.InForce) != 1 || overview.InForce[0].ID != warning.ID || overview.ActiveCount != 1 {
		t.Fatalf("in force = %+v, count %d", overview.InForce, overview.ActiveCount)
	}
	if len(overview.Banner) != 1 || overview.Banner[0].Title != warning.Title || overview.Banner[0].Key != "bulletin:"+warning.ID {
		t.Fatalf("banner = %+v", overview.Banner)
	}
	mentioned := false
	for _, district := range overview.Districts {
		if district.ID == "LK-42" && district.Name == "Mannar" && len(district.Mentioned) == 1 && district.Mentioned[0].ID == warning.ID {
			mentioned = true
		}
	}
	if !mentioned {
		t.Fatalf("districts = %+v", overview.Districts)
	}

	// Filters.
	flood, err := service.Overview(ctx, warnings.Filter{Type: "water_level"})
	if err != nil {
		t.Fatal(err)
	}
	for _, bulletin := range flood.Bulletins {
		if bulletin.Type != "water_level" {
			t.Fatalf("filtered list has %+v", bulletin)
		}
	}
	mannar, err := service.Overview(ctx, warnings.Filter{District: "LK-42"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mannar.Bulletins) == 0 || mannar.Bulletins[0].ID != warning.ID {
		t.Fatalf("district list = %+v", mannar.Bulletins)
	}

	// A second run does not read text again, and keeps what it read.
	fetcher.requests = nil
	if _, err := collector.Collect(ctx, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, url := range fetcher.requests {
		if strings.HasSuffix(url, "doc.txt") && strings.Contains(url, "warning-for-stron-22c40aa8") {
			t.Fatalf("read text again: %s", url)
		}
	}
	again, err := service.Overview(ctx, warnings.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, bulletin := range again.Bulletins {
		if bulletin.ID == warning.ID && (bulletin.Text == nil || bulletin.Severity != "warning") {
			t.Fatalf("second run lost text: %+v", bulletin)
		}
	}
}

func TestOverviewIsStaleWhenTheUpstreamHasStopped(t *testing.T) {
	pool := dbtest.Open(t, tables...)
	ctx := context.Background()
	// Fetches succeed, but the index was last rebuilt days before now.
	now := time.Date(2026, 11, 6, 19, 0, 0, 0, time.UTC)
	collector := &dmcbulletins.Collector{Pool: pool, Fetcher: &replay{t: t}}
	if _, err := collector.Collect(ctx, now); err != nil {
		t.Fatal(err)
	}
	service := &warnings.Service{Pool: pool, Now: func() time.Time { return now }, BulletinStaleAfter: 6 * time.Hour, LandslideStaleAfter: 12 * time.Hour}
	overview, err := service.Overview(ctx, warnings.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if !overview.Meta.Stale {
		t.Fatalf("meta = %+v", overview.Meta)
	}
}
