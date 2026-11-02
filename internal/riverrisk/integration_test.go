//go:build integration

package riverrisk_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/hydrobasins"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/hydrorivers"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigationgis"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/rivernet"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/riverseed"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

var (
	now     = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	longAgo = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
)

// A tiny river network near Colombo: a main stem with one gauge and a tributary,
// drawn so its lines lie inside the official basin below.
const riversGeoJSON = `{"type":"FeatureCollection","features":[
 {"properties":{"HYRIV_ID":101,"NEXT_DOWN":102,"MAIN_RIV":104,"LENGTH_KM":10,"CATCH_SKM":500,"UPLAND_SKM":500,"HYBAS_L12":9001.0},
  "geometry":{"type":"MultiLineString","coordinates":[[[80.30,6.95],[80.20,6.95]]]}},
 {"properties":{"HYRIV_ID":102,"NEXT_DOWN":103,"MAIN_RIV":104,"LENGTH_KM":10,"CATCH_SKM":300,"UPLAND_SKM":800,"HYBAS_L12":9002.0},
  "geometry":{"type":"MultiLineString","coordinates":[[[80.20,6.95],[80.10,6.95]]]}},
 {"properties":{"HYRIV_ID":103,"NEXT_DOWN":104,"MAIN_RIV":104,"LENGTH_KM":10,"CATCH_SKM":300,"UPLAND_SKM":1200,"HYBAS_L12":9002.0},
  "geometry":{"type":"MultiLineString","coordinates":[[[80.10,6.95],[80.00,6.95]]]}},
 {"properties":{"HYRIV_ID":104,"NEXT_DOWN":0,"MAIN_RIV":104,"LENGTH_KM":10,"CATCH_SKM":300,"UPLAND_SKM":1500,"HYBAS_L12":9003.0},
  "geometry":{"type":"MultiLineString","coordinates":[[[80.00,6.95],[79.90,6.95]]]}},
 {"properties":{"HYRIV_ID":105,"NEXT_DOWN":103,"MAIN_RIV":104,"LENGTH_KM":12,"CATCH_SKM":100,"UPLAND_SKM":100,"HYBAS_L12":9002.0},
  "geometry":{"type":"MultiLineString","coordinates":[[[80.10,7.05],[80.10,6.95]]]}},
 {"properties":{"HYRIV_ID":201,"NEXT_DOWN":0,"MAIN_RIV":201,"LENGTH_KM":12,"CATCH_SKM":150,"UPLAND_SKM":150,"HYBAS_L12":9004.0},
  "geometry":{"type":"MultiLineString","coordinates":[[[80.10,7.40],[80.00,7.40]]]}}
]}`

const namesJSON = `{"104":"Kelani Ganga","105":"Wak Oya","201":""}`

// square is a clockwise ArcGIS ring.
func square(west, south, east, north float64) string {
	return fmt.Sprintf(`[[[%g,%g],[%g,%g],[%g,%g],[%g,%g],[%g,%g]]]`, west, south, west, north, east, north, east, south, west, south)
}

func feature(attributes, rings string) string {
	return `{"attributes":` + attributes + `,"geometry":{"rings":` + rings + `}}`
}

func page(features ...string) []byte {
	return []byte(`{"features":[` + strings.Join(features, ",") + `]}`)
}

// fakeServices answers the GitHub and ArcGIS requests the collectors make.
type fakeServices struct{}

func (fakeServices) Fetch(_ context.Context, request source.Request) ([]byte, error) {
	address := request.URL
	switch {
	case strings.Contains(address, "/commits"):
		return []byte(`[{"sha":"8473820c499fc3463aee4412bf8ac87440cdab7b","commit":{"author":{"date":"2025-12-06T15:29:41Z"},"committer":{"date":"2025-12-06T15:29:41Z"}}}]`), nil
	case strings.HasSuffix(address, riversSuffix):
		return []byte(riversGeoJSON), nil
	case strings.HasSuffix(address, namesSuffix):
		return []byte(namesJSON), nil
	}
	query := request.Query
	switch {
	case strings.Contains(address, "/Flood_Map/FeatureServer") && !strings.Contains(address, "/query") && strings.HasSuffix(address, "/FeatureServer"):
		return []byte(`{"layers":[{"id":2,"name":"Flood Inundation Area"},{"id":1,"name":"River_Basin"}]}`), nil
	case !strings.Contains(address, "/query"):
		return []byte(`{"editingInfo":{"lastEditDate":1662458967537}}`), nil
	case strings.Contains(address, "/river_basins/"):
		return page(
			feature(`{"FID":1,"Name":"Kelani Ganga"}`, square(79.85, 6.8, 80.4, 7.2)),
			feature(`{"FID":2,"Name":"Placemark"}`, square(79.95, 7.3, 80.2, 7.5)),
			feature(`{"FID":3,"Name":"Placemark"}`, square(80.5, 7.3, 80.5001, 7.3001)),
		), nil
	case strings.Contains(address, "/Srilanka_River/"):
		return page(feature(`{"FID":1,"NAME":"Kuda Oya"}`, square(80.5, 7.3, 80.51, 7.31))), nil
	case strings.Contains(address, "/hydrostations/"):
		return []byte(`{"features":[
		  {"attributes":{"station":"Hanwella","basin":"Kelani Ganga","Tributory":"Kelani Ganga","latitude":6.95,"longitude":80.08}}]}`), nil
	case strings.Contains(address, "/Flood_Map/"):
		return page(feature(`{"Flooding_area":"Kelani Ganag Flood 2016 Major"}`, square(80.0, 6.9, 80.1, 7.0))), nil
	case strings.Contains(address, "/Flood__Nov2025/"):
		if query.Get("resultOffset") != "0" {
			return page(), nil
		}
		return page(feature(`{"RiverBasin":"Kelani Ganga Flood 2025 Nov"}`, square(79.9, 6.92, 80.0, 6.98))), nil
	case strings.Contains(address, "/Puttalam_Flood"):
		return page(
			feature(`{"FID":1}`, square(80.0, 7.3, 80.01, 7.31)),
			feature(`{"FID":2}`, square(80.01, 7.3, 80.02, 7.31)),
		), nil
	}
	return nil, fmt.Errorf("unexpected request %s", address)
}

const (
	riversSuffix = "sri_lanka_rivers.geojson"
	namesSuffix  = "riv_id_to_name.json"
)

// hydroBasinsArchive builds a zip holding a one-record shapefile.
func hydroBasinsArchive(t *testing.T) []byte {
	t.Helper()
	ring := [][2]float64{{80.0, 6.9}, {80.0, 7.0}, {80.2, 7.0}, {80.2, 6.9}, {80.0, 6.9}}
	var body bytes.Buffer
	_ = binary.Write(&body, binary.BigEndian, int32(1))
	_ = binary.Write(&body, binary.BigEndian, int32((44+4+len(ring)*16)/2))
	_ = binary.Write(&body, binary.LittleEndian, int32(5))
	for _, value := range []float64{80.0, 6.9, 80.2, 7.0} {
		_ = binary.Write(&body, binary.LittleEndian, value)
	}
	_ = binary.Write(&body, binary.LittleEndian, int32(1))
	_ = binary.Write(&body, binary.LittleEndian, int32(len(ring)))
	_ = binary.Write(&body, binary.LittleEndian, int32(0))
	for _, point := range ring {
		_ = binary.Write(&body, binary.LittleEndian, point[0])
		_ = binary.Write(&body, binary.LittleEndian, point[1])
	}
	header := make([]byte, 100)
	binary.BigEndian.PutUint32(header[0:], 9994)
	binary.LittleEndian.PutUint32(header[32:], 5)
	shp := append(header, body.Bytes()...)

	var table bytes.Buffer
	fields := []string{"HYBAS_ID", "NEXT_DOWN", "MAIN_BAS", "SUB_AREA", "UP_AREA"}
	dbfHeader := make([]byte, 32)
	binary.LittleEndian.PutUint16(dbfHeader[8:], uint16(32+len(fields)*32+1))
	binary.LittleEndian.PutUint16(dbfHeader[10:], uint16(1+len(fields)*13))
	table.Write(dbfHeader)
	for _, field := range fields {
		descriptor := make([]byte, 32)
		copy(descriptor, field)
		descriptor[11] = 'N'
		descriptor[16] = 13
		table.Write(descriptor)
	}
	table.WriteByte(0x0D)
	table.WriteByte(' ')
	fmt.Fprintf(&table, "%13d%13d%13d%13.1f%13.1f", 9002, 0, 9002, 240.5, 240.5)

	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, content := range map[string][]byte{"a.shp": shp, "a.dbf": table.Bytes()} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func open(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("LANKAWEATHER_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("LANKAWEATHER_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, url); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE river_segments, river_source_names, official_basins, official_stream_names,
		official_stations, river_subbasins, rivers, river_gauge_matches, past_flood_events, source_versions,
		river_readings, river_stations, flood_gauges, flood_areas CASCADE`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func addStation(t *testing.T, pool *pgxpool.Pool, id, name, basin, sourceName string, lon, lat float64, alert, minor, major *float64) {
	t.Helper()
	queries := store.New(pool)
	if err := queries.UpsertRiverStation(context.Background(), store.UpsertRiverStationParams{
		ID: id, Name: name, Basin: basin, Lon: lon, Lat: lat, Unit: "m", AlertLevel: alert, MinorFloodLevel: minor,
		MajorFloodLevel: major, Source: sourceName, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}

func addReading(t *testing.T, pool *pgxpool.Pool, id string, at time.Time, level float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO river_readings (station_id, observed_at, water_level, fetched_at)
		VALUES ($1, $2, $3, $4)`, id, at, level, now); err != nil {
		t.Fatal(err)
	}
}

func f(value float64) *float64 { return &value }

func load(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	fake := fakeServices{}
	if n, err := (&hydrorivers.Collector{Pool: pool, Fetcher: fake, Minimum: 1}).Collect(ctx, now); err != nil || n != 6 {
		t.Fatalf("hydrorivers: %d, %v", n, err)
	}
	if n, err := (&irrigationgis.BasinsCollector{Pool: pool, Fetcher: fake, Minimum: 1}).Collect(ctx, now); err != nil || n != 2 {
		t.Fatalf("official basins: %d, %v", n, err)
	}
	if n, err := (&irrigationgis.PastFloodsCollector{Pool: pool, Fetcher: fake, Minimum: 1}).Collect(ctx, now); err != nil || n != 3 {
		t.Fatalf("past floods: %d, %v", n, err)
	}
	archive := hydroBasinsArchive(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write(archive)
	}))
	t.Cleanup(server.Close)
	if n, err := (&hydrobasins.Collector{Pool: pool, Client: source.NewClient("test"), URL: server.URL, Minimum: 1}).Collect(ctx, now); err != nil || n != 1 {
		t.Fatalf("hydrobasins: %d, %v", n, err)
	}

	// A station from the irrigation collector with thresholds; one from another
	// source with none, and with a basin name no table knows; and a Google gauge.
	addStation(t, pool, "hanwella", "Hanwella", "Kelani Ganga", "irrigation-gauges", 80.08, 6.95, f(6.5), f(7.5), f(9.5))
	addReading(t, pool, "hanwella", now.Add(-2*time.Hour), 6.0)
	addReading(t, pool, "hanwella", now.Add(-time.Hour), 6.8)
	addStation(t, pool, "other-station", "Other Source", "A Basin Nobody Named", "dmc-bulletins", 80.25, 6.95, nil, nil, nil)
	addReading(t, pool, "other-station", now.Add(-time.Hour), 3)
	if err := store.New(pool).InsertFloodGauge(ctx, store.InsertFloodGaugeParams{
		GaugeID: "hybas_9003", Source: "google-flood-forecasting", GaugeSource: "HYBAS", Lon: 79.95, Lat: 6.95, QualityVerified: true,
		Severity: "no_flooding", Trend: "rise", IssuedAt: now, ValueUnit: "m3/s", FetchedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if n, err := (&rivernet.Collector{Pool: pool}).Collect(ctx, now); err != nil || n == 0 {
		t.Fatalf("network: %d, %v", n, err)
	}
}

func service(pool *pgxpool.Pool) *riverrisk.Service {
	clock := func() time.Time { return now }
	return &riverrisk.Service{Pool: pool, Now: clock,
		Rivers: &rivers.Service{Pool: pool, StaleAfter: 3 * time.Hour, Now: clock},
		Floods: &floods.Service{Pool: pool, Configured: true, StaleAfter: 30 * time.Hour, Now: clock}}
}

func TestCollectorsAndRiskEndToEnd(t *testing.T) {
	pool := open(t)
	load(t, pool)
	ctx := context.Background()

	// Rebuilds only when an input changes.
	if n, err := (&rivernet.Collector{Pool: pool}).Collect(ctx, now.Add(time.Hour)); err != nil || n != 0 {
		t.Fatalf("an unchanged network should not rebuild: %d, %v", n, err)
	}

	document, err := service(pool).Risk(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !document.Estimate || document.Meta.NetworkVersion == "" || document.Meta.Seeded {
		t.Fatalf("meta = %+v", document.Meta)
	}
	byID := map[string]riverrisk.RiverRisk{}
	for _, river := range document.Rivers {
		byID[river.ID] = river
	}
	kelani, ok := byID["kelani-ganga"]
	if !ok {
		t.Fatalf("rivers = %v", document.Rivers)
	}
	if kelani.Level != riverrisk.LevelHigh {
		t.Fatalf("Kelani = %s: %+v", kelani.Level, kelani.Reasons)
	}
	if kelani.BBox[0] == 0 || kelani.GaugedReaches == 0 {
		t.Fatalf("Kelani = %+v", kelani)
	}
	reaches := map[int64]riverrisk.ReachRisk{}
	for _, reach := range kelani.Reaches {
		reaches[reach.FID] = reach
	}
	// The null-threshold station from another source is tolerated: it was placed
	// by position, and proves nothing about its stretch, which is not called safe.
	var matchedOther bool
	matches, err := store.New(pool).ListRiverGaugeMatches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range matches {
		if match.GaugeID == "other-station" {
			matchedOther = match.Method == "position"
		}
	}
	if !matchedOther {
		t.Fatalf("matches = %+v", matches)
	}
	if reach := reaches[101]; reach.Level == riverrisk.LevelNormal {
		t.Fatalf("the stretch at a thresholdless gauge read safe: %+v", reach)
	}

	// Past floods keep the event name and year, and drop the source's misspelling.
	past, err := service(pool).PastFloods(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, event := range past.Events {
		names = append(names, event.Name)
	}
	joined := strings.Join(names, " | ")
	for _, want := range []string{"Kelani Ganga flood, 2016 (major)", "Kelani Ganga flood, 2025", "Puttalam flood, Cyclone Ditwah"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("events = %s", joined)
		}
	}
	if past.Note == "" {
		t.Fatal("past floods must say they are past")
	}

	// Official basins, with sliver dropped and coastal area kept.
	basins, err := service(pool).Basins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(basins.Basins) != 2 {
		t.Fatalf("basins = %+v", basins.Basins)
	}
	states := map[string]riverrisk.BasinStatus{}
	for _, basin := range basins.Basins {
		states[basin.ID] = basin
	}
	if states["kelani-ganga"].State != riverrisk.BasinAboveNormal {
		t.Fatalf("Kelani basin = %+v", states["kelani-ganga"])
	}
	if states["coastal-2"].State != riverrisk.BasinNoForecast || states["coastal-2"].Named {
		t.Fatalf("coastal = %+v", states["coastal-2"])
	}
	if len(states["kelani-ganga"].Gauges) != 3 {
		t.Fatalf("gauges = %+v", states["kelani-ganga"].Gauges)
	}
	if len(basins.Subbasins) == 0 || basins.Subbasins[0].FID != 9002 {
		t.Fatalf("sub-basins = %+v", basins.Subbasins)
	}

	// Near a point.
	near, err := service(pool).Near(ctx, 80.08, 6.96)
	if err != nil {
		t.Fatal(err)
	}
	if len(near) == 0 || near[0].ID != "kelani-ganga" || !near[0].Estimate {
		t.Fatalf("near = %+v", near)
	}
}

func TestRiverDetailListsGaugesForecastsAndAffectedAreas(t *testing.T) {
	pool := open(t)
	load(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO regions (id, fid, level, name, province_id, area_km2, centroid, label_point, west, south, east, north,
			geom, geom_simple, geom_3857, geom_simple_3857, source, source_version, source_time, fetched_at)
		SELECT v.id, v.fid, v.level, v.name, 'LK-1', 100, ST_SetSRID(ST_MakePoint(80,7),4326), ST_SetSRID(ST_MakePoint(80,7),4326), 79, 6, 81, 8,
		       e, e, ST_Transform(e,3857), ST_Transform(e,3857), 't', 't', $1, $1
		FROM (VALUES ('LK-11', 11, 'district', 'Colombo'), ('LK-1103', 1103, 'ds', 'Colombo DS')) AS v(id, fid, level, name),
		     (SELECT ST_Multi(ST_MakeEnvelope(79.8, 6.8, 80.4, 7.1, 4326)) AS e) q
		ON CONFLICT (id) DO NOTHING`, longAgo); err != nil {
		t.Fatal(err)
	}
	detail, err := service(pool).River(ctx, "kelani-ganga")
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Estimate || detail.Note == "" {
		t.Fatalf("detail = %+v", detail)
	}
	var hanwella *riverrisk.RiverGauge
	for index := range detail.Gauges {
		if detail.Gauges[index].ID == "hanwella" {
			hanwella = &detail.Gauges[index]
		}
	}
	if hanwella == nil || len(hanwella.Readings) != 2 {
		t.Fatalf("gauges = %+v", detail.Gauges)
	}
	if len(detail.Forecasts) != 1 || detail.Forecasts[0].GaugeID != "hybas_9003" {
		t.Fatalf("forecasts = %+v", detail.Forecasts)
	}
	if len(detail.Affected) == 0 || len(detail.Affected[0].Districts) != 1 || detail.Affected[0].Districts[0].Name != "Colombo" ||
		detail.Affected[0].DSDivisions[0].Name != "Colombo DS" {
		t.Fatalf("affected = %+v", detail.Affected)
	}
	if _, err := service(pool).River(ctx, "nowhere"); err != riverrisk.ErrNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestTilesCarryTheLayers(t *testing.T) {
	pool := open(t)
	load(t, pool)
	ctx := context.Background()
	svc := service(pool)
	// The z7 tile over Colombo.
	tile, err := svc.RiverTile(ctx, 7, 92, 61)
	if err != nil {
		t.Fatal(err)
	}
	if len(tile.Body) == 0 || !bytes.Contains(tile.Body, []byte("reaches")) || !bytes.Contains(tile.Body, []byte("labels")) {
		t.Fatalf("river tile of %d bytes", len(tile.Body))
	}
	if !bytes.Contains(tile.Body, []byte("Kelani Ganga")) {
		t.Fatal("the label is missing from the tile")
	}
	basin, err := svc.BasinTile(ctx, 7, 92, 61)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(basin.Body, []byte("basins")) || bytes.Contains(basin.Body, []byte("subbasins")) {
		t.Fatalf("basin tile of %d bytes at z7", len(basin.Body))
	}
	fine, err := svc.BasinTile(ctx, 10, 739, 492)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(fine.Body, []byte("subbasins")) {
		t.Fatalf("sub-basins missing at z10, %d bytes", len(fine.Body))
	}
	past, err := svc.PastFloodTile(ctx, 7, 92, 61)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(past.Body, []byte("events")) {
		t.Fatalf("past flood tile of %d bytes", len(past.Body))
	}
	near, err := svc.PastFloodTile(ctx, 12, 2958, 1968)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(near.Body, []byte("events")) {
		t.Fatalf("close-up past flood tile of %d bytes", len(near.Body))
	}
}

func TestGoogleRowsAreDroppedWhenTheyLapse(t *testing.T) {
	pool := open(t)
	load(t, pool)
	ctx := context.Background()
	queries := store.New(pool)
	collector := &rivernet.Collector{Pool: pool}
	later := now.Add(25 * time.Hour)
	if err := collector.Prune(ctx, later); err != nil {
		t.Fatal(err)
	}
	matches, err := queries.ListRiverGaugeMatches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range matches {
		if match.Kind == "google" {
			t.Fatalf("a Google match outlived its day: %+v", match)
		}
	}
	// A later rebuild no longer splits at it, or mentions it.
	if _, err := collector.Collect(ctx, later); err != nil {
		t.Fatal(err)
	}
	gauges, err := queries.ListRiverReachGauges(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, gauge := range gauges {
		if gauge.Kind == "google" {
			t.Fatalf("a reach still names Google's gauge: %+v", gauge)
		}
	}
}

func TestSeedRefusesTheMainDatabaseAndIsAlwaysFlagged(t *testing.T) {
	pool := open(t)
	load(t, pool)
	ctx := context.Background()
	if err := riverseed.Apply(ctx, pool, now); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name == "lankaweather" {
		t.Fatal("tests must never run on the main database")
	}
	var seeded bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM river_stations WHERE source = 'local-seed')`).Scan(&seeded); err != nil {
		t.Fatal(err)
	}
	if !seeded {
		t.Fatal("hanwella is one of the seeded stations")
	}
	document, err := service(pool).Risk(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !document.Meta.Seeded {
		t.Fatal("seeded data must be flagged in the document")
	}
	if err := riverseed.Clear(ctx, pool); err != nil {
		t.Fatal(err)
	}
	cleared, err := service(pool).Risk(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Meta.Seeded {
		t.Fatal("clearing should remove the flag")
	}
}

func TestPayloadsAreValidJSON(t *testing.T) {
	pool := open(t)
	load(t, pool)
	document, err := service(pool).Risk(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(document); err != nil {
		t.Fatal(err)
	}
}
