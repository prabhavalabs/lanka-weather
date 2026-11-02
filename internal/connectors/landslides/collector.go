package landslides

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// DefaultURL is the parsed history of reports (about 550 kB).
	DefaultURL = "https://raw.githubusercontent.com/nuuuwan/lk_dmc_landslides/main/data/all.json"
	// rewriteWindow is how far back stored reports are rewritten on every run,
	// in case a report was parsed again upstream.
	rewriteWindow = 14 * 24 * time.Hour
)

// Collector stores the reports it has not seen and refreshes recent ones.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// URL overrides DefaultURL, for tests.
	URL string
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector. NBRO issues about one report a day
// while a risk lasts, usually at 16:00, and it reaches this feed a few hours
// later; checking every two hours picks it up the same evening.
func (collector *Collector) Interval() time.Duration { return 2 * time.Hour }

// Collect fetches the history and stores new and recent reports, returning
// the number of division levels written.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	url := collector.URL
	if url == "" {
		url = DefaultURL
	}
	payload, err := collector.Fetcher.Fetch(ctx, source.Request{URL: url})
	if err != nil {
		return 0, fmt.Errorf("fetch landslide reports: %w", err)
	}
	reports, _, err := Decode(payload)
	if err != nil {
		return 0, err
	}
	stored := 0
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		known, err := knownReports(ctx, tx)
		if err != nil {
			return err
		}
		for _, report := range reports {
			date := report.Date.Format(time.DateOnly)
			if known[date] && now.Sub(report.Date) > rewriteWindow {
				continue
			}
			params := pgtype.Date{Time: report.Date, Valid: true}
			if err := queries.UpsertLandslideReport(ctx, store.UpsertLandslideReportParams{
				ReportDate: params, Source: SourceName, SourceTime: report.Date, FetchedAt: now,
			}); err != nil {
				return fmt.Errorf("store report %s: %w", date, err)
			}
			if err := queries.DeleteLandslideLevels(ctx, params); err != nil {
				return fmt.Errorf("replace report %s: %w", date, err)
			}
			rows := make([]store.InsertLandslideLevelsParams, 0, len(report.Levels))
			for _, level := range report.Levels {
				rows = append(rows, store.InsertLandslideLevelsParams{
					ReportDate: params, DsID: level.DS, DistrictID: level.District, Level: int16(level.Level),
				})
			}
			written, err := queries.InsertLandslideLevels(ctx, rows)
			if err != nil {
				return fmt.Errorf("store levels of %s: %w", date, err)
			}
			stored += int(written)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	newest := reports[0]
	if err := store.New(collector.Pool).PutSourceVersion(ctx, store.PutSourceVersionParams{
		Source: SourceName, Version: newest.Date.Format(time.DateOnly), SourceTime: newest.Date,
		FetchedAt: now, Records: int32(len(newest.Levels)),
	}); err != nil {
		return stored, errors.New("record landslide version: " + err.Error())
	}
	return stored, nil
}

func knownReports(ctx context.Context, tx pgx.Tx) (map[string]bool, error) {
	rows, err := tx.Query(ctx, `SELECT report_date::text FROM landslide_reports`)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()
	known := map[string]bool{}
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		known[date] = true
	}
	return known, rows.Err()
}
