package rivers

import (
	"testing"
	"time"
)

func ptr(v float64) *float64 { return &v }

func TestClassify(t *testing.T) {
	thresholds := Thresholds{Alert: ptr(4), MinorFlood: ptr(5), MajorFlood: ptr(7)}
	cases := []struct {
		level float64
		want  Status
	}{
		{1.8, StatusNormal},
		{3.99, StatusNormal},
		{4, StatusAlert},
		{5, StatusMinorFlood},
		{6.9, StatusMinorFlood},
		{7, StatusMajorFlood},
		{12, StatusMajorFlood},
	}
	for _, c := range cases {
		if got := Classify(c.level, thresholds); got != c.want {
			t.Errorf("Classify(%v) = %s, want %s", c.level, got, c.want)
		}
	}
	if got := Classify(10, Thresholds{}); got != StatusUnknown {
		t.Errorf("without thresholds = %s, want unknown", got)
	}
}

func TestEstimateTrend(t *testing.T) {
	at := func(minutesAgo int) time.Time {
		return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC).Add(-time.Duration(minutesAgo) * time.Minute)
	}
	cases := []struct {
		name     string
		samples  []Sample
		want     Trend
		wantRate float64
	}{
		{"rising", []Sample{{at(0), 3.0}, {at(60), 2.8}, {at(120), 2.6}}, TrendRising, 0.2},
		{"falling", []Sample{{at(0), 2.0}, {at(120), 2.4}}, TrendFalling, -0.2},
		{"steady", []Sample{{at(0), 2.0}, {at(120), 2.02}}, TrendSteady, -0.01},
		{"compares with the oldest reading inside the window", []Sample{{at(0), 3.0}, {at(60), 3.0}, {at(180), 2.4}, {at(240), 0}}, TrendRising, 0.2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			trend, rate := EstimateTrend(c.samples)
			if trend != c.want || rate == nil || abs(*rate-c.wantRate) > 1e-9 {
				t.Fatalf("got %s %v, want %s %v", trend, deref(rate), c.want, c.wantRate)
			}
		})
	}

	if trend, _ := EstimateTrend([]Sample{{at(0), 1}}); trend != TrendUnknown {
		t.Error("one sample must be unknown")
	}
	if trend, _ := EstimateTrend([]Sample{{at(0), 1}, {at(10), 0}}); trend != TrendUnknown {
		t.Error("a span below the minimum must be unknown")
	}
	if trend, _ := EstimateTrend([]Sample{{at(0), 1}, {at(400), 0}}); trend != TrendUnknown {
		t.Error("no sample inside the window must be unknown")
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func deref(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}
