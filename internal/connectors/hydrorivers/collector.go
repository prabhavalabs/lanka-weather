package hydrorivers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/adminregions"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Where the data lives. Both files are read at the commit that last changed
// the river file, so one load never mixes versions.
const (
	Repository     = "nuuuwan/lk_rivers"
	riversPath     = "data/sri_lanka_rivers.geojson"
	namesPath      = "data/static/riv_id_to_name.json"
	defaultAPIBase = "https://api.github.com"
	defaultRawBase = "https://raw.githubusercontent.com"
	maxRiverBytes  = 16 << 20 // the file is about 3.5 MB
	riverTimeout   = 2 * time.Minute
	// MinimumSegments guards against a truncated upstream file deleting rivers.
	MinimumSegments = 5000
)

// Collector loads the network weekly, when upstream has changed.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// APIBase and RawBase override GitHub's URLs, for tests.
	APIBase, RawBase string
	// Minimum overrides MinimumSegments, for tests with small fixtures.
	Minimum int
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector. HydroRIVERS is a static dataset; only
// the extract can change.
func (collector *Collector) Interval() time.Duration { return 7 * 24 * time.Hour }

// RetryAfter implements source.Retrier. Nothing depends on a fresh copy, so a
// failure can wait.
func (collector *Collector) RetryAfter() time.Duration { return 6 * time.Hour }

// Collect loads the segments and names when a newer commit exists and returns
// the number of segments stored.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	commit, err := collector.latestCommit(ctx)
	if err != nil {
		return 0, fmt.Errorf("river version: %w", err)
	}
	queries := store.New(collector.Pool)
	stored, err := queries.GetSourceVersion(ctx, SourceName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("read river version: %w", err)
	}
	if err == nil && stored.Version == commit.SHA {
		if count, err := queries.CountRiverSegments(ctx); err == nil && count > 0 {
			return 0, nil
		}
	}
	payload, err := collector.raw(ctx, commit.SHA, riversPath)
	if err != nil {
		return 0, fmt.Errorf("fetch rivers: %w", err)
	}
	segments, _, err := DecodeSegments(payload)
	if err != nil {
		return 0, err
	}
	if minimum := collector.minimum(); len(segments) < minimum {
		return 0, fmt.Errorf("river file has %d segments, want at least %d", len(segments), minimum)
	}
	payload, err = collector.raw(ctx, commit.SHA, namesPath)
	if err != nil {
		return 0, fmt.Errorf("fetch river names: %w", err)
	}
	names, err := DecodeNames(payload)
	if err != nil {
		return 0, err
	}
	if err := collector.save(ctx, segments, names, commit, now); err != nil {
		return 0, err
	}
	return len(segments), nil
}

func (collector *Collector) minimum() int {
	if collector.Minimum > 0 {
		return collector.Minimum
	}
	return MinimumSegments
}

func (collector *Collector) latestCommit(ctx context.Context) (adminregions.Commit, error) {
	base := collector.APIBase
	if base == "" {
		base = defaultAPIBase
	}
	payload, err := collector.Fetcher.Fetch(ctx, source.Request{
		URL:    base + "/repos/" + Repository + "/commits",
		Query:  url.Values{"path": {riversPath}, "per_page": {"1"}},
		Accept: "application/vnd.github+json",
	})
	if err != nil {
		return adminregions.Commit{}, err
	}
	return adminregions.DecodeCommits(payload)
}

func (collector *Collector) raw(ctx context.Context, sha, path string) ([]byte, error) {
	base := collector.RawBase
	if base == "" {
		base = defaultRawBase
	}
	return collector.Fetcher.Fetch(ctx, source.Request{
		URL:      base + "/" + Repository + "/" + sha + "/" + path,
		Accept:   "application/json, text/plain, */*",
		MaxBytes: maxRiverBytes,
		Timeout:  riverTimeout,
	})
}

func (collector *Collector) save(ctx context.Context, segments []Segment, names map[int64]string, commit adminregions.Commit, now time.Time) error {
	rows := make([]store.StageRiverSegmentsParams, len(segments))
	for index, segment := range segments {
		rows[index] = store.StageRiverSegmentsParams{
			HyrivID: segment.ID, NextDown: segment.NextDown, MainRiv: segment.MainRiv, LengthKm: segment.LengthKM,
			DistDnKm: segment.DistDownKM, DistUpKm: segment.DistUpKM, CatchKm2: segment.CatchKM2,
			UplandKm2: segment.UplandKM2, DischargeCms: segment.DischargeCMS, OrdStra: int16(segment.OrdStra),
			OrdClas: int16(segment.OrdClas), OrdFlow: int16(segment.OrdFlow), HybasL12: segment.HybasL12,
			Wkb: lineWKB(segment),
		}
	}
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		if err := queries.ClearRiverSegmentStaging(ctx); err != nil {
			return err
		}
		if _, err := queries.StageRiverSegments(ctx, rows); err != nil {
			return fmt.Errorf("stage river segments: %w", err)
		}
		if _, err := queries.MergeStagedRiverSegments(ctx, store.MergeStagedRiverSegmentsParams{
			Source: SourceName, SourceVersion: commit.SHA, SourceTime: commit.Time, FetchedAt: now,
		}); err != nil {
			return fmt.Errorf("merge river segments: %w", err)
		}
		if _, err := queries.DeleteRiverSegmentsNotStaged(ctx); err != nil {
			return fmt.Errorf("delete removed segments: %w", err)
		}
		if err := queries.ClearRiverSegmentStaging(ctx); err != nil {
			return err
		}
		if err := queries.DeleteRiverSourceNames(ctx); err != nil {
			return err
		}
		for id, name := range names {
			if err := queries.InsertRiverSourceName(ctx, store.InsertRiverSourceNameParams{
				RivID: id, Name: name, Source: SourceName, FetchedAt: now,
			}); err != nil {
				return fmt.Errorf("store river name: %w", err)
			}
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: SourceName, Version: commit.SHA, SourceTime: commit.Time, FetchedAt: now, Records: int32(len(rows)),
		})
	})
}
