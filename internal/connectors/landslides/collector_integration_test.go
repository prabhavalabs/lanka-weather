//go:build integration

package landslides_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/dmcbulletins"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/landslides"
	"github.com/prabhavalabs/lanka-weather/internal/platform/database/dbtest"
	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

type replay struct{ payload []byte }

func (r replay) Fetch(context.Context, source.Request) ([]byte, error) { return r.payload, nil }

// seed loads the report's division names and its bulletin.
func seed(t *testing.T, pool *pgxpool.Pool, issued, until time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO regions (id, fid, level, name, parent_id, province_id, district_id, ds_id, area_km2, centroid, label_point, west, south, east, north,
		                     geom, geom_simple, geom_3857, geom_simple_3857, source, source_version, source_time, fetched_at)
		SELECT v.id, v.fid, v.level, v.name, v.parent, 'LK-2', v.district, v.ds, 1,
		       ST_SetSRID(ST_MakePoint(80.5, 7.2), 4326), ST_SetSRID(ST_MakePoint(80.5, 7.2), 4326), 80.4, 7.1, 80.6, 7.3,
		       m.g, m.g, ST_Transform(m.g, 3857), ST_Transform(m.g, 3857), 'test', 'test', now(), now()
		FROM (VALUES ('LK-21', 21, 'district', 'Kandy', 'LK-2', 'LK-21', NULL),
		             ('LK-2136', 2136, 'ds', 'Yatinuwara', 'LK-21', 'LK-21', 'LK-2136'),
		             ('LK-92', 92, 'district', 'Kegalle', 'LK-9', 'LK-92', NULL),
		             ('LK-9206', 9206, 'ds', 'Mawanella', 'LK-92', 'LK-92', 'LK-9206')) AS v(id, fid, level, name, parent, district, ds),
		     (SELECT ST_Multi(ST_MakeEnvelope(80.4, 7.1, 80.6, 7.3, 4326)) AS g) AS m`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO official_bulletins (doc_id, bulletin_type, title, severity, issued_at, valid_until, message_id, pdf_url, source_page_url, lang, text_status,
		                                source, source_time, fetched_at)
		VALUES ('report', 'landslide', 'Landslide EW Report', 'routine', $1, $2, 'LEWM-31-10-2026-1600-38', 'https://www.dmc.gov.lk/a.pdf', 'https://www.dmc.gov.lk/page', 'en', 'unusable',
		        'dmc-reports', $1, $1)`, issued, until); err != nil {
		t.Fatal(err)
	}
}

var tables = []string{"regions", "official_bulletins", "landslide_levels", "landslide_reports", "source_versions"}

func TestCollectAndServeTheLatestReport(t *testing.T) {
	pool := dbtest.Open(t, tables...)
	ctx := context.Background()
	payload, err := os.ReadFile("testdata/all.json")
	if err != nil {
		t.Fatal(err)
	}
	issued := time.Date(2026, 10, 31, 10, 30, 0, 0, time.UTC)
	until := issued.Add(24 * time.Hour)
	seed(t, pool, issued, until)

	// During the report's validity.
	now := issued.Add(6 * time.Hour)
	collector := &landslides.Collector{Pool: pool, Fetcher: replay{payload}}
	stored, err := collector.Collect(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if stored == 0 {
		t.Fatal("stored nothing")
	}
	if err := putUpstream(ctx, pool, now); err != nil {
		t.Fatal(err)
	}

	service := &warnings.Service{Pool: pool, Now: func() time.Time { return now }, LandslideStaleAfter: 12 * time.Hour, BulletinStaleAfter: 6 * time.Hour}
	layer, err := service.Landslides(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if layer.Meta.Status != warnings.StatusActive || layer.Meta.Stale {
		t.Fatalf("meta = %+v", layer.Meta)
	}
	if layer.Meta.MessageID == nil || *layer.Meta.MessageID != "LEWM-31-10-2026-1600-38" || layer.Meta.PDFURL == nil {
		t.Fatalf("meta = %+v", layer.Meta)
	}
	if !layer.Meta.ValidityStated || !layer.Meta.ValidUntil.Equal(until) {
		t.Fatalf("meta = %+v", layer.Meta)
	}
	// LK-9209 is in the report and in no region of this small seed.
	if len(layer.Features) != 2 || strings.Join(layer.Meta.Unresolved, ",") != "LK-9209" || layer.Meta.Counts["1"] != 3 {
		t.Fatalf("features = %+v, meta = %+v", layer.Features, layer.Meta)
	}
	first := layer.Features[0]
	if first.ID != "LK-2136" || first.Properties.Name != "Yatinuwara" || first.Properties.District != "Kandy" || first.Properties.Label != "Level 1 (Yellow)" {
		t.Fatalf("first = %+v", first)
	}

	division, err := service.Division(ctx, "lk-2136")
	if err != nil {
		t.Fatal(err)
	}
	if division.Level == nil || division.Level.Level != 1 || division.District != "Kandy" || len(division.History) == 0 || division.History[0].Date != "2026-10-31" {
		t.Fatalf("division = %+v", division)
	}
	if _, err := service.Division(ctx, "LK-9999"); err != warnings.ErrDivisionNotFound {
		t.Fatalf("unknown division: %v", err)
	}

	here, err := service.Official(ctx, "LK-21", "LK-2136")
	if err != nil {
		t.Fatal(err)
	}
	if here.Landslide.Level == nil || here.Landslide.Level.Label != "Level 1 (Yellow)" || *here.Landslide.DS != "Yatinuwara" {
		t.Fatalf("official = %+v", here.Landslide)
	}
	elsewhere, err := service.Official(ctx, "LK-11", "LK-1103")
	if err != nil {
		t.Fatal(err)
	}
	if elsewhere.Landslide.Level != nil || elsewhere.Landslide.Status != warnings.StatusActive {
		t.Fatalf("official elsewhere = %+v", elsewhere.Landslide)
	}

	// Long after the validity, the map is empty and the words say why.
	now = until.Add(33 * time.Hour)
	if err := putUpstream(ctx, pool, now); err != nil {
		t.Fatal(err)
	}
	if _, err := collector.Collect(ctx, now); err != nil {
		t.Fatal(err)
	}
	layer, err = service.Landslides(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(layer.Features) != 0 || layer.Meta.Status != warnings.StatusNone {
		t.Fatalf("features = %+v, meta = %+v", layer.Features, layer.Meta)
	}
	if layer.Meta.Note != "Last report 31 Oct 2026; none currently active as far as we know." {
		t.Fatalf("note = %q", layer.Meta.Note)
	}
	// History stays reachable after the report ends.
	if division, err = service.Division(ctx, "LK-2136"); err != nil || division.Level != nil || len(division.History) == 0 {
		t.Fatalf("division = %+v, err %v", division, err)
	}
}

func TestCollectKeepsHistoryAndDoesNotDuplicateIt(t *testing.T) {
	pool := dbtest.Open(t, tables...)
	ctx := context.Background()
	payload, err := os.ReadFile("testdata/all.json")
	if err != nil {
		t.Fatal(err)
	}
	collector := &landslides.Collector{Pool: pool, Fetcher: replay{payload}}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	if _, err := collector.Collect(ctx, now); err != nil {
		t.Fatal(err)
	}
	if _, err := collector.Collect(ctx, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	var reports, levels int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM landslide_reports), (SELECT count(*) FROM landslide_levels)`).Scan(&reports, &levels); err != nil {
		t.Fatal(err)
	}
	if reports != 3 || levels == 0 {
		t.Fatalf("%d reports, %d levels", reports, levels)
	}
}

func putUpstream(ctx context.Context, pool *pgxpool.Pool, at time.Time) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO source_versions (source, version, source_time, fetched_at, records) VALUES ($1, 'v', $2, $2, 1)
		ON CONFLICT (source) DO UPDATE SET source_time = EXCLUDED.source_time, fetched_at = EXCLUDED.fetched_at`, dmcbulletins.SourceName, at)
	return err
}
