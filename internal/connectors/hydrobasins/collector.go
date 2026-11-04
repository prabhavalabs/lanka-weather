package hydrobasins

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// ArchiveURL is the Asia level 12 archive, about 80 MB.
	ArchiveURL = "https://data.hydrosheds.org/file/hydrobasins/standard/hybas_as_lev12_v1c.zip"
	// maxArchiveBytes is an explicit limit well above the published size.
	maxArchiveBytes = 256 << 20
	downloadTimeout = 25 * time.Minute
	loadTimeout     = "4min"
	// MinimumBasins guards against a broken archive deleting sub-basins. Sri
	// Lanka has several hundred at level 12.
	MinimumBasins = 300
)

// Client is what the collector needs from source.Client.
type Client interface {
	Do(ctx context.Context, request source.Request) (source.Response, error)
	Download(ctx context.Context, request source.Request, path string) (http.Header, int64, error)
}

// Collector loads the sub-basins monthly, when the archive has changed. It
// downloads to a temporary file, so the archive never sits in memory, and runs
// apart from other collectors (source.LongRunner).
type Collector struct {
	Pool   *pgxpool.Pool
	Client Client
	Logger *slog.Logger
	// URL overrides ArchiveURL, and Minimum MinimumBasins, for tests.
	URL     string
	Minimum int
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector. HydroBASINS is a static dataset.
func (collector *Collector) Interval() time.Duration { return 30 * 24 * time.Hour }

// RetryAfter implements source.Retrier: an 80 MB file is not retried often.
func (collector *Collector) RetryAfter() time.Duration { return 24 * time.Hour }

// LongRunTimeout implements source.LongRunner.
func (collector *Collector) LongRunTimeout() time.Duration { return 30 * time.Minute }

func (collector *Collector) url() string {
	if collector.URL != "" {
		return collector.URL
	}
	return ArchiveURL
}

// Collect loads the sub-basins when the archive's validator has changed, and
// returns the number stored.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	queries := store.New(collector.Pool)
	stored, err := queries.GetSourceVersion(ctx, SourceName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("read sub-basin version: %w", err)
	}
	head, err := collector.Client.Do(ctx, source.Request{Method: http.MethodHead, URL: collector.url(), Timeout: time.Minute})
	if err != nil {
		return 0, fmt.Errorf("check the archive: %w", err)
	}
	version := validator(head.Header)
	if version != "" && stored.Version == version {
		if count, err := queries.CountRiverSubbasins(ctx); err == nil && count > 0 {
			return 0, nil
		}
	}

	directory, err := os.MkdirTemp("", "hydrobasins-")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(directory) }()
	path := filepath.Join(directory, "archive.zip")
	header, size, err := collector.Client.Download(ctx, source.Request{
		URL: collector.url(), Accept: "application/zip, */*", MaxBytes: maxArchiveBytes, Timeout: downloadTimeout,
	}, path)
	if err != nil {
		return 0, fmt.Errorf("download the archive: %w", err)
	}
	if version == "" {
		version = validator(header)
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return 0, fmt.Errorf("open the archive: %w", err)
	}
	defer func() { _ = archive.Close() }()
	basins, err := DecodeArchive(&archive.Reader, Bounds)
	if err != nil {
		return 0, err
	}
	minimum := collector.Minimum
	if minimum <= 0 {
		minimum = MinimumBasins
	}
	if len(basins) < minimum {
		return 0, fmt.Errorf("the archive has %d sub-basins in Sri Lanka, want at least %d", len(basins), minimum)
	}
	sourceTime := now
	for _, file := range archive.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".shp") && !file.Modified.IsZero() {
			sourceTime = file.Modified
		}
	}
	if err := collector.save(ctx, basins, version, sourceTime, now); err != nil {
		return 0, err
	}
	if collector.Logger != nil {
		collector.Logger.Info("sub-basins stored", "basins", len(basins), "archive_bytes", size)
	}
	return len(basins), nil
}

// validator names a version of the archive from its response headers. The
// host sends no entity tag, and its Last-Modified moves with no change to the
// file, so the size stands in: HydroBASINS v1c has not changed since 2014.
func validator(header http.Header) string {
	if etag := strings.Trim(header.Get("ETag"), `"`); etag != "" {
		return etag
	}
	if length := header.Get("Content-Length"); length != "" {
		return "size-" + length
	}
	return ""
}

func (collector *Collector) save(ctx context.Context, basins []Basin, version string, sourceTime, now time.Time) error {
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '"+loadTimeout+"'"); err != nil {
			return err
		}
		queries := store.New(tx)
		if err := queries.DeleteRiverSubbasins(ctx); err != nil {
			return err
		}
		for _, basin := range basins {
			if err := queries.InsertRiverSubbasin(ctx, store.InsertRiverSubbasinParams{
				HybasID: basin.ID, NextDown: basin.NextDown, MainBas: basin.MainBasin, AreaKm2: basin.AreaKM2,
				UpAreaKm2: basin.UpAreaKM2, Source: SourceName, Version: version, SourceTime: sourceTime, FetchedAt: now,
				Wkb: basin.Shape.WKB(),
			}); err != nil {
				return fmt.Errorf("store sub-basin %d: %w", basin.ID, err)
			}
		}
		if _, err := queries.AssignSubbasinsToOfficialBasins(ctx); err != nil {
			return fmt.Errorf("assign sub-basins: %w", err)
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: SourceName, Version: version, SourceTime: sourceTime, FetchedAt: now, Records: int32(len(basins)),
		})
	})
}
