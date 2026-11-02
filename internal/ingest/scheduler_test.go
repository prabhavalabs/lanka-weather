package ingest

import (
	"testing"
	"time"
)

func TestDueAfter(t *testing.T) {
	if got := DueAfter(3*time.Hour, "ok"); got != 3*time.Hour {
		t.Fatalf("after success = %v", got)
	}
	if got := DueAfter(3*time.Hour, "error"); got != 15*time.Minute {
		t.Fatalf("after failure = %v, want a retry in 15 minutes", got)
	}
	if got := DueAfter(5*time.Minute, "error"); got != 5*time.Minute {
		t.Fatalf("a short interval is kept after failure: %v", got)
	}
}
