package adminregions

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Where the data lives. Files are fetched at the commit that last changed
// them, so one load never mixes versions and each record names its version.
const (
	BoundaryRepository = "nuuuwan/lk_admin_regions"
	// BoundaryDir holds the TopoJSON files at about 10 m precision.
	BoundaryDir       = "data/geo/topojson/e4_medium"
	CensusRepository  = "nuuuwan/lk_census_2024"
	CensusDir         = "data/Person-Sex"
	censusFile        = "data.tsv"
	defaultAPIBase    = "https://api.github.com"
	defaultRawBase    = "https://raw.githubusercontent.com"
	maxBoundaryBytes  = 96 << 20 // the GN file is about 35 MB
	boundaryTimeout   = 3 * time.Minute
	loadTimeout       = "4min"
	maxOrphanFraction = 0.01
)

// files names each level's TopoJSON file within BoundaryDir.
var files = map[regions.Level]string{
	regions.LevelProvince: "provinces.topojson",
	regions.LevelDistrict: "districts.topojson",
	regions.LevelDS:       "dsds.topojson",
	regions.LevelGN:       "gnds.topojson",
}

// MinimumCounts are the fewest regions a complete load has at each level.
// A load with fewer is rejected, so a truncated upstream file cannot delete
// regions.
var MinimumCounts = map[regions.Level]int{
	regions.LevelProvince: 9,
	regions.LevelDistrict: 25,
	regions.LevelDS:       300,
	regions.LevelGN:       13000,
}

// Collector loads boundaries and populations. It checks weekly for a new
// upstream commit and reloads only when one exists.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// APIBase and RawBase override GitHub's URLs, for tests.
	APIBase, RawBase string
	// Minimums overrides MinimumCounts, for tests with small fixtures.
	Minimums map[regions.Level]int
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return regions.BoundarySource }

// Interval implements source.Collector. Boundaries change a few times a
// year at most.
func (collector *Collector) Interval() time.Duration { return 7 * 24 * time.Hour }

// RetryAfter implements source.Retrier.
func (collector *Collector) RetryAfter() time.Duration { return time.Hour }

// Collect loads every region when the boundary or census files changed
// since the last load, and returns the number of regions stored.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	boundary, err := collector.latestCommit(ctx, BoundaryRepository, BoundaryDir)
	if err != nil {
		return 0, fmt.Errorf("boundary version: %w", err)
	}
	census, err := collector.latestCommit(ctx, CensusRepository, CensusDir)
	if err != nil {
		return 0, fmt.Errorf("census version: %w", err)
	}
	queries := store.New(collector.Pool)
	if current, err := collector.unchanged(ctx, queries, boundary, census); err != nil {
		return 0, err
	} else if current {
		return 0, nil
	}

	payload, err := collector.raw(ctx, CensusRepository, census.SHA, CensusDir+"/"+censusFile)
	if err != nil {
		return 0, fmt.Errorf("fetch census: %w", err)
	}
	population, err := DecodeCensus(payload)
	if err != nil {
		return 0, err
	}

	var all []Region
	for _, level := range regions.Levels {
		payload, err := collector.raw(ctx, BoundaryRepository, boundary.SHA, BoundaryDir+"/"+files[level])
		if err != nil {
			return 0, fmt.Errorf("fetch %s boundaries: %w", level, err)
		}
		decoded, _, err := DecodeLevel(level, payload)
		if err != nil {
			return 0, err
		}
		if minimum := collector.minimum(level); len(decoded) < minimum {
			return 0, fmt.Errorf("%s boundaries: %d regions, want at least %d", level, len(decoded), minimum)
		}
		all = append(all, decoded...)
	}
	all, err = dropOrphans(all)
	if err != nil {
		return 0, err
	}
	for index := range all {
		if count, ok := population[all[index].ID]; ok {
			all[index].Population = &count
		}
	}
	if err := collector.save(ctx, all, boundary, census, now); err != nil {
		return 0, err
	}
	return len(all), nil
}

func (collector *Collector) minimum(level regions.Level) int {
	if collector.Minimums != nil {
		return collector.Minimums[level]
	}
	return MinimumCounts[level]
}

// unchanged reports whether both versions are already loaded.
func (collector *Collector) unchanged(ctx context.Context, queries *store.Queries, boundary, census Commit) (bool, error) {
	for _, check := range []struct {
		name   string
		commit Commit
	}{{regions.BoundarySource, boundary}, {regions.PopulationSource, census}} {
		stored, err := queries.GetSourceVersion(ctx, check.name)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("read %s version: %w", check.name, err)
		}
		if stored.Version != check.commit.SHA {
			return false, nil
		}
	}
	counts, err := queries.CountRegionsByLevel(ctx)
	if err != nil {
		return false, fmt.Errorf("count regions: %w", err)
	}
	return len(counts) == len(regions.Levels), nil
}

// latestCommit asks the GitHub API for the newest commit touching path.
func (collector *Collector) latestCommit(ctx context.Context, repository, path string) (Commit, error) {
	base := collector.APIBase
	if base == "" {
		base = defaultAPIBase
	}
	payload, err := collector.Fetcher.Fetch(ctx, source.Request{
		URL:    base + "/repos/" + repository + "/commits",
		Query:  url.Values{"path": {path}, "per_page": {"1"}},
		Accept: "application/vnd.github+json",
	})
	if err != nil {
		return Commit{}, err
	}
	return DecodeCommits(payload)
}

// raw fetches a file of a repository at a commit.
func (collector *Collector) raw(ctx context.Context, repository, sha, path string) ([]byte, error) {
	base := collector.RawBase
	if base == "" {
		base = defaultRawBase
	}
	return collector.Fetcher.Fetch(ctx, source.Request{
		URL:      base + "/" + repository + "/" + sha + "/" + path,
		Accept:   "application/json, text/plain, */*",
		MaxBytes: maxBoundaryBytes,
		Timeout:  boundaryTimeout,
	})
}

// dropOrphans removes regions whose parent is missing from the load, so the
// hierarchy stays complete. More than a few orphans means a broken load.
func dropOrphans(all []Region) ([]Region, error) {
	known := make(map[string]bool, len(all))
	for _, region := range all {
		known[region.ID] = true
	}
	kept := all[:0]
	orphans := 0
	for _, region := range all {
		if region.ParentID != "" && !known[region.ParentID] {
			orphans++
			continue
		}
		kept = append(kept, region)
	}
	if float64(orphans) > maxOrphanFraction*float64(len(all)) {
		return nil, fmt.Errorf("%d of %d regions have no parent in the load", orphans, len(all))
	}
	return kept, nil
}

func (collector *Collector) save(ctx context.Context, all []Region, boundary, census Commit, now time.Time) error {
	rows := make([]store.StageRegionsParams, len(all))
	populated := 0
	for index, region := range all {
		row := store.StageRegionsParams{
			ID: region.ID, Fid: region.FID, Level: string(region.Level), Name: region.Name,
			NameSi: optional(region.NameSI), NameTa: optional(region.NameTA),
			ParentID: optional(region.ParentID), ProvinceID: region.ProvinceID,
			DistrictID: optional(region.DistrictID), DsID: optional(region.DSID),
			IsoCode: optional(region.ISOCode), GnNumber: optional(region.GNNumber),
			AreaKm2: region.AreaKM2, CenterLon: region.CenterLon, CenterLat: region.CenterLat,
			GeomWkb: region.Shape.WKB(), SimpleWkb: region.Simple.WKB(),
		}
		if region.Population != nil {
			total, year := int32(region.Population.Total), int16(region.Population.Year)
			row.Population, row.PopulationYear = &total, &year
			populated++
		}
		rows[index] = row
	}
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		// Validating and projecting 14,000 boundaries takes longer than the
		// pool's statement timeout allows a request.
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '"+loadTimeout+"'"); err != nil {
			return err
		}
		queries := store.New(tx)
		if err := queries.ClearRegionStaging(ctx); err != nil {
			return err
		}
		if _, err := queries.StageRegions(ctx, rows); err != nil {
			return fmt.Errorf("stage regions: %w", err)
		}
		merged, err := queries.MergeStagedRegions(ctx, store.MergeStagedRegionsParams{
			Source: regions.BoundarySource, SourceVersion: boundary.SHA,
			SourceTime: boundary.Time, FetchedAt: now,
		})
		if err != nil {
			return fmt.Errorf("merge regions: %w", err)
		}
		if merged < int64(len(rows))*99/100 {
			return fmt.Errorf("only %d of %d regions had a usable boundary", merged, len(rows))
		}
		if _, err := queries.DeleteRegionsNotStaged(ctx); err != nil {
			return fmt.Errorf("delete removed regions: %w", err)
		}
		if err := queries.ClearRegionStaging(ctx); err != nil {
			return err
		}
		if err := queries.DeleteSearchNames(ctx, "region"); err != nil {
			return err
		}
		if _, err := queries.InsertRegionSearchNames(ctx); err != nil {
			return fmt.Errorf("index region names: %w", err)
		}
		// Places keep their GN division in step with the boundaries.
		if _, err := queries.AssignPlaceRegions(ctx); err != nil {
			return fmt.Errorf("assign places to regions: %w", err)
		}
		for _, version := range []store.PutSourceVersionParams{
			{Source: regions.BoundarySource, Version: boundary.SHA, SourceTime: boundary.Time, FetchedAt: now, Records: int32(merged)},
			{Source: regions.PopulationSource, Version: census.SHA, SourceTime: census.Time, FetchedAt: now, Records: int32(populated)},
		} {
			if err := queries.PutSourceVersion(ctx, version); err != nil {
				return err
			}
		}
		return nil
	})
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
