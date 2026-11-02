//go:build integration

package irrigation_test

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// replay serves the recorded page and remembers the queries it was asked.
type replay struct {
	payload []byte
	queries []url.Values
}

func (r *replay) GetJSON(_ context.Context, _ string, query url.Values) ([]byte, error) {
	r.queries = append(r.queries, query)
	return r.payload, nil
}

func TestCollectStoresAndServesRecordedPage(t *testing.T) {
	databaseURL := os.Getenv("LANKAWEATHER_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LANKAWEATHER_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE river_readings, river_stations, source_runs`); err != nil {
		t.Fatal(err)
	}

	payload, err := os.ReadFile("testdata/gauges_page.json")
	if err != nil {
		t.Fatal(err)
	}
	fetcher := &replay{payload: payload}
	collector := &irrigation.Collector{Pool: pool, Fetcher: fetcher}
	// The recorded readings were observed on 2026-11-02 around 11:30 UTC.
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 9 {
		t.Fatalf("stored %d readings, want 9", stored)
	}
	if got := fetcher.queries[0].Get("where"); got != "CreationDate >= TIMESTAMP '2026-10-26 12:00:00'" {
		t.Fatalf("first run should backfill a week, got where=%q", got)
	}

	// A second run starts from shortly before the newest stored reading and
	// re-stores the same readings without duplicating them.
	if _, err := collector.Collect(ctx, now.Add(15*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var newest time.Time
	if err := pool.QueryRow(ctx, `SELECT max(observed_at) FROM river_readings`).Scan(&newest); err != nil {
		t.Fatal(err)
	}
	want := "CreationDate >= TIMESTAMP '" + newest.Add(-3*time.Hour).UTC().Format("2006-01-02 15:04:05") + "'"
	if got := fetcher.queries[1].Get("where"); got != want {
		t.Fatalf("second run where = %q, want %q", got, want)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM river_readings`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 9 {
		t.Fatalf("readings after re-run = %d, want 9", count)
	}

	if _, err := store.New(pool).StartSourceRun(ctx, store.StartSourceRunParams{Source: irrigation.SourceName, StartedAt: now}); err != nil {
		t.Fatal(err)
	}
	service := &rivers.Service{Pool: pool, StaleAfter: 3 * time.Hour, Now: func() time.Time { return now }}
	layer, err := service.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(layer.Features) != 5 {
		t.Fatalf("features = %d, want 5", len(layer.Features))
	}
	byID := map[string]rivers.StationProperties{}
	for _, feature := range layer.Features {
		byID[feature.ID] = feature.Properties
	}
	if got := byID["ampara"].Status; got != rivers.StatusUnknown {
		t.Fatalf("ampara status = %s, want unknown", got)
	}
	colombo := byID["nagalagam-street"]
	if colombo.Status != rivers.StatusNormal || colombo.Unit != "ft" || colombo.Stale || colombo.WaterLevel == nil {
		t.Fatalf("nagalagam-street = %+v", colombo)
	}

	history, err := service.StationHistory(ctx, "nagalagam-street", 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Readings) != 2 || !history.Readings[0].ObservedAt.Before(history.Readings[1].ObservedAt) {
		t.Fatalf("history = %+v", history.Readings)
	}
	if _, err := service.StationHistory(ctx, "nowhere", time.Hour); err != rivers.ErrStationNotFound {
		t.Fatalf("unknown station error = %v", err)
	}
}
