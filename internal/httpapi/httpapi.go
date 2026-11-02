// Package httpapi exposes lanka-weather's read API.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/layers"
	"github.com/prabhavalabs/lanka-weather/internal/places"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/summary"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// Envelope is the shape of every JSON response.
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error is a machine-readable error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Rivers serves the river-levels layer.
type Rivers interface {
	Layer(ctx context.Context) (rivers.FeatureCollection, error)
	StationHistory(ctx context.Context, id string, period time.Duration) (rivers.History, error)
}

// Regions serves administrative regions.
type Regions interface {
	Collection(ctx context.Context, level regions.Level, ancestor string) (regions.Document, error)
	Region(ctx context.Context, id string, geometry bool) (regions.Detail, error)
	Lookup(ctx context.Context, lon, lat float64) (regions.Lookup, error)
	Tile(ctx context.Context, z, x, y int) (regions.Document, error)
}

// Places searches places and regions by name.
type Places interface {
	Search(ctx context.Context, query string, limit int) (places.Result, error)
}

// Weather serves the weather grid, point forecasts, regions and wind.
type Weather interface {
	CurrentGridJSON(ctx context.Context) (plain, gzipped []byte, err error)
	Point(ctx context.Context, lon, lat float64, hours, days int) (*weather.Point, error)
	WindJSON(ctx context.Context, at time.Time) (plain, gzipped []byte, err error)
	RegionWeather(ctx context.Context, level, parent string) (*weather.RegionsResponse, error)
}

// Floods serves the Google flood-forecast layer.
type Floods interface {
	Layer(ctx context.Context) (floods.FeatureCollection, error)
}

// Summaries serves the location summary.
type Summaries interface {
	Summary(ctx context.Context, lon, lat float64) (*summary.Summary, error)
}

// Warnings serves official warnings: landslide levels and DMC bulletins.
type Warnings interface {
	Landslides(ctx context.Context) (warnings.LandslideLayer, error)
	Division(ctx context.Context, id string) (*warnings.Division, error)
	Overview(ctx context.Context, filter warnings.Filter) (*warnings.Overview, error)
}

// Hazards serves the earthquake and cyclone layers.
type Hazards interface {
	Earthquakes(ctx context.Context) (hazards.EarthquakeLayer, error)
	Cyclones(ctx context.Context) (hazards.CycloneLayer, error)
}

// Dependencies are the services the API reads from.
type Dependencies struct {
	Logger      *slog.Logger
	Rivers      Rivers
	Regions     Regions
	Places      Places
	Weather     Weather
	Floods      Floods
	Summary     Summaries
	Stations    Stations
	Rain        Rain
	Environment Environment
	// RiverRisk serves the river network, its risk estimate and flood basins.
	RiverRisk RiverRisk

	// Warnings and Hazards serve official warnings and the hazard layers.
	Warnings Warnings
	Hazards  Hazards

	// Now is the clock for requests without an explicit time; nil is
	// time.Now.
	Now func() time.Time
	// Ready reports whether the service can serve requests (database reachable).
	Ready func(ctx context.Context) error
}

const (
	defaultHistoryHours = 72
	maxHistoryHours     = 30 * 24
)

// NewRouter returns the API's HTTP handler.
func NewRouter(deps Dependencies) http.Handler {
	api := &api{deps: deps}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(deps.Logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, Envelope{Data: map[string]string{"status": "ok"}})
	})
	r.Get("/readyz", api.ready)

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.Compress(5, "application/json", "application/geo+json", tileContentType))
		r.Get("/layers", listLayers)
		r.Get("/layers/{id}", getLayer)
		r.Get("/rivers/levels", api.riverLevels)
		r.Get("/rivers/stations/{id}/readings", api.stationReadings)
		r.Get("/regions", api.regionCollection)
		r.Get("/regions/lookup", api.regionLookup)
		r.Get("/regions/{id}", api.region)
		r.Get("/tiles/regions/{z}/{x}/{y}.pbf", api.regionTile)
		r.Get("/places/search", api.searchPlaces)
		r.Get("/weather/current", api.weatherCurrent)
		r.Get("/weather/point", api.weatherPoint)
		r.Get("/weather/wind", api.weatherWind)
		r.Get("/weather/regions", api.weatherRegions)
		r.Get("/floods/forecast", api.floodForecast)
		r.Get("/summary", api.summary)
		api.riverRoutes(r)
		r.Get("/warnings", api.warningsOverview)
		r.Get("/warnings/landslides", api.landslideWarnings)
		r.Get("/warnings/landslides/{id}", api.landslideDivision)
		r.Get("/hazards/earthquakes", api.earthquakes)
		r.Get("/hazards/cyclones", api.cyclones)
		r.Get("/stations", api.stationLayer)
		r.Get("/stations/{id}", api.stationDetail)
		r.Get("/rain/steps", api.rainSteps)
		r.Get("/rain/observed", api.rainObserved)
		r.Get("/rain/tiles/{time}/{z}/{x}/{y}.png", api.rainTile)
		r.Get("/air/current", api.airCurrent)
		r.Get("/sea/current", api.seaCurrent)
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	})
	return r
}

type api struct {
	deps Dependencies
}

func (api *api) ready(w http.ResponseWriter, r *http.Request) {
	if api.deps.Ready != nil {
		if err := api.deps.Ready(r.Context()); err != nil {
			api.deps.Logger.Warn("not ready", "error", err)
			writeError(w, http.StatusServiceUnavailable, "not_ready", "service is not ready")
			return
		}
	}
	writeJSON(w, http.StatusOK, Envelope{Data: map[string]string{"status": "ready"}})
}

func listLayers(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, Envelope{Data: layers.All()})
}

func getLayer(w http.ResponseWriter, r *http.Request) {
	layer, ok := layers.ByID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "layer_not_found", "unknown layer")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, Envelope{Data: layer})
}

// riverLevels returns the river-levels layer as a GeoJSON feature collection.
// It is returned bare, not in the envelope, so map libraries can load it
// directly; errors still use the envelope.
func (api *api) riverLevels(w http.ResponseWriter, r *http.Request) {
	if api.deps.Rivers == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "river levels are not configured")
		return
	}
	collection, err := api.deps.Rivers.Layer(r.Context())
	if err != nil {
		api.internalError(w, "river levels", err)
		return
	}
	w.Header().Set("Content-Type", "application/geo+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(collection)
}

func (api *api) stationReadings(w http.ResponseWriter, r *http.Request) {
	if api.deps.Rivers == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "river levels are not configured")
		return
	}
	hours := defaultHistoryHours
	if raw := r.URL.Query().Get("hours"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxHistoryHours {
			writeError(w, http.StatusBadRequest, "invalid_hours", "hours must be a whole number from 1 to 720")
			return
		}
		hours = parsed
	}
	history, err := api.deps.Rivers.StationHistory(r.Context(), chi.URLParam(r, "id"), time.Duration(hours)*time.Hour)
	if errors.Is(err, rivers.ErrStationNotFound) {
		writeError(w, http.StatusNotFound, "station_not_found", "unknown station")
		return
	}
	if err != nil {
		api.internalError(w, "station readings", err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, Envelope{Data: history})
}

func (api *api) internalError(w http.ResponseWriter, what string, err error) {
	api.deps.Logger.Error("request failed", "what", what, "error", err)
	writeError(w, http.StatusInternalServerError, "internal", "the request could not be completed")
}

func writeJSON(w http.ResponseWriter, status int, body Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, Envelope{Error: &Error{Code: code, Message: message}})
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
