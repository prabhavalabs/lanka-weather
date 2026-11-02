package overview

import (
	"context"
	"log/slog"
	"time"

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

// WarningSource serves official warnings.
type WarningSource interface {
	Overview(ctx context.Context, filter warnings.Filter) (*warnings.Overview, error)
	Landslides(ctx context.Context) (warnings.LandslideLayer, error)
}

// RiskSource serves the river estimate.
type RiskSource interface {
	Risk(ctx context.Context) (*riverrisk.RiskDocument, error)
}

// GaugeSource serves the river gauges.
type GaugeSource interface {
	Layer(ctx context.Context) (rivers.FeatureCollection, error)
}

// ReservoirSource serves the reservoirs.
type ReservoirSource interface {
	Layer(ctx context.Context) (reservoirs.FeatureCollection, error)
}

// HazardSource serves earthquakes and cyclones.
type HazardSource interface {
	Earthquakes(ctx context.Context) (hazards.EarthquakeLayer, error)
	Cyclones(ctx context.Context) (hazards.CycloneLayer, error)
}

// WeatherSource serves region weather.
type WeatherSource interface {
	RegionWeather(ctx context.Context, level, parent string) (*weather.RegionsResponse, error)
}

// RainSource serves measured 24-hour totals.
type RainSource interface {
	ObservedRainfall(ctx context.Context) (*stations.RainCollection, error)
}

// EnvironmentSource serves the air-quality and sea grids' extremes.
type EnvironmentSource interface {
	WorstAir(ctx context.Context) (*environment.Located[environment.AirPoint], error)
	HighestSea(ctx context.Context) (*environment.Located[environment.SeaPoint], error)
}

// RegionLookup names where a place is.
type RegionLookup interface {
	Lookup(ctx context.Context, lon, lat float64) (regions.Lookup, error)
}

// Service composes the overview. Any source may be nil, and any may fail: a part
// that cannot be read is said to be unread, as /v1/summary does.
type Service struct {
	Warnings    WarningSource
	Risk        RiskSource
	Gauges      GaugeSource
	Reservoirs  ReservoirSource
	Hazards     HazardSource
	Weather     WeatherSource
	Rain        RainSource
	Environment EnvironmentSource
	Regions     RegionLookup
	// WeatherStaleAfter is how old region weather may be.
	WeatherStaleAfter time.Duration
	Logger            *slog.Logger
	Now               func() time.Time
}

func (service *Service) warn(what string, err error) {
	if service.Logger != nil {
		service.Logger.Warn("overview input failed", "input", what, "error", err)
	}
}

// Overview reads every source and builds the answer. It does not fail: a source
// that fails leaves its part unavailable.
func (service *Service) Overview(ctx context.Context) (*Overview, error) {
	now := service.now()
	in := Inputs{}
	if service.Warnings != nil {
		if value, err := service.Warnings.Overview(ctx, warnings.Filter{Limit: 1}); err != nil {
			service.warn("warnings", err)
		} else {
			in.Warnings = value
		}
		if value, err := service.Warnings.Landslides(ctx); err != nil {
			service.warn("landslides", err)
		} else {
			in.Landslides = &value
		}
	}
	if service.Risk != nil {
		if value, err := service.Risk.Risk(ctx); err != nil {
			service.warn("river risk", err)
		} else {
			in.Risk = value
		}
	}
	if service.Gauges != nil {
		if value, err := service.Gauges.Layer(ctx); err != nil {
			service.warn("gauges", err)
		} else {
			in.Gauges = &value
		}
	}
	if service.Reservoirs != nil {
		if value, err := service.Reservoirs.Layer(ctx); err != nil {
			service.warn("reservoirs", err)
		} else {
			in.Reservoirs = &value
		}
	}
	if service.Hazards != nil {
		if value, err := service.Hazards.Cyclones(ctx); err != nil {
			service.warn("cyclones", err)
		} else {
			in.Cyclones = &value
		}
		if value, err := service.Hazards.Earthquakes(ctx); err != nil {
			service.warn("earthquakes", err)
		} else {
			in.Earthquakes = &value
		}
	}
	if service.Weather != nil {
		if value, err := service.Weather.RegionWeather(ctx, "district", ""); err != nil {
			service.warn("region weather", err)
		} else {
			in.Weather = value
			in.WeatherStale = value.FetchedAt == nil || (service.WeatherStaleAfter > 0 && now.Sub(*value.FetchedAt) > service.WeatherStaleAfter)
		}
	}
	if service.Rain != nil {
		if value, err := service.Rain.ObservedRainfall(ctx); err != nil {
			service.warn("gauge rainfall", err)
		} else {
			in.Rain = value
		}
	}
	if service.Environment != nil {
		if value, err := service.Environment.WorstAir(ctx); err != nil {
			service.warn("air quality", err)
		} else if value != nil {
			in.Air = &AirWorst{AirPoint: value.Value, Lon: value.Lon, Lat: value.Lat, Place: service.district(ctx, value.Lon, value.Lat)}
		}
		if value, err := service.Environment.HighestSea(ctx); err != nil {
			service.warn("sea conditions", err)
		} else if value != nil {
			in.Sea = &SeaHighest{SeaPoint: value.Value, Lon: value.Lon, Lat: value.Lat}
		}
	}
	return Build(in, now), nil
}

// district names the district a place is in, or "".
func (service *Service) district(ctx context.Context, lon, lat float64) string {
	if service.Regions == nil {
		return ""
	}
	lookup, err := service.Regions.Lookup(ctx, lon, lat)
	if err != nil {
		service.warn("region lookup", err)
		return ""
	}
	if lookup.District == nil {
		return ""
	}
	return lookup.District.Name + " District"
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
