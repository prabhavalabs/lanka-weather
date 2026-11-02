// Package raingauges serves the live rain gauges the hmisrain collector stores:
// about 275 devices run by NBRO, the Department of Meteorology, the Irrigation
// Department and ACCIMT, each with a rolling 24-hour total. Nothing shorter than
// 24 hours can be derived from a rolling sum, and none is.
//
// A gauge whose newest reading is old is out of date, not dry: its total is left out
// and only given separately as the last known figure.
package raingauges

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/hmisrain"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// Bands of a 24-hour total, as waterlevels.lk words them.
const (
	BandUnknown   = "unknown"
	BandNone      = "none"
	BandLight     = "light"
	BandModerate  = "moderate"
	BandHeavy     = "heavy"
	BandVeryHeavy = "very_heavy"

	lightBelowMm    = 10.0
	moderateBelowMm = 50.0
	heavyBelowMm    = 100.0

	// NearbyKm bounds the nearest gauge offered for a place.
	NearbyKm = 30.0
	// seriesWindow is the history a gauge's panel shows.
	seriesWindow = 48 * time.Hour

	// Note says what these figures are, wherever they are served.
	Note = "Rolling 24-hour totals measured at each gauge. Not an official warning."
)

// ErrNotFound means no such gauge.
var ErrNotFound = errors.New("no such rain gauge")

// BandOf names a 24-hour total's band: light under 10 mm, moderate 10 to 50, heavy
// 50 to 100, very heavy 100 or more.
func BandOf(mm float64) string {
	switch {
	case mm <= 0:
		return BandNone
	case mm < lightBelowMm:
		return BandLight
	case mm < moderateBelowMm:
		return BandModerate
	case mm < heavyBelowMm:
		return BandHeavy
	}
	return BandVeryHeavy
}

// Service reads the stored gauges.
type Service struct {
	Pool *pgxpool.Pool
	// StaleAfter is the age of the newest reading after which a gauge is out of date.
	StaleAfter time.Duration
	Now        func() time.Time
}

// FeatureCollection is GET /v1/rain/gauges.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
	Meta     Meta      `json:"meta"`
}

// Meta describes the gauges' origin and how many are current.
type Meta struct {
	Source      string `json:"source"`
	Attribution string `json:"attribution"`
	Note        string `json:"note"`
	// FetchedAt is the last time the file was stored, and SourceTime the file's own.
	FetchedAt   *time.Time `json:"fetchedAt"`
	SourceTime  *time.Time `json:"sourceTime"`
	GeneratedAt time.Time  `json:"generatedAt"`
	// StaleAfterSeconds is how old a reading may be.
	StaleAfterSeconds int `json:"staleAfterSeconds"`
	Total             int `json:"total"`
	Fresh             int `json:"fresh"`
	Stale             int `json:"stale"`
	// Wet counts current gauges with a meaningful total, 10 mm or more.
	Wet int `json:"wet"`
	// Institutions counts gauges by who runs them.
	Institutions map[string]int `json:"institutions"`
}

// Feature is one gauge.
type Feature struct {
	Type       string     `json:"type"`
	ID         string     `json:"id"`
	Geometry   Point      `json:"geometry"`
	Properties Properties `json:"properties"`
}

// Point is a GeoJSON point in longitude, latitude order.
type Point struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

// Properties describe a gauge.
type Properties struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Institution string `json:"institution"`
	// Rain24hMm is the current rolling 24-hour total; null when out of date, so it
	// cannot read as dry. LastRain24hMm is the newest figure there is, however old.
	Rain24hMm     *float64 `json:"rain24hMm"`
	LastRain24hMm *float64 `json:"lastRain24hMm"`
	// Band is light, moderate, heavy, very_heavy, none or, when out of date, unknown.
	Band       string     `json:"band"`
	ObservedAt *time.Time `json:"observedAt"`
	Stale      bool       `json:"stale"`
	Source     string     `json:"source"`
	SourceTime time.Time  `json:"sourceTime"`
	FetchedAt  time.Time  `json:"fetchedAt"`
}

// SeriesPoint is one reading of a gauge's history.
type SeriesPoint struct {
	At        time.Time `json:"at"`
	Rain24hMm float64   `json:"rain24hMm"`
}

// Detail is GET /v1/rain/gauges/{id}.
type Detail struct {
	Gauge Feature `json:"gauge"`
	// Series is up to 48 hours of the rolling total, oldest first.
	Series []SeriesPoint `json:"series"`
	Meta   Meta          `json:"meta"`
}

// Nearby is the gauge nearest a place.
type Nearby struct {
	Properties
	DistanceKm  float64 `json:"distanceKm"`
	Lon         float64 `json:"lon"`
	Lat         float64 `json:"lat"`
	Attribution string  `json:"attribution"`
}

func (service *Service) properties(row store.RainGauge, now time.Time) Properties {
	out := Properties{ID: row.ID, Name: row.Name, Institution: row.Institution, ObservedAt: row.LatestObservedAt, LastRain24hMm: row.LatestRainMm,
		Band: BandUnknown, Stale: true, Source: row.Source, SourceTime: row.SourceTime.UTC(), FetchedAt: row.FetchedAt.UTC()}
	if row.LatestObservedAt != nil && row.LatestRainMm != nil && now.Sub(*row.LatestObservedAt) <= service.StaleAfter {
		rain := math.Round(*row.LatestRainMm*10) / 10
		out.Rain24hMm, out.Band, out.Stale = &rain, BandOf(rain), false
	}
	return out
}

// Layer returns every gauge.
func (service *Service) Layer(ctx context.Context) (FeatureCollection, error) {
	now := service.now()
	rows, err := store.New(service.Pool).ListRainGauges(ctx)
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("list rain gauges: %w", err)
	}
	collection := FeatureCollection{Type: "FeatureCollection", Features: make([]Feature, 0, len(rows)), Meta: Meta{
		Source: hmisrain.SourceName, Attribution: hmisrain.Attribution, Note: Note, GeneratedAt: now,
		StaleAfterSeconds: int(service.StaleAfter / time.Second), Total: len(rows), Institutions: map[string]int{},
	}}
	for _, row := range rows {
		properties := service.properties(row, now)
		collection.Features = append(collection.Features, Feature{Type: "Feature", ID: row.ID,
			Geometry: Point{Type: "Point", Coordinates: [2]float64{row.Lon, row.Lat}}, Properties: properties})
		collection.Meta.Institutions[row.Institution]++
		if properties.Stale {
			collection.Meta.Stale++
		} else {
			collection.Meta.Fresh++
			if *properties.Rain24hMm >= lightBelowMm {
				collection.Meta.Wet++
			}
		}
		fetched, sourced := row.FetchedAt.UTC(), row.SourceTime.UTC()
		if collection.Meta.FetchedAt == nil || fetched.After(*collection.Meta.FetchedAt) {
			collection.Meta.FetchedAt, collection.Meta.SourceTime = &fetched, &sourced
		}
	}
	return collection, nil
}

// Gauge returns one gauge and its recent readings.
func (service *Service) Gauge(ctx context.Context, id string) (*Detail, error) {
	now := service.now()
	queries := store.New(service.Pool)
	row, err := queries.GetRainGauge(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read rain gauge: %w", err)
	}
	readings, err := queries.RainGaugeReadingsSince(ctx, store.RainGaugeReadingsSinceParams{GaugeID: id, ObservedAt: now.Add(-seriesWindow)})
	if err != nil {
		return nil, fmt.Errorf("read rain gauge readings: %w", err)
	}
	detail := &Detail{Gauge: Feature{Type: "Feature", ID: row.ID, Geometry: Point{Type: "Point", Coordinates: [2]float64{row.Lon, row.Lat}},
		Properties: service.properties(row, now)}, Series: make([]SeriesPoint, 0, len(readings))}
	for _, reading := range readings {
		detail.Series = append(detail.Series, SeriesPoint{At: reading.ObservedAt.UTC(), Rain24hMm: reading.RainMm})
	}
	fetched, sourced := row.FetchedAt.UTC(), row.SourceTime.UTC()
	detail.Meta = Meta{Source: hmisrain.SourceName, Attribution: hmisrain.Attribution, Note: Note, GeneratedAt: now,
		FetchedAt: &fetched, SourceTime: &sourced, StaleAfterSeconds: int(service.StaleAfter / time.Second)}
	return detail, nil
}

// Nearest finds the gauge nearest a place within 30 km: the nearest current one, and
// only if none is current the nearest out-of-date one, which says so.
func (service *Service) Nearest(ctx context.Context, lon, lat float64) (*Nearby, error) {
	layer, err := service.Layer(ctx)
	if err != nil {
		return nil, err
	}
	return NearestIn(layer, lon, lat), nil
}

// NearestIn is Nearest over a layer already read.
func NearestIn(layer FeatureCollection, lon, lat float64) *Nearby {
	var fresh, any *Nearby
	for _, feature := range layer.Features {
		distance := distanceKm(lon, lat, feature.Geometry.Coordinates[0], feature.Geometry.Coordinates[1])
		if distance > NearbyKm {
			continue
		}
		near := &Nearby{Properties: feature.Properties, DistanceKm: math.Round(distance*10) / 10, Lon: feature.Geometry.Coordinates[0],
			Lat: feature.Geometry.Coordinates[1], Attribution: layer.Meta.Attribution}
		if any == nil || near.DistanceKm < any.DistanceKm {
			any = near
		}
		if !feature.Properties.Stale && (fresh == nil || near.DistanceKm < fresh.DistanceKm) {
			fresh = near
		}
	}
	if fresh != nil {
		return fresh
	}
	return any
}

// Wettest lists the current gauges with rain, wettest first, up to limit.
func Wettest(layer FeatureCollection, limit int) []Feature {
	wet := make([]Feature, 0, limit)
	for _, feature := range layer.Features {
		if feature.Properties.Rain24hMm != nil && *feature.Properties.Rain24hMm > 0 {
			wet = append(wet, feature)
		}
	}
	sort.SliceStable(wet, func(i, j int) bool { return *wet[i].Properties.Rain24hMm > *wet[j].Properties.Rain24hMm })
	if len(wet) > limit {
		wet = wet[:limit]
	}
	return wet
}

func distanceKm(lon1, lat1, lon2, lat2 float64) float64 {
	const earthRadiusKm = 6371.0088
	toRadians := math.Pi / 180
	dLat, dLon := (lat2-lat1)*toRadians, (lon2-lon1)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
