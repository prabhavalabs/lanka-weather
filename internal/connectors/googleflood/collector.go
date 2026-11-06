package googleflood

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// DefaultBaseURL is the Flood Forecasting API.
const DefaultBaseURL = "https://floodforecasting.googleapis.com"

const (
	// Interval: gauge statuses are reissued about daily; hourly picks a new
	// issue up soon. A calm run is about 11 requests with extended coverage: 2
	// for statuses, 7 for 320 gauges' models and 2 for the events search, whose
	// last page is empty. Flash floods have their own collector (FlashCollector).
	Interval = time.Hour
	// Retention is the longest Google's terms let gauge search results be
	// kept.
	Retention = 24 * time.Hour
	// searchPageSize keeps requests low: the 131 flash floods live worldwide on
	// 2 November 2026 came on one page of this size, against seven at 20.
	searchPageSize = 1000
	// maxPages bounds one search. Reaching it is logged, not silent: significant
	// events are worldwide and filtered here, so one can be on any page.
	maxPages = 25
	// maxPolygons bounds the polygons read in a run. Flash floods and events are
	// read first, then verified gauges, so lower-confidence gauges in a large
	// flood cannot use the budget up.
	maxPolygons = 120
	// modelBatch is the number of gauge models asked for in one request.
	modelBatch = 50
)

// IncludeUnverified reads LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED: lower-confidence
// gauges are collected unless it is set to false. The map keeps them off until
// someone turns on Extended coverage.
func IncludeUnverified(setting string) bool {
	return !strings.EqualFold(strings.TrimSpace(setting), "false")
}

// Doer sends a request to a source.
type Doer interface {
	Do(ctx context.Context, request source.Request) (source.Response, error)
}

// Collector stores the latest flood forecasts for Sri Lanka.
type Collector struct {
	Pool   *pgxpool.Pool
	Client Doer
	// APIKey is the Google Cloud API key; it is sent in a header only.
	APIKey string
	// IncludeUnverified also reads gauges Google has not quality-verified,
	// which Flood Hub calls Extended coverage. cmd/lankaweather turns it on
	// unless LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED is false.
	IncludeUnverified bool
	// BaseURL overrides DefaultBaseURL, for tests.
	BaseURL string
	Logger  *slog.Logger
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration { return Interval }

// Collect reads the latest statuses, thresholds, significant events and
// polygons for Sri Lanka and replaces what is stored, except the flash floods,
// which FlashCollector keeps. It returns the number of gauges and areas stored.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	if strings.TrimSpace(collector.APIKey) == "" {
		return 0, errors.New("LANKAWEATHER_GOOGLE_API_KEY is not set")
	}
	statuses, err := collector.statuses(ctx)
	if err != nil {
		return 0, fmt.Errorf("flood statuses: %w", err)
	}
	models, err := collector.models(ctx, statuses)
	if err != nil {
		return 0, fmt.Errorf("gauge models: %w", err)
	}
	events, err := collector.search(ctx, "/v1/significantEvents:search", nil, DecodeSignificantEvents)
	if err != nil {
		return 0, fmt.Errorf("significant events: %w", err)
	}
	var areas []Area
	for _, status := range statuses {
		areas = append(areas, GaugeAreas(status)...)
	}
	areas = append(areas, events...)
	shapes := collector.polygons(ctx, PrioritiseAreas(areas, statuses), nil)
	if err := collector.save(ctx, statuses, models, areas, shapes, now); err != nil {
		return 0, err
	}
	if collector.Logger != nil {
		collector.Logger.Info("google flood forecasts stored", "gauges", len(statuses), "areas", len(shapes))
	}
	return len(statuses) + len(shapes), nil
}

func (collector *Collector) base() string {
	if collector.BaseURL != "" {
		return strings.TrimRight(collector.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (collector *Collector) header() http.Header {
	return http.Header{"X-Goog-Api-Key": {collector.APIKey}, "Content-Type": {"application/json"}, "Accept": {"application/json"}}
}

func (collector *Collector) post(ctx context.Context, path string, body any) ([]byte, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	response, err := collector.Client.Do(ctx, source.Request{Method: http.MethodPost, URL: collector.base() + path, Header: collector.header(), Body: encoded})
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func (collector *Collector) get(ctx context.Context, path string) ([]byte, error) {
	response, err := collector.Client.Do(ctx, source.Request{URL: collector.base() + path, Header: collector.header()})
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

type statusRequest struct {
	RegionCode                string `json:"regionCode"`
	IncludeNonQualityVerified bool   `json:"includeNonQualityVerified,omitempty"`
	PageSize                  int    `json:"pageSize"`
	PageToken                 string `json:"pageToken,omitempty"`
}

type searchRequest struct {
	// CountryCodes narrows flashFloods:search; significantEvents:search takes
	// no such filter, so its results are read worldwide and kept by country.
	CountryCodes []string `json:"countryCodes,omitempty"`
	PageSize     int      `json:"pageSize"`
	PageToken    string   `json:"pageToken,omitempty"`
}

// statuses pages through Sri Lanka's gauges.
func (collector *Collector) statuses(ctx context.Context) ([]Status, error) {
	var all []Status
	seen := map[string]bool{}
	token := ""
	for range maxPages {
		payload, err := collector.post(ctx, "/v1/floodStatus:searchLatestFloodStatusByArea", statusRequest{
			RegionCode: RegionCode, IncludeNonQualityVerified: collector.IncludeUnverified, PageSize: 1000, PageToken: token,
		})
		if err != nil {
			return nil, err
		}
		page, next, _, err := DecodeStatuses(payload)
		if err != nil {
			return nil, err
		}
		for _, status := range page {
			if !seen[status.GaugeID] {
				seen[status.GaugeID] = true
				all = append(all, status)
			}
		}
		// A page can be empty and still have a next one, and DecodeStatuses
		// leaves out anything unusable, so only the token ends the loop.
		if next == "" || next == token {
			return all, nil
		}
		token = next
	}
	collector.warnTruncated("/v1/floodStatus:searchLatestFloodStatusByArea")
	return all, nil
}

// models reads the thresholds of the gauges, in batches.
func (collector *Collector) models(ctx context.Context, statuses []Status) (map[string]Model, error) {
	models := map[string]Model{}
	for start := 0; start < len(statuses); start += modelBatch {
		query := url.Values{}
		for _, status := range statuses[start:min(start+modelBatch, len(statuses))] {
			query.Add("names", "gaugeModels/"+status.GaugeID)
		}
		payload, err := collector.get(ctx, "/v1/gaugeModels:batchGet?"+query.Encode())
		if err != nil {
			return nil, err
		}
		batch, err := DecodeModels(payload)
		if err != nil {
			return nil, err
		}
		for id, model := range batch {
			models[id] = model
		}
	}
	return models, nil
}

// search pages through a search and keeps what affects Sri Lanka. Even a
// filtered search is checked by country again: decode keeps only LK.
func (collector *Collector) search(ctx context.Context, path string, countries []string, decode func([]byte) ([]Area, string, error)) ([]Area, error) {
	var all []Area
	token := ""
	for range maxPages {
		payload, err := collector.post(ctx, path, searchRequest{CountryCodes: countries, PageSize: searchPageSize, PageToken: token})
		if err != nil {
			return nil, err
		}
		areas, next, err := decode(payload)
		if err != nil {
			return nil, err
		}
		all = append(all, areas...)
		if next == "" || next == token {
			return all, nil
		}
		token = next
	}
	collector.warnTruncated(path)
	return all, nil
}

func (collector *Collector) warnTruncated(path string) {
	if collector.Logger != nil {
		collector.Logger.Warn("google flood search stopped at its page limit; later pages were not read", "path", path, "pages", maxPages)
	}
}

// PrioritiseAreas orders areas for reading when polygons are limited: flash
// floods and significant events, then verified gauges' areas, then lower-confidence
// gauges' areas, each from the most severe down. It does no I/O.
func PrioritiseAreas(areas []Area, statuses []Status) []Area {
	verified := make(map[string]bool, len(statuses))
	severities := make(map[string]int, len(statuses))
	for _, status := range statuses {
		verified[status.GaugeID] = status.QualityVerified
		severities[status.GaugeID] = status.Severity.Rank()
	}
	tier := func(area Area) int {
		switch {
		case area.GaugeID == "":
			return 0
		case verified[area.GaugeID]:
			return 1
		}
		return 2
	}
	ordered := slices.Clone(areas)
	slices.SortStableFunc(ordered, func(a, b Area) int {
		if order := cmp.Compare(tier(a), tier(b)); order != 0 {
			return order
		}
		return cmp.Compare(severities[b.GaugeID], severities[a.GaugeID])
	})
	return ordered
}

// polygons reads each area's polygon, up to maxPolygons. A polygon in known is
// used as it is: Google's serialized polygon IDs name one shape. An area whose
// polygon cannot be read is left out and logged; the gauges still count.
func (collector *Collector) polygons(ctx context.Context, areas []Area, known map[string][]byte) map[string][]byte {
	shapes := map[string][]byte{}
	skipped := 0
	for _, area := range areas {
		if shapes[area.PolygonID] != nil {
			continue
		}
		if len(shapes) == maxPolygons {
			skipped++
			continue
		}
		if shape := known[area.PolygonID]; shape != nil {
			shapes[area.PolygonID] = shape
			continue
		}
		payload, err := collector.get(ctx, "/v1/serializedPolygons/"+url.PathEscape(area.PolygonID))
		if err == nil {
			var shape []byte
			if shape, err = ParsePolygon(payload); err == nil {
				shapes[area.PolygonID] = shape
				continue
			}
		}
		if collector.Logger != nil {
			collector.Logger.Warn("google flood polygon unreadable", "kind", area.Kind, "error", err)
		}
	}
	if skipped > 0 && collector.Logger != nil {
		collector.Logger.Warn("google flood polygons over the limit were not read", "limit", maxPolygons, "skipped", skipped)
	}
	return shapes
}

func (collector *Collector) save(ctx context.Context, statuses []Status, models map[string]Model, areas []Area, shapes map[string][]byte, now time.Time) error {
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		if err := queries.DeleteFloodGauges(ctx); err != nil {
			return err
		}
		for _, status := range statuses {
			model := models[status.GaugeID]
			if err := queries.InsertFloodGauge(ctx, store.InsertFloodGaugeParams{
				GaugeID: status.GaugeID, Source: SourceName, GaugeSource: status.GaugeSource, SiteName: status.SiteName,
				River: status.River, Lon: status.Lon, Lat: status.Lat, QualityVerified: status.QualityVerified,
				Severity: string(status.Severity), Trend: status.Trend, IssuedAt: status.IssuedAt,
				ForecastStart: status.ForecastStart, ForecastEnd: status.ForecastEnd,
				WarningLevel: model.WarningLevel, DangerLevel: model.DangerLevel, ExtremeDangerLevel: model.ExtremeLevel,
				ValueUnit: model.Unit, FetchedAt: now,
			}); err != nil {
				return fmt.Errorf("store gauge %s: %w", status.GaugeID, err)
			}
		}
		seen, err := firstSeen(ctx, queries, otherKinds)
		if err != nil {
			return err
		}
		if err := queries.DeleteFloodAreasNotOfKinds(ctx, FlashKinds); err != nil {
			return err
		}
		_, err = insertAreas(ctx, queries, areas, shapes, seen, now)
		return err
	})
}

// FlashKinds are the area kinds FlashCollector keeps.
var FlashKinds = []string{KindFlashLikely, KindFlashHighly}

// otherKinds are the kinds Collector keeps: every kind but the flash floods.
var otherKinds = []string{KindNotification, "inundation_high", "inundation_medium", "inundation_low", KindSignificantEvent}

// sighting is a stored area's first sighting and shape.
type sighting struct {
	at    time.Time
	shape []byte
}

// firstSeen reads when the stored areas of some kinds were first seen.
func firstSeen(ctx context.Context, queries *store.Queries, kinds []string) (map[string]sighting, error) {
	rows, err := queries.ListFloodAreasSeen(ctx, kinds)
	if err != nil {
		return nil, fmt.Errorf("read stored areas: %w", err)
	}
	seen := make(map[string]sighting, len(rows))
	for _, row := range rows {
		seen[row.ID] = sighting{at: row.FirstSeenAt, shape: []byte(row.Geojson)}
	}
	return seen, nil
}

// insertAreas stores the areas that have a shape, keeping an area's first
// sighting. It returns the areas seen for the first time.
func insertAreas(ctx context.Context, queries *store.Queries, areas []Area, shapes map[string][]byte, seen map[string]sighting, now time.Time) ([]Area, error) {
	var fresh []Area
	for _, area := range areas {
		shape := shapes[area.PolygonID]
		if shape == nil {
			continue
		}
		properties, err := json.Marshal(area.Properties)
		if err != nil || area.Properties == nil {
			properties = []byte("{}")
		}
		var gauge *string
		if area.GaugeID != "" {
			gauge = &area.GaugeID
		}
		first := now
		if earlier, ok := seen[area.PolygonID]; ok {
			first = earlier.at
		} else {
			fresh = append(fresh, area)
		}
		if err := queries.InsertFloodArea(ctx, store.InsertFloodAreaParams{
			ID: area.PolygonID, Kind: area.Kind, Source: SourceName, GaugeID: gauge, Severity: area.Severity,
			IssuedAt: area.IssuedAt, ValidFrom: area.ValidFrom, ValidUntil: area.ValidUntil,
			Properties: properties, Geojson: string(shape), FetchedAt: now, FirstSeenAt: first,
		}); err != nil {
			return nil, fmt.Errorf("store %s area: %w", area.Kind, err)
		}
	}
	return fresh, nil
}

// Prune implements source.Pruner: nothing fetched more than a day ago is
// kept, as Google's terms require, even when collection has stopped.
func (collector *Collector) Prune(ctx context.Context, now time.Time) error {
	queries := store.New(collector.Pool)
	if err := queries.DeleteFloodGaugesFetchedBefore(ctx, now.Add(-Retention)); err != nil {
		return err
	}
	return queries.DeleteFloodAreasFetchedBefore(ctx, now.Add(-Retention))
}
