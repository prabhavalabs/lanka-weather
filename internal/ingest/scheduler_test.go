package ingest

import (
	"context"
	"testing"
	"time"
)

type fakeCollector struct{ interval time.Duration }

func (fakeCollector) Name() string                                    { return "fake" }
func (collector fakeCollector) Interval() time.Duration               { return collector.interval }
func (fakeCollector) Collect(context.Context, time.Time) (int, error) { return 0, nil }

type retryingCollector struct {
	fakeCollector
	retry time.Duration
}

func (collector retryingCollector) RetryAfter() time.Duration { return collector.retry }

const week = 7 * 24 * time.Hour

func TestDueAfterUsesTheIntervalUnlessTheRunFailed(t *testing.T) {
	for _, status := range []string{"ok", "running"} {
		if got := DueAfter(fakeCollector{3 * time.Hour}, status); got != 3*time.Hour {
			t.Fatalf("after %s = %v, want the interval", status, got)
		}
		if got := DueAfter(retryingCollector{fakeCollector{week}, time.Hour}, status); got != week {
			t.Fatalf("retrier after %s = %v, want the interval", status, got)
		}
	}
}

func TestDueAfterRetriesFailedRunsSooner(t *testing.T) {
	if got := DueAfter(fakeCollector{3 * time.Hour}, "error"); got != 15*time.Minute {
		t.Fatalf("after failure = %v, want a retry in 15 minutes", got)
	}
	if got := DueAfter(fakeCollector{5 * time.Minute}, "error"); got != 5*time.Minute {
		t.Fatalf("a short interval is kept after failure: %v", got)
	}
}

func TestDueAfterHonoursACollectorsOwnRetryDelay(t *testing.T) {
	if got := DueAfter(retryingCollector{fakeCollector{week}, time.Hour}, "error"); got != time.Hour {
		t.Fatalf("weekly with a 1 h retry = %v", got)
	}
	if got := DueAfter(retryingCollector{fakeCollector{week}, 3 * time.Hour}, "error"); got != 3*time.Hour {
		t.Fatalf("weekly with a 3 h retry = %v", got)
	}
	if got := DueAfter(retryingCollector{fakeCollector{30 * time.Minute}, time.Hour}, "error"); got != 30*time.Minute {
		t.Fatalf("a retry longer than the interval = %v, want the interval", got)
	}
	if got := DueAfter(retryingCollector{fakeCollector{week}, 0}, "error"); got != DefaultRetryAfter {
		t.Fatalf("a zero retry = %v, want the default", got)
	}
}
