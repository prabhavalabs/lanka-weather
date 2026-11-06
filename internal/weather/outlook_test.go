package weather

import (
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
)

func TestOutlooksTakeTheWettestCellAheadAndTheAreaMean(t *testing.T) {
	snapshot := testSnapshot(testNow)
	lattice := snapshot.Lattice
	size := lattice.Size()
	from := testNow.Truncate(time.Hour).Add(-2 * time.Hour)
	steps := 48
	values := make([]float32, steps*size)
	for step := range steps {
		for node := range size {
			// 2 mm an hour everywhere; node 1 has a 10 mm/h burst in the next 3 hours.
			value := float32(2)
			if node == 1 && step >= 3 && step < 6 {
				value = 10
			}
			values[step*size+node] = value
		}
	}
	snapshot.Forecast.Fields[openmeteo.HourlyPrefix+"precipitation"] = &Field{ValidFrom: from, Step: time.Hour, Steps: steps, Values: values}
	regions := []Region{{ID: "LK-11", Nodes: []int{0, 1}, Weights: []float64{1, 1}}, {ID: "LK-99", Lon: 70, Lat: 7}}
	out := OutlooksFor(snapshot, regions, testNow)
	if out == nil || out.Stale {
		t.Fatalf("outlooks %+v", out)
	}
	colombo, ok := out.Regions["LK-11"]
	// Now is step 3 (09:40 is nearest 10:00): node 1 has 3 × 10 + 3 × 2 in the next 6 hours.
	if !ok || colombo.Next6hMaxMm != 36 {
		t.Fatalf("LK-11 %+v", colombo)
	}
	// 24 hours: node 0 has 48, node 1 has 30 + 21 × 2 = 72; the mean is 60.
	if colombo.Next24hMeanMm != 60 {
		t.Fatalf("LK-11 mean %+v", colombo)
	}
	if _, ok := out.Regions["LK-99"]; ok {
		t.Fatal("a region off the grid has no outlook")
	}
	snapshot.Forecast.FetchedAt = testNow.Add(-13 * time.Hour)
	if !OutlooksFor(snapshot, regions, testNow).Stale {
		t.Fatal("a 13-hour-old forecast is stale")
	}
}
