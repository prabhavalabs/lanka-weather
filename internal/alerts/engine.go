package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/floodrisk"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/store"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// SourceName is the engine's name in source_runs.
const SourceName = "push-alerts"

// Interval is how often the engine runs: within a minute of data landing.
const Interval = 30 * time.Second

// Retention keeps events and what was sent for 30 days.
const Retention = 30 * 24 * time.Hour

// The services the engine reads, the same ones that serve the map, so an
// alert never disagrees with the panel it opens.
type (
	WarningsSource interface {
		Overview(ctx context.Context, filter warnings.Filter) (*warnings.Overview, error)
		Landslides(ctx context.Context) (warnings.LandslideLayer, error)
		LandslideWarning(ctx context.Context) (*warnings.Bulletin, error)
	}
	RiversSource interface {
		Layer(ctx context.Context) (rivers.FeatureCollection, error)
	}
	FloodsSource interface {
		Layer(ctx context.Context) (floods.FeatureCollection, error)
	}
	HazardsSource interface {
		Earthquakes(ctx context.Context) (hazards.EarthquakeLayer, error)
		Cyclones(ctx context.Context) (hazards.CycloneLayer, error)
	}
	ReservoirsSource interface {
		Layer(ctx context.Context) (reservoirs.FeatureCollection, error)
	}
	RainGaugesSource interface {
		Intensities(ctx context.Context) (raingauges.Intensities, error)
	}
	WeatherSource interface {
		RainOutlooks(ctx context.Context, level string) (*weather.RainOutlooks, error)
	}
)

// Engine raises events from our own tables and has them sent. It is a
// collector, so the scheduler runs it only in the worker holding the lock.
type Engine struct {
	Pool       *pgxpool.Pool
	Warnings   WarningsSource
	Rivers     RiversSource
	Floods     FloodsSource
	Hazards    HazardsSource
	Reservoirs ReservoirsSource
	RainGauges RainGaugesSource
	// Weather gives the rain forecast ahead, for the combined estimate.
	Weather    WeatherSource
	Dispatcher *push.Dispatcher
	Logger     *slog.Logger
	// Locator places gauges and areas in districts; one is made if nil.
	Locator *floodrisk.Locator

	once sync.Once
}

// Name implements source.Collector.
func (engine *Engine) Name() string { return SourceName }

// Interval implements source.Collector.
func (engine *Engine) Interval() time.Duration { return Interval }

// LongRunTimeout makes the scheduler run the engine beside the collectors, so
// a slow download never delays an alert.
func (engine *Engine) LongRunTimeout() time.Duration { return 2 * time.Minute }

// RetryAfter is the interval: a failed run is retried at the next tick.
func (engine *Engine) RetryAfter() time.Duration { return Interval }

// Collect runs the engine once and returns the number of notices sent.
func (engine *Engine) Collect(ctx context.Context, now time.Time) (int, error) {
	raised, err := engine.Detect(ctx, now)
	if err != nil {
		return 0, err
	}
	if engine.Dispatcher == nil {
		return 0, nil
	}
	stats, err := engine.Dispatcher.Run(ctx)
	if err != nil {
		return 0, err
	}
	if raised > 0 || stats.Notices > 0 || stats.Gone > 0 || stats.Dropped > 0 {
		engine.logger().Info("push alerts", "raised", raised, "events", stats.Events, "subscribers", stats.Subscribers,
			"notices", stats.Notices, "sent", stats.Sent, "failed", stats.Failed, "gone", stats.Gone, "held", stats.Held,
			"backing_off", stats.BackingOff, "dropped", stats.Dropped)
	}
	return stats.Sent, nil
}

// Prune applies the retention limit.
func (engine *Engine) Prune(ctx context.Context, now time.Time) error {
	_, err := store.New(engine.Pool).DeletePushEventsBefore(ctx, now.Add(-Retention))
	return err
}

func (engine *Engine) logger() *slog.Logger {
	if engine.Logger != nil {
		return engine.Logger
	}
	return slog.Default()
}

// Detect reads every source, applies hysteresis and stores the events raised.
// A source that fails is skipped and its subjects keep their state.
func (engine *Engine) Detect(ctx context.Context, now time.Time) (int, error) {
	queries := store.New(engine.Pool)
	rows, err := queries.ListPushAlertState(ctx)
	if err != nil {
		return 0, fmt.Errorf("read alert state: %w", err)
	}
	states := make(map[string]State, len(rows))
	for _, row := range rows {
		states[row.Subject] = State{Subject: row.Subject, Kind: row.Kind, Level: int(row.Level), AlertedLevel: int(row.AlertedLevel),
			AlertedAt: row.AlertedAt, BelowSince: row.BelowSince, UpdatedAt: row.UpdatedAt}
	}
	raised := 0
	for _, reading := range engine.read(ctx, now) {
		count, err := engine.apply(ctx, reading, states, now)
		if err != nil {
			return raised, err
		}
		raised += count
	}
	return raised, nil
}

// read gathers what each source says now, and the combined flood-risk
// estimate from the same layers.
func (engine *Engine) read(ctx context.Context, now time.Time) []Reading {
	readings := []Reading{}
	layers := floodrisk.Layers{}
	fail := func(what string, err error) {
		engine.logger().Warn("alert source failed", "source", what, "error", err)
	}
	places := engine.places(ctx)
	if engine.Warnings != nil {
		if overview, err := engine.Warnings.Overview(ctx, warnings.Filter{Limit: 1}); err == nil {
			readings = append(readings, WithBasinDistricts(Bulletins(overview, now), places.BasinDistricts))
			layers.Bulletins = overview
		} else {
			fail("bulletins", err)
		}
		if layer, err := engine.Warnings.Landslides(ctx); err == nil {
			readings = append(readings, Landslides(layer, now))
		} else {
			fail("landslides", err)
		}
		if bulletin, err := engine.Warnings.LandslideWarning(ctx); err == nil {
			readings = append(readings, LandslideIssued(bulletin, now))
		} else {
			fail("landslide warning", err)
		}
	}
	if engine.Rivers != nil {
		if layer, err := engine.Rivers.Layer(ctx); err == nil {
			readings = append(readings, Rivers(layer, places.DistrictOf, now))
			layers.Rivers = &layer
		} else {
			fail("rivers", err)
		}
	}
	if engine.Floods != nil {
		if layer, err := engine.Floods.Layer(ctx); err == nil {
			readings = append(readings, FloodForecasts(layer, places.DistrictOf, now), FlashFloods(layer, places.AreaDistricts, now))
			layers.Floods = &layer
		} else {
			fail("flood forecasts", err)
		}
	}
	if engine.Hazards != nil {
		if layer, err := engine.Hazards.Cyclones(ctx); err == nil {
			readings = append(readings, WithNearestDistrict(Cyclones(layer, now), places.DistrictOf))
		} else {
			fail("cyclones", err)
		}
		if layer, err := engine.Hazards.Earthquakes(ctx); err == nil {
			readings = append(readings, Earthquakes(layer, now))
		} else {
			fail("earthquakes", err)
		}
	}
	if engine.Reservoirs != nil {
		if layer, err := engine.Reservoirs.Layer(ctx); err == nil {
			readings = append(readings, Reservoirs(layer, places.DistrictNamed, now))
			layers.Reservoirs = &layer
		} else {
			fail("reservoirs", err)
		}
	}
	if engine.RainGauges != nil {
		if intensities, err := engine.RainGauges.Intensities(ctx); err == nil {
			readings = append(readings, HeavyRain(intensities, places.DistrictOf, now))
			layers.Rain = &intensities
		} else {
			fail("rain gauges", err)
		}
	}
	if engine.Weather != nil {
		if outlook, err := engine.Weather.RainOutlooks(ctx, "district"); err == nil {
			layers.Outlook = outlook
		} else {
			fail("rain forecast", err)
		}
	}
	// The estimate needs the rain gauges at least; without them it would rest
	// on forecasts alone.
	if layers.Rain != nil {
		assessments := map[string]floodrisk.Assessment{}
		for district, inputs := range floodrisk.Gather(layers, places.Places, now) {
			assessments[district] = floodrisk.Assess(inputs)
		}
		readings = append(readings, FloodRisk(assessments, layers.Complete(), now))
	}
	return readings
}

// Transition is one subject's step: its new state and the event to raise.
type Transition struct {
	State State
	Event *push.Event
}

// Apply works out a reading's transitions against the states known. It is
// pure, so tests can replay fixtures through it.
func Apply(reading Reading, states map[string]State, now time.Time) []Transition {
	seen := map[string]bool{}
	transitions := []Transition{}
	for _, observation := range reading.Observations {
		seen[observation.Subject] = true
		previous, ok := states[observation.Subject]
		if !ok {
			previous = State{Subject: observation.Subject, Kind: reading.Kind}
		}
		next, alert := Step(previous, observation.Level, observation.Hold, reading.HoldFor, now)
		next.Kind = reading.Kind
		transition := Transition{State: next}
		if alert && observation.Level > 0 {
			event := observation.Event
			event.Subject, event.Key, event.DetectedAt = observation.Subject, next.EventKey(), now
			transition.Event = &event
		}
		if !observation.Hold && (ok || !next.Idle()) {
			transitions = append(transitions, transition)
		}
	}
	if !reading.Complete {
		return transitions
	}
	// A complete reading that no longer mentions a subject says its level is 0.
	for subject, previous := range states {
		if seen[subject] || previous.Kind != reading.Kind {
			continue
		}
		next, _ := Step(previous, 0, false, reading.HoldFor, now)
		transitions = append(transitions, Transition{State: next})
	}
	return transitions
}

// apply stores a reading's transitions in one transaction, so an event and the
// state that raised it are written together.
func (engine *Engine) apply(ctx context.Context, reading Reading, states map[string]State, now time.Time) (int, error) {
	transitions := Apply(reading, states, now)
	if len(transitions) == 0 {
		return 0, nil
	}
	tx, err := engine.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	queries := store.New(engine.Pool).WithTx(tx)
	raised := 0
	for _, transition := range transitions {
		state := transition.State
		if transition.Event != nil {
			params, err := push.InsertEventParams(*transition.Event)
			if err != nil {
				return 0, err
			}
			inserted, err := queries.InsertPushEvent(ctx, params)
			if err != nil {
				return 0, fmt.Errorf("store event %s: %w", transition.Event.Key, err)
			}
			raised += int(inserted)
		}
		if state.Idle() {
			if err := queries.DeletePushAlertState(ctx, state.Subject); err != nil {
				return 0, err
			}
			continue
		}
		if err := queries.UpsertPushAlertState(ctx, store.UpsertPushAlertStateParams{Subject: state.Subject, Kind: state.Kind,
			Level: int16(state.Level), AlertedLevel: int16(state.AlertedLevel), AlertedAt: state.AlertedAt,
			BelowSince: state.BelowSince, UpdatedAt: state.UpdatedAt}); err != nil {
			return 0, fmt.Errorf("store state of %s: %w", state.Subject, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	for _, transition := range transitions {
		if transition.State.Idle() {
			delete(states, transition.State.Subject)
		} else {
			states[transition.State.Subject] = transition.State
		}
	}
	return raised, nil
}

// places finds districts through the shared locator, reading district names
// once per run.
func (engine *Engine) places(ctx context.Context) places {
	engine.once.Do(func() {
		if engine.Locator == nil {
			engine.Locator = &floodrisk.Locator{Pool: engine.Pool, Logger: engine.Logger}
		}
	})
	locator := engine.Locator
	return places{
		Places: floodrisk.Places{
			DistrictOf:    func(lon, lat float64) string { return locator.DistrictOf(ctx, lon, lat) },
			AreaDistricts: func(id string) []string { return locator.AreaDistricts(ctx, id) },
			DistrictNamed: locator.Named(ctx),
		},
		BasinDistricts: func(title string) []string { return locator.BasinDistricts(ctx, title) },
	}
}

// places are the estimate's lookups and the basins of flood warnings.
type places struct {
	floodrisk.Places
	BasinDistricts func(title string) []string
}

// WithBasinDistricts places a flood warning that names no district in the
// districts of the basin its title names, "Flood Warning for Gin River Basin"
// in Galle and Matara: the bulletin's text often names only rivers and
// reservoirs, and such a warning had gone to the whole country as "Sri
// Lanka" (November 2026). One with no basin we know stays national.
func WithBasinDistricts(reading Reading, basinDistricts func(title string) []string) Reading {
	if basinDistricts == nil {
		return reading
	}
	for index := range reading.Observations {
		event := &reading.Observations[index].Event
		if len(event.Districts) > 0 || event.Facts["type"] != "water_level" {
			continue
		}
		districts := basinDistricts(event.Facts["title"])
		if len(districts) == 0 {
			continue
		}
		facts := maps.Clone(event.Facts)
		facts["districts"] = strings.Join(districts, ",")
		facts["areaFrom"] = "basin"
		event.Facts, event.Districts, event.National = facts, districts, false
		if len(districts) == 1 {
			event.URL += "&district=" + districts[0]
		}
	}
	return reading
}

// WithNearestDistrict names the district of the coast nearest each cyclone.
func WithNearestDistrict(reading Reading, districtOf DistrictOf) Reading {
	if districtOf == nil {
		return reading
	}
	for index := range reading.Observations {
		event := &reading.Observations[index].Event
		lon, errLon := strconv.ParseFloat(event.Facts["coastLon"], 64)
		lat, errLat := strconv.ParseFloat(event.Facts["coastLat"], 64)
		if errLon != nil || errLat != nil {
			continue
		}
		facts := maps.Clone(event.Facts)
		delete(facts, "coastLon")
		delete(facts, "coastLat")
		if district := districtOf(lon, lat); district != "" {
			facts["nearDistrict"] = district
		}
		event.Facts = facts
	}
	return reading
}
