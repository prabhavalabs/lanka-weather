// Package reservoirs turns stored reservoir readings into the reservoirs map
// layer: each reservoir's status and storage on its latest reading, where it is
// when it has been matched to OpenStreetMap, and whether the reading has aged
// out. Nothing here is an official warning.
package reservoirs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/osmreservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// ErrNotFound is returned for an unknown reservoir.
var ErrNotFound = errors.New("reservoir not found")

// Note is the caution every answer carries.
const Note = "Not an official warning. In an emergency call 117."

// Attribution credits the readings and the positions.
const Attribution = "Irrigation Department and Mahaweli Authority via " + waterlevels.Attribution +
	"; positions " + osmreservoirs.Attribution

// Service reads the reservoirs layer from the database.
type Service struct {
	Pool *pgxpool.Pool
	// StaleAfter is how long after the end of its reading day a reading is
	// marked stale.
	StaleAfter time.Duration
	Now        func() time.Time
}

// FeatureCollection is a GeoJSON feature collection with layer metadata.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
	Meta     Meta      `json:"meta"`
}

// Meta describes the layer as a whole.
type Meta struct {
	Source      string     `json:"source"`
	MapSource   string     `json:"mapSource"`
	Attribution string     `json:"attribution"`
	Note        string     `json:"note"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	// SourceTime is when the source compiled the newest response.
	SourceTime  *time.Time `json:"sourceTime"`
	MapTime     *time.Time `json:"mapTime"`
	GeneratedAt time.Time  `json:"generatedAt"`
	Counts      Counts     `json:"counts"`
}

// Counts are how many reservoirs there are of each kind.
type Counts struct {
	Total     int `json:"total"`
	Mapped    int `json:"mapped"`
	Spilling  int `json:"spilling"`
	Releasing int `json:"releasing"`
	Holding   int `json:"holding"`
	NoData    int `json:"nodata"`
}

// Feature is a reservoir. Its geometry is null when it has no known position,
// which GeoJSON allows, so the layer also holds the ones the map cannot show.
type Feature struct {
	Type       string     `json:"type"`
	ID         string     `json:"id"`
	Geometry   *Point     `json:"geometry"`
	Properties Properties `json:"properties"`
}

// Point is a GeoJSON point in longitude, latitude order.
type Point struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

// Flow is water leaving a reservoir; a null flow means unknown, never zero.
type Flow struct {
	// Kind is spill, outlets or downstream, as the source writes it. A
	// downstream flow can include power-station flow: it is not a flood release.
	Kind   string   `json:"kind"`
	Cusecs *float64 `json:"cusecs"`
	Agency *string  `json:"agency"`
	Day    *string  `json:"day"`
	// Stale says the figure's day is too old to use, which can differ from the
	// day the reservoir's status was read.
	Stale bool `json:"stale"`
}

// Properties are a reservoir's properties.
type Properties struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	District *string `json:"district"`
	River    *string `json:"river"`
	// Type is major, medium or mahaweli.
	Type string `json:"type"`
	// Status is spilling, releasing, holding or nodata, as the source gives it.
	Status string `json:"status"`
	// StoragePercent is the last figure the source has; StorageFresh says
	// whether it is current. Null means unknown, never zero.
	StoragePercent *float64 `json:"storagePercent"`
	StorageFresh   *bool    `json:"storageFresh"`
	Flow           *Flow    `json:"flow"`
	// ReadingDay is the day the figures were read, in Sri Lanka.
	ReadingDay *string `json:"readingDay"`
	// Agency is the agency that reported the reading.
	Agency    *string   `json:"agency"`
	Source    string    `json:"source"`
	SourceURL *string   `json:"sourceUrl"`
	FetchedAt time.Time `json:"fetchedAt"`
	// Stale says the reading is older than the layer allows. A stale reading
	// says nothing about today, even if it said "spilling".
	Stale bool `json:"stale"`
	// Mapped says the reservoir has a position; OSMName and OSMKind say what it
	// was matched to.
	Mapped  bool    `json:"mapped"`
	OSMName *string `json:"osmName"`
	OSMKind *string `json:"osmKind"`
	// AltNames are other spellings of the name, for search.
	AltNames []string `json:"altNames"`
}

// Active says the reservoir is spilling or releasing on a current reading.
func (properties Properties) Active() bool {
	return !properties.Stale &&
		(properties.Status == waterlevels.StatusSpilling || properties.Status == waterlevels.StatusReleasing)
}

// IsStale says whether a reading day is older than limit, counted from the end
// of that day in Sri Lanka: the sheets are read through the day, and the source
// marks yesterday's reading as still fresh. No reading day is stale.
func IsStale(readingDay pgtype.Date, now time.Time, limit time.Duration) bool {
	if !readingDay.Valid {
		return true
	}
	end := time.Date(readingDay.Time.Year(), readingDay.Time.Month(), readingDay.Time.Day()+1, 0, 0, 0, 0, waterlevels.ColomboZone)
	return now.Sub(end) > limit
}

func day(value pgtype.Date) *string {
	if !value.Valid {
		return nil
	}
	text := value.Time.Format(time.DateOnly)
	return &text
}

// Layer returns every reservoir with its latest reading. Those without a
// position have a null geometry.
func (service *Service) Layer(ctx context.Context) (FeatureCollection, error) {
	now := service.now()
	queries := store.New(service.Pool)
	rows, err := queries.LatestReservoirs(ctx)
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("read reservoirs: %w", err)
	}
	collection := FeatureCollection{
		Type: "FeatureCollection", Features: make([]Feature, 0, len(rows)),
		Meta: Meta{
			Source: waterlevels.ReservoirsSource, MapSource: osmreservoirs.SourceName, Attribution: Attribution, Note: Note,
			GeneratedAt: now,
		},
	}
	if run, err := queries.LastSuccessfulRun(ctx, waterlevels.ReservoirsSource); err == nil {
		collection.Meta.FetchedAt = run.FinishedAt
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return FeatureCollection{}, fmt.Errorf("read last run: %w", err)
	}
	for _, row := range rows {
		feature := Feature{Type: "Feature", ID: row.ID, Properties: Properties{
			ID: row.ID, Name: row.Name, District: row.District, River: row.River, Type: row.Kind,
			Status: row.Status, StoragePercent: row.StoragePercent, StorageFresh: row.StorageFresh,
			Flow:       flow(row.FlowKind, row.FlowCusecs, row.FlowAgency, day(row.FlowDay), flowStale(row.FlowDay, row.ReadingDay, now, service.StaleAfter)),
			ReadingDay: day(row.ReadingDay),
			Agency:     row.Agency, Source: row.Source, SourceURL: row.SourceUrl, FetchedAt: row.FetchedAt,
			Stale: IsStale(row.ReadingDay, now, service.StaleAfter), OSMName: row.OsmName, OSMKind: row.OsmKind,
			AltNames: osmreservoirs.Aliases[row.ID],
		}}
		if row.Lon != nil && row.Lat != nil {
			feature.Geometry = &Point{Type: "Point", Coordinates: [2]float64{*row.Lon, *row.Lat}}
			feature.Properties.Mapped = true
			collection.Meta.Counts.Mapped++
		}
		if source := row.SourceTime; collection.Meta.SourceTime == nil || source.After(*collection.Meta.SourceTime) {
			collection.Meta.SourceTime = &source
		}
		if collection.Meta.MapTime == nil {
			collection.Meta.MapTime = row.MapSourceTime
		}
		switch row.Status {
		case waterlevels.StatusSpilling:
			collection.Meta.Counts.Spilling++
		case waterlevels.StatusReleasing:
			collection.Meta.Counts.Releasing++
		case waterlevels.StatusHolding:
			collection.Meta.Counts.Holding++
		default:
			collection.Meta.Counts.NoData++
		}
		collection.Meta.Counts.Total++
		collection.Features = append(collection.Features, feature)
	}
	return collection, nil
}

func flow(kind *string, cusecs *float64, agency, flowDay *string, stale bool) *Flow {
	if kind == nil {
		return nil
	}
	return &Flow{Kind: *kind, Cusecs: cusecs, Agency: agency, Day: flowDay, Stale: stale}
}

// flowStale says whether a flow figure is too old: by its own day, or the
// reading's when it gives none.
func flowStale(flowDay, readingDay pgtype.Date, now time.Time, limit time.Duration) bool {
	if flowDay.Valid {
		return IsStale(flowDay, now, limit)
	}
	return IsStale(readingDay, now, limit)
}

// Areas is a GeoJSON feature collection of the mapped water bodies.
type Areas struct {
	Type     string        `json:"type"`
	Features []AreaFeature `json:"features"`
}

// AreaFeature is one reservoir's polygon, carrying what the map colours it by.
type AreaFeature struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties AreaProperties  `json:"properties"`
}

// AreaProperties are a polygon's properties.
type AreaProperties struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Stale  bool   `json:"stale"`
}

// Areas returns the polygons of mapped reservoirs, simplified for the map.
func (service *Service) Areas(ctx context.Context) (Areas, error) {
	layer, err := service.Layer(ctx)
	if err != nil {
		return Areas{}, err
	}
	byID := make(map[string]Properties, len(layer.Features))
	for _, feature := range layer.Features {
		byID[feature.ID] = feature.Properties
	}
	rows, err := store.New(service.Pool).ReservoirAreas(ctx)
	if err != nil {
		return Areas{}, fmt.Errorf("read reservoir areas: %w", err)
	}
	areas := Areas{Type: "FeatureCollection", Features: make([]AreaFeature, 0, len(rows))}
	for _, row := range rows {
		properties, ok := byID[row.ID]
		if !ok {
			continue
		}
		areas.Features = append(areas.Features, AreaFeature{
			Type: "Feature", ID: row.ID, Geometry: json.RawMessage(row.Geometry),
			Properties: AreaProperties{ID: row.ID, Name: properties.Name, Status: properties.Status, Stale: properties.Stale},
		})
	}
	return areas, nil
}

// History is one reservoir's daily readings.
type History struct {
	Reservoir Properties `json:"reservoir"`
	Note      string     `json:"note"`
	Readings  []Reading  `json:"readings"`
}

// Reading is one day read.
type Reading struct {
	Day            string   `json:"day"`
	Status         string   `json:"status"`
	StoragePercent *float64 `json:"storagePercent"`
	StorageFresh   *bool    `json:"storageFresh"`
	Flow           *Flow    `json:"flow"`
	Agency         *string  `json:"agency"`
}

// History returns a reservoir's readings over the last days, oldest first.
func (service *Service) History(ctx context.Context, id string, days int) (History, error) {
	now := service.now()
	queries := store.New(service.Pool)
	row, err := queries.GetReservoir(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return History{}, ErrNotFound
	}
	if err != nil {
		return History{}, fmt.Errorf("read reservoir: %w", err)
	}
	since := waterlevels.SnapshotDay(now).AddDate(0, 0, -days)
	rows, err := queries.ReservoirHistory(ctx, store.ReservoirHistoryParams{
		ReservoirID: id, ReadingDay: pgtype.Date{Time: since, Valid: true},
	})
	if err != nil {
		return History{}, fmt.Errorf("read readings: %w", err)
	}
	history := History{
		Note: Note,
		Reservoir: Properties{
			ID: row.ID, Name: row.Name, District: row.District, River: row.River, Type: row.Kind,
			Status: row.Status, StoragePercent: row.StoragePercent, StorageFresh: row.StorageFresh,
			Flow:       flow(row.FlowKind, row.FlowCusecs, row.FlowAgency, day(row.FlowDay), flowStale(row.FlowDay, row.ReadingDay, now, service.StaleAfter)),
			ReadingDay: day(row.ReadingDay),
			Agency:     row.Agency, Source: row.Source, SourceURL: row.SourceUrl, FetchedAt: row.FetchedAt,
			Stale: IsStale(row.ReadingDay, now, service.StaleAfter), Mapped: row.Lon != nil,
			OSMName: row.OsmName, OSMKind: row.OsmKind, AltNames: osmreservoirs.Aliases[row.ID],
		},
		Readings: make([]Reading, 0, len(rows)),
	}
	for _, reading := range rows {
		history.Readings = append(history.Readings, Reading{
			Day: reading.ReadingDay.Time.Format(time.DateOnly), Status: reading.Status,
			StoragePercent: reading.StoragePercent, StorageFresh: reading.StorageFresh,
			Flow: flow(reading.FlowKind, reading.FlowCusecs, reading.FlowAgency, nil, false), Agency: reading.Agency,
		})
	}
	return history, nil
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
