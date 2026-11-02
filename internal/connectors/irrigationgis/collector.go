package irrigationgis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/geo"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// DefaultBase is the department's public ArcGIS organisation.
const DefaultBase = "https://services3.arcgis.com/J7ZFXmR8rSmQ3FGf/arcgis/rest/services"

const (
	pageSize = 2000
	// extentOffset is the generalisation of flood extents, in degrees.
	extentOffset = 0.00005
	maxPages     = 40 // the Puttalam map is 16 pages
	maxBytes     = 256 << 20
	fetchWait    = 3 * time.Minute
	// Guards against a truncated or emptied service replacing good data.
	minimumBasins    = 80
	minimumEvents    = 15
	unionTimeout     = "4min"
	inundationLayer  = "Flood Inundation Area"
	floodMapService  = "Flood_Map"
	novemberService  = "Flood__Nov2025"
	puttalamService  = "Puttalam_Flood__Detwah_30Nov2025"
	basinsService    = "river_basins"
	streamsService   = "Srilanka_River"
	stationsService  = "hydrostations"
	novemberEventDay = 30
)

// reader queries the department's services.
type reader struct {
	fetcher source.Fetcher
	base    string
}

func (r reader) url(service string, layer int, suffix string) string {
	base := r.base
	if base == "" {
		base = DefaultBase
	}
	return base + "/" + service + "/FeatureServer/" + strconv.Itoa(layer) + suffix
}

func (r reader) get(ctx context.Context, address string, query url.Values) ([]byte, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("f", "json")
	return r.fetcher.Fetch(ctx, source.Request{URL: address, Query: query, MaxBytes: maxBytes, Timeout: fetchWait})
}

// features pages through a layer in WGS 84, whatever the layer's own spatial
// reference (several use EPSG:5234).
func (r reader) features(ctx context.Context, service string, layer int, fields string, geometry bool) ([]Feature, error) {
	return r.pages(ctx, service, layer, fields, geometry, pageSize, 0)
}

// extents pages through flood extents, generalised to about 5 m: raster
// polygons carry far more vertices than a map can show. A layer of very large
// features is asked for a few at a time, or the service times out.
func (r reader) extents(ctx context.Context, service string, layer int, fields string, records int) ([]Feature, error) {
	return r.pages(ctx, service, layer, fields, true, records, extentOffset)
}

func (r reader) pages(ctx context.Context, service string, layer int, fields string, geometry bool, records int, offset float64) ([]Feature, error) {
	var all []Feature
	for page := range maxPages {
		query := url.Values{
			"where": {"1=1"}, "outFields": {fields}, "outSR": {"4326"}, "returnGeometry": {strconv.FormatBool(geometry)},
			"resultOffset": {strconv.Itoa(page * records)}, "resultRecordCount": {strconv.Itoa(records)},
		}
		if offset > 0 {
			query.Set("maxAllowableOffset", strconv.FormatFloat(offset, 'f', -1, 64))
			query.Set("geometryPrecision", "6")
		}
		payload, err := r.get(ctx, r.url(service, layer, "/query"), query)
		if err != nil {
			return nil, fmt.Errorf("%s/%d page %d: %w", service, layer, page, err)
		}
		features, more, err := DecodePage(payload)
		if err != nil {
			return nil, fmt.Errorf("%s/%d page %d: %w", service, layer, page, err)
		}
		all = append(all, features...)
		if !more || len(features) == 0 {
			return all, nil
		}
	}
	return nil, fmt.Errorf("%s/%d has more than %d pages", service, layer, maxPages)
}

// version names the state of some layers from when they were last edited.
func (r reader) version(ctx context.Context, service string, layer int) (int64, error) {
	payload, err := r.get(ctx, r.url(service, layer, ""), nil)
	if err != nil {
		return 0, fmt.Errorf("%s/%d: %w", service, layer, err)
	}
	return DecodeLastEdit(payload), nil
}

func newest(times ...int64) time.Time {
	var latest int64
	for _, at := range times {
		latest = max(latest, at)
	}
	return time.UnixMilli(latest).UTC()
}

func versionOf(times ...int64) string {
	parts := make([]string, len(times))
	for index, at := range times {
		parts[index] = strconv.FormatInt(at, 10)
	}
	return strings.Join(parts, "-")
}

// unchanged reports whether a version is already loaded. A layer that does not
// say when it was edited reads as changed.
func unchanged(ctx context.Context, queries *store.Queries, name, version string, count func(context.Context) (int32, error)) (bool, error) {
	if strings.Contains("-"+version+"-", "-0-") {
		return false, nil
	}
	stored, err := queries.GetSourceVersion(ctx, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s version: %w", name, err)
	}
	if stored.Version != version {
		return false, nil
	}
	records, err := count(ctx)
	return err == nil && records > 0, nil
}

// BasinsCollector loads the official river basins, the named streams and the
// station list, which together give the map its boundaries and names. It
// checks weekly and reloads only what changed upstream.
type BasinsCollector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// Base overrides DefaultBase, and Minimum minimumBasins, for tests.
	Base    string
	Minimum int
	Logger  *slog.Logger
}

// Name implements source.Collector.
func (collector *BasinsCollector) Name() string { return BasinsSource }

// Interval implements source.Collector. river_basins was last edited in 2022.
func (collector *BasinsCollector) Interval() time.Duration { return 7 * 24 * time.Hour }

// RetryAfter implements source.Retrier.
func (collector *BasinsCollector) RetryAfter() time.Duration { return 6 * time.Hour }

// Collect implements source.Collector.
func (collector *BasinsCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	r := reader{fetcher: collector.Fetcher, base: collector.Base}
	basinsEdit, err := r.version(ctx, basinsService, 0)
	if err != nil {
		return 0, err
	}
	streamsEdit, err := r.version(ctx, streamsService, 0)
	if err != nil {
		return 0, err
	}
	stationsEdit, err := r.version(ctx, stationsService, 0)
	if err != nil {
		return 0, err
	}
	version := versionOf(basinsEdit, streamsEdit, stationsEdit)
	queries := store.New(collector.Pool)
	if same, err := unchanged(ctx, queries, BasinsSource, version, queries.CountOfficialBasins); err != nil {
		return 0, err
	} else if same {
		return 0, nil
	}

	features, err := r.features(ctx, basinsService, 0, "FID,Name", true)
	if err != nil {
		return 0, err
	}
	basins := CleanBasins(features)
	minimum := collector.Minimum
	if minimum <= 0 {
		minimum = minimumBasins
	}
	if len(basins) < minimum {
		return 0, fmt.Errorf("river_basins gave %d basins, want at least %d", len(basins), minimum)
	}
	features, err = r.features(ctx, streamsService, 0, "FID,NAME", true)
	if err != nil {
		return 0, err
	}
	streams := DecodeStreams(features)
	features, err = r.features(ctx, stationsService, 0, "station,basin,Tributory,latitude,longitude", false)
	if err != nil {
		return 0, err
	}
	stations := DecodeStations(features)

	sourceTime := newest(basinsEdit, streamsEdit, stationsEdit)
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '"+unionTimeout+"'"); err != nil {
			return err
		}
		queries := store.New(tx)
		if err := queries.DeleteOfficialBasins(ctx); err != nil {
			return err
		}
		for _, basin := range basins {
			if err := queries.InsertOfficialBasin(ctx, store.InsertOfficialBasinParams{
				Fid: basin.FID, ID: basin.ID, Name: basin.Name, Named: basin.Named, Source: BasinsSource,
				Version: version, SourceTime: sourceTime, FetchedAt: now, Wkb: basin.Shape.WKB(),
			}); err != nil {
				return fmt.Errorf("store basin %s: %w", basin.ID, err)
			}
		}
		if err := queries.DeleteOfficialStreamNames(ctx); err != nil {
			return err
		}
		for _, stream := range streams {
			if err := queries.InsertOfficialStreamName(ctx, store.InsertOfficialStreamNameParams{
				Fid: stream.FID, Name: stream.Name, Source: BasinsSource, FetchedAt: now, Wkb: stream.Shape.WKB(),
			}); err != nil {
				return fmt.Errorf("store stream %d: %w", stream.FID, err)
			}
		}
		if err := queries.DeleteOfficialStations(ctx); err != nil {
			return err
		}
		for _, station := range stations {
			if err := queries.InsertOfficialStation(ctx, store.InsertOfficialStationParams{
				Station: station.Name, Basin: station.Basin, Tributary: station.Tributary,
				Lon: station.Lon, Lat: station.Lat, Source: BasinsSource, FetchedAt: now,
			}); err != nil {
				return fmt.Errorf("store station %s: %w", station.Name, err)
			}
		}
		if _, err := queries.AssignSubbasinsToOfficialBasins(ctx); err != nil {
			return fmt.Errorf("assign sub-basins: %w", err)
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: BasinsSource, Version: version, SourceTime: sourceTime, FetchedAt: now, Records: int32(len(basins)),
		})
	})
	if err != nil {
		return 0, err
	}
	if collector.Logger != nil {
		collector.Logger.Info("official basins stored", "basins", len(basins), "streams", len(streams), "stations", len(stations))
	}
	return len(basins), nil
}

// PastFloodsCollector loads the mapped extents of past floods. It checks
// monthly, and runs apart from other collectors (source.LongRunner) because
// one service is 16 pages of raster cells.
type PastFloodsCollector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// Base overrides DefaultBase, and Minimum minimumEvents, for tests.
	Base    string
	Minimum int
	Logger  *slog.Logger
}

// Name implements source.Collector.
func (collector *PastFloodsCollector) Name() string { return PastFloodsSource }

// Interval implements source.Collector. New maps appear a few times a year.
func (collector *PastFloodsCollector) Interval() time.Duration { return 30 * 24 * time.Hour }

// RetryAfter implements source.Retrier.
func (collector *PastFloodsCollector) RetryAfter() time.Duration { return 24 * time.Hour }

// LongRunTimeout implements source.LongRunner.
func (collector *PastFloodsCollector) LongRunTimeout() time.Duration { return 20 * time.Minute }

// Collect implements source.Collector.
func (collector *PastFloodsCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	r := reader{fetcher: collector.Fetcher, base: collector.Base}
	payload, err := r.get(ctx, strings.TrimSuffix(r.url(floodMapService, 0, ""), "/0"), nil)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", floodMapService, err)
	}
	layers, err := DecodeServiceLayers(payload)
	if err != nil {
		return 0, err
	}
	var inundation []LayerInfo
	for _, layer := range layers {
		if strings.EqualFold(layer.Name, inundationLayer) {
			inundation = append(inundation, layer)
		}
	}
	if len(inundation) == 0 {
		return 0, fmt.Errorf("%s has no %q layers", floodMapService, inundationLayer)
	}
	mapEdit, err := r.version(ctx, floodMapService, inundation[0].ID)
	if err != nil {
		return 0, err
	}
	novemberEdit, err := r.version(ctx, novemberService, 0)
	if err != nil {
		return 0, err
	}
	puttalamEdit, err := r.version(ctx, puttalamService, 0)
	if err != nil {
		return 0, err
	}
	version := versionOf(mapEdit, novemberEdit, puttalamEdit)
	queries := store.New(collector.Pool)
	if same, err := unchanged(ctx, queries, PastFloodsSource, version, queries.CountPastFloodEvents); err != nil {
		return 0, err
	} else if same {
		return 0, nil
	}

	events := map[string]*PastEvent{}
	var order []string
	add := func(event PastEvent, shape geo.MultiPolygon) {
		if len(shape) == 0 {
			return
		}
		if existing, ok := events[event.ID]; ok {
			existing.Shape = append(existing.Shape, shape...)
			return
		}
		event.Shape = shape
		events[event.ID] = &event
		order = append(order, event.ID)
	}
	var skipped []string
	for _, layer := range inundation {
		features, err := r.extents(ctx, floodMapService, layer.ID, "Flooding_area", pageSize)
		if err != nil {
			return 0, err
		}
		for _, feature := range features {
			event, ok := ParseEvent(feature.Text("Flooding_area"))
			if !ok {
				skipped = append(skipped, feature.Text("Flooding_area"))
				continue
			}
			event.Service = floodMapService
			add(event, feature.Shape())
		}
	}
	// Each of its seven features is several megabytes.
	features, err := r.extents(ctx, novemberService, 0, "RiverBasin", 1)
	if err != nil {
		return 0, err
	}
	// The layer is named Flood_30Nov2025; its records say only the month.
	day := time.Date(2025, time.November, novemberEventDay, 0, 0, 0, 0, time.UTC)
	for _, feature := range features {
		event, ok := ParseEvent(feature.Text("RiverBasin"))
		if !ok {
			skipped = append(skipped, feature.Text("RiverBasin"))
			continue
		}
		event.Service = novemberService
		if event.Year == day.Year() {
			date := day
			event.Date = &date
		}
		add(event, feature.Shape())
	}
	cells, err := r.extents(ctx, puttalamService, 0, "FID", pageSize)
	if err != nil {
		return 0, err
	}
	puttalam := PuttalamEvent()
	puttalam.Service = puttalamService
	combined := make(geo.MultiPolygon, 0, len(cells))
	for _, cell := range cells {
		combined = append(combined, cell.Shape()...)
	}
	add(puttalam, combined)

	minimum := collector.Minimum
	if minimum <= 0 {
		minimum = minimumEvents
	}
	if len(order) < minimum {
		return 0, fmt.Errorf("the flood services gave %d events, want at least %d", len(order), minimum)
	}
	if collector.Logger != nil && len(skipped) > 0 {
		collector.Logger.Warn("past flood events without a year were skipped", "values", skipped)
	}
	sourceTime := newest(mapEdit, novemberEdit, puttalamEdit)
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '"+unionTimeout+"'"); err != nil {
			return err
		}
		queries := store.New(tx)
		if err := queries.DeletePastFloodEvents(ctx); err != nil {
			return err
		}
		for _, id := range order {
			event := events[id]
			row := store.InsertPastFloodEventParams{
				Fid: event.FID, ID: event.ID, Name: event.Name, SourceName: event.SourceName, River: event.River,
				Year: int32(event.Year), Service: event.Service, Source: PastFloodsSource, Version: version,
				SourceTime: sourceTime, FetchedAt: now, Wkb: event.Shape.WKB(),
			}
			if event.Magnitude != "" {
				row.Magnitude = &event.Magnitude
			}
			if event.Date != nil {
				row.EventDate = pgtype.Date{Time: *event.Date, Valid: true}
			}
			if err := queries.InsertPastFloodEvent(ctx, row); err != nil {
				return fmt.Errorf("store %s: %w", event.ID, err)
			}
			if err := queries.BuildPastFloodDetail(ctx, event.FID); err != nil {
				return fmt.Errorf("detail of %s: %w", event.ID, err)
			}
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: PastFloodsSource, Version: version, SourceTime: sourceTime, FetchedAt: now, Records: int32(len(order)),
		})
	})
	if err != nil {
		return 0, err
	}
	return len(order), nil
}
