package rivers

import (
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/store"
)

func TestNextHourConvertsToTheStationsUnit(t *testing.T) {
	issued := time.Date(2026, 11, 2, 18, 32, 0, 0, time.UTC)
	low, high := 5.79, 6.06
	row := store.RiverNextHour{IssuedAt: issued, LevelFt: 5.92, LowFt: &low, HighFt: &high}

	metres := nextHour(row, "m")
	if metres == nil || metres.Level != 1.8 || *metres.Low != 1.76 || *metres.High != 1.85 || metres.Unit != "m" || !metres.Experimental {
		t.Fatalf("metres = %+v", metres)
	}
	feet := nextHour(row, "ft")
	if feet == nil || feet.Level != 5.92 || *feet.Low != 5.79 || *feet.High != 6.06 || feet.Unit != "ft" {
		t.Fatalf("feet = %+v", feet)
	}
	if forecast := nextHour(row, "unknown"); forecast != nil {
		t.Fatalf("a station with no unit got a forecast: %+v", forecast)
	}
	// A trend estimate has no range.
	if forecast := nextHour(store.RiverNextHour{IssuedAt: issued, LevelFt: 3}, "m"); forecast.Low != nil || forecast.High != nil {
		t.Fatalf("range = %v %v", forecast.Low, forecast.High)
	}
	// Half a range is no range.
	if forecast := nextHour(store.RiverNextHour{IssuedAt: issued, LevelFt: 3, LowFt: &low}, "m"); forecast.Low != nil {
		t.Fatalf("half a range was kept: %+v", forecast)
	}
}

func TestAttributionNamesWhoMeasured(t *testing.T) {
	river := "RiverNet, DMC / Leecom SCADA (rivernet.lk)"
	cases := []struct {
		source   string
		upstream *string
		want     string
	}{
		{"irrigation-gauges", nil, "Irrigation Department of Sri Lanka"},
		{"waterlevels-lk-gauges", &river, "RiverNet, DMC / Leecom SCADA (rivernet.lk) via Sri Lanka Water Levels (waterlevels.lk)"},
		{"waterlevels-lk-gauges", nil, "Sri Lanka Water Levels (waterlevels.lk) (upstream agency not stated)"},
	}
	for _, test := range cases {
		if got := attribution(test.source, test.upstream); got != test.want {
			t.Errorf("attribution(%s) = %q, want %q", test.source, got, test.want)
		}
	}
}
