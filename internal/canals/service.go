// Package canals serves Colombo's canal levels. The readings are CUrW's and SLLDC's,
// reached through waterlevels.lk, and they are context rather than a flood warning.
// The feed gives no positions and none are guessed, so this is a list, not map pins.
package canals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// Note says what these levels are.
	Note = "Context, not a flood warning. Levels are metres above mean sea level."
	// staleAfter is how old a reading may be: the gauges report every 15 minutes.
	staleAfter = 3 * time.Hour
	// Colombo's middle and how near it counts as the Colombo area, for the location
	// panel's link. It says nothing of where any one gauge is.
	colomboLon, colomboLat = 79.8612, 6.9271
	areaKm                 = 20.0
)

// Service reads the stored canal gauges.
type Service struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

// Document is GET /v1/rivers/canals.
type Document struct {
	Gauges []Gauge `json:"gauges"`
	Sea    *Sea    `json:"sea"`
	Tide   Tide    `json:"tide"`
	Meta   Meta    `json:"meta"`
}

// Meta says where the levels come from and how current they are.
type Meta struct {
	Source string `json:"source"`
	// Credit is the source's own credit field, and Attribution the line it asks for.
	Credit      string     `json:"credit"`
	Attribution string     `json:"attribution"`
	Via         string     `json:"via"`
	Licence     string     `json:"licence"`
	UpstreamURL string     `json:"upstreamUrl"`
	Note        string     `json:"note"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	SourceTime  *time.Time `json:"sourceTime"`
	GeneratedAt time.Time  `json:"generatedAt"`
	Total       int        `json:"total"`
	Reporting   int        `json:"reporting"`
	// Raised counts gauges in the source's states "rising" and "higher".
	Raised int  `json:"raised"`
	Stale  bool `json:"stale"`
}

// Gauge is one canal gauge.
type Gauge struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
	// State is the source's word: normal, rising, higher or offline.
	State      string                   `json:"state"`
	LevelM     *float64                 `json:"levelM"`
	ObservedAt *time.Time               `json:"observedAt"`
	Change3hM  *float64                 `json:"change3hM"`
	Last24h    []waterlevels.CanalPoint `json:"last24h"`
	// Stale says the reading is too old, whatever the state says.
	Stale bool `json:"stale"`
}

// Sea is the sea level at the Kelani mouth.
type Sea struct {
	LevelM     *float64   `json:"levelM"`
	Trend      *string    `json:"trend"`
	ObservedAt *time.Time `json:"observedAt"`
}

// Tide is the predicted tide, as the source words it.
type Tide struct {
	Note   string                  `json:"note"`
	Events []waterlevels.TideEvent `json:"events"`
}

// Brief is what a place's summary says of the canals: how many report and how many
// are raised, never a position.
type Brief struct {
	Total       int        `json:"total"`
	Reporting   int        `json:"reporting"`
	Raised      int        `json:"raised"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	Stale       bool       `json:"stale"`
	Attribution string     `json:"attribution"`
	Note        string     `json:"note"`
}

// Current reads the canals.
func (service *Service) Current(ctx context.Context) (*Document, error) {
	now := service.now()
	queries := store.New(service.Pool)
	rows, err := queries.ListCanalGauges(ctx)
	if err != nil {
		return nil, fmt.Errorf("list canal gauges: %w", err)
	}
	document := &Document{Gauges: make([]Gauge, 0, len(rows)), Tide: Tide{Events: []waterlevels.TideEvent{}}, Meta: Meta{
		Source: waterlevels.CanalsSource, Attribution: waterlevels.CanalCredit, Via: waterlevels.Attribution, Note: Note, GeneratedAt: now,
		Total: len(rows), Stale: true,
	}}
	for _, row := range rows {
		gauge := Gauge{ID: row.ID, Name: row.Name, Group: row.CanalGroup, State: row.State, LevelM: row.LevelM, ObservedAt: row.ObservedAt,
			Change3hM: row.Change3hM, Last24h: []waterlevels.CanalPoint{}}
		if err := json.Unmarshal(row.Last24h, &gauge.Last24h); err != nil {
			return nil, fmt.Errorf("read canal gauge %s: %w", row.ID, err)
		}
		gauge.Stale = row.ObservedAt == nil || row.LevelM == nil || now.Sub(*row.ObservedAt) > staleAfter
		if !gauge.Stale && row.State != "offline" {
			document.Meta.Reporting++
			if row.State == "rising" || row.State == "higher" {
				document.Meta.Raised++
			}
		}
		document.Gauges = append(document.Gauges, gauge)
	}
	stored, err := queries.GetCanalContext(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return document, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read canal context: %w", err)
	}
	fetched, sourced := stored.FetchedAt.UTC(), stored.SourceTime.UTC()
	document.Meta.Credit, document.Meta.Licence, document.Meta.UpstreamURL = stored.Credit, stored.Licence, stored.UpstreamUrl
	document.Meta.FetchedAt, document.Meta.SourceTime = &fetched, &sourced
	document.Meta.Stale = document.Meta.Reporting == 0
	document.Sea = &Sea{LevelM: stored.SeaLevelM, Trend: stored.SeaTrend, ObservedAt: stored.SeaObservedAt}
	document.Tide.Note = stored.TideNote
	if err := json.Unmarshal(stored.Tide, &document.Tide.Events); err != nil {
		return nil, fmt.Errorf("read the tide: %w", err)
	}
	return document, nil
}

// InArea says whether a place is in the Colombo area: within 20 km of its middle.
func InArea(lon, lat float64) bool {
	return distanceKm(lon, lat) <= areaKm
}

// BriefAt says what a place's summary says of the canals; nil outside the Colombo area.
func (service *Service) BriefAt(ctx context.Context, lon, lat float64) (*Brief, error) {
	if !InArea(lon, lat) {
		return nil, nil
	}
	document, err := service.Current(ctx)
	if err != nil {
		return nil, err
	}
	return &Brief{Total: document.Meta.Total, Reporting: document.Meta.Reporting, Raised: document.Meta.Raised, FetchedAt: document.Meta.FetchedAt,
		Stale: document.Meta.Stale, Attribution: document.Meta.Attribution, Note: Note}, nil
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}

func distanceKm(lon, lat float64) float64 {
	const earthRadiusKm = 6371.0088
	toRadians := math.Pi / 180
	dLat, dLon := (colomboLat-lat)*toRadians, (colomboLon-lon)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat*toRadians)*math.Cos(colomboLat*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}
