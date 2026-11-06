package weather

import (
	"context"
	"math"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
)

// OutlookStaleAfter is the age after which a forecast says nothing about the
// hours ahead, as for the point forecast.
const OutlookStaleAfter = 12 * time.Hour

// RainOutlook is a region's forecast rain in the hours ahead: the wettest cell
// over the next 6 hours, which is where a burst would flood, and the mean over
// its area for the next 24 hours.
type RainOutlook struct {
	Next6hMaxMm   float64 `json:"next6hMaxMm"`
	Next24hMeanMm float64 `json:"next24hMeanMm"`
}

// RainOutlooks is every region's outlook at one level, from one forecast run.
type RainOutlooks struct {
	Regions   map[string]RainOutlook
	FetchedAt time.Time
	Stale     bool
}

// RainOutlooks reads the forecast rain ahead for each region of a level.
func (service *Service) RainOutlooks(ctx context.Context, level string) (*RainOutlooks, error) {
	if !ValidLevel(level) {
		return nil, ErrUnknownLevel
	}
	if service.Regions == nil {
		return nil, ErrRegionsUnavailable
	}
	regions, err := service.Regions.Regions(ctx, level)
	if err != nil {
		return nil, err
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return OutlooksFor(snapshot, regions, service.now()), nil
}

// OutlooksFor computes the outlooks from a snapshot; nil without an hourly
// rain forecast covering now. Pure.
func OutlooksFor(snapshot *Snapshot, regions []Region, now time.Time) *RainOutlooks {
	if snapshot == nil || snapshot.Forecast == nil {
		return nil
	}
	field := snapshot.Forecast.Fields[openmeteo.HourlyPrefix+"precipitation"]
	if field == nil || field.Step != time.Hour {
		return nil
	}
	start, ok := field.StepAt(now)
	if !ok {
		return nil
	}
	size := len(field.At(start))
	next6, next24 := make([]float32, size), make([]float32, size)
	covered := make([]bool, size)
	for step := start; step < min(start+24, field.Steps); step++ {
		for node, value := range field.At(step) {
			if math.IsNaN(float64(value)) {
				continue
			}
			covered[node] = true
			next24[node] += value
			if step < start+6 {
				next6[node] += value
			}
		}
	}
	nan := float32(math.NaN())
	for node, ok := range covered {
		if !ok {
			next6[node], next24[node] = nan, nan
		}
	}
	out := &RainOutlooks{Regions: make(map[string]RainOutlook, len(regions)), FetchedAt: snapshot.Forecast.FetchedAt,
		Stale: now.Sub(snapshot.Forecast.FetchedAt) > OutlookStaleAfter}
	for _, region := range regions {
		nodes, weights := region.Nodes, region.Weights
		if len(nodes) == 0 || len(weights) != len(nodes) {
			around, ok := corners(snapshot.Lattice, region.Lon, region.Lat)
			if !ok {
				continue
			}
			nodes, weights = nil, nil
			for _, corner := range around {
				if corner.Weight > 0 {
					nodes, weights = append(nodes, corner.Index), append(weights, corner.Weight)
				}
			}
		}
		wettest := extreme(next6, nodes, weights, math.Max)
		mean := weightedMean(next24, nodes, weights)
		if math.IsNaN(float64(wettest)) || math.IsNaN(float64(mean)) {
			continue
		}
		out.Regions[region.ID] = RainOutlook{Next6hMaxMm: round1(float64(wettest)), Next24hMeanMm: round1(float64(mean))}
	}
	return out
}

func round1(value float64) float64 { return math.Round(value*10) / 10 }
