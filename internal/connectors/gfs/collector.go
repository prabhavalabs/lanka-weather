package gfs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/grib2"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Interval is how often the collector looks for a new run. A run's hours
// appear over about an hour; half-hourly checks store it soon after its
// last hour is out.
const Interval = 30 * time.Minute

// retention is how long a run is kept once a newer complete run exists.
const retention = 30 * time.Hour

// Doer sends a request to a source.
type Doer interface {
	Do(ctx context.Context, request source.Request) (source.Response, error)
}

// Collector stores the newest GFS run whose hours up to MaxForecast are all
// published.
type Collector struct {
	Pool    *pgxpool.Pool
	Client  Doer
	BaseURL string
	Logger  *slog.Logger
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration { return Interval }

// errMissing means a file is not published (yet).
var errMissing = errors.New("not published")

// Collect stores the missing hours of the newest complete run. A run counts
// as complete when its last hour is published; until then the previous run
// stays in use. It returns the number of hours stored.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	queries := store.New(collector.Pool)
	hours := Hours()
	for _, run := range Runs(now) {
		stored, err := queries.WindRunHours(ctx, run)
		if err != nil {
			return 0, fmt.Errorf("read stored hours: %w", err)
		}
		have := map[int]bool{}
		for _, hour := range stored {
			have[int(hour)] = true
		}
		if len(have) >= len(hours) {
			return 0, nil // the newest complete run is already stored
		}
		last := hours[len(hours)-1]
		if _, err := collector.index(ctx, run, last); errors.Is(err, errMissing) {
			continue // not complete yet; try the run before
		} else if err != nil {
			return 0, err
		}
		count := 0
		for _, hour := range hours {
			if have[hour] {
				continue
			}
			if err := collector.storeHour(ctx, queries, run, hour, now); err != nil {
				return count, fmt.Errorf("run %s hour %d: %w", run.Format("2006010215"), hour, err)
			}
			count++
		}
		if collector.Logger != nil {
			collector.Logger.Info("gfs wind run stored", "run", run, "hours", count)
		}
		return count, nil
	}
	return 0, errors.New("no GFS run of the last day is published to its last hour")
}

func (collector *Collector) base() string {
	if collector.BaseURL != "" {
		return collector.BaseURL
	}
	return DefaultBaseURL
}

func (collector *Collector) index(ctx context.Context, run time.Time, hour int) ([]indexEntry, error) {
	response, err := collector.Client.Do(ctx, source.Request{URL: FileURL(collector.base(), run, hour) + ".idx"})
	// S3 answers 403 for a key that is not there when listing is not allowed.
	if source.IsStatus(err, http.StatusNotFound, http.StatusForbidden) {
		return nil, errMissing
	}
	if err != nil {
		return nil, fmt.Errorf("GFS inventory: %w", err)
	}
	return ParseIndex(response.Body)
}

// storeHour reads one hour's u and v messages by byte range, crops them to
// the region and stores them.
func (collector *Collector) storeHour(ctx context.Context, queries *store.Queries, run time.Time, hour int, now time.Time) error {
	entries, err := collector.index(ctx, run, hour)
	if err != nil {
		return err
	}
	file := FileURL(collector.base(), run, hour)
	var components [2][]float32
	var grid grib2.Grid
	for index, variable := range []string{"UGRD", "VGRD"} {
		span, err := MessageRange(entries, run, variable, Level)
		if err != nil {
			return err
		}
		if span.Size() > maxMessageBytes {
			return fmt.Errorf("GFS %s message is %d bytes", variable, span.Size())
		}
		response, err := collector.Client.Do(ctx, source.Request{
			URL: file, Header: http.Header{"Range": {span.Header()}}, Accept: []int{http.StatusPartialContent},
		})
		if err != nil {
			return fmt.Errorf("GFS %s: %w", variable, err)
		}
		if int64(len(response.Body)) != span.Size() {
			return fmt.Errorf("GFS %s: %d bytes for %d asked", variable, len(response.Body), span.Size())
		}
		field, err := grib2.Decode(response.Body)
		if err != nil {
			return fmt.Errorf("GFS %s: %w", variable, err)
		}
		if field.Discipline != 0 || field.Category != 2 || field.Number != 2+index || !field.Reference.Equal(run) {
			return fmt.Errorf("GFS %s: message is parameter %d.%d.%d of run %s", variable, field.Discipline, field.Category, field.Number, field.Reference)
		}
		cropped, values, err := Crop(field.Grid, field.Values, Region)
		if err != nil {
			return err
		}
		if index == 1 && cropped != grid {
			return errors.New("GFS u and v are on different grids")
		}
		grid, components[index] = cropped, values
	}
	return queries.UpsertWindField(ctx, store.UpsertWindFieldParams{
		RunAt: run, ForecastHour: int32(hour), ValidAt: run.Add(time.Duration(hour) * time.Hour),
		Source: SourceName, Model: Model, Level: Level,
		Lon0: grid.Lon0, Lat0: grid.Lat0, Dlon: grid.DLon, Dlat: grid.DLat, Nx: int32(grid.Nx), Ny: int32(grid.Ny),
		Scale: Scale, U: Quantise(components[0]), V: Quantise(components[1]), FetchedAt: now,
	})
}

// Prune implements source.Pruner: runs more than 30 hours old go once a newer
// run is complete, so the layer never empties for want of a new run.
func (collector *Collector) Prune(ctx context.Context, now time.Time) error {
	queries := store.New(collector.Pool)
	runs, err := queries.LatestWindRuns(ctx, 8)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if int(run.Steps) >= len(Hours()) {
			cutoff := now.Add(-retention)
			if run.RunAt.Before(cutoff) {
				cutoff = run.RunAt
			}
			return queries.DeleteWindRunsBefore(ctx, cutoff)
		}
	}
	return nil
}
