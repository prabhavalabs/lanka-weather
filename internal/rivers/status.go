// Package rivers turns stored gauge readings into the river-levels map layer:
// each station's current level, its flood status against the station's
// thresholds, and whether the river is rising or falling.
package rivers

import "time"

// Status is a station's level relative to its published thresholds.
type Status string

const (
	StatusNormal     Status = "normal"
	StatusAlert      Status = "alert"
	StatusMinorFlood Status = "minor_flood"
	StatusMajorFlood Status = "major_flood"
	// StatusUnknown means the station publishes no usable thresholds.
	StatusUnknown Status = "unknown"
)

// Thresholds are a station's alert, minor-flood and major-flood levels, in
// the station's unit. All three are set, or none.
type Thresholds struct {
	Alert, MinorFlood, MajorFlood *float64
}

// Classify returns the status of a water level.
func Classify(level float64, thresholds Thresholds) Status {
	if thresholds.Alert == nil || thresholds.MinorFlood == nil || thresholds.MajorFlood == nil {
		return StatusUnknown
	}
	switch {
	case level >= *thresholds.MajorFlood:
		return StatusMajorFlood
	case level >= *thresholds.MinorFlood:
		return StatusMinorFlood
	case level >= *thresholds.Alert:
		return StatusAlert
	default:
		return StatusNormal
	}
}

// Trend is the direction a river is moving.
type Trend string

const (
	TrendRising  Trend = "rising"
	TrendFalling Trend = "falling"
	TrendSteady  Trend = "steady"
	// TrendUnknown means there is no earlier reading close enough to compare.
	TrendUnknown Trend = "unknown"
)

// Sample is one reading used for trend estimation.
type Sample struct {
	ObservedAt time.Time
	Level      float64
}

const (
	// trendWindow is how far back the comparison reading is looked for.
	trendWindow = 3 * time.Hour
	// minimumSpan avoids dividing tiny level changes by tiny time spans.
	minimumSpan = 30 * time.Minute
	// steadyRate is the change per hour, in the station's unit, below which
	// a river is called steady. Gauges are read to a centimetre or a tenth of
	// a foot; 5 per hour is well above reading noise.
	steadyRate = 0.05
)

// EstimateTrend compares the latest sample with the oldest one within the
// trend window and returns the direction and the rate of change per hour.
// Samples must be ordered newest first.
func EstimateTrend(samples []Sample) (Trend, *float64) {
	if len(samples) < 2 {
		return TrendUnknown, nil
	}
	latest := samples[0]
	var earlier *Sample
	for i := 1; i < len(samples); i++ {
		if latest.ObservedAt.Sub(samples[i].ObservedAt) > trendWindow {
			break
		}
		earlier = &samples[i]
	}
	if earlier == nil {
		return TrendUnknown, nil
	}
	span := latest.ObservedAt.Sub(earlier.ObservedAt)
	if span < minimumSpan {
		return TrendUnknown, nil
	}
	rate := (latest.Level - earlier.Level) / span.Hours()
	switch {
	case rate >= steadyRate:
		return TrendRising, &rate
	case rate <= -steadyRate:
		return TrendFalling, &rate
	default:
		return TrendSteady, &rate
	}
}
