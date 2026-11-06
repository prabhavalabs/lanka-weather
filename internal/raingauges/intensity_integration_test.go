//go:build integration

package raingauges_test

import (
	"context"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

func TestIntensitiesReadEveryGaugesRiseAndFollowNewFiles(t *testing.T) {
	pool := dbtest.Open(t, "rain_gauge_readings", "rain_gauges")
	ctx := context.Background()
	queries := store.New(pool)
	at := func(hour, minute int) time.Time { return time.Date(2026, 11, 6, hour, minute, 0, 0, time.UTC) }
	save := func(fetched time.Time, points [][2]float64) {
		t.Helper()
		latest := points[len(points)-1]
		latestAt := at(int(latest[0])/60, int(latest[0])%60)
		if err := queries.UpsertRainGauge(ctx, store.UpsertRainGaugeParams{ID: "argcbo01", Device: "argcbo01", Name: "DOM HEAD OFFICE",
			Institution: "Department of Meteorology", Lon: 79.87, Lat: 6.9, LatestObservedAt: &latestAt, LatestRainMm: &latest[1],
			Source: "hmis-rain-live", SourceTime: fetched, FetchedAt: fetched}); err != nil {
			t.Fatal(err)
		}
		params := []store.InsertRainGaugeReadingParams{}
		for _, point := range points {
			params = append(params, store.InsertRainGaugeReadingParams{GaugeID: "argcbo01", ObservedAt: at(int(point[0])/60, int(point[0])%60),
				RainMm: point[1], Source: "hmis-rain-live", SourceTime: fetched, FetchedAt: fetched})
		}
		if err := queries.InsertRainGaugeReading(ctx, params).Close(); err != nil {
			t.Fatal(err)
		}
	}
	now := at(13, 5)
	service := &raingauges.Service{Pool: pool, StaleAfter: 3 * time.Hour, Now: func() time.Time { return now }}
	save(at(12, 33), [][2]float64{{11*60 + 50, 16}, {12*60 + 20, 26.5}})
	first, err := service.Intensities(ctx)
	if err != nil || len(first.Gauges) != 1 || first.Gauges[0].Rise1hMm != 10.5 {
		t.Fatalf("first %+v, %v", first, err)
	}
	// A new file brings the burst.
	save(at(13, 3), [][2]float64{{12*60 + 50, 83.5}})
	second, err := service.Intensities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	gauge := second.Gauges[0]
	if gauge.Rise1hMm != 67.5 || gauge.Rain24hMm != 83.5 || gauge.Stale || gauge.Suspect || !second.FetchedAt.Equal(at(13, 3)) {
		t.Fatalf("second %+v", second)
	}
	// Hours on with no new file, the gauge is out of date.
	now = at(17, 0)
	if third, _ := service.Intensities(ctx); !third.Gauges[0].Stale {
		t.Fatalf("third %+v", third)
	}
}
