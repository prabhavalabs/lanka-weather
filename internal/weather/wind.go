package weather

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/gfs"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// ErrNoWind means no stored run has a wind field for the time asked.
var ErrNoWind = errors.New("no wind field for that time")

// WindField is the wind for the particle layer, in the format TerraGuard's
// renderer reads (apps/web wind/field.ts in TerraGuard): u and v are
// base64-encoded little-endian int16s of Scale m/s; value j × nx + i lies
// at lat0 + j × dlat, lon0 + i × dlon. Rows run north to south (dlat < 0).
type WindField struct {
	Source       string    `json:"source"`
	Model        string    `json:"model"`
	Level        string    `json:"level"`
	RunAt        time.Time `json:"runAt"`
	ForecastHour int       `json:"forecastHour"`
	ValidAt      time.Time `json:"validAt"`
	// FirstValidAt and LastValidAt bound the times this run is served for,
	// StepHours apart.
	FirstValidAt time.Time `json:"firstValidAt"`
	LastValidAt  time.Time `json:"lastValidAt"`
	StepHours    int       `json:"stepHours"`
	Grid         WindGrid  `json:"grid"`
	Scale        float64   `json:"scale"`
	U            string    `json:"u"`
	V            string    `json:"v"`
	Attribution  string    `json:"attribution"`
	FetchedAt    time.Time `json:"fetchedAt"`
}

// WindGrid is the wind field's grid.
type WindGrid struct {
	Nx   int     `json:"nx"`
	Ny   int     `json:"ny"`
	Lon0 float64 `json:"lon0"`
	Lat0 float64 `json:"lat0"`
	DLon float64 `json:"dlon"`
	DLat float64 `json:"dlat"`
}

type encodedWind struct {
	json, gzipped []byte
}

// windRunsRecheck is how often the list of stored runs is re-read.
const windRunsRecheck = time.Minute

// maxCachedWinds bounds the encoded fields kept, about 60 KB each.
const maxCachedWinds = 64

// ValidTime rounds a time to the nearest forecast step.
func ValidTime(at time.Time) time.Time {
	return at.UTC().Add(gfs.Step / 2).Truncate(gfs.Step)
}

// WindJSON returns the field valid at the step nearest `at` from the newest
// complete run that has it, encoded and gzipped.
func (service *Service) WindJSON(ctx context.Context, at time.Time) (plain, gzipped []byte, err error) {
	runs, err := service.windRunList(ctx)
	if err != nil {
		return nil, nil, err
	}
	valid := ValidTime(at)
	for _, run := range runs {
		if valid.Before(run.FirstValidAt) || valid.After(run.LastValidAt) {
			continue
		}
		hour := int(valid.Sub(run.RunAt) / time.Hour)
		key := run.RunAt.Format("2006010215") + fmt.Sprintf("f%03d", hour)
		service.windMutex.Lock()
		cached := service.winds[key]
		service.windMutex.Unlock()
		if cached != nil {
			return cached.json, cached.gzipped, nil
		}
		row, err := store.New(service.Pool).WindField(ctx, store.WindFieldParams{RunAt: run.RunAt, ForecastHour: int32(hour)})
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read wind field: %w", err)
		}
		body, err := json.Marshal(NewWindField(row, run.FirstValidAt, run.LastValidAt))
		if err != nil {
			return nil, nil, err
		}
		encoded := &encodedWind{json: body, gzipped: compress(body)}
		service.windMutex.Lock()
		if service.winds == nil || len(service.winds) >= maxCachedWinds {
			service.winds = map[string]*encodedWind{}
		}
		service.winds[key] = encoded
		service.windMutex.Unlock()
		return encoded.json, encoded.gzipped, nil
	}
	return nil, nil, ErrNoWind
}

// NewWindField converts a stored field to the response.
func NewWindField(row store.WindField, first, last time.Time) WindField {
	return WindField{
		Source: gfs.SourceName, Model: row.Model, Level: row.Level, RunAt: row.RunAt, ForecastHour: int(row.ForecastHour),
		ValidAt: row.ValidAt, FirstValidAt: first, LastValidAt: last, StepHours: int(gfs.Step / time.Hour),
		Grid:  WindGrid{Nx: int(row.Nx), Ny: int(row.Ny), Lon0: row.Lon0, Lat0: row.Lat0, DLon: row.Dlon, DLat: row.Dlat},
		Scale: row.Scale, U: base64.StdEncoding.EncodeToString(row.U), V: base64.StdEncoding.EncodeToString(row.V),
		Attribution: gfs.Attribution, FetchedAt: row.FetchedAt,
	}
}

// windRunList returns the stored runs that hold every forecast hour, newest
// first, re-read at most once a minute.
func (service *Service) windRunList(ctx context.Context) ([]store.LatestWindRunsRow, error) {
	service.windMutex.Lock()
	defer service.windMutex.Unlock()
	if service.windRuns != nil && time.Since(service.windAt) < windRunsRecheck {
		return service.windRuns, nil
	}
	rows, err := store.New(service.Pool).LatestWindRuns(ctx, 8)
	if err != nil {
		return nil, fmt.Errorf("read wind runs: %w", err)
	}
	complete := make([]store.LatestWindRunsRow, 0, len(rows))
	for _, row := range rows {
		if int(row.Steps) >= len(gfs.Hours()) {
			complete = append(complete, row)
		}
	}
	service.windRuns, service.windAt = complete, time.Now()
	return complete, nil
}
