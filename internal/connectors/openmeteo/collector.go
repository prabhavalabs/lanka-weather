package openmeteo

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Default cadences. Together they cost about 7,200 of the 10,000 weighted
// calls a day the free tier allows on the 0.2° grid (ADR 0007).
const (
	CurrentInterval  = time.Hour
	ForecastInterval = 3 * time.Hour
)

const (
	// chunkSize is the number of nodes in one request.
	chunkSize = 50
	// weightPerMinute paces requests at half the free tier's 600 a minute,
	// so the two collectors running back to back stay well under it.
	weightPerMinute = 300
	// retention is how long runs are kept; the newest of each kind is
	// always kept.
	retention = 24 * time.Hour
)

// Doer sends a request to a source.
type Doer interface {
	Do(ctx context.Context, request source.Request) (source.Response, error)
}

// Collector fetches one kind of data for every node of the grid.
type Collector struct {
	Pool   *pgxpool.Pool
	Client Doer
	Grid   grid.Grid
	Kind   Kind
	// Every overrides the default interval for the kind.
	Every time.Duration
	// Endpoint overrides the API address, for tests.
	Endpoint string
	// NoPacing sends requests back to back, for tests.
	NoPacing bool
	Logger   *slog.Logger
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName + "-" + string(collector.Kind) }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration {
	if collector.Every > 0 {
		return collector.Every
	}
	if collector.Kind == KindCurrent {
		return CurrentInterval
	}
	return ForecastInterval
}

// Collect fetches the whole grid and stores it as one run. A run is stored
// whole or not at all; on a failure the previous run stays current.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	nodes := collector.Grid.Nodes
	locations := make([]Location, len(nodes))
	weight := 0.0
	for index, chunk := range Chunks(nodes, chunkSize) {
		if index > 0 && !collector.NoPacing {
			pause := time.Duration(Weight(collector.Kind, chunkSize) / weightPerMinute * float64(time.Minute))
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(pause):
			}
		}
		endpoint := collector.Endpoint
		if endpoint == "" {
			endpoint = Endpoint
		}
		response, err := collector.fetch(ctx, endpoint+"?"+Query(collector.Kind, nodes, chunk).Encode())
		if err != nil {
			return 0, fmt.Errorf("request %d: %w", index+1, err)
		}
		decoded, err := Decode(collector.Kind, response.Body, len(chunk.Positions))
		if err != nil {
			return 0, fmt.Errorf("request %d: %w", index+1, err)
		}
		for k, position := range chunk.Positions {
			locations[position] = decoded[k]
		}
		weight += Weight(collector.Kind, len(chunk.Positions))
	}

	var sourceTime time.Time
	var fields []Field
	var err error
	if collector.Kind == KindCurrent {
		sourceTime, fields, err = CurrentFields(nodes, locations)
	} else {
		sourceTime, fields, err = ForecastFields(nodes, locations, now)
	}
	if err != nil {
		return 0, err
	}
	if err := collector.save(ctx, locations, sourceTime, fields, now); err != nil {
		return 0, err
	}
	if collector.Logger != nil {
		collector.Logger.Info("open-meteo run stored", "kind", collector.Kind, "nodes", len(nodes),
			"fields", len(fields), "weighted_calls", weight)
	}
	return len(nodes) * len(fields), nil
}

// retryDelays are the pauses before retrying a request the service failed
// with a server error ("The service is overloaded" is a 503). Rate limiting
// (429) is not retried: the client backs off from the host instead.
var retryDelays = []time.Duration{20 * time.Second, 60 * time.Second}

func (collector *Collector) fetch(ctx context.Context, target string) (source.Response, error) {
	for attempt := 0; ; attempt++ {
		response, err := collector.Client.Do(ctx, source.Request{URL: target})
		retryable := source.IsStatus(err, http.StatusInternalServerError, http.StatusBadGateway,
			http.StatusServiceUnavailable, http.StatusGatewayTimeout)
		if !retryable || attempt == len(retryDelays) || collector.NoPacing {
			return response, err
		}
		select {
		case <-ctx.Done():
			return source.Response{}, ctx.Err()
		case <-time.After(retryDelays[attempt]):
		}
	}
}

func (collector *Collector) save(ctx context.Context, locations []Location, sourceTime time.Time, fields []Field, now time.Time) error {
	nodes := collector.Grid.Nodes
	indexes := make([]int32, len(nodes))
	for position, node := range nodes {
		indexes[position] = int32(node.Index)
	}
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		half := collector.Grid.Step / 2
		nodeParams := make([]store.UpsertWeatherGridNodeParams, len(nodes))
		for position, node := range nodes {
			nodeParams[position] = store.UpsertWeatherGridNodeParams{
				Grid: collector.Grid.ID, Node: int32(node.Index), Lon: node.Lon, Lat: node.Lat, Land: node.Land,
				ElevationM: locations[position].Elevation,
				West:       node.Lon - half, South: node.Lat - half, East: node.Lon + half, North: node.Lat + half,
				Source: SourceName, UpdatedAt: now,
			}
		}
		if err := batch(queries.UpsertWeatherGridNode(ctx, nodeParams)); err != nil {
			return fmt.Errorf("store grid nodes: %w", err)
		}
		runID, err := queries.InsertWeatherRun(ctx, store.InsertWeatherRunParams{
			Kind: string(collector.Kind), Source: SourceName, Model: Model, Grid: collector.Grid.ID,
			Nodes: indexes, SourceTime: sourceTime, FetchedAt: now,
		})
		if err != nil {
			return fmt.Errorf("store run: %w", err)
		}
		params := make([]store.InsertWeatherFieldParams, len(fields))
		for index, field := range fields {
			params[index] = store.InsertWeatherFieldParams{
				RunID: runID, Variable: field.Variable, Source: SourceName, ValidFrom: field.ValidFrom,
				StepSeconds: int32(field.Step / time.Second), Steps: int32(field.Steps), NodeValues: field.Values,
				FetchedAt: now,
			}
		}
		if err := batch(queries.InsertWeatherField(ctx, params)); err != nil {
			return fmt.Errorf("store fields: %w", err)
		}
		return nil
	})
}

// Prune implements source.Pruner: runs older than a day go, except the
// newest of each kind.
func (collector *Collector) Prune(ctx context.Context, now time.Time) error {
	return store.New(collector.Pool).DeleteWeatherRunsBefore(ctx, now.Add(-retention))
}

// batchResults is what sqlc's :batchexec queries return.
type batchResults interface {
	Exec(func(int, error))
	Close() error
}

func batch(results batchResults) error {
	var first error
	results.Exec(func(_ int, err error) {
		if err != nil && first == nil {
			first = err
		}
	})
	if err := results.Close(); err != nil && first == nil {
		first = err
	}
	return first
}
