package ingest

import (
	"context"
	"testing"
	"time"
)

type weekly struct{}

func (weekly) Name() string                                    { return "weekly" }
func (weekly) Interval() time.Duration                         { return 7 * 24 * time.Hour }
func (weekly) Collect(context.Context, time.Time) (int, error) { return 0, nil }

type retryingWeekly struct{ weekly }

func (retryingWeekly) RetryAfter() time.Duration { return time.Hour }

func TestWaitRetriesFailedRunsSooner(t *testing.T) {
	if got := wait(retryingWeekly{}, "error"); got != time.Hour {
		t.Fatalf("after a failure = %v, want 1h", got)
	}
	if got := wait(retryingWeekly{}, "ok"); got != 7*24*time.Hour {
		t.Fatalf("after a success = %v, want the interval", got)
	}
	if got := wait(retryingWeekly{}, "running"); got != 7*24*time.Hour {
		t.Fatalf("while running = %v, want the interval", got)
	}
	if got := wait(weekly{}, "error"); got != 7*24*time.Hour {
		t.Fatalf("a collector without retries = %v, want the interval", got)
	}
}
