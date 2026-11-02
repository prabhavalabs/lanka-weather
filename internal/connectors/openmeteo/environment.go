package openmeteo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Air quality and sea state come from Open-Meteo's air-quality and marine
// APIs, on coarser grids of their own and with less frequent runs than the
// weather grid. They are assumed to share its free-tier budget (ADR 0012).
const (
	KindAirQuality Kind = "air_quality"
	KindMarine     Kind = "marine"
)

// Source names, addresses and credits. CAMS is the model behind air quality
// over Sri Lanka, and its licence asks to be named.
const (
	SourceAirQuality = "open-meteo-air-quality"
	SourceMarine     = "open-meteo-marine"

	EndpointAirQuality = "https://air-quality-api.open-meteo.com/v1/air-quality"
	EndpointMarine     = "https://marine-api.open-meteo.com/v1/marine"

	AttributionAirQuality = "Air quality by Open-Meteo.com (CC BY 4.0), from the Copernicus Atmosphere Monitoring Service (CAMS)"
	AttributionMarine     = "Marine data by Open-Meteo.com (CC BY 4.0)"

	ModelAirQuality = "cams_global"
	ModelMarine     = "best_match"
)

// Variables of each kind. Fewer than ten keeps a node's weight at one call.
var (
	AirQualityVariables = []string{"us_aqi", "pm2_5"}
	MarineVariables     = []string{
		"wave_height", "wave_direction", "wave_period",
		"swell_wave_height", "swell_wave_direction", "swell_wave_period",
	}
)

// EnvironmentInterval is how often both run: CAMS is issued twice a day and
// the wave models a few times, so more would mostly re-read the same forecast.
const EnvironmentInterval = 3 * time.Hour

// EnvironmentDailyCalls is the weighted calls a day of air-quality and marine
// collecting costs on the two grids, one per node per run.
func EnvironmentDailyCalls(airNodes, marineNodes int, every time.Duration) float64 {
	return float64(24*time.Hour/every) * float64(airNodes+marineNodes)
}

// environment returns the kind's source, endpoint, model and variables.
func environment(kind Kind) (sourceName, endpoint, model string, variables []string) {
	if kind == KindAirQuality {
		return SourceAirQuality, EndpointAirQuality, ModelAirQuality, AirQualityVariables
	}
	return SourceMarine, EndpointMarine, ModelMarine, MarineVariables
}

// EnvironmentQuery builds one request for a chunk of nodes. The APIs take many
// coordinates, like the forecast API. Air quality has one model domain
// and no cell selection; the marine API is asked for sea cells.
func EnvironmentQuery(kind Kind, nodes []grid.Node, positions []int) url.Values {
	lats := make([]string, len(positions))
	lons := make([]string, len(positions))
	for index, position := range positions {
		lats[index] = strconv.FormatFloat(nodes[position].Lat, 'f', -1, 64)
		lons[index] = strconv.FormatFloat(nodes[position].Lon, 'f', -1, 64)
	}
	_, _, _, variables := environment(kind)
	query := url.Values{
		"latitude":   {strings.Join(lats, ",")},
		"longitude":  {strings.Join(lons, ",")},
		"current":    {strings.Join(variables, ",")},
		"timeformat": {"unixtime"},
		"timezone":   {"GMT"},
	}
	if kind == KindMarine {
		query.Set("cell_selection", "sea")
	}
	return query
}

// EnvironmentLocation is one location of a response. Missing values are NaN,
// as for a node the model treats as land.
type EnvironmentLocation struct {
	Time   time.Time
	Values map[string]float32
}

// DecodeEnvironment reads an air-quality or marine response for `count`
// locations. It does no I/O.
func DecodeEnvironment(kind Kind, payload []byte, count int) ([]EnvironmentLocation, error) {
	trimmed := strings.TrimSpace(string(payload))
	var wire []wireLocation
	switch {
	case strings.HasPrefix(trimmed, "["):
		if err := json.Unmarshal(payload, &wire); err != nil {
			return nil, fmt.Errorf("decode Open-Meteo response: %w", err)
		}
	case strings.HasPrefix(trimmed, "{"):
		var failure wireError
		if err := json.Unmarshal(payload, &failure); err == nil && failure.Error {
			return nil, fmt.Errorf("Open-Meteo error: %s", failure.Reason)
		}
		var single wireLocation
		if err := json.Unmarshal(payload, &single); err != nil {
			return nil, fmt.Errorf("decode Open-Meteo response: %w", err)
		}
		wire = []wireLocation{single}
	default:
		return nil, errors.New("Open-Meteo response is not JSON")
	}
	if len(wire) != count {
		return nil, fmt.Errorf("Open-Meteo returned %d locations for %d asked", len(wire), count)
	}
	_, _, _, variables := environment(kind)
	locations := make([]EnvironmentLocation, len(wire))
	for index, item := range wire {
		if item.Current == nil {
			return nil, fmt.Errorf("location %d: no current block", index)
		}
		var unix int64
		if err := json.Unmarshal(item.Current["time"], &unix); err != nil || unix <= 0 {
			return nil, fmt.Errorf("location %d: current block has no unix time", index)
		}
		values := make(map[string]float32, len(variables))
		for _, variable := range variables {
			value, err := number(item.Current[variable])
			if err != nil {
				return nil, fmt.Errorf("location %d: %s: %w", index, variable, err)
			}
			values[variable] = value
		}
		locations[index] = EnvironmentLocation{Time: time.Unix(unix, 0).UTC(), Values: values}
	}
	return locations, nil
}

// EnvironmentFields turns locations into one field per variable, at the
// earliest time they report. It fails when a whole variable has no value,
// which would otherwise replace a good run with an empty one.
func EnvironmentFields(kind Kind, locations []EnvironmentLocation) (time.Time, []Field, error) {
	if len(locations) == 0 {
		return time.Time{}, nil, errors.New("no locations")
	}
	var at time.Time
	for _, location := range locations {
		if at.IsZero() || location.Time.Before(at) {
			at = location.Time
		}
	}
	_, _, _, variables := environment(kind)
	fields := make([]Field, 0, len(variables))
	for _, variable := range variables {
		values := make([]float32, len(locations))
		present := 0
		for position, location := range locations {
			values[position] = location.Values[variable]
			if !math.IsNaN(float64(values[position])) {
				present++
			}
		}
		if present == 0 {
			return time.Time{}, nil, fmt.Errorf("no location has %s", variable)
		}
		fields = append(fields, Field{Variable: variable, ValidFrom: at, Steps: 1, Values: values})
	}
	return at, fields, nil
}

// EnvironmentCollector fetches one kind for every node of its grid and stores
// it as one run, whole or not at all.
type EnvironmentCollector struct {
	Pool   *pgxpool.Pool
	Client Doer
	Grid   grid.Grid
	Kind   Kind
	// Endpoint overrides the API address, for tests.
	Endpoint string
	// NoPacing sends requests back to back, for tests.
	NoPacing bool
	Logger   *slog.Logger
}

// Name implements source.Collector.
func (collector *EnvironmentCollector) Name() string {
	name, _, _, _ := environment(collector.Kind)
	return name
}

// Interval implements source.Collector.
func (collector *EnvironmentCollector) Interval() time.Duration { return EnvironmentInterval }

// Collect implements source.Collector. Requests are paced like the weather
// grid's and retried the same way when the service is overloaded.
func (collector *EnvironmentCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	sourceName, endpoint, model, _ := environment(collector.Kind)
	if collector.Endpoint != "" {
		endpoint = collector.Endpoint
	}
	nodes := collector.Grid.Nodes
	locations := make([]EnvironmentLocation, len(nodes))
	for index, chunk := range environmentChunks(len(nodes), chunkSize) {
		if index > 0 && !collector.NoPacing {
			pause := time.Duration(float64(len(chunk)) / weightPerMinute * float64(time.Minute))
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(pause):
			}
		}
		response, err := fetchRetrying(ctx, collector.Client, collector.NoPacing, endpoint+"?"+EnvironmentQuery(collector.Kind, nodes, chunk).Encode())
		if err != nil {
			return 0, fmt.Errorf("request %d: %w", index+1, err)
		}
		decoded, err := DecodeEnvironment(collector.Kind, response.Body, len(chunk))
		if err != nil {
			return 0, fmt.Errorf("request %d: %w", index+1, err)
		}
		for k, position := range chunk {
			locations[position] = decoded[k]
		}
	}
	sourceTime, fields, err := EnvironmentFields(collector.Kind, locations)
	if err != nil {
		return 0, err
	}
	indexes := make([]int32, len(nodes))
	for position, node := range nodes {
		indexes[position] = int32(node.Index)
	}
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		runID, err := queries.InsertWeatherRun(ctx, store.InsertWeatherRunParams{
			Kind: string(collector.Kind), Source: sourceName, Model: model, Grid: collector.Grid.ID,
			Nodes: indexes, SourceTime: sourceTime, FetchedAt: now,
		})
		if err != nil {
			return fmt.Errorf("store run: %w", err)
		}
		params := make([]store.InsertWeatherFieldParams, len(fields))
		for index, field := range fields {
			params[index] = store.InsertWeatherFieldParams{RunID: runID, Variable: field.Variable, Source: sourceName,
				ValidFrom: field.ValidFrom, Steps: 1, NodeValues: field.Values, FetchedAt: now}
		}
		if err := batch(queries.InsertWeatherField(ctx, params)); err != nil {
			return fmt.Errorf("store fields: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if collector.Logger != nil {
		collector.Logger.Info("open-meteo environment run stored", "kind", collector.Kind, "nodes", len(nodes), "fields", len(fields))
	}
	return len(nodes) * len(fields), nil
}

// environmentChunks splits node positions into requests of at most size.
func environmentChunks(nodes, size int) [][]int {
	var chunks [][]int
	for start := 0; start < nodes; start += size {
		chunk := make([]int, 0, size)
		for position := start; position < min(start+size, nodes); position++ {
			chunk = append(chunk, position)
		}
		chunks = append(chunks, chunk)
	}
	return chunks
}

// fetchRetrying sends a request, retrying as the weather grid does.
func fetchRetrying(ctx context.Context, client Doer, noPacing bool, target string) (source.Response, error) {
	collector := Collector{Client: client, NoPacing: noPacing}
	return collector.fetch(ctx, target)
}
