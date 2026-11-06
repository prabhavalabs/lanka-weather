package floodrisk

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// Note says what the estimate is, wherever it is served.
const Note = "Our estimate from rain measured at the live gauges (lower bounds), rain forecast, Google Flood Hub, " +
	"river gauges, DMC and Department of Meteorology bulletins and spilling reservoirs. Not an official warning."

// recompute is how long an estimate is served before it is worked out again.
const recompute = time.Minute

// The sources the estimate reads: the services that serve the map.
type (
	RainSource interface {
		Intensities(ctx context.Context) (raingauges.Intensities, error)
	}
	FloodsSource interface {
		Layer(ctx context.Context) (floods.FeatureCollection, error)
	}
	RiversSource interface {
		Layer(ctx context.Context) (rivers.FeatureCollection, error)
	}
	WarningsSource interface {
		Overview(ctx context.Context, filter warnings.Filter) (*warnings.Overview, error)
	}
	ReservoirsSource interface {
		Layer(ctx context.Context) (reservoirs.FeatureCollection, error)
	}
	WeatherSource interface {
		RainOutlooks(ctx context.Context, level string) (*weather.RainOutlooks, error)
	}
)

// Service serves the estimate for every district.
type Service struct {
	Rain       RainSource
	Floods     FloodsSource
	Rivers     RiversSource
	Warnings   WarningsSource
	Reservoirs ReservoirsSource
	Weather    WeatherSource
	Locator    *Locator
	Logger     *slog.Logger
	Now        func() time.Time

	mu       sync.Mutex
	document *Document
	at       time.Time
}

// District is one district's estimate and what it rests on.
type District struct {
	Assessment
	Inputs Inputs `json:"inputs"`
}

// Document is GET /v1/flood-risk.
type Document struct {
	Districts []District `json:"districts"`
	Meta      Meta       `json:"meta"`
}

// Meta says how current the estimate is.
type Meta struct {
	Note        string    `json:"note"`
	GeneratedAt time.Time `json:"generatedAt"`
	// Complete is false when a source could not be read: Missing names them,
	// and their families say nothing.
	Complete bool     `json:"complete"`
	Missing  []string `json:"missing"`
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}

// Estimate returns every district's estimate, worked out at most once a minute.
func (service *Service) Estimate(ctx context.Context) (*Document, error) {
	now := service.now()
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.document != nil && now.Sub(service.at) < recompute {
		return service.document, nil
	}
	layers, missing := service.read(ctx)
	locator := service.Locator
	places := Places{
		DistrictOf:    func(lon, lat float64) string { return locator.DistrictOf(ctx, lon, lat) },
		AreaDistricts: func(id string) []string { return locator.AreaDistricts(ctx, id) },
		DistrictNamed: locator.Named(ctx),
	}
	document := &Document{Districts: []District{}, Meta: Meta{Note: Note, GeneratedAt: now, Complete: len(missing) == 0, Missing: missing}}
	for _, inputs := range Gather(layers, places, now) {
		document.Districts = append(document.Districts, District{Assessment: Assess(inputs), Inputs: inputs})
	}
	sort.Slice(document.Districts, func(i, j int) bool {
		a, b := document.Districts[i], document.Districts[j]
		if a.Level != b.Level {
			return a.Level > b.Level
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return a.District < b.District
	})
	service.document, service.at = document, now
	return document, nil
}

// read gathers the layers, naming those that failed.
func (service *Service) read(ctx context.Context) (Layers, []string) {
	layers := Layers{}
	missing := []string{}
	fail := func(what string, err error) {
		missing = append(missing, what)
		if service.Logger != nil {
			service.Logger.Warn("flood-risk source failed", "source", what, "error", err)
		}
	}
	if service.Rain != nil {
		if value, err := service.Rain.Intensities(ctx); err == nil {
			layers.Rain = &value
		} else {
			fail("rain-gauges", err)
		}
	}
	if service.Floods != nil {
		if value, err := service.Floods.Layer(ctx); err == nil {
			layers.Floods = &value
		} else {
			fail("flood-forecast", err)
		}
	}
	if service.Rivers != nil {
		if value, err := service.Rivers.Layer(ctx); err == nil {
			layers.Rivers = &value
		} else {
			fail("river-levels", err)
		}
	}
	if service.Warnings != nil {
		if value, err := service.Warnings.Overview(ctx, warnings.Filter{Limit: 1}); err == nil {
			layers.Bulletins = value
		} else {
			fail("official-warnings", err)
		}
	}
	if service.Reservoirs != nil {
		if value, err := service.Reservoirs.Layer(ctx); err == nil {
			layers.Reservoirs = &value
		} else {
			fail("reservoirs", err)
		}
	}
	if service.Weather != nil {
		if value, err := service.Weather.RainOutlooks(ctx, "district"); err == nil {
			layers.Outlook = value
		} else {
			fail("rain-forecast", err)
		}
	}
	return layers, missing
}
