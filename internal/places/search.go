// Package places searches populated places and administrative regions by
// name, in English, Sinhala or Tamil, tolerating typos.
//
// Every name of every region and place is stored folded (lower case,
// accents and joiners removed) in search_names with a trigram index. A
// query matches a name exactly, as a prefix, as the prefix of a later word,
// or, from three letters, by trigram word similarity. Results are ranked by
// match quality (70%) and importance (30%): cities, then provinces and
// districts, towns, DS divisions, suburbs, villages, GN divisions and
// hamlets, with population breaking ties.
package places

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Limits on a search.
const (
	MaxQueryLength = 100
	DefaultLimit   = 10
	MaxLimit       = 50
	// fuzzyFrom is the query length from which typos are matched; shorter
	// queries have too few trigrams to compare.
	fuzzyFrom = 3
	// similarityThreshold is the trigram word similarity a fuzzy match needs.
	similarityThreshold = "0.5"
)

// Attribution credits the searched data.
const Attribution = "Places © OpenStreetMap contributors (ODbL); regions: " + regions.Attribution

// Service searches the database.
type Service struct {
	Pool *pgxpool.Pool
}

// Result is a search's answer.
type Result struct {
	Query       string  `json:"query"`
	Results     []Match `json:"results"`
	Attribution string  `json:"attribution"`
}

// Match is one place or region found.
type Match struct {
	// ID is a region code (LK-21) or a place id (osm-node-2908772560).
	ID string `json:"id"`
	// Type is "region" or "place".
	Type string `json:"type"`
	// Kind is the region's level (province, district, ds, gn) or the place's
	// kind (city, town, suburb, village, quarter, neighbourhood, hamlet,
	// locality, isolated_dwelling).
	Kind  string        `json:"kind"`
	Name  string        `json:"name"`
	Names regions.Names `json:"names"`
	// MatchedName is the name the query matched, which may be a Sinhala,
	// Tamil or alternative name.
	MatchedName string  `json:"matchedName"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	// BBox is [west, south, east, north] for regions and null for places.
	BBox       *[4]float64 `json:"bbox"`
	Population *int32      `json:"population"`
	// Hierarchy lists the regions containing the match, province first.
	Hierarchy []regions.Ref `json:"hierarchy"`
	Score     float64       `json:"score"`
}

// NormalizeQuery trims a query and collapses its spaces.
func NormalizeQuery(query string) string {
	return strings.Join(strings.Fields(query), " ")
}

// Search finds up to limit places and regions matching query, best first.
func (service *Service) Search(ctx context.Context, query string, limit int) (Result, error) {
	query = NormalizeQuery(query)
	result := Result{Query: query, Results: []Match{}, Attribution: Attribution}
	var rows []store.SearchNamesRow
	err := pgx.BeginFunc(ctx, service.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		if err := queries.SetWordSimilarityThreshold(ctx, similarityThreshold); err != nil {
			return err
		}
		var err error
		rows, err = queries.SearchNames(ctx, store.SearchNamesParams{
			Query: query, Fuzzy: utf8.RuneCountInString(query) >= fuzzyFrom, MaxResults: int32(limit),
		})
		return err
	})
	if err != nil {
		return Result{}, fmt.Errorf("search %q: %w", query, err)
	}
	for _, row := range rows {
		match := Match{
			ID: row.TargetID, Type: row.TargetKind, Kind: row.Kind, Name: row.Name,
			Names:       regions.Names{EN: row.Name, SI: row.NameSi, TA: row.NameTa},
			MatchedName: row.MatchedName,
			Lat:         round(row.Lat, 6), Lon: round(row.Lon, 6),
			Population: row.Population, Hierarchy: []regions.Ref{}, Score: round(row.Score, 3),
		}
		if row.West != nil && row.South != nil && row.East != nil && row.North != nil {
			match.BBox = &[4]float64{round(*row.West, 6), round(*row.South, 6), round(*row.East, 6), round(*row.North, 6)}
		}
		for _, ancestor := range []struct {
			level    regions.Level
			id, name *string
		}{
			{regions.LevelProvince, row.ProvinceID, row.ProvinceName},
			{regions.LevelDistrict, row.DistrictID, row.DistrictName},
			{regions.LevelDS, row.DsID, row.DsName},
			{regions.LevelGN, row.GnID, row.GnName},
		} {
			if ancestor.id == nil || ancestor.name == nil || *ancestor.id == row.TargetID {
				continue
			}
			match.Hierarchy = append(match.Hierarchy, regions.Ref{ID: *ancestor.id, Level: ancestor.level, Name: *ancestor.name})
		}
		result.Results = append(result.Results, match)
	}
	return result, nil
}

func round(value float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(value*scale) / scale
}
