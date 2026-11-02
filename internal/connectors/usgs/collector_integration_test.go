//go:build integration

package usgs_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/usgs"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
)

// replay answers the wide query with payload and the near one with nothing.
type replay struct {
	payload []byte
	queries []string
}

const emptyAnswer = `{"type":"FeatureCollection","metadata":{"status":200},"features":[]}`

func (r *replay) Fetch(_ context.Context, request source.Request) ([]byte, error) {
	r.queries = append(r.queries, request.Query.Encode())
	if request.Query.Get("minmagnitude") == "4.5" {
		return r.payload, nil
	}
	return []byte(emptyAnswer), nil
}

func TestCollectFollowsRevisionsAndDeletions(t *testing.T) {
	pool := dbtest.Open(t, "earthquakes", "source_versions")
	ctx := context.Background()
	payload, err := os.ReadFile("testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	fetcher := &replay{payload: payload}
	collector := &usgs.Collector{Pool: pool, Fetcher: fetcher}
	now := time.Date(2026, 11, 2, 19, 0, 0, 0, time.UTC)

	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 4 || len(fetcher.queries) != 2 {
		t.Fatalf("stored %d after %d queries", stored, len(fetcher.queries))
	}
	if !strings.Contains(fetcher.queries[0], "includedeleted=true") || !strings.Contains(fetcher.queries[0], "starttime=2026-10-03T19%3A00%3A00") {
		t.Fatalf("query = %s", fetcher.queries[0])
	}

	service := &hazards.Service{Pool: pool, Now: func() time.Time { return now }, EarthquakesStaleAfter: 2 * time.Hour}
	layer, err := service.Earthquakes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(layer.Features) != 4 || layer.Meta.Count != 4 || layer.Meta.Stale || layer.Meta.WindowDays != 30 || layer.Meta.FetchedAt == nil {
		t.Fatalf("meta = %+v, %d features", layer.Meta, len(layer.Features))
	}
	first := layer.Features[0]
	if first.Properties.DistanceKm < 500 || first.Properties.DepthKm != 10 || first.Properties.URL == "" || first.Properties.Source != usgs.SourceName {
		t.Fatalf("first = %+v", first.Properties)
	}

	// USGS revises one magnitude, deletes one event and merges another away.
	revised := strings.Replace(string(payload), `"mag": 5.1`, `"mag": 5.4`, 1)
	revised = strings.Replace(revised, `"status": "reviewed"`, `"status": "deleted"`, 2)
	fetcher.payload = []byte(revised)
	if _, err := collector.Collect(ctx, now.Add(15*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var magnitude float64
	var remaining int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM earthquakes), coalesce((SELECT magnitude FROM earthquakes ORDER BY occurred_at DESC, id LIMIT 1), 0)`).Scan(&remaining, &magnitude); err != nil {
		t.Fatal(err)
	}
	if remaining != 2 {
		t.Fatalf("%d events remain after deletions", remaining)
	}

	// An event no longer listed at all leaves the table too.
	fetcher.payload = []byte(strings.Replace(string(payload), `"fixture-tsunami"`, `"another-id"`, 1))
	if _, err := collector.Collect(ctx, now.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var gone bool
	if err := pool.QueryRow(ctx, `SELECT NOT EXISTS (SELECT 1 FROM earthquakes WHERE id = 'fixture-tsunami')`).Scan(&gone); err != nil {
		t.Fatal(err)
	}
	if !gone {
		t.Fatal("an event USGS stopped listing is still stored")
	}

	// An empty answer changes nothing: it may be a broken one.
	fetcher.payload = []byte(emptyAnswer)
	if _, err := collector.Collect(ctx, now.Add(45*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM earthquakes`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining == 0 {
		t.Fatal("an empty answer wiped the table")
	}
}
