package summary

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/environment"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
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
	return []floods.Gauge{{QualityVerified: true, GaugeID: "hybas_near", Severity: "no_flooding", Lon: 80.0, Lat: 7.0}, {GaugeID: "hybas_far", Severity: "extreme", Lon: 81.3, Lat: 7.9}}, fake.err
}

func (fake fakeFloods) AreasAt(context.Context, float64, float64) ([]floods.AreaAt, error) {
	return nil, nil
}

type fakeRegions struct{ err error }

func (fake fakeRegions) Lookup(_ context.Context, lon, lat float64) (regions.Lookup, error) {
	if fake.err != nil || lon < 79 {
		return regions.Lookup{Lat: lat, Lon: lon}, fake.err
	}
	return regions.Lookup{Lat: lat, Lon: lon,
		Province: &regions.Ref{ID: "LK-1", Level: regions.LevelProvince, Name: "Western"},
		District: &regions.Ref{ID: "LK-11", Level: regions.LevelDistrict, Name: "Colombo"},
		DS:       &regions.Ref{ID: "LK-1103", Level: regions.LevelDS, Name: "Colombo"},
		GN:       &regions.Ref{ID: "LK-1103005", Level: regions.LevelGN, Name: "Fort"},
	}, nil
}

type fakeOfficial struct {
	err                error
	district, division *string
}

func (fake fakeOfficial) Official(_ context.Context, districtID, dsID string) (*warnings.OfficialAt, error) {
	if fake.err != nil {
		return nil, fake.err
	}
	*fake.district, *fake.division = districtID, dsID
	return &warnings.OfficialAt{Status: "available", Warnings: []warnings.Bulletin{{ID: "bulletin"}}, General: []warnings.Bulletin{}}, nil
}

func TestSummaryForColombo(t *testing.T) {
	var district, division string
	service := &Service{Weather: fakeWeather{}, Rivers: fakeRivers{}, Floods: fakeFloods{}, Regions: fakeRegions{},
		Official: fakeOfficial{district: &district, division: &division}, Now: func() time.Time { return start }}
	summary, err := service.Summary(context.Background(), 79.8612, 6.9271)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Weather == nil || len(summary.Unavailable) != 0 {
		t.Fatalf("unavailable = %v", summary.Unavailable)
	}
	if summary.Regions == nil || summary.Regions.District.ID != "LK-11" || summary.Regions.GN.Name != "Fort" {
		t.Fatalf("regions = %+v", summary.Regions)
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
	if summary.Official.Status != "available" || len(summary.Official.Warnings) != 1 || district != "LK-11" || division != "LK-1103" {
		t.Fatalf("official = %+v for %q %q", summary.Official, district, division)
	}
}

func TestSummarySaysWhenOfficialWarningsCannotBeRead(t *testing.T) {
	service := &Service{Official: fakeOfficial{err: errors.New("down")}}
	summary, err := service.Summary(context.Background(), 79.8612, 6.9271)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Official.Status != "not_available" || !summary.Official.Stale || summary.Official.Warnings == nil ||
		!slices.Contains(summary.Unavailable, "official") {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestSummaryListsUnavailableParts(t *testing.T) {
	service := &Service{Weather: fakeWeather{err: weather.ErrOutsideGrid}, Floods: fakeFloods{err: errors.New("down")}}
	summary, err := service.Summary(context.Background(), 70, 10)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Weather != nil || len(summary.Unavailable) != 5 || summary.Rivers == nil || summary.Alerts == nil {
		t.Fatalf("summary = %+v", summary)
	}

	// At sea the lookup finds nothing: no hierarchy, but nothing failed.
	service = &Service{Regions: fakeRegions{}}
	if summary, _ := service.Summary(context.Background(), 70, 10); summary.Regions != nil || slices.Contains(summary.Unavailable, "regions") {
		t.Fatalf("at sea = %+v, unavailable %v", summary.Regions, summary.Unavailable)
	}
	service = &Service{Regions: fakeRegions{err: errors.New("down")}}
	if summary, _ := service.Summary(context.Background(), 80, 7); summary.Regions != nil || !slices.Contains(summary.Unavailable, "regions") {
		t.Fatalf("failed lookup = %+v, unavailable %v", summary.Regions, summary.Unavailable)
	}
}

type fakeRiverRisk struct {
	rivers []riverrisk.NearbyRiver
	err    error
}

func (fake fakeRiverRisk) Near(context.Context, float64, float64) ([]riverrisk.NearbyRiver, error) {
	return fake.rivers, fake.err
}

func TestSummaryListsNearbyRiversWithTheirEstimatedRisk(t *testing.T) {
	service := &Service{RiverRisk: fakeRiverRisk{rivers: []riverrisk.NearbyRiver{
		{ID: "kelani-ganga", Name: "Kelani Ganga", Level: riverrisk.LevelWatch, DistanceKm: 2.1, Estimate: true},
	}}}
	summary, err := service.Summary(context.Background(), 79.88, 6.95)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.NearbyRivers) != 1 || summary.NearbyRivers[0].ID != "kelani-ganga" || !summary.NearbyRivers[0].Estimate {
		t.Fatalf("nearby = %+v", summary.NearbyRivers)
	}
}

func TestSummarySurvivesARiverRiskFailure(t *testing.T) {
	service := &Service{RiverRisk: fakeRiverRisk{err: errors.New("database is down")}}
	summary, err := service.Summary(context.Background(), 79.88, 6.95)
	if err != nil {
		t.Fatal(err)
	}
	if summary.NearbyRivers == nil || len(summary.NearbyRivers) != 0 {
		t.Fatalf("nearby = %+v", summary.NearbyRivers)
	}
	found := false
	for _, part := range summary.Unavailable {
		found = found || part == "nearbyRivers"
	}
	if !found {
		t.Fatalf("unavailable = %v", summary.Unavailable)
	}
}

type fakeStations struct{ err error }

func (fake fakeStations) Nearest(context.Context, float64, float64) (*stations.Nearby, error) {
	if fake.err != nil {
		return nil, fake.err
	}
	return &stations.Nearby{Properties: stations.Properties{ID: "colombo", Stale: true}, DistanceKm: 3}, nil
}

type fakeEnvironment struct{ err error }

func (fake fakeEnvironment) AirAt(context.Context, float64, float64) (*environment.AirPoint, error) {
	if fake.err != nil {
		return nil, fake.err
	}
	return &environment.AirPoint{USAQI: 60, Category: environment.AQIModerate, Estimate: true}, nil
}

func (fake fakeEnvironment) SeaAt(_ context.Context, lon, _ float64) (*environment.SeaPoint, error) {
	if lon > 80.5 {
		return nil, nil
	}
	return &environment.SeaPoint{WaveHeightM: 1.2, State: "slight", Estimate: true}, nil
}

func TestSummaryAddsObservedAirAndSea(t *testing.T) {
	service := &Service{Stations: fakeStations{}, Environment: fakeEnvironment{}, Now: func() time.Time { return start }}
	summary, _ := service.Summary(context.Background(), 79.86, 6.92)
	if summary.Observed == nil || summary.Observed.ID != "colombo" || !summary.Observed.Stale {
		t.Fatalf("observed = %+v", summary.Observed)
	}
	if summary.AirQuality == nil || summary.Sea == nil || summary.Sea.State != "slight" {
		t.Fatalf("air = %+v, sea = %+v", summary.AirQuality, summary.Sea)
	}
	inland, _ := service.Summary(context.Background(), 80.77, 7.0)
	if inland.Sea != nil {
		t.Fatalf("no sea inland: %+v", inland.Sea)
	}
	if slices.Contains(inland.Unavailable, "sea") {
		t.Fatal("being inland is not a failure")
	}
}

func TestSummaryListsFailedObservedAndAirParts(t *testing.T) {
	failure := errors.New("down")
	service := &Service{Stations: fakeStations{err: failure}, Environment: fakeEnvironment{err: failure}}
	summary, err := service.Summary(context.Background(), 79.86, 6.92)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(summary.Unavailable, "observed") || !slices.Contains(summary.Unavailable, "airQuality") {
		t.Fatalf("unavailable = %v", summary.Unavailable)
	}
}

func TestSummaryListsOnlyCurrentNearbyReservoirsSpillingOrReleasing(t *testing.T) {
	point := func(lon, lat float64) *reservoirs.Point {
		return &reservoirs.Point{Type: "Point", Coordinates: [2]float64{lon, lat}}
	}
	feature := func(id string, geometry *reservoirs.Point, status string, stale bool) reservoirs.Feature {
		return reservoirs.Feature{ID: id, Geometry: geometry, Properties: reservoirs.Properties{ID: id, Name: id, Status: status, Stale: stale, Mapped: geometry != nil}}
	}
	service := &Service{Reservoirs: fakeReservoirs{layer: reservoirs.FeatureCollection{Features: []reservoirs.Feature{
		feature("further", point(80.2, 7.0), "releasing", false),
		feature("nearest", point(80.0, 7.0), "spilling", false),
		feature("calm", point(80.0, 7.0), "holding", false),
		feature("no-news", point(80.0, 7.0), "nodata", false),
		feature("old-news", point(80.0, 7.0), "spilling", true),
		feature("not-mapped", nil, "spilling", false),
		feature("far-away", point(81.5, 8.5), "spilling", false),
	}}}}
	summary, err := service.Summary(context.Background(), 80.0, 7.01)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Reservoirs) != 2 || summary.Reservoirs[0].ID != "nearest" || summary.Reservoirs[1].ID != "further" {
		t.Fatalf("reservoirs = %+v", summary.Reservoirs)
	}
	if slices.Contains(summary.Unavailable, "reservoirs") {
		t.Fatalf("unavailable = %v", summary.Unavailable)
	}

	failing := &Service{Reservoirs: fakeReservoirs{err: errors.New("down")}}
	summary, err = failing.Summary(context.Background(), 80.0, 7.01)
	if err != nil || !slices.Contains(summary.Unavailable, "reservoirs") || len(summary.Reservoirs) != 0 {
		t.Fatalf("summary = %+v, err %v", summary, err)
	}
}

// A gauge read through another site is credited to whoever measured it.
func TestRiverAlertBasisNamesTheSource(t *testing.T) {
	alerts := riverAlerts([]NearbyRiver{
		{Name: "Pugoda", DistanceKm: 3, Status: "alert", Source: "waterlevels-lk-gauges", Attribution: "RiverNet via waterlevels.lk"},
		{Name: "Hanwella", DistanceKm: 5, Status: "alert", Source: "irrigation-gauges", Attribution: "Irrigation Department of Sri Lanka"},
	})
	if len(alerts) != 2 || alerts[0].Basis != "RiverNet via waterlevels.lk gauge reading and the station's published thresholds" || alerts[1].Basis != basisRivers {
		t.Fatalf("alerts = %+v", alerts)
	}
}

type fakeReservoirs struct {
	layer reservoirs.FeatureCollection
	err   error
}

func (fake fakeReservoirs) Layer(context.Context) (reservoirs.FeatureCollection, error) {
	return fake.layer, fake.err
}

func TestLowerConfidenceGaugesRaiseNoAlertAndAreNotListed(t *testing.T) {
	near := []floods.Gauge{
		{GaugeID: "hybas_unverified", Severity: "extreme", QualityVerified: false, Lon: 79.87, Lat: 6.93},
		{GaugeID: "hybas_verified", Severity: "no_flooding", QualityVerified: true, Lon: 79.9, Lat: 6.95},
	}
	listed := nearestFloods(near, 79.87, 6.93)
	if len(listed) != 1 || listed[0].GaugeID != "hybas_verified" {
		t.Fatalf("nearest = %+v", listed)
	}
}
