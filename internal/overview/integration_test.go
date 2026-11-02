//go:build integration

package overview_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/prabhavalabs/lanka-weather/internal/environment"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/overview"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
	"github.com/prabhavalabs/lanka-weather/internal/store"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

var tables = []string{"official_bulletins", "landslide_reports", "source_versions", "cyclones", "earthquakes",
	"reservoirs", "river_stations", "rivers", "weather_runs"}

func service(t *testing.T, now time.Time) *overview.Service {
	t.Helper()
	pool := dbtest.Open(t, tables...)
	clock := func() time.Time { return now }
	riverService := &rivers.Service{Pool: pool, StaleAfter: 3 * time.Hour, Now: clock}
	reservoirService := &reservoirs.Service{Pool: pool, StaleAfter: 48 * time.Hour, Now: clock}
	return &overview.Service{
		Warnings:    &warnings.Service{Pool: pool, LandslideStaleAfter: 12 * time.Hour, BulletinStaleAfter: 6 * time.Hour, Now: clock},
		Risk:        &riverrisk.Service{Pool: pool, Rivers: riverService, Reservoirs: reservoirService, Now: clock},
		Gauges:      riverService,
		Reservoirs:  reservoirService,
		Hazards:     &hazards.Service{Pool: pool, EarthquakesStaleAfter: time.Hour, CyclonesStaleAfter: 3 * time.Hour, Now: clock},
		Rain:        &stations.Service{Pool: pool, StaleAfter: 6 * time.Hour, Now: clock},
		Environment: &environment.Service{Pool: pool, AirStaleAfter: 6 * time.Hour, SeaStaleAfter: 6 * time.Hour, Now: clock},
		Now:         clock,
	}
}

func section(t *testing.T, got *overview.Overview, id string) overview.Section {
	t.Helper()
	for _, candidate := range got.Sections {
		if candidate.ID == id {
			return candidate
		}
	}
	t.Fatalf("no section %s", id)
	return overview.Section{}
}

func TestAnEmptyDatabaseIsNeverCalm(t *testing.T) {
	now := time.Date(2026, 11, 2, 21, 0, 0, 0, time.UTC)
	got, err := service(t, now).Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Status, "No river gauge") {
		t.Fatalf("no gauges have been read: %q", got.Status)
	}
	if !strings.Contains(got.Status, "Out of date") || !strings.Contains(got.Status, "river gauges") {
		t.Fatalf("status = %q", got.Status)
	}
	if cyclones := section(t, got, overview.SectionCyclones); !cyclones.Stale {
		t.Fatalf("cyclones = %+v", cyclones)
	}
}

func TestStoredRowsReachTheOverview(t *testing.T) {
	now := time.Date(2026, 11, 2, 21, 0, 0, 0, time.UTC)
	composed := service(t, now)
	ctx := context.Background()
	pool := dbtest.Open(t)
	queries := store.New(pool)

	// A reservoir spilling on a current reading, and one on an old reading.
	day := pgtype.Date{Time: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), Valid: true}
	old := pgtype.Date{Time: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	spill, kind := 1551.0, "spill"
	for _, row := range []struct {
		id, name string
		reading  pgtype.Date
	}{{"kimbulwana-oya", "Kimbulwana Oya", day}, {"old-tank", "Old Tank", old}} {
		if err := queries.UpsertReservoir(ctx, store.UpsertReservoirParams{ID: row.id, Name: row.name, Kind: "major", Source: "test", UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if err := queries.UpsertReservoirReading(ctx, store.UpsertReservoirReadingParams{ReservoirID: row.id, SnapshotDay: row.reading, ReadingDay: row.reading,
			Status: "spilling", FlowKind: &kind, FlowCusecs: &spill, FlowDay: row.reading, Source: "test", SourceTime: now, FetchedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	// A warning in force.
	until := now.Add(6 * time.Hour)
	if _, err := pool.Exec(ctx, `INSERT INTO official_bulletins (doc_id, bulletin_type, title, severity, issued_at, valid_until, pdf_url, source_page_url, lang, text_status, source, source_time, fetched_at)
		VALUES ('w', 'weather', 'Warning for strong winds and rough seas', 'warning', $1, $2, 'https://dmc.gov.lk/w.pdf', 'https://dmc.gov.lk/', 'en', 'unusable', 'test', $1, $1)`, now.Add(-time.Hour), until); err != nil {
		t.Fatal(err)
	}

	got, err := composed.Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Status, "1 reservoir spilling") {
		t.Fatalf("a stale reservoir must not count: %q", got.Status)
	}
	if !strings.Contains(got.Status, "1 official warning or advisory in force") {
		t.Fatalf("status = %q", got.Status)
	}
	reservoirSection := section(t, got, overview.SectionReservoirs)
	if len(reservoirSection.Items) != 1 || reservoirSection.Items[0].Detail != "1,551 cusecs" || reservoirSection.Items[0].Target.ID != "kimbulwana-oya" {
		t.Fatalf("reservoirs = %+v", reservoirSection.Items)
	}
	official := section(t, got, overview.SectionWarnings)
	if !official.Official || official.Items[0].Title != "Warning for strong winds and rough seas" || official.Items[0].Href == nil {
		t.Fatalf("warnings = %+v", official)
	}
}
