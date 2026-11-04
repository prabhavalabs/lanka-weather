package imerg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// collectEvery is how often new images are looked for.
	collectEvery = 30 * time.Minute
	// StepsKept is how many half hours the time slider offers: 12 hours.
	StepsKept = 24
	// retention keeps what the slider can reach, counted back from the newest
	// image so an outage does not empty the layer.
	retention = (StepsKept - 1) * Step
	// lookBack is the time range asked of GIBS. It reaches well past the
	// kept steps because the service publishes hours late.
	lookBack = 36 * time.Hour
	// tilePause spaces tile requests, so a first run of 216 tiles stays gentle.
	tilePause = 100 * time.Millisecond
)

// Doer sends a request to a source.
type Doer interface {
	Do(ctx context.Context, request source.Request) (source.Response, error)
}

// Collector stores the newest half hours of tiles that are not stored yet.
type Collector struct {
	Pool   *pgxpool.Pool
	Client Doer
	// NoPacing sends requests back to back, for tests.
	NoPacing bool
	Logger   *slog.Logger
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration { return collectEvery }

// Collect fetches the half hours GIBS has and we do not. Each half hour is
// stored whole, in one transaction, newest first, so an interrupted run still
// leaves the newest images. A half hour GIBS lists but cannot serve yet (404)
// is left for the next run and is not a failure.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	response, err := collector.Client.Do(ctx, source.Request{URL: DomainsURL(now.Add(-lookBack), now.Add(time.Hour)), Accept: "text/xml"})
	if err != nil {
		return 0, fmt.Errorf("read available times: %w", err)
	}
	intervals, err := DecodeDomains(response.Body)
	if err != nil {
		return 0, err
	}
	stored, err := store.New(collector.Pool).RainSteps(ctx)
	if err != nil {
		return 0, fmt.Errorf("read stored steps: %w", err)
	}
	complete := make(map[time.Time]bool, len(stored))
	for _, step := range stored {
		complete[step.ValidAt.UTC()] = int(step.Tiles) == len(Tiles())
	}

	records := 0
	var failure error
	for _, at := range LatestSteps(intervals, now.Add(-lookBack), StepsKept) {
		if complete[at] {
			continue
		}
		tiles, err := collector.fetchStep(ctx, at)
		if source.IsStatus(err, http.StatusNotFound) {
			collector.info("imerg step is not served yet", "valid_at", at)
			continue
		}
		if err != nil {
			failure = fmt.Errorf("step %s: %w", TimeID(at), err)
			break
		}
		if err := collector.save(ctx, at, tiles, now); err != nil {
			return records, err
		}
		records += len(tiles)
	}
	// Stored half hours stay even when a later one fails; the failure is
	// reported only when the run stored nothing new, as the next run retries it.
	if failure != nil && records == 0 {
		return 0, failure
	}
	if failure != nil {
		collector.info("imerg run stopped early", "error", failure)
	}
	return records, nil
}

type fetchedTile struct {
	tile Tile
	body []byte
}

func (collector *Collector) fetchStep(ctx context.Context, at time.Time) ([]fetchedTile, error) {
	tiles := make([]fetchedTile, 0, len(Tiles()))
	for index, tile := range Tiles() {
		if index > 0 && !collector.NoPacing {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(tilePause):
			}
		}
		response, err := collector.Client.Do(ctx, source.Request{URL: TileURL(at, tile), Accept: "image/png"})
		if err != nil {
			return nil, err
		}
		empty, err := Inspect(response.Body)
		if err != nil {
			return nil, fmt.Errorf("tile %d/%d/%d: %w", tile.Z, tile.X, tile.Y, err)
		}
		body := response.Body
		if empty {
			body = nil
		}
		tiles = append(tiles, fetchedTile{tile: tile, body: body})
	}
	return tiles, nil
}

func (collector *Collector) save(ctx context.Context, at time.Time, tiles []fetchedTile, now time.Time) error {
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		params := make([]store.UpsertRainTileParams, len(tiles))
		for index, tile := range tiles {
			params[index] = store.UpsertRainTileParams{
				ValidAt: at, Z: int32(tile.tile.Z), X: int32(tile.tile.X), Y: int32(tile.tile.Y),
				Png: tile.body, Source: SourceName, FetchedAt: now,
			}
		}
		var first error
		results := store.New(tx).UpsertRainTile(ctx, params)
		results.Exec(func(_ int, err error) {
			if err != nil && first == nil {
				first = err
			}
		})
		if err := results.Close(); err != nil && first == nil {
			first = err
		}
		if first != nil {
			return fmt.Errorf("store tiles of %s: %w", TimeID(at), first)
		}
		return nil
	})
}

// Prune implements source.Pruner: half hours beyond the slider's reach go,
// counted from the newest image rather than from now.
func (collector *Collector) Prune(ctx context.Context, _ time.Time) error {
	queries := store.New(collector.Pool)
	newest, err := queries.NewestRainStep(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return queries.DeleteRainTilesBefore(ctx, newest.Add(-retention))
}

func (collector *Collector) info(message string, args ...any) {
	if collector.Logger != nil {
		collector.Logger.Info(message, args...)
	}
}
