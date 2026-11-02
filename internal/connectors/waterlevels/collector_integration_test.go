//go:build integration

package waterlevels_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
	"github.com/prabhavalabs/lanka-weather/internal/layers"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// replay answers every request with a recorded file.
type replay struct{ file string }

func (r replay) Fetch(context.Context, source.Request) ([]byte, error) {
	return os.ReadFile(r.file)
}

func TestGaugesAndForecastsIntegration(t *testing.T) {
	databaseURL := os.Getenv("LANKAWEATHER_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LANKAWEATHER_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE river_next_hour, river_readings, river_stations, source_runs`); err != nil {
		t.Fatal(err)
	}
	queries := store.New(pool)

	now := time.Date(2026, 11, 2, 19, 45, 0, 0, time.UTC)
	gauges := &waterlevels.GaugeCollector{Pool: pool, Fetcher: replay{"testdata/levels.json"}, Minimum: 1}

	// Without the department's stations every report would look new, so the
	// collector waits for them.
	if _, err := gauges.Collect(ctx, now); !errors.Is(err, waterlevels.ErrIrrigationNotLoaded) {
		t.Fatalf("collect before the department's feed: %v", err)
	}

	// The department's own stations, as its collector stores them.
	station := func(id, name string, lon, lat float64, unit string, alert, minor, major *float64) {
		t.Helper()
		if err := queries.UpsertRiverStation(ctx, store.UpsertRiverStationParams{
			ID: id, Name: name, Basin: "Basin", Lon: lon, Lat: lat, Unit: unit, AlertLevel: alert, MinorFloodLevel: minor,
			MajorFloodLevel: major, Source: irrigation.SourceName, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	mark := func(value float64) *float64 { return &value }
	station("rathnapura", "Rathnapura", 80.39, 6.68, "m", mark(5.2), mark(7.5), mark(9.5))
	station("ampara", "Ampara", 81.67, 7.29, "unknown", nil, nil, nil)
	station("dunamale", "Dunamale", 80.084, 7.116, "m", mark(3.3), mark(4.4), mark(5.5))
	station("nagalagam-street", "Nagalagam Street", 79.879, 6.959, "ft", mark(4), mark(5), mark(7))

	stored, err := gauges.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	// Karawanella Bridge, Kuru Ganga, Wattala and Pugoda have readings; Calidonia has none.
	if stored != 4 {
		t.Fatalf("stored %d readings, want 4", stored)
	}

	// Stations of the department are untouched and not duplicated.
	var sources = map[string]string{}
	rows, err := queries.RiverStationSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		sources[row.ID] = row.Source
	}
	if len(sources) != 9 {
		t.Fatalf("stations = %v", sources)
	}
	for _, id := range []string{"rathnapura", "ampara", "dunamale", "nagalagam-street"} {
		if sources[id] != irrigation.SourceName {
			t.Errorf("%s now belongs to %s", id, sources[id])
		}
	}
	if _, found := sources["ratnapura"]; found {
		t.Fatal("Ratnapura was stored under the alias as well")
	}
	rathnapura, err := queries.GetRiverStation(ctx, "rathnapura")
	if err != nil {
		t.Fatal(err)
	}
	if rathnapura.Name != "Rathnapura" || *rathnapura.AlertLevel != 5.2 {
		t.Fatalf("the department's station changed: %+v", rathnapura)
	}
	for _, id := range []string{"rathnapura", "ampara", "dunamale", "nagalagam-street"} {
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM river_readings WHERE station_id = $1`, id).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("%s received %d readings from the community feed", id, count)
		}
	}

	// A second run with the same readings stores nothing new.
	if _, err := gauges.Collect(ctx, now.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var readings int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM river_readings`).Scan(&readings); err != nil {
		t.Fatal(err)
	}
	if readings != 4 {
		t.Fatalf("readings after a second run = %d", readings)
	}

	// The extra gauges' readings do not move the point the department's
	// collector resumes from.
	if _, err := queries.LatestRiverObservation(ctx, irrigation.SourceName); err == nil {
		t.Fatal("the department's collector would resume from another source's reading")
	}
	if _, err := queries.LatestRiverObservation(ctx, waterlevels.GaugesSource); err != nil {
		t.Fatal(err)
	}

	limit := func(id string) time.Duration {
		layer, _ := layers.ByID(id)
		return time.Duration(layer.StaleAfterSeconds) * time.Second
	}
	service := &rivers.Service{Pool: pool, StaleAfter: limit("river-levels"), Now: func() time.Time { return now }}
	layer, err := service.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]rivers.StationProperties{}
	for _, feature := range layer.Features {
		byID[feature.ID] = feature.Properties
	}

	// A gauge with marks gets a verdict, in the station's own unit.
	pugoda := byID["pugoda"]
	if pugoda.Status != rivers.StatusNormal || pugoda.Unit != "m" || *pugoda.WaterLevel != 3.963 || *pugoda.AlertLevel != 8 || pugoda.Stale {
		t.Fatalf("pugoda = %+v", pugoda)
	}
	if !strings.Contains(pugoda.Attribution, "RiverNet") || !strings.Contains(pugoda.Attribution, "waterlevels.lk") ||
		pugoda.Source != waterlevels.GaugesSource || pugoda.SourceURL == nil {
		t.Fatalf("pugoda credit = %+v", pugoda)
	}
	// Karawanella's page names no agency: nothing is invented.
	if karawanella := byID["karawanella-bridge"]; strings.Contains(karawanella.Attribution, "RiverNet") {
		t.Fatalf("karawanella credit = %q", karawanella.Attribution)
	}
	// No marks: a level, without a verdict.
	kuru := byID["kuru-ganga-kuruvita"]
	if kuru.Status != rivers.StatusUnknown || *kuru.WaterLevel != 2.24 || kuru.AlertLevel != nil {
		t.Fatalf("kuru = %+v", kuru)
	}
	// A gauge that has gone quiet: no reading, no verdict, out of date.
	calidonia := byID["calidonia"]
	if calidonia.Status != rivers.StatusUnknown || !calidonia.Stale || calidonia.WaterLevel != nil || calidonia.Unit != "unknown" {
		t.Fatalf("calidonia = %+v", calidonia)
	}
	// The forecast of a department station, converted to metres, with the
	// department's status unaffected.
	dunamale := byID["dunamale"]
	if dunamale.NextHour == nil || dunamale.NextHour.Level != 1.8 || *dunamale.NextHour.Low != 1.76 || *dunamale.NextHour.High != 1.85 ||
		dunamale.NextHour.Unit != "m" || !dunamale.NextHour.Experimental || dunamale.Attribution != rivers.IrrigationAttribution {
		t.Fatalf("dunamale = %+v, next hour %+v", dunamale, dunamale.NextHour)
	}
	if street := byID["nagalagam-street"]; street.NextHour == nil || street.NextHour.Unit != "ft" {
		t.Fatalf("nagalagam = %+v", street)
	}
	// Ratnapura's forecast is filed under the department's identifier.
	if byID["rathnapura"].NextHour == nil {
		t.Fatal("Ratnapura's forecast did not reach Rathnapura")
	}
	// Ampara has no unit and no forecast.
	if byID["ampara"].NextHour != nil || byID["ampara"].Unit != "unknown" {
		t.Fatalf("ampara = %+v", byID["ampara"])
	}
	if len(layer.Meta.Sources) != 2 {
		t.Fatalf("sources = %+v", layer.Meta.Sources)
	}

	// A station going quiet ages exactly as the department's do.
	later := &rivers.Service{Pool: pool, StaleAfter: limit("river-levels"), Now: func() time.Time { return now.Add(4 * time.Hour) }}
	laterLayer, err := later.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, feature := range laterLayer.Features {
		// Forecasts older than two hours disappear with it.
		if feature.Properties.NextHour != nil {
			t.Errorf("%s still has a forecast four hours on", feature.ID)
		}
		if feature.ID == "pugoda" && !feature.Properties.Stale {
			t.Errorf("pugoda was not marked stale four hours on: %+v", feature.Properties)
		}
	}

	// The community feed must not take over a station if the department later
	// publishes one of its gauges, and the department's feed takes over cleanly.
	station("pugoda", "Pugoda", 80.1, 6.97, "m", mark(8), mark(10), mark(11.5))
	if _, err := gauges.Collect(ctx, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	pugodaRow, err := queries.GetRiverStation(ctx, "pugoda")
	if err != nil {
		t.Fatal(err)
	}
	if pugodaRow.Source != irrigation.SourceName || pugodaRow.SourceUrl != nil || pugodaRow.Upstream != nil {
		t.Fatalf("pugoda after the department took over = %+v", pugodaRow)
	}

	if err := gauges.Prune(ctx, now.Add(40*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	var forecasts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM river_next_hour`).Scan(&forecasts); err != nil {
		t.Fatal(err)
	}
	if forecasts != 0 {
		t.Fatalf("forecasts after pruning = %d", forecasts)
	}
}

func TestReservoirsIntegration(t *testing.T) {
	databaseURL := os.Getenv("LANKAWEATHER_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LANKAWEATHER_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `TRUNCATE reservoirs, reservoir_readings, source_runs CASCADE`); err != nil {
		t.Fatal(err)
	}

	// The response was generated on 2 November at 19:39 UTC, 3 November in Sri Lanka.
	now := time.Date(2026, 11, 2, 19, 45, 0, 0, time.UTC)
	collector := &waterlevels.ReservoirCollector{Pool: pool, Fetcher: replay{"testdata/reservoirs.json"}, Minimum: 1}
	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 9 {
		t.Fatalf("stored %d", stored)
	}
	// A later fetch on the same day replaces the day's row; nothing is added.
	if _, err := collector.Collect(ctx, now.Add(6*time.Hour)); err != nil {
		t.Fatal(err)
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM reservoir_readings`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 9 {
		t.Fatalf("rows after a second fetch = %d", rows)
	}
	// Coordinates belong to the map collector and survive the feed.
	if _, err := pool.Exec(ctx, `UPDATE reservoirs SET lon = 80.86, lat = 8.02, osm_kind = 'water', osm_name = 'Minneriya Reservoir' WHERE id = 'minneriya'`); err != nil {
		t.Fatal(err)
	}
	if _, err := collector.Collect(ctx, now.Add(7*time.Hour)); err != nil {
		t.Fatal(err)
	}

	if _, err := store.New(pool).StartSourceRun(ctx, store.StartSourceRunParams{Source: waterlevels.ReservoirsSource, StartedAt: now}); err != nil {
		t.Fatal(err)
	}
	service := &reservoirs.Service{Pool: pool, StaleAfter: 48 * time.Hour, Now: func() time.Time { return now }}
	layer, err := service.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	counts := layer.Meta.Counts
	if counts.Total != 9 || counts.Mapped != 1 || counts.Spilling+counts.Releasing+counts.Holding+counts.NoData != 9 {
		t.Fatalf("counts = %+v", counts)
	}
	byID := map[string]reservoirs.Feature{}
	for _, feature := range layer.Features {
		byID[feature.ID] = feature
	}
	minneriya := byID["minneriya"]
	if minneriya.Geometry == nil || !minneriya.Properties.Mapped || minneriya.Properties.Status != "releasing" ||
		minneriya.Properties.Flow == nil || minneriya.Properties.Flow.Kind != "downstream" || !minneriya.Properties.Active() {
		t.Fatalf("minneriya = %+v", minneriya)
	}
	// Not matched to the map: listed, without a position.
	denagama := byID["denagama"]
	if denagama.Geometry != nil || denagama.Properties.Mapped || !denagama.Properties.Active() {
		t.Fatalf("denagama = %+v", denagama)
	}
	// An old reading stays unknown-for-today rather than current, and null
	// stays null rather than becoming zero.
	if old := byID["alikota-ara"].Properties; !old.Stale || *old.StoragePercent != 68.4 || *old.StorageFresh {
		t.Fatalf("alikota-ara = %+v", old)
	}
	if empty := byID["kandiyapita-right-bank"].Properties; empty.StoragePercent != nil || empty.Status != waterlevels.StatusNoData {
		t.Fatalf("kandiyapita-right-bank = %+v", empty)
	}
	// Reading a spilling reservoir long after the fact is not current news.
	later := &reservoirs.Service{Pool: pool, StaleAfter: 48 * time.Hour, Now: func() time.Time { return now.Add(10 * 24 * time.Hour) }}
	laterLayer, err := later.Layer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, feature := range laterLayer.Features {
		if feature.Properties.Active() {
			t.Errorf("%s is still active ten days on", feature.ID)
		}
	}

	history, err := service.History(ctx, "minneriya", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Readings) != 1 || history.Readings[0].Day != "2026-11-02" || *history.Readings[0].StoragePercent != 67 || history.Note == "" {
		t.Fatalf("history = %+v", history)
	}
	// A day the source only repeats an old figure is not a day of history.
	stale, err := service.History(ctx, "alikota-ara", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale.Readings) != 0 {
		t.Fatalf("stale history = %+v", stale.Readings)
	}
	if _, err := service.History(ctx, "nowhere", 30); !errors.Is(err, reservoirs.ErrNotFound) {
		t.Fatalf("unknown reservoir: %v", err)
	}
	if areas, err := service.Areas(ctx); err != nil || len(areas.Features) != 0 {
		t.Fatalf("areas = %+v, err %v", areas, err)
	}
}
