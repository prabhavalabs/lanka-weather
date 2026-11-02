// Package weather serves the stored weather grid and wind field: the grid of
// current values for the map, forecasts interpolated to a point, per-region
// aggregates, and the wind field for the animated particle layer. It never
// calls a source; collectors fill the database on a schedule (ADR 0007).
package weather

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/openmeteo"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// ErrNoData means no run of the kind asked for has been stored yet.
var ErrNoData = errors.New("no weather data has been collected yet")

// recheck is how often the newest run IDs are read; a new run is served
// within this time of being stored.
const recheck = 30 * time.Second

// Service reads the weather layers from the database and keeps the newest
// runs decoded in memory.
type Service struct {
	Pool *pgxpool.Pool
	// Regions yields administrative regions for /v1/weather/regions.
	Regions RegionSource
	// CurrentStaleAfter and ForecastStaleAfter are the ages after which data
	// is marked stale.
	CurrentStaleAfter  time.Duration
	ForecastStaleAfter time.Duration
	Now                func() time.Time

	mutex     sync.Mutex
	snapshot  *Snapshot
	checkedAt time.Time

	windMutex sync.Mutex
	winds     map[string]*encodedWind
	windRuns  []store.LatestWindRunsRow
	windAt    time.Time
}

// Run is one stored run with its fields spread over the whole lattice.
type Run struct {
	ID         int64
	Source     string
	Model      string
	SourceTime time.Time
	FetchedAt  time.Time
	Fields     map[string]*Field
}

// Field is one variable of a run: Steps × lattice-size values, step-major,
// NaN where masked or missing.
type Field struct {
	ValidFrom time.Time
	Step      time.Duration
	Steps     int
	Values    []float32
}

// At returns the values of one step.
func (field *Field) At(step int) []float32 {
	size := len(field.Values) / max(field.Steps, 1)
	return field.Values[step*size : (step+1)*size]
}

// StepAt is the step nearest a time, and whether the time is within half a
// step of the field's span.
func (field *Field) StepAt(at time.Time) (int, bool) {
	if field.Step <= 0 {
		return 0, true
	}
	step := int(math.Round(float64(at.Sub(field.ValidFrom)) / float64(field.Step)))
	return step, step >= 0 && step < field.Steps
}

// Snapshot is the newest current and forecast runs on one lattice.
type Snapshot struct {
	Lattice  grid.Lattice
	GridID   string
	Current  *Run
	Forecast *Run

	once    sync.Once
	encoded *encodedGrid
	regions sync.Map // level → *regionResult
}

// lattices are the grids runs may be stored on.
var lattices = map[string]grid.Lattice{grid.SriLankaID: grid.SriLankaLattice}

// Snapshot returns the newest runs, reloading them when a newer run has
// been stored.
func (service *Service) Snapshot(ctx context.Context) (*Snapshot, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	now := time.Now()
	if service.snapshot != nil && now.Sub(service.checkedAt) < recheck {
		return service.snapshot, nil
	}
	queries := store.New(service.Pool)
	ids, err := queries.LatestWeatherRunIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("read newest runs: %w", err)
	}
	service.checkedAt = now
	if current := service.snapshot; current != nil && runID(current.Current) == ids.CurrentID && runID(current.Forecast) == ids.ForecastID {
		return current, nil
	}
	snapshot := &Snapshot{}
	for _, kind := range []string{string(openmeteo.KindCurrent), string(openmeteo.KindForecast)} {
		run, err := loadRun(ctx, queries, kind, snapshot)
		if err != nil {
			return nil, err
		}
		if kind == string(openmeteo.KindCurrent) {
			snapshot.Current = run
		} else {
			snapshot.Forecast = run
		}
	}
	if snapshot.Current == nil && snapshot.Forecast == nil {
		return nil, ErrNoData
	}
	service.snapshot = snapshot
	return snapshot, nil
}

func runID(run *Run) int64 {
	if run == nil {
		return 0
	}
	return run.ID
}

// loadRun reads the newest run of a kind; nil when there is none.
func loadRun(ctx context.Context, queries *store.Queries, kind string, snapshot *Snapshot) (*Run, error) {
	stored, err := queries.LatestWeatherRun(ctx, kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s run: %w", kind, err)
	}
	lattice, ok := lattices[stored.Grid]
	if !ok {
		return nil, fmt.Errorf("run %d is on unknown grid %q", stored.ID, stored.Grid)
	}
	if snapshot.GridID != "" && snapshot.GridID != stored.Grid {
		return nil, fmt.Errorf("current and forecast runs are on different grids")
	}
	snapshot.GridID, snapshot.Lattice = stored.Grid, lattice
	rows, err := queries.WeatherRunFields(ctx, stored.ID)
	if err != nil {
		return nil, fmt.Errorf("read %s fields: %w", kind, err)
	}
	run := &Run{ID: stored.ID, Source: stored.Source, Model: stored.Model, SourceTime: stored.SourceTime,
		FetchedAt: stored.FetchedAt, Fields: make(map[string]*Field, len(rows))}
	size, nodes := lattice.Size(), len(stored.Nodes)
	for _, row := range rows {
		steps := int(row.Steps)
		if len(row.NodeValues) != steps*nodes {
			return nil, fmt.Errorf("field %s of run %d has %d values for %d steps of %d nodes", row.Variable, stored.ID, len(row.NodeValues), steps, nodes)
		}
		values := make([]float32, steps*size)
		for index := range values {
			values[index] = float32(math.NaN())
		}
		for step := range steps {
			for position, node := range stored.Nodes {
				if int(node) < size {
					values[step*size+int(node)] = row.NodeValues[step*nodes+position]
				}
			}
		}
		run.Fields[row.Variable] = &Field{ValidFrom: row.ValidFrom, Step: time.Duration(row.StepSeconds) * time.Second, Steps: steps, Values: values}
	}
	return run, nil
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
