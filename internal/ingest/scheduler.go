// Package ingest runs collectors on their schedules and records every run.
package ingest

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// schedulerLockID is the PostgreSQL advisory lock that elects one scheduler
// across worker replicas ("LKWX").
const schedulerLockID = 0x4c4b5758

// Retention limits for stored data.
const (
	readingRetention = 400 * 24 * time.Hour // a year of history, plus a margin to compare seasons
	runRetention     = 30 * 24 * time.Hour
	runTimeout       = 5 * time.Minute
)

// Scheduler runs each collector when its interval has passed since its last
// run. Every worker replica runs a Scheduler; only the one holding the
// advisory lock collects, and the others wait to take over.
type Scheduler struct {
	Pool       *pgxpool.Pool
	Collectors []source.Collector
	Logger     *slog.Logger
	// Tick is how often due collectors are checked.
	Tick time.Duration
	Now  func() time.Time
}

// Run schedules until ctx is cancelled.
func (scheduler *Scheduler) Run(ctx context.Context) {
	for ctx.Err() == nil {
		led, err := scheduler.lead(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			scheduler.Logger.Warn("scheduler stopped leading", "error", err)
		}
		wait := 15 * time.Second
		if led {
			wait = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// lead holds the advisory lock on one connection and schedules while it holds
// it. It returns false when another replica leads.
func (scheduler *Scheduler) lead(ctx context.Context) (bool, error) {
	connection, err := scheduler.Pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	defer connection.Release()
	var acquired bool
	if err := connection.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, schedulerLockID).Scan(&acquired); err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}
	defer func() {
		_, _ = connection.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, schedulerLockID)
	}()
	scheduler.Logger.Info("scheduler leading", "collectors", len(scheduler.Collectors))

	queries := store.New(scheduler.Pool)
	// A previous leader may have stopped mid-run; its runs never finish.
	now := scheduler.now()
	if err := queries.AbandonRunningRuns(ctx, store.AbandonRunningRunsParams{StartedAt: now, FinishedAt: &now}); err != nil {
		return true, err
	}

	tick := scheduler.Tick
	if tick <= 0 {
		tick = 30 * time.Second
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	lastCleanup := time.Time{}
	for {
		// The lock lives on this connection; a failed ping means it is gone.
		if err := connection.Ping(ctx); err != nil {
			return true, err
		}
		scheduler.RunDue(ctx)
		if now := scheduler.now(); now.Sub(lastCleanup) >= time.Hour {
			scheduler.cleanup(ctx, now)
			lastCleanup = now
		}
		select {
		case <-ctx.Done():
			return true, nil
		case <-ticker.C:
		}
	}
}

// RunDue runs, one after another, every collector whose interval has passed.
func (scheduler *Scheduler) RunDue(ctx context.Context) {
	queries := store.New(scheduler.Pool)
	for _, collector := range scheduler.Collectors {
		if ctx.Err() != nil {
			return
		}
		last, err := queries.LastRunStart(ctx, collector.Name())
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			scheduler.Logger.Warn("read last run failed", "source", collector.Name(), "error", err)
			continue
		}
		if err == nil && scheduler.now().Sub(last) < collector.Interval() {
			continue
		}
		scheduler.RunOne(ctx, collector)
	}
}

// RunOne runs one collector now and records the run.
func (scheduler *Scheduler) RunOne(ctx context.Context, collector source.Collector) {
	queries := store.New(scheduler.Pool)
	started := scheduler.now()
	id, err := queries.StartSourceRun(ctx, store.StartSourceRunParams{Source: collector.Name(), StartedAt: started})
	if err != nil {
		scheduler.Logger.Warn("record run start failed", "source", collector.Name(), "error", err)
		return
	}

	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	records, collectErr := collector.Collect(runCtx, started)
	cancel()

	finished := scheduler.now()
	params := store.FinishSourceRunParams{ID: id, FinishedAt: &finished, Status: "ok", Records: int32(records)}
	if collectErr != nil {
		message := collectErr.Error()
		params.Status, params.Error, params.Records = "error", &message, 0
		scheduler.Logger.Warn("collect failed", "source", collector.Name(), "error", collectErr)
	} else {
		scheduler.Logger.Info("collected", "source", collector.Name(), "records", records,
			"duration_ms", finished.Sub(started).Milliseconds())
	}
	// Record the outcome even when ctx was cancelled during the run.
	if err := queries.FinishSourceRun(context.WithoutCancel(ctx), params); err != nil {
		scheduler.Logger.Warn("record run finish failed", "source", collector.Name(), "error", err)
	}
}

func (scheduler *Scheduler) cleanup(ctx context.Context, now time.Time) {
	queries := store.New(scheduler.Pool)
	if err := queries.DeleteRiverReadingsBefore(ctx, now.Add(-readingRetention)); err != nil {
		scheduler.Logger.Warn("delete old readings failed", "error", err)
	}
	if err := queries.DeleteSourceRunsBefore(ctx, now.Add(-runRetention)); err != nil {
		scheduler.Logger.Warn("delete old runs failed", "error", err)
	}
	for _, collector := range scheduler.Collectors {
		if pruner, ok := collector.(source.Pruner); ok {
			if err := pruner.Prune(ctx, now); err != nil {
				scheduler.Logger.Warn("prune failed", "source", collector.Name(), "error", err)
			}
		}
	}
}

// Prune applies every collector's retention limit now, as the worker does
// hourly; `collect` calls it after its run.
func (scheduler *Scheduler) Prune(ctx context.Context) {
	scheduler.cleanup(ctx, scheduler.now())
}

func (scheduler *Scheduler) now() time.Time {
	if scheduler.Now != nil {
		return scheduler.Now().UTC()
	}
	return time.Now().UTC()
}
