package dmcbulletins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	defaultRawBase = "https://raw.githubusercontent.com/nuuuwan/lk_dmc"
	// textWindow is how far back bulletin text is read; older bulletins keep
	// their title and link.
	textWindow = 14 * 24 * time.Hour
	// maxTextPerRun bounds the files read in one run. A first run fills the
	// newest and later runs the rest.
	maxTextPerRun = 80
	// textGiveUpAfter is when a missing text file stops being retried.
	textGiveUpAfter = 3 * 24 * time.Hour
)

// dataset names each family's branch and directory in nuuuwan/lk_dmc.
var datasets = map[Type]string{
	TypeLandslide:  "lk_dmc_landslide_warnings",
	TypeWaterLevel: "lk_dmc_river_water_level_and_flood_warnings",
	TypeWeather:    "lk_dmc_weather_forecasts",
	TypeSituation:  "lk_dmc_situation_reports",
}

// Collector stores the bulletin indexes and reads the text of new bulletins.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// RawBase overrides the raw file URL, for tests.
	RawBase string
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector. The upstream rebuilds its indexes
// about hourly, and warnings are the reason to look.
func (collector *Collector) Interval() time.Duration { return time.Hour }

// Collect reads every index, then the text of bulletins that have none yet.
// A failing index does not lose the others; the run still reports the error.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	queries := store.New(collector.Pool)
	stored := 0
	var failures []error
	var newestUpdate time.Time
	for _, kind := range Types {
		count, updated, err := collector.collectIndex(ctx, queries, kind, now)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s index: %w", kind, err))
			continue
		}
		stored += count
		if updated.After(newestUpdate) {
			newestUpdate = updated
		}
	}
	if err := collector.collectText(ctx, queries, now); err != nil {
		failures = append(failures, err)
	}
	if !newestUpdate.IsZero() {
		// The upstream's rebuild time, not ours: a fetch can succeed while the
		// upstream has quietly stopped, and that must read as out of date.
		if err := queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: SourceName, Version: newestUpdate.Format(time.RFC3339), SourceTime: newestUpdate,
			FetchedAt: now, Records: int32(stored),
		}); err != nil {
			failures = append(failures, err)
		}
	}
	return stored, errors.Join(failures...)
}

func (collector *Collector) collectIndex(ctx context.Context, queries *store.Queries, kind Type, now time.Time) (int, time.Time, error) {
	dataset := datasets[kind]
	summary, err := collector.raw(ctx, dataset, "summary.json")
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("fetch summary: %w", err)
	}
	updated, err := DecodeUpdatedAt(summary)
	if err != nil {
		return 0, time.Time{}, err
	}
	index, err := collector.raw(ctx, dataset, "docs_last100.tsv")
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("fetch index: %w", err)
	}
	bulletins, _, err := DecodeIndex(index, kind)
	if err != nil {
		return 0, time.Time{}, err
	}
	if len(bulletins) == 0 {
		return 0, time.Time{}, errors.New("index is empty")
	}
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		within := queries.WithTx(tx)
		for _, bulletin := range bulletins {
			if err := within.UpsertBulletin(ctx, store.UpsertBulletinParams{
				DocID: bulletin.DocID, BulletinType: string(bulletin.Type), Title: bulletin.Title,
				Severity: string(bulletin.Severity), IssuedAt: bulletin.IssuedAt, PdfUrl: bulletin.PDFURL,
				SourcePageUrl: bulletin.SourcePageURL, Lang: bulletin.Language, Source: SourceName,
				SourceTime: updated, FetchedAt: now,
			}); err != nil {
				return fmt.Errorf("upsert %s: %w", bulletin.DocID, err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	return len(bulletins), updated, nil
}

func (collector *Collector) collectText(ctx context.Context, queries *store.Queries, now time.Time) error {
	rows, err := queries.BulletinsNeedingText(ctx, store.BulletinsNeedingTextParams{
		Since: now.Add(-textWindow), MaxRows: maxTextPerRun,
	})
	if err != nil {
		return fmt.Errorf("list bulletins needing text: %w", err)
	}
	var failures []error
	for _, row := range rows {
		kind := Type(row.BulletinType)
		dataset, ok := datasets[kind]
		if !ok {
			continue
		}
		path := fmt.Sprintf("2020s/%d/%s/doc.txt", row.IssuedAt.In(sriLanka).Year(), row.DocID)
		text, err := collector.raw(ctx, dataset, path)
		if source.IsStatus(err, http.StatusNotFound) {
			if now.Sub(row.IssuedAt) > textGiveUpAfter {
				if err := queries.MarkBulletinTextUnavailable(ctx, store.MarkBulletinTextUnavailableParams{
					DocID: row.DocID, TextFetchedAt: &now,
				}); err != nil {
					failures = append(failures, err)
				}
			}
			continue
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("fetch text of %s: %w", row.DocID, err))
			continue
		}
		extract := ExtractText(string(text), kind, row.IssuedAt)
		if err := queries.UpdateBulletinText(ctx, store.UpdateBulletinTextParams{
			DocID: row.DocID, Severity: string(Classify(kind, row.Title, extract.Colour)),
			Colour: optional(extract.Colour), MessageID: optional(extract.MessageID),
			ValidFrom: extract.ValidFrom, ValidUntil: extract.ValidUntil, Body: optional(extract.Body),
			TextStatus: extract.Status, Districts: nonNil(extract.Districts), TextFetchedAt: &now,
		}); err != nil {
			failures = append(failures, fmt.Errorf("store text of %s: %w", row.DocID, err))
		}
	}
	return errors.Join(failures...)
}

func (collector *Collector) raw(ctx context.Context, dataset, path string) ([]byte, error) {
	base := collector.RawBase
	if base == "" {
		base = defaultRawBase
	}
	return collector.Fetcher.Fetch(ctx, source.Request{
		URL:    fmt.Sprintf("%s/data_%s/data/%s/%s", base, dataset, dataset, path),
		Accept: "text/plain, application/json, */*",
	})
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
