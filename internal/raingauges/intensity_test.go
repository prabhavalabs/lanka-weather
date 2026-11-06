package raingauges

import (
	"testing"
	"time"
)

// colombo is the Department of Meteorology's gauge at its head office in
// Colombo on 6 November 2026 (UTC), as stored: a rolling 24-hour total.
func colombo() []Reading {
	at := func(hour, minute int) time.Time { return time.Date(2026, 11, 6, hour, minute, 0, 0, time.UTC) }
	return []Reading{
		{at(11, 10), 17}, {at(11, 20), 17}, {at(11, 50), 16}, {at(12, 20), 26.5}, {at(12, 50), 83.5},
		{at(13, 0), 91}, {at(14, 0), 128.5}, {at(14, 40), 129},
	}
}

func upTo(readings []Reading, at time.Time) []Reading {
	out := []Reading{}
	for _, reading := range readings {
		if !reading.At.After(at) {
			out = append(out, reading)
		}
	}
	return out
}

func TestRiseOverBoundsTheCloudburstFromBelow(t *testing.T) {
	readings := colombo()
	cases := []struct {
		at          time.Time
		hour, three float64
	}{
		// Before the burst the total falls as old rain leaves it: no rise.
		{time.Date(2026, 11, 6, 11, 50, 0, 0, time.UTC), 0, 0},
		{time.Date(2026, 11, 6, 12, 20, 0, 0, time.UTC), 10.5, 10.5},
		// 18:20 in Colombo: at least 67.5 mm in the hour.
		{time.Date(2026, 11, 6, 12, 50, 0, 0, time.UTC), 67.5, 67.5},
		{time.Date(2026, 11, 6, 14, 0, 0, 0, time.UTC), 37.5, 112.5},
		// Hours later the three-hour rise still holds the burst.
		{time.Date(2026, 11, 6, 14, 40, 0, 0, time.UTC), 0.5, 113},
	}
	for _, c := range cases {
		window := upTo(readings, c.at)
		if got := RiseOver(window, Window1h); got != c.hour {
			t.Errorf("%s: 1 h rise %.1f, want %.1f", c.at.Format("15:04"), got, c.hour)
		}
		if got := RiseOver(window, Window3h); got != c.three {
			t.Errorf("%s: 3 h rise %.1f, want %.1f", c.at.Format("15:04"), got, c.three)
		}
	}
}

func TestRiseOverNeedsTwoReadings(t *testing.T) {
	if RiseOver(nil, Window1h) != 0 || RiseOver(colombo()[:1], Window1h) != 0 {
		t.Fatal("one reading has no rise")
	}
}
