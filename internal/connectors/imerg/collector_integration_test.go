//go:build integration

package imerg_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/imerg"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/rainfall"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

var now = time.Date(2026, 11, 2, 20, 0, 0, 0, time.UTC)

func tile(t *testing.T, rain bool) []byte {
	t.Helper()
	picture := image.NewPaletted(image.Rect(0, 0, 8, 8), color.Palette{color.NRGBA{191, 191, 191, 0}, color.NRGBA{0, 165, 3, 255}})
	if rain {
		picture.SetColorIndex(1, 1, 1)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, picture); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// fakeGIBS answers the domains request with `domain` and tiles with rain only
// in the tile holding Sri Lanka. Steps in `missing` answer 404.
type fakeGIBS struct {
	t        *testing.T
	domain   string
	missing  string
	requests int
}

func (fake *fakeGIBS) Do(_ context.Context, request source.Request) (source.Response, error) {
	fake.requests++
	if strings.HasSuffix(request.URL, ".xml") {
		return source.Response{Status: 200, Body: []byte(`<Domains xmlns:ows="x"><DimensionDomain><ows:Identifier>time</ows:Identifier><Domain>` + fake.domain + `</Domain></DimensionDomain></Domains>`)}, nil
	}
	if fake.missing != "" && strings.Contains(request.URL, fake.missing) {
		return source.Response{Status: 404}, &source.StatusError{Status: http.StatusNotFound}
	}
	return source.Response{Status: 200, Body: tile(fake.t, strings.HasSuffix(request.URL, "/6/30/46.png"))}, nil
}

func TestCollectStoresStepsAndServesThem(t *testing.T) {
	pool := dbtest.Open(t, "rain_tiles")
	ctx := context.Background()
	fake := &fakeGIBS{t: t, domain: "2026-11-02/2026-11-02T13:30:00Z/PT30M", missing: "2026-11-02T13:30:00Z"}
	collector := &imerg.Collector{Pool: pool, Client: fake, NoPacing: true}

	// 24 half hours are listed; the newest is not served yet, which is no
	// failure, and the rest are stored whole.
	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if want := 23 * 9; stored != want {
		t.Fatalf("stored %d tiles, want %d", stored, want)
	}

	// The next run fetches only the step it does not have.
	fake.missing, fake.requests = "", 0
	stored, err = collector.Collect(ctx, now.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if stored != 9 || fake.requests != 10 {
		t.Fatalf("second run stored %d tiles in %d requests", stored, fake.requests)
	}

	service := &rainfall.Service{Pool: pool, StaleAfter: 10 * time.Hour, Now: func() time.Time { return now }}
	steps, err := service.Steps(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps.Steps) != 24 || steps.Steps[0].ValidAt.After(steps.Steps[23].ValidAt) || steps.Stale {
		t.Fatalf("steps = %d, stale %v", len(steps.Steps), steps.Stale)
	}
	newest := steps.Steps[23]
	if newest.Time != "20261102T1330Z" || newest.Tiles != 9 || newest.EmptyTiles != 8 {
		t.Fatalf("newest = %+v", newest)
	}

	rainy, err := service.Tile(ctx, newest.Time, 6, 46, 30)
	if err != nil || bytes.Equal(rainy, rainfall.EmptyTile()) {
		t.Fatalf("rainy tile = %d bytes, %v", len(rainy), err)
	}
	empty, err := service.Tile(ctx, newest.Time, 6, 45, 30)
	if err != nil || !bytes.Equal(empty, rainfall.EmptyTile()) {
		t.Fatalf("empty tile = %d bytes, %v", len(empty), err)
	}
	if _, err := service.Tile(ctx, "20200101T0000Z", 6, 46, 30); err != rainfall.ErrTileNotFound {
		t.Fatalf("old step error = %v", err)
	}
	if _, err := service.Tile(ctx, newest.Time, 6, 10, 10); err != rainfall.ErrTileNotFound {
		t.Fatalf("outside error = %v", err)
	}

	// Pruning counts back from the newest image, not from now, so an outage
	// leaves the layer as it was. Add an old step and see only that go.
	if err := pool.QueryRow(ctx, `SELECT 1`).Scan(new(int)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO rain_tiles SELECT valid_at - interval '3 days', z, x, y, png, source, fetched_at FROM rain_tiles WHERE valid_at = $1`, newest.ValidAt); err != nil {
		t.Fatal(err)
	}
	if err := collector.Prune(ctx, now.Add(30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	rows, err := store.New(pool).RainSteps(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != imerg.StepsKept {
		t.Fatalf("after pruning %d steps, want %d", len(rows), imerg.StepsKept)
	}
}

func TestCollectReportsAFailureOnlyWhenNothingWasStored(t *testing.T) {
	pool := dbtest.Open(t, "rain_tiles")
	fake := &brokenTiles{}
	collector := &imerg.Collector{Pool: pool, Client: fake, NoPacing: true}
	if _, err := collector.Collect(context.Background(), now); err == nil {
		t.Fatal("expected an error when a server failure stores nothing")
	}
}

type brokenTiles struct{}

func (*brokenTiles) Do(_ context.Context, request source.Request) (source.Response, error) {
	if strings.HasSuffix(request.URL, ".xml") {
		return source.Response{Status: 200, Body: []byte(`<Domains xmlns:ows="x"><DimensionDomain><ows:Identifier>time</ows:Identifier><Domain>2026-11-02/2026-11-02T13:30:00Z/PT30M</Domain></DimensionDomain></Domains>`)}, nil
	}
	return source.Response{Status: 500}, &source.StatusError{Status: 500}
}
