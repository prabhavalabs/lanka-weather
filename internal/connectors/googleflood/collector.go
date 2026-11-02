package googleflood

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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
	// Interval: gauge statuses are reissued about daily and flash floods a
	// few times a day; hourly picks a new issue up soon at about five
	// requests a run.
	Interval = time.Hour
	// Retention is the longest Google's terms let gauge search results be
	// kept.
	Retention = 24 * time.Hour
	// maxPages bounds one search, maxPolygons the polygons read in a run.
	maxPages    = 5
	maxPolygons = 40
	// modelBatch is the number of gauge models asked for in one request.
	modelBatch = 50
)

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
	// which Flood Hub shows only in its expert mode. Off by default.
	IncludeUnverified bool
	// BaseURL overrides DefaultBaseURL, for tests.
	BaseURL string
	Logger  *slog.Logger
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector.
func (collector *Collector) Interval() time.Duration { return Interval }

// Collect reads the latest statuses, thresholds, flash floods, significant
// events and polygons for Sri Lanka and replaces what is stored. It returns
// the number of gauges and areas stored.
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
	flash, err := collector.search(ctx, "/v1/flashFloods:search", DecodeFlashFloods)
	if err != nil {
		return 0, fmt.Errorf("flash floods: %w", err)
	}
	events, err := collector.search(ctx, "/v1/significantEvents:search", DecodeSignificantEvents)
	if err != nil {
		return 0, fmt.Errorf("significant events: %w", err)
	}
	var areas []Area
	for _, status := range statuses {
		areas = append(areas, GaugeAreas(status)...)
	}
	areas = append(append(areas, flash...), events...)
	shapes := collector.polygons(ctx, areas)
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
	PageSize  int    `json:"pageSize"`
	PageToken string `json:"pageToken,omitempty"`
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
		if next == "" || next == token || len(page) == 0 {
			break
		}
		token = next
	}
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

// search pages through a worldwide search and keeps what affects Sri
// Lanka; these endpoints take no region filter.
func (collector *Collector) search(ctx context.Context, path string, decode func([]byte) ([]Area, string, error)) ([]Area, error) {
	var all []Area
	token := ""
	for range maxPages {
		payload, err := collector.post(ctx, path, searchRequest{PageSize: 1000, PageToken: token})
		if err != nil {
			return nil, err
		}
		areas, next, err := decode(payload)
		if err != nil {
			return nil, err
		}
		all = append(all, areas...)
		if next == "" || next == token {
			break
		}
		token = next
	}
	return all, nil
}

// polygons reads each area's polygon, up to maxPolygons. An area whose
// polygon cannot be read is left out and logged; the gauges still count.
func (collector *Collector) polygons(ctx context.Context, areas []Area) map[string][]byte {
	shapes := map[string][]byte{}
	for _, area := range areas {
		if len(shapes) == maxPolygons || shapes[area.PolygonID] != nil {
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
		if err := queries.DeleteFloodAreas(ctx); err != nil {
			return err
		}
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
			if err := queries.InsertFloodArea(ctx, store.InsertFloodAreaParams{
				ID: area.PolygonID, Kind: area.Kind, Source: SourceName, GaugeID: gauge, Severity: area.Severity,
				IssuedAt: area.IssuedAt, ValidFrom: area.ValidFrom, ValidUntil: area.ValidUntil,
				Properties: properties, Geojson: string(shape), FetchedAt: now,
			}); err != nil {
				return fmt.Errorf("store %s area: %w", area.Kind, err)
			}
		}
		return nil
	})
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
