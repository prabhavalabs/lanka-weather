package summary

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

type fakeWeather struct{ err error }

func (fake fakeWeather) Point(_ context.Context, lon, lat float64, hours, _ int) (*weather.Point, error) {
	if fake.err != nil {
		return nil, fake.err
	}
	times := make([]time.Time, hours)
	rain := make(weather.Values, hours)
	for index := range times {
		times[index] = start.Add(time.Duration(index) * time.Hour)
		rain[index] = 3 // 72 mm in 24 hours
	}
	return &weather.Point{Location: weather.Location{Lat: lat, Lon: lon}, Hourly: map[string]any{"times": times, "precipitationMm": rain}}, nil
}

type fakeRivers struct{}

func (fakeRivers) Layer(context.Context) (rivers.FeatureCollection, error) {
	station := func(id string, lon, lat float64, status rivers.Status) rivers.Feature {
		return rivers.Feature{ID: id, Geometry: rivers.Point{Type: "Point", Coordinates: [2]float64{lon, lat}},
			Properties: rivers.StationProperties{ID: id, Name: id, Status: status}}
	}
	return rivers.FeatureCollection{Features: []rivers.Feature{
		station("nagalagam-street", 79.879, 6.959, rivers.StatusAlert),
		station("hanwella", 80.08, 6.91, rivers.StatusNormal),
		station("glencourse", 80.18, 6.98, rivers.StatusNormal),
		station("kitulgala", 80.42, 6.99, rivers.StatusNormal),
		station("trincomalee", 81.2, 8.6, rivers.StatusMajorFlood),
	}}, nil
}

type fakeFloods struct{ err error }

func (fake fakeFloods) Gauges(context.Context) ([]floods.Gauge, error) {
	return []floods.Gauge{{GaugeID: "hybas_near", Severity: "no_flooding", Lon: 80.0, Lat: 7.0}, {GaugeID: "hybas_far", Severity: "extreme", Lon: 81.3, Lat: 7.9}}, fake.err
}

func (fake fakeFloods) AreasAt(context.Context, float64, float64) ([]floods.AreaAt, error) {
	return nil, nil
}

func TestSummaryForColombo(t *testing.T) {
	service := &Service{Weather: fakeWeather{}, Rivers: fakeRivers{}, Floods: fakeFloods{}, Now: func() time.Time { return start }}
	summary, err := service.Summary(context.Background(), 79.8612, 6.9271)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Weather == nil || len(summary.Unavailable) != 0 {
		t.Fatalf("unavailable = %v", summary.Unavailable)
	}
	if len(summary.Rivers) != 3 || summary.Rivers[0].ID != "nagalagam-street" || summary.Rivers[0].DistanceKm > 5 {
		t.Fatalf("rivers = %+v", summary.Rivers)
	}
	if len(summary.FloodForecast.Gauges) != 1 || summary.FloodForecast.Gauges[0].GaugeID != "hybas_near" {
		t.Fatalf("flood gauges = %+v", summary.FloodForecast.Gauges)
	}
	if find(summary.Alerts, "heavy_rain") == nil || find(summary.Alerts, "river_level") == nil {
		t.Fatalf("alerts = %+v", summary.Alerts)
	}
	if summary.Official.Status != "not_available" || summary.Official.Warnings == nil {
		t.Fatalf("official = %+v", summary.Official)
	}
}

func TestSummaryListsUnavailableParts(t *testing.T) {
	service := &Service{Weather: fakeWeather{err: weather.ErrOutsideGrid}, Floods: fakeFloods{err: errors.New("down")}}
	summary, err := service.Summary(context.Background(), 70, 10)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Weather != nil || len(summary.Unavailable) != 3 || summary.Rivers == nil || summary.Alerts == nil {
		t.Fatalf("summary = %+v", summary)
	}
}
