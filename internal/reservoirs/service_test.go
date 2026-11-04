package reservoirs

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
)

func TestIsStale(t *testing.T) {
	day := func(text string) pgtype.Date {
		parsed, err := time.Parse(time.DateOnly, text)
		if err != nil {
			t.Fatal(err)
		}
		return pgtype.Date{Time: parsed, Valid: true}
	}
	// 01:09 on 3 November in Sri Lanka.
	now := time.Date(2026, 11, 2, 19, 39, 0, 0, time.UTC)
	limit := 48 * time.Hour
	cases := []struct {
		day  pgtype.Date
		want bool
	}{
		{day("2026-11-03"), false},
		{day("2026-11-02"), false},
		{day("2026-11-01"), false},
		// The end of 31 October in Sri Lanka is 18:30 UTC that day: just over
		// 49 hours ago.
		{day("2026-10-31"), true},
		{day("2026-10-29"), true},
		{pgtype.Date{}, true},
	}
	for _, test := range cases {
		if got := IsStale(test.day, now, limit); got != test.want {
			t.Errorf("IsStale(%v) = %v, want %v", test.day.Time, got, test.want)
		}
	}
}

func TestActive(t *testing.T) {
	cases := []struct {
		status string
		stale  bool
		want   bool
	}{
		{waterlevels.StatusSpilling, false, true},
		{waterlevels.StatusReleasing, false, true},
		{waterlevels.StatusHolding, false, false},
		{waterlevels.StatusNoData, false, false},
		// An old report of spilling says nothing about today.
		{waterlevels.StatusSpilling, true, false},
	}
	for _, test := range cases {
		if got := (Properties{Status: test.status, Stale: test.stale}).Active(); got != test.want {
			t.Errorf("Active(%s, stale=%v) = %v", test.status, test.stale, got)
		}
	}
}
