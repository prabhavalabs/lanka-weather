package osmreservoirs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Endpoints are public Overpass API instances, tried in order. The main
// instance asks for no more than 10,000 queries and 1 GB a day; this collector
// sends one query of about 15 MB a week.
var Endpoints = []string{
	"https://overpass-api.de/api/interpreter",
	"https://overpass.kumi.systems/api/interpreter",
}

// box is the query's area around Sri Lanka, as for place search. It reaches no
// part of India.
const box = "5.85,79.5,9.9,81.95"

// Query selects named water bodies and dams, with their geometry.
var Query = strings.NewReplacer("BOX", box).Replace(`[out:json][timeout:180];
(
  way["natural"="water"]["name"](BOX);
  relation["natural"="water"]["name"](BOX);
  way["landuse"="reservoir"]["name"](BOX);
  relation["landuse"="reservoir"]["name"](BOX);
  nwr["man_made"="dam"]["name"](BOX);
  nwr["waterway"="dam"]["name"](BOX);
);
out geom qt;`)

const (
	queryTimeout = 4 * time.Minute
	// MinimumElements is the fewest elements a complete response has; the
	// query returned about 3,100 when this was written.
	MinimumElements = 1500
	// MinimumMatches is the fewest reservoirs a sound run maps. Fewer means
	// the answer or the regions are wrong, so the last good map is kept.
	MinimumMatches = 40
	loadTimeout    = "2min"
)

// ErrNoReservoirs stops a run until the reservoir feed has been read, so the
// weekly query is not spent on nothing.
var ErrNoReservoirs = errors.New("no reservoirs stored yet")

// Collector maps reservoirs weekly.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	Logger  *slog.Logger
	// Endpoints, Minimum and MinimumMatched override the defaults, for tests.
	Endpoints      []string
	Minimum        int
	MinimumMatched int
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration { return 7 * 24 * time.Hour }

// RetryAfter implements source.Retrier. Public Overpass instances are often
// busy, and a failed run should not hammer them.
func (collector *Collector) RetryAfter() time.Duration { return 3 * time.Hour }

// Collect queries OpenStreetMap, matches reservoirs and replaces their positions.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	rows, err := store.New(collector.Pool).ListReservoirTargets(ctx)
	if err != nil {
		return 0, fmt.Errorf("read reservoirs: %w", err)
	}
	if len(rows) == 0 {
		return 0, ErrNoReservoirs
	}
	targets := make([]Target, len(rows))
	wanted := map[string]bool{}
	for index, row := range rows {
		targets[index] = Target{ID: row.ID, Name: row.Name, District: row.District, Kind: row.Kind}
		for _, key := range Keys(targets[index]) {
			wanted[key] = true
		}
	}

	payload, err := collector.fetch(ctx)
	if err != nil {
		return 0, err
	}
	snapshot, err := Decode(payload)
	if err != nil {
		return 0, err
	}
	minimum := MinimumElements
	if collector.Minimum > 0 {
		minimum = collector.Minimum
	}
	if snapshot.Elements < minimum {
		return 0, fmt.Errorf("overpass returned %d elements, want at least %d", snapshot.Elements, minimum)
	}

	// Only candidates whose name could match are measured.
	byID := map[string]Candidate{}
	var staged []store.StageReservoirCandidatesParams
	for _, candidate := range snapshot.Candidates {
		for _, key := range CandidateKeys(candidate.Names) {
			if wanted[key] {
				byID[candidate.OSMID] = candidate
				staged = append(staged, store.StageReservoirCandidatesParams{OsmID: candidate.OSMID, Lines: candidate.Lines})
				break
			}
		}
	}

	minimumMatched := MinimumMatches
	if collector.MinimumMatched > 0 {
		minimumMatched = collector.MinimumMatched
	}
	mapped := 0
	var results []Result
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '"+loadTimeout+"'"); err != nil {
			return err
		}
		queries := store.New(tx)
		if err := queries.ClearReservoirCandidates(ctx); err != nil {
			return err
		}
		if _, err := queries.StageReservoirCandidates(ctx, staged); err != nil {
			return fmt.Errorf("stage candidates: %w", err)
		}
		measured, err := queries.MeasureReservoirCandidates(ctx)
		if err != nil {
			return fmt.Errorf("measure candidates: %w", err)
		}
		candidates := make([]Measured, 0, len(measured))
		for _, row := range measured {
			candidate := byID[row.OsmID]
			candidates = append(candidates, Measured{
				OSMID: row.OsmID, Name: candidate.Name, Kind: candidate.Kind, Names: candidate.Names,
				Lon: row.Lon, Lat: row.Lat, AreaHa: row.AreaHa, Districts: row.Districts, NearDistricts: row.NearDistricts,
			})
		}
		results = Match(targets, candidates)
		for _, result := range results {
			if result.Candidate != nil {
				mapped++
			}
		}
		if mapped < minimumMatched {
			return fmt.Errorf("matched %d of %d reservoirs, want at least %d; keeping the last map", mapped, len(targets), minimumMatched)
		}
		if err := queries.ClearReservoirMap(ctx, store.ClearReservoirMapParams{
			MapSource: SourceName, MapSourceTime: snapshot.SourceTime, MapFetchedAt: now,
		}); err != nil {
			return err
		}
		for _, result := range results {
			match := result.Candidate
			if match == nil {
				continue
			}
			if err := queries.SetReservoirMap(ctx, store.SetReservoirMapParams{
				Lon: match.Lon, Lat: match.Lat, OsmKind: match.Kind, OsmID: match.OSMID,
				OsmName: match.Name, ID: result.Target.ID,
			}); err != nil {
				return fmt.Errorf("store map of %s: %w", result.Target.ID, err)
			}
		}
		return queries.ClearReservoirCandidates(ctx)
	})
	if err != nil {
		return 0, err
	}
	collector.report(results, mapped)
	return mapped, nil
}

// report logs the match rate and every reservoir left off the map, which is
// how a new alias is found.
func (collector *Collector) report(results []Result, mapped int) {
	logger := collector.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("reservoirs mapped", "mapped", mapped, "total", len(results))
	for _, result := range results {
		switch {
		case result.Candidate == nil:
			logger.Info("reservoir not mapped", "id", result.Target.ID, "reason", result.Reason)
		case result.Near:
			logger.Info("reservoir mapped just outside its listed district", "id", result.Target.ID, "osm", result.Candidate.OSMID)
		}
	}
}

// fetch tries each endpoint in turn and returns the first complete answer.
func (collector *Collector) fetch(ctx context.Context) ([]byte, error) {
	endpoints := collector.Endpoints
	if len(endpoints) == 0 {
		endpoints = Endpoints
	}
	var errs []error
	for _, endpoint := range endpoints {
		payload, err := collector.Fetcher.Fetch(ctx, source.Request{
			Method: "POST", URL: endpoint,
			Form:    url.Values{"data": {Query}},
			Timeout: queryTimeout,
		})
		if err == nil {
			return payload, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", endpoint, err))
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("query overpass: %w", errors.Join(errs...))
}
