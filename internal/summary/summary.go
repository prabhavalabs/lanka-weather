// Package summary answers "what is happening here?" for one place: the
// weather and a short forecast, the nearest river gauges and Google flood
// forecasts, and alerts derived from them. Derived alerts are estimates;
// official warnings, when they are collected, go in their own slot.
package summary

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// WeatherSource interpolates the weather grid at a point.
type WeatherSource interface {
	Point(ctx context.Context, lon, lat float64, hours, days int) (*weather.Point, error)
}

// RiverSource serves the river-levels layer.
type RiverSource interface {
	Layer(ctx context.Context) (rivers.FeatureCollection, error)
}

// FloodSource serves the Google flood forecasts.
type FloodSource interface {
	Gauges(ctx context.Context) ([]floods.Gauge, error)
	AreasAt(ctx context.Context, lon, lat float64) ([]floods.AreaAt, error)
}

// RegionLookup finds the administrative regions containing a point.
type RegionLookup interface {
	Lookup(ctx context.Context, lon, lat float64) (regions.Lookup, error)
}

// Service builds summaries. Any source may be nil.
type Service struct {
	Weather WeatherSource
	Rivers  RiverSource
	Floods  FloodSource
	Regions RegionLookup
	Logger  *slog.Logger
	Now     func() time.Time
}

const (
	// Short forecast in a summary.
	summaryHours = 24
	summaryDays  = 3
	// nearbyRivers and nearbyFloods bound the gauges listed, within
	// riverSearchKm and floodSearchKm.
	nearbyRivers  = 3
	nearbyFloods  = 3
	riverSearchKm = 50.0
	floodSearchKm = 75.0
)

// Summary is GET /v1/summary.
type Summary struct {
	Location    weather.Location `json:"location"`
	GeneratedAt time.Time        `json:"generatedAt"`
	// Regions are the province, district, DS and GN division containing
	// the place; null outside Sri Lanka or before regions are loaded.
	Regions *Regions `json:"regions"`
	// Weather is the current conditions, the next 24 hours and three days;
	// null when the place is outside the grid or nothing is collected.
	Weather       *weather.Point `json:"weather"`
	Rivers        []River        `json:"rivers"`
	FloodForecast FloodForecast  `json:"floodForecast"`
	// Alerts are derived by lanka-weather and are estimates.
	Alerts []Alert `json:"alerts"`
	// Official will hold Department of Meteorology and DMC warnings, which
	// are authoritative; they are not collected yet.
	Official Official `json:"official"`
	// Unavailable lists the parts that could not be filled: weather,
	// regions, rivers or floodForecast.
	Unavailable []string `json:"unavailable"`
}

// Regions is the administrative hierarchy of the place. A level is null
// where no region of it contains the place (at sea, for example).
type Regions struct {
	Province *regions.Ref `json:"province"`
	District *regions.Ref `json:"district"`
	DS       *regions.Ref `json:"ds"`
	GN       *regions.Ref `json:"gn"`
}

// River is a river gauge near the place.
type River struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Basin           string     `json:"basin"`
	DistanceKm      float64    `json:"distanceKm"`
	Status          string     `json:"status"`
	Trend           string     `json:"trend"`
	WaterLevel      *float64   `json:"waterLevel"`
	Unit            string     `json:"unit"`
	AlertLevel      *float64   `json:"alertLevel"`
	MinorFloodLevel *float64   `json:"minorFloodLevel"`
	MajorFloodLevel *float64   `json:"majorFloodLevel"`
	ObservedAt      *time.Time `json:"observedAt"`
	Stale           bool       `json:"stale"`
	Lon             float64    `json:"lon"`
	Lat             float64    `json:"lat"`
}

// FloodForecast is the Google flood forecast near the place.
type FloodForecast struct {
	Gauges []FloodGauge    `json:"gauges"`
	Areas  []floods.AreaAt `json:"areas"`
}

// FloodGauge is a Google flood-forecast gauge near the place.
type FloodGauge struct {
	GaugeID         string     `json:"gaugeId"`
	DistanceKm      float64    `json:"distanceKm"`
	Severity        string     `json:"severity"`
	Trend           string     `json:"trend"`
	IssuedAt        time.Time  `json:"issuedAt"`
	ForecastStart   *time.Time `json:"forecastStart"`
	ForecastEnd     *time.Time `json:"forecastEnd"`
	QualityVerified bool       `json:"qualityVerified"`
	Lon             float64    `json:"lon"`
	Lat             float64    `json:"lat"`
}

// Official is the slot for authoritative warnings.
type Official struct {
	Status   string `json:"status"`
	Warnings []any  `json:"warnings"`
	Note     string `json:"note"`
}

// Summary builds the summary for a place. A failing source leaves its part
// empty and is listed in Unavailable; the summary itself does not fail.
func (service *Service) Summary(ctx context.Context, lon, lat float64) (*Summary, error) {
	summary := &Summary{
		Location: weather.Location{Lat: lat, Lon: lon}, GeneratedAt: service.now(),
		Rivers: []River{}, FloodForecast: FloodForecast{Gauges: []FloodGauge{}, Areas: []floods.AreaAt{}},
		Official: Official{Status: "not_available", Warnings: []any{},
			Note: "Department of Meteorology and Disaster Management Centre warnings are authoritative; they will appear here once collected."},
		Unavailable: []string{},
	}
	inputs := Inputs{}

	if service.Weather != nil {
		point, err := service.Weather.Point(ctx, lon, lat, summaryHours, summaryDays)
		switch {
		case err == nil:
			summary.Weather = point
			inputs.Hours, inputs.Days = hours(point), days(point)
		case errors.Is(err, weather.ErrOutsideGrid) || errors.Is(err, weather.ErrNoData):
			summary.Unavailable = append(summary.Unavailable, "weather")
		default:
			service.warn("summary weather", err)
			summary.Unavailable = append(summary.Unavailable, "weather")
		}
	} else {
		summary.Unavailable = append(summary.Unavailable, "weather")
	}

	if service.Regions != nil {
		lookup, err := service.Regions.Lookup(ctx, lon, lat)
		switch {
		case err != nil:
			service.warn("summary regions", err)
			summary.Unavailable = append(summary.Unavailable, "regions")
		case lookup.Province != nil || lookup.District != nil || lookup.DS != nil || lookup.GN != nil:
			summary.Regions = &Regions{Province: lookup.Province, District: lookup.District, DS: lookup.DS, GN: lookup.GN}
		}
	} else {
		summary.Unavailable = append(summary.Unavailable, "regions")
	}

	if service.Rivers != nil {
		layer, err := service.Rivers.Layer(ctx)
		if err != nil {
			service.warn("summary rivers", err)
			summary.Unavailable = append(summary.Unavailable, "rivers")
		} else {
			summary.Rivers = nearestRivers(layer, lon, lat)
			for _, river := range summary.Rivers {
				inputs.Rivers = append(inputs.Rivers, NearbyRiver{Name: river.Name, DistanceKm: river.DistanceKm, Status: river.Status, Stale: river.Stale})
			}
		}
	} else {
		summary.Unavailable = append(summary.Unavailable, "rivers")
	}

	if service.Floods != nil {
		gauges, err := service.Floods.Gauges(ctx)
		var areas []floods.AreaAt
		if err == nil {
			areas, err = service.Floods.AreasAt(ctx, lon, lat)
		}
		if err != nil {
			service.warn("summary flood forecast", err)
			summary.Unavailable = append(summary.Unavailable, "floodForecast")
		} else {
			summary.FloodForecast.Gauges = nearestFloods(gauges, lon, lat)
			summary.FloodForecast.Areas = areas
			for _, gauge := range summary.FloodForecast.Gauges {
				inputs.Floods = append(inputs.Floods, NearbyFloodGauge{GaugeID: gauge.GaugeID, DistanceKm: gauge.DistanceKm,
					Severity: gauge.Severity, ForecastStart: gauge.ForecastStart, ForecastEnd: gauge.ForecastEnd})
			}
			for _, area := range areas {
				inputs.Areas = append(inputs.Areas, FloodArea{Kind: area.Kind, Severity: area.Severity, ValidFrom: area.ValidFrom, ValidUntil: area.ValidUntil})
			}
		}
	} else {
		summary.Unavailable = append(summary.Unavailable, "floodForecast")
	}

	summary.Alerts = Evaluate(inputs)
	return summary, nil
}

// hours reads the point's hourly forecast for the alert rules.
func hours(point *weather.Point) []Hour {
	if point.Hourly == nil {
		return nil
	}
	times, _ := point.Hourly["times"].([]time.Time)
	series := func(name string) []float64 { return values(point.Hourly, name, len(times)) }
	rain, temperature, humidity := series("precipitationMm"), series("temperatureC"), series("relativeHumidityPct")
	speed, gusts := series("windSpeedKmh"), series("windGustsKmh")
	out := make([]Hour, len(times))
	for index, at := range times {
		out[index] = Hour{At: at, PrecipitationMm: rain[index], TemperatureC: temperature[index], HumidityPct: humidity[index],
			WindSpeedKmh: speed[index], WindGustsKmh: gusts[index]}
	}
	return out
}

// days reads the point's daily forecast for the alert rules.
func days(point *weather.Point) []Day {
	if point.Daily == nil {
		return nil
	}
	dates, _ := point.Daily["dates"].([]string)
	rain := values(point.Daily, "precipitationSumMm", len(dates))
	speed := values(point.Daily, "windSpeedMaxKmh", len(dates))
	gusts := values(point.Daily, "windGustsMaxKmh", len(dates))
	out := make([]Day, 0, len(dates))
	for index, date := range dates {
		day, err := time.Parse(time.DateOnly, date)
		if err != nil {
			continue
		}
		start := day.Add(-5*time.Hour - 30*time.Minute) // local midnight in UTC
		out = append(out, Day{Start: start, PrecipitationMm: rain[index], WindSpeedKmh: speed[index], WindGustsKmh: gusts[index]})
	}
	return out
}

func values(series map[string]any, name string, count int) []float64 {
	out := make([]float64, count)
	list, _ := series[name].(weather.Values)
	for index := range out {
		out[index] = math.NaN()
		if index < len(list) {
			out[index] = float64(list[index])
		}
	}
	return out
}

func nearestRivers(layer rivers.FeatureCollection, lon, lat float64) []River {
	var out []River
	for _, feature := range layer.Features {
		distance := DistanceKm(lon, lat, feature.Geometry.Coordinates[0], feature.Geometry.Coordinates[1])
		if distance > riverSearchKm {
			continue
		}
		properties := feature.Properties
		out = append(out, River{
			ID: properties.ID, Name: properties.Name, Basin: properties.Basin, DistanceKm: round1(distance),
			Status: string(properties.Status), Trend: string(properties.Trend), WaterLevel: properties.WaterLevel,
			Unit: properties.Unit, AlertLevel: properties.AlertLevel, MinorFloodLevel: properties.MinorFloodLevel,
			MajorFloodLevel: properties.MajorFloodLevel, ObservedAt: properties.ObservedAt, Stale: properties.Stale,
			Lon: feature.Geometry.Coordinates[0], Lat: feature.Geometry.Coordinates[1],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DistanceKm < out[j].DistanceKm })
	if len(out) > nearbyRivers {
		out = out[:nearbyRivers]
	}
	if out == nil {
		out = []River{}
	}
	return out
}

func nearestFloods(gauges []floods.Gauge, lon, lat float64) []FloodGauge {
	var out []FloodGauge
	for _, gauge := range gauges {
		distance := DistanceKm(lon, lat, gauge.Lon, gauge.Lat)
		if distance > floodSearchKm {
			continue
		}
		out = append(out, FloodGauge{
			GaugeID: gauge.GaugeID, DistanceKm: round1(distance), Severity: gauge.Severity, Trend: gauge.Trend,
			IssuedAt: gauge.IssuedAt, ForecastStart: gauge.ForecastStart, ForecastEnd: gauge.ForecastEnd,
			QualityVerified: gauge.QualityVerified, Lon: gauge.Lon, Lat: gauge.Lat,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DistanceKm < out[j].DistanceKm })
	if len(out) > nearbyFloods {
		out = out[:nearbyFloods]
	}
	if out == nil {
		out = []FloodGauge{}
	}
	return out
}

// DistanceKm is the great-circle distance between two points.
func DistanceKm(lon1, lat1, lon2, lat2 float64) float64 {
	const earthRadiusKm = 6371.0088
	toRadians := math.Pi / 180
	dLat, dLon := (lat2-lat1)*toRadians, (lon2-lon1)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

func round1(value float64) float64 { return math.Round(value*10) / 10 }

func (service *Service) warn(what string, err error) {
	if service.Logger != nil {
		service.Logger.Warn("summary part failed", "part", what, "error", err)
	}
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
