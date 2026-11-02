package overview

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/environment"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

var errDown = errors.New("down")

type sources struct{ in Inputs }

func (s sources) Overview(context.Context, warnings.Filter) (*warnings.Overview, error) {
	return s.in.Warnings, nil
}
func (s sources) Landslides(context.Context) (warnings.LandslideLayer, error) {
	return *s.in.Landslides, nil
}
func (s sources) Risk(context.Context) (*riverrisk.RiskDocument, error) { return s.in.Risk, nil }
func (s sources) Earthquakes(context.Context) (hazards.EarthquakeLayer, error) {
	return *s.in.Earthquakes, nil
}
func (s sources) Cyclones(context.Context) (hazards.CycloneLayer, error) { return *s.in.Cyclones, nil }
func (s sources) RegionWeather(context.Context, string, string) (*weather.RegionsResponse, error) {
	return s.in.Weather, nil
}
func (s sources) ObservedRainfall(context.Context) (*stations.RainCollection, error) {
	return s.in.Rain, nil
}
func (s sources) WorstAir(context.Context) (*environment.Located[environment.AirPoint], error) {
	return &environment.Located[environment.AirPoint]{Value: s.in.Air.AirPoint, Lon: s.in.Air.Lon, Lat: s.in.Air.Lat}, nil
}
func (s sources) HighestSea(context.Context) (*environment.Located[environment.SeaPoint], error) {
	return &environment.Located[environment.SeaPoint]{Value: s.in.Sea.SeaPoint, Lon: s.in.Sea.Lon, Lat: s.in.Sea.Lat}, nil
}
func (s sources) Lookup(context.Context, float64, float64) (regions.Lookup, error) {
	return regions.Lookup{District: &regions.Ref{ID: "LK-11", Name: "Colombo"}}, nil
}

type gauges struct {
	layer *rivers.FeatureCollection
	err   error
}

func (g gauges) Layer(context.Context) (rivers.FeatureCollection, error) {
	if g.err != nil {
		return rivers.FeatureCollection{}, g.err
	}
	return *g.layer, nil
}

type reservoirLayer struct{ layer *reservoirs.FeatureCollection }

func (r reservoirLayer) Layer(context.Context) (reservoirs.FeatureCollection, error) {
	return *r.layer, nil
}

type failing struct{}

func (failing) Risk(context.Context) (*riverrisk.RiskDocument, error) { return nil, errDown }
func (failing) Overview(context.Context, warnings.Filter) (*warnings.Overview, error) {
	return nil, errDown
}
func (failing) Landslides(context.Context) (warnings.LandslideLayer, error) {
	return warnings.LandslideLayer{}, errDown
}

func fullService(in Inputs) *Service {
	s := sources{in: in}
	return &Service{Warnings: s, Risk: s, Gauges: gauges{layer: in.Gauges}, Reservoirs: reservoirLayer{layer: in.Reservoirs},
		Hazards: s, Weather: s, Rain: s, Environment: s, Regions: s}
}

func TestServiceComposesEverySource(t *testing.T) {
	got, err := fullService(quiet()).Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Unavailable) != 0 || got.Status != "No river gauge above its alert level." {
		t.Fatalf("overview = %+v", got)
	}
	if title := section(t, got, SectionAir).Items[0].Title; !strings.Contains(title, "Colombo District") {
		t.Fatalf("air = %q", title)
	}
}

func TestServiceToleratesAnySourceFailing(t *testing.T) {
	service := fullService(quiet())
	service.Warnings, service.Risk, service.Gauges = failing{}, failing{}, gauges{err: errDown}
	got, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("a failing source should not fail the overview: %v", err)
	}
	joined := strings.Join(got.Unavailable, ",")
	for _, want := range []string{"landslide levels", "official warnings", "river estimates", "river gauges"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("unavailable = %v", got.Unavailable)
		}
	}
	if !section(t, got, SectionReservoirs).Available {
		t.Fatal("the other parts should still answer")
	}
	if strings.Contains(got.Status, "No river gauge") {
		t.Fatalf("no calm statement without gauges: %q", got.Status)
	}
}

func TestServiceWithNoSourcesAtAllStillAnswers(t *testing.T) {
	got, err := (&Service{}).Overview(context.Background())
	if err != nil || len(got.Unavailable) != len(got.Sections) {
		t.Fatalf("overview = %+v, %v", got, err)
	}
}
