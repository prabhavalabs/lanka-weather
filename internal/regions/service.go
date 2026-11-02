package regions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// ErrRegionNotFound is returned for an unknown region code.
var ErrRegionNotFound = errors.New("region not found")

// TileMinZoom is the lowest zoom at which each level's layer is in the
// vector tiles. Lower zooms would draw thousands of lines a few pixels apart.
var TileMinZoom = map[Level]int{
	LevelProvince: 0,
	LevelDistrict: 6,
	LevelDS:       8,
	LevelGN:       10,
}

const (
	// TileMaxZoom is the highest zoom served; boundaries carry no more
	// detail beyond it, so clients overzoom.
	TileMaxZoom = 14
	// simpleBelowZoom: below this zoom tiles use the boundaries simplified
	// per level; from it they use the full boundaries.
	simpleBelowZoom = 9
	// formatVersion changes when the shape of responses changes, so entity
	// tags change even if the data did not.
	formatVersion  = "1"
	versionMemo    = 30 * time.Second
	tileCacheBytes = 64 << 20
)

// Service reads regions from the database. It caches rendered GeoJSON and
// tiles in memory by data version.
type Service struct {
	Pool *pgxpool.Pool

	once  sync.Once
	cache *byteCache

	mu          sync.Mutex
	version     Version
	versionRead time.Time
}

// Version identifies the loaded data.
type Version struct {
	// Tag changes whenever the boundaries, the populations or the response
	// format change; it is used as the entity tag.
	Tag               string
	BoundaryVersion   string
	PopulationVersion string
	SourceTime        *time.Time
	FetchedAt         *time.Time
}

// Document is a rendered response body with its entity tag.
type Document struct {
	Body []byte
	ETag string
}

func (service *Service) init() {
	service.once.Do(func() { service.cache = newByteCache(tileCacheBytes) })
}

// Version returns the version of the loaded data, read at most every
// 30 seconds.
func (service *Service) Version(ctx context.Context) (Version, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if !service.versionRead.IsZero() && time.Since(service.versionRead) < versionMemo {
		return service.version, nil
	}
	queries := store.New(service.Pool)
	version := Version{Tag: formatVersion + "-empty"}
	boundary, err := queries.GetSourceVersion(ctx, BoundarySource)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return Version{}, fmt.Errorf("read boundary version: %w", err)
	default:
		population, err := queries.GetSourceVersion(ctx, PopulationSource)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return Version{}, fmt.Errorf("read population version: %w", err)
		}
		version = Version{
			Tag:               formatVersion + "-" + short(boundary.Version) + "-" + short(population.Version) + "-" + strconv.FormatInt(boundary.FetchedAt.Unix(), 36),
			BoundaryVersion:   boundary.Version,
			PopulationVersion: population.Version,
			SourceTime:        &boundary.SourceTime,
			FetchedAt:         &boundary.FetchedAt,
		}
	}
	service.version, service.versionRead = version, time.Now()
	return version, nil
}

func short(version string) string {
	if len(version) > 12 {
		return version[:12]
	}
	return version
}

// FeatureCollection is a GeoJSON feature collection of regions.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
	Meta     Meta      `json:"meta"`
}

// Feature is one region with its simplified boundary.
type Feature struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties Properties      `json:"properties"`
}

// Properties are a region feature's properties.
type Properties struct {
	ID             string     `json:"id"`
	Level          Level      `json:"level"`
	Name           string     `json:"name"`
	NameSI         *string    `json:"nameSi"`
	NameTA         *string    `json:"nameTa"`
	ParentID       *string    `json:"parentId"`
	ProvinceID     string     `json:"provinceId"`
	DistrictID     *string    `json:"districtId"`
	DSID           *string    `json:"dsId"`
	ISOCode        *string    `json:"isoCode"`
	GNNumber       *string    `json:"gnNumber"`
	AreaKM2        float64    `json:"areaKm2"`
	Population     *int32     `json:"population"`
	PopulationYear *int16     `json:"populationYear"`
	LabelPoint     [2]float64 `json:"labelPoint"`
}

// Meta describes the data's origin and version.
type Meta struct {
	Source            string     `json:"source"`
	PopulationSource  string     `json:"populationSource"`
	Attribution       string     `json:"attribution"`
	Version           string     `json:"version"`
	SourceVersion     string     `json:"sourceVersion"`
	SourceTime        *time.Time `json:"sourceTime"`
	FetchedAt         *time.Time `json:"fetchedAt"`
	SimplifiedDegrees float64    `json:"simplifiedDegrees"`
}

// Collection returns the regions of a level, optionally only those within an
// ancestor region, as a GeoJSON feature collection with simplified
// boundaries. Collections of a whole level are cached.
func (service *Service) Collection(ctx context.Context, level Level, ancestor string) (Document, error) {
	service.init()
	version, err := service.Version(ctx)
	if err != nil {
		return Document{}, err
	}
	key := "collection/" + version.Tag + "/" + string(level) + "/" + ancestor
	etag := `W/"` + version.Tag + `-` + string(level) + `-` + ancestor + `"`
	if body, ok := service.cache.get(key); ok {
		return Document{Body: body, ETag: etag}, nil
	}
	rows, err := store.New(service.Pool).ListRegionFeatures(ctx, store.ListRegionFeaturesParams{Level: string(level), Ancestor: ancestor})
	if err != nil {
		return Document{}, fmt.Errorf("list %s regions: %w", level, err)
	}
	collection := FeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]Feature, 0, len(rows)),
		Meta:     service.meta(version, Tolerances[level]),
	}
	for _, row := range rows {
		collection.Features = append(collection.Features, Feature{
			Type: "Feature", ID: row.ID, Geometry: json.RawMessage(row.Geometry),
			Properties: Properties{
				ID: row.ID, Level: Level(row.Level), Name: row.Name, NameSI: row.NameSi, NameTA: row.NameTa,
				ParentID: row.ParentID, ProvinceID: row.ProvinceID, DistrictID: row.DistrictID, DSID: row.DsID,
				ISOCode: row.IsoCode, GNNumber: row.GnNumber, AreaKM2: round(row.AreaKm2, 2),
				Population: row.Population, PopulationYear: row.PopulationYear,
				LabelPoint: [2]float64{round(row.LabelLon, 6), round(row.LabelLat, 6)},
			},
		})
	}
	body, err := json.Marshal(collection)
	if err != nil {
		return Document{}, err
	}
	service.cache.put(key, body)
	return Document{Body: body, ETag: etag}, nil
}

func (service *Service) meta(version Version, tolerance float64) Meta {
	return Meta{
		Source: BoundarySource, PopulationSource: PopulationSource, Attribution: Attribution,
		Version: version.Tag, SourceVersion: version.BoundaryVersion,
		SourceTime: version.SourceTime, FetchedAt: version.FetchedAt, SimplifiedDegrees: tolerance,
	}
}

// Ref names a region.
type Ref struct {
	ID     string  `json:"id"`
	Level  Level   `json:"level"`
	Name   string  `json:"name"`
	NameSI *string `json:"nameSi,omitempty"`
	NameTA *string `json:"nameTa,omitempty"`
}

// Names are a region's names in English, Sinhala and Tamil.
type Names struct {
	EN string  `json:"en"`
	SI *string `json:"si"`
	TA *string `json:"ta"`
}

// Detail describes one region.
type Detail struct {
	ID       string  `json:"id"`
	Level    Level   `json:"level"`
	Name     string  `json:"name"`
	Names    Names   `json:"names"`
	ISOCode  *string `json:"isoCode"`
	GNNumber *string `json:"gnNumber"`
	ParentID *string `json:"parentId"`
	// Hierarchy lists the regions containing this one, province first.
	Hierarchy  []Ref `json:"hierarchy"`
	ChildCount int   `json:"childCount"`
	// Centroid is the source's area-weighted centre; LabelPoint is inside
	// the region. Both are [longitude, latitude].
	Centroid   [2]float64 `json:"centroid"`
	LabelPoint [2]float64 `json:"labelPoint"`
	// BBox is [west, south, east, north].
	BBox           [4]float64      `json:"bbox"`
	AreaKM2        float64         `json:"areaKm2"`
	Population     *int32          `json:"population"`
	PopulationYear *int16          `json:"populationYear"`
	Geometry       json.RawMessage `json:"geometry,omitempty"`
	Source         SourceInfo      `json:"source"`
}

// SourceInfo is a record's provenance.
type SourceInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	SourceTime  time.Time `json:"sourceTime"`
	FetchedAt   time.Time `json:"fetchedAt"`
	Attribution string    `json:"attribution"`
}

// Region returns one region. With geometry, the detail includes its
// simplified boundary.
func (service *Service) Region(ctx context.Context, id string, geometry bool) (Detail, error) {
	queries := store.New(service.Pool)
	row, err := queries.GetRegion(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrRegionNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("read region: %w", err)
	}
	detail := Detail{
		ID: row.ID, Level: Level(row.Level), Name: row.Name,
		Names:   Names{EN: row.Name, SI: row.NameSi, TA: row.NameTa},
		ISOCode: row.IsoCode, GNNumber: row.GnNumber, ParentID: row.ParentID,
		Hierarchy: []Ref{}, ChildCount: int(row.ChildCount),
		Centroid:   [2]float64{round(row.CentroidLon, 6), round(row.CentroidLat, 6)},
		LabelPoint: [2]float64{round(row.LabelLon, 6), round(row.LabelLat, 6)},
		BBox:       [4]float64{round(row.West, 6), round(row.South, 6), round(row.East, 6), round(row.North, 6)},
		AreaKM2:    round(row.AreaKm2, 2), Population: row.Population, PopulationYear: row.PopulationYear,
		Source: SourceInfo{
			Name: row.Source, Version: row.SourceVersion, SourceTime: row.SourceTime,
			FetchedAt: row.FetchedAt, Attribution: Attribution,
		},
	}
	var ancestors []string
	for _, ancestor := range []*string{&row.ProvinceID, row.DistrictID, row.DsID} {
		if ancestor != nil && *ancestor != row.ID {
			ancestors = append(ancestors, *ancestor)
		}
	}
	if len(ancestors) > 0 {
		names, err := queries.RegionNames(ctx, ancestors)
		if err != nil {
			return Detail{}, fmt.Errorf("read hierarchy: %w", err)
		}
		byID := make(map[string]store.RegionNamesRow, len(names))
		for _, name := range names {
			byID[name.ID] = name
		}
		for _, ancestor := range ancestors {
			if name, ok := byID[ancestor]; ok {
				detail.Hierarchy = append(detail.Hierarchy, Ref{ID: name.ID, Level: Level(name.Level), Name: name.Name, NameSI: name.NameSi, NameTA: name.NameTa})
			}
		}
	}
	if geometry {
		shape, err := queries.GetRegionGeometry(ctx, id)
		if err != nil {
			return Detail{}, fmt.Errorf("read geometry: %w", err)
		}
		detail.Geometry = json.RawMessage(shape)
	}
	return detail, nil
}

// Lookup is the region containing a point at each level. A level is nil
// when the point is outside every region of it, for example at sea.
type Lookup struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Province *Ref    `json:"province"`
	District *Ref    `json:"district"`
	DS       *Ref    `json:"ds"`
	GN       *Ref    `json:"gn"`
}

// Lookup finds the regions containing a point.
func (service *Service) Lookup(ctx context.Context, lon, lat float64) (Lookup, error) {
	rows, err := store.New(service.Pool).LookupRegions(ctx, store.LookupRegionsParams{Lon: lon, Lat: lat})
	if err != nil {
		return Lookup{}, fmt.Errorf("look up point: %w", err)
	}
	lookup := Lookup{Lat: lat, Lon: lon}
	for _, row := range rows {
		ref := &Ref{ID: row.ID, Level: Level(row.Level), Name: row.Name, NameSI: row.NameSi, NameTA: row.NameTa}
		slot := map[Level]**Ref{LevelProvince: &lookup.Province, LevelDistrict: &lookup.District, LevelDS: &lookup.DS, LevelGN: &lookup.GN}[ref.Level]
		// On a shared border the first region by code is kept.
		if slot != nil && *slot == nil {
			*slot = ref
		}
	}
	return lookup, nil
}

// TileLevels returns the levels drawn at a zoom.
func TileLevels(z int) []string {
	var levels []string
	for _, level := range Levels {
		if z >= TileMinZoom[level] {
			levels = append(levels, string(level))
		}
	}
	return levels
}

// Tile returns the Mapbox Vector Tile at z/x/y; its body is empty when no
// region crosses the tile.
func (service *Service) Tile(ctx context.Context, z, x, y int) (Document, error) {
	service.init()
	version, err := service.Version(ctx)
	if err != nil {
		return Document{}, err
	}
	key := fmt.Sprintf("tile/%s/%d/%d/%d", version.Tag, z, x, y)
	etag := `W/"` + version.Tag + `"`
	if body, ok := service.cache.get(key); ok {
		return Document{Body: body, ETag: etag}, nil
	}
	body, err := store.New(service.Pool).RegionTile(ctx, store.RegionTileParams{
		Z: int32(z), X: int32(x), Y: int32(y), Simple: z < simpleBelowZoom, Levels: TileLevels(z),
	})
	if err != nil {
		return Document{}, fmt.Errorf("render tile %d/%d/%d: %w", z, x, y, err)
	}
	service.cache.put(key, body)
	return Document{Body: body, ETag: etag}, nil
}

func round(value float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(value*scale) / scale
}
