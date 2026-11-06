package raingauges

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// A rolling 24-hour total R cannot be split into shorter totals, but its rise
// bounds them from below: R(b) - R(a) is the rain in (a, b] less the rain that
// left the 24 hours, which is never negative. So the rise over the last hour is
// at least as much as fell in it, never more. Alerts read these lower bounds:
// they can miss rain, never invent it.

// Windows the rise is measured over.
const (
	Window1h = time.Hour
	Window3h = 3 * time.Hour
)

// Rises bounding a window's rain no gauge could measure are a faulty sensor or
// a reset, not rain: Sri Lanka's heaviest hours are under 200 mm.
const (
	implausible1hMm = 200.0
	implausible3hMm = 400.0
)

// Reading is one point of a gauge's rolling 24-hour total.
type Reading struct {
	At     time.Time
	Rain24 float64
}

// RiseOver is the most the rolling total rose within a window ending at the
// newest reading: the newest total less the lowest in the window. It is a lower
// bound on the rain that fell in the window. Readings are oldest first.
func RiseOver(readings []Reading, window time.Duration) float64 {
	if len(readings) < 2 {
		return 0
	}
	newest := readings[len(readings)-1]
	lowest := newest.Rain24
	for index := len(readings) - 2; index >= 0; index-- {
		reading := readings[index]
		if newest.At.Sub(reading.At) > window {
			break
		}
		lowest = math.Min(lowest, reading.Rain24)
	}
	return math.Max(0, newest.Rain24-lowest)
}

// Intensity is what a gauge measured lately.
type Intensity struct {
	ID          string
	Name        string
	Institution string
	Lon, Lat    float64
	// ObservedAt is the newest reading and Rain24hMm its rolling total.
	ObservedAt time.Time
	Rain24hMm  float64
	// Rise1hMm and Rise3hMm are at least the rain of the last hour and three.
	Rise1hMm, Rise3hMm float64
	// Stale means the newest reading is too old to say anything now.
	Stale bool
	// Suspect means a rise no rain could make: the gauge is left out.
	Suspect bool
}

// Intensities is every gauge's recent rain.
type Intensities struct {
	Gauges []Intensity
	// FetchedAt is the last time the feed was stored; nil before the first.
	FetchedAt *time.Time
}

// intensityCache keeps the last computation until a new file is stored.
type intensityCache struct {
	mu        sync.Mutex
	fetchedAt time.Time
	result    Intensities
}

// lookback is the readings read: the longest window and a reading before it.
const lookback = Window3h + 30*time.Minute

// Intensities reads every gauge's recent rises. The readings change only when
// the collector stores a file, about every 30 minutes, so the result is kept
// until then.
func (service *Service) Intensities(ctx context.Context) (Intensities, error) {
	now := service.now()
	queries := store.New(service.Pool)
	gauges, err := queries.ListRainGauges(ctx)
	if err != nil {
		return Intensities{}, fmt.Errorf("list rain gauges: %w", err)
	}
	var fetched time.Time
	for _, gauge := range gauges {
		if gauge.FetchedAt.After(fetched) {
			fetched = gauge.FetchedAt
		}
	}
	service.intensity.mu.Lock()
	defer service.intensity.mu.Unlock()
	if !fetched.IsZero() && service.intensity.fetchedAt.Equal(fetched) {
		return service.restale(service.intensity.result, now), nil
	}
	rows, err := queries.RainGaugeReadingsAllSince(ctx, now.Add(-lookback))
	if err != nil {
		return Intensities{}, fmt.Errorf("read rain gauge readings: %w", err)
	}
	series := map[string][]Reading{}
	for _, row := range rows {
		series[row.GaugeID] = append(series[row.GaugeID], Reading{At: row.ObservedAt, Rain24: row.RainMm})
	}
	result := Intensities{Gauges: make([]Intensity, 0, len(gauges))}
	if !fetched.IsZero() {
		at := fetched.UTC()
		result.FetchedAt = &at
	}
	for _, gauge := range gauges {
		readings := series[gauge.ID]
		intensity := Intensity{ID: gauge.ID, Name: gauge.Name, Institution: gauge.Institution, Lon: gauge.Lon, Lat: gauge.Lat, Stale: true}
		if len(readings) > 0 {
			newest := readings[len(readings)-1]
			intensity.ObservedAt, intensity.Rain24hMm = newest.At.UTC(), newest.Rain24
			intensity.Rise1hMm, intensity.Rise3hMm = round1(RiseOver(readings, Window1h)), round1(RiseOver(readings, Window3h))
			intensity.Suspect = intensity.Rise1hMm > implausible1hMm || intensity.Rise3hMm > implausible3hMm
		}
		result.Gauges = append(result.Gauges, intensity)
	}
	service.intensity.fetchedAt, service.intensity.result = fetched, result
	return service.restale(result, now), nil
}

// restale marks gauges stale against now, on a copy.
func (service *Service) restale(result Intensities, now time.Time) Intensities {
	out := result
	out.Gauges = make([]Intensity, len(result.Gauges))
	for index, gauge := range result.Gauges {
		gauge.Stale = gauge.ObservedAt.IsZero() || now.Sub(gauge.ObservedAt) > service.StaleAfter
		out.Gauges[index] = gauge
	}
	return out
}

func round1(value float64) float64 { return math.Round(value*10) / 10 }
