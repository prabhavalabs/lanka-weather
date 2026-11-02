package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/prabhavalabs/lanka-weather/internal/rainfall"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
)

// Stations serves Department of Meteorology observations and rainfall totals.
type Stations interface {
	Layer(ctx context.Context) (stations.FeatureCollection, error)
	Station(ctx context.Context, id string, days int) (*stations.Detail, error)
	ObservedRainfall(ctx context.Context) (*stations.RainCollection, error)
}

// Rain serves the satellite rain estimate.
type Rain interface {
	Steps(ctx context.Context) (*rainfall.Steps, error)
	Tile(ctx context.Context, timeID string, z, x, y int) ([]byte, error)
}

// Environment serves the air-quality and sea grids.
type Environment interface {
	AirJSON(ctx context.Context) (plain, gzipped []byte, err error)
	SeaJSON(ctx context.Context) (plain, gzipped []byte, err error)
}

const (
	defaultStationDays = 7
	maxStationDays     = 45
)

// stationLayer returns the weather-stations layer as GeoJSON, bare like other
// map data.
func (api *api) stationLayer(w http.ResponseWriter, r *http.Request) {
	if api.deps.Stations == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "weather stations are not configured")
		return
	}
	collection, err := api.deps.Stations.Layer(r.Context())
	if err != nil {
		api.internalError(w, "weather stations", err)
		return
	}
	writeCachedGeoJSON(w, collection, "public, max-age=300")
}

func (api *api) stationDetail(w http.ResponseWriter, r *http.Request) {
	if api.deps.Stations == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "weather stations are not configured")
		return
	}
	days := defaultStationDays
	if raw := r.URL.Query().Get("days"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxStationDays {
			writeError(w, http.StatusBadRequest, "invalid_days", "days must be a whole number from 1 to 45")
			return
		}
		days = parsed
	}
	detail, err := api.deps.Stations.Station(r.Context(), chi.URLParam(r, "id"), days)
	if errors.Is(err, stations.ErrStationNotFound) {
		writeError(w, http.StatusNotFound, "station_not_found", "unknown station")
		return
	}
	if err != nil {
		api.internalError(w, "weather station", err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, Envelope{Data: detail})
}

// rainObserved returns 24-hour totals at stations and gauges as GeoJSON.
func (api *api) rainObserved(w http.ResponseWriter, r *http.Request) {
	if api.deps.Stations == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "weather stations are not configured")
		return
	}
	collection, err := api.deps.Stations.ObservedRainfall(r.Context())
	if err != nil {
		api.internalError(w, "observed rainfall", err)
		return
	}
	writeCachedGeoJSON(w, collection, "public, max-age=300")
}

// rainSteps lists the satellite images there are, bare like other map data.
func (api *api) rainSteps(w http.ResponseWriter, r *http.Request) {
	if api.deps.Rain == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "the rain layer is not configured")
		return
	}
	steps, err := api.deps.Rain.Steps(r.Context())
	if err != nil {
		api.internalError(w, "rain steps", err)
		return
	}
	if len(steps.Steps) == 0 {
		writeError(w, http.StatusServiceUnavailable, "no_data", "satellite rain has not been collected yet")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=120")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(steps)
}

// rainTile serves one stored tile. A half hour's image never changes once
// stored, so it can be cached for good.
func (api *api) rainTile(w http.ResponseWriter, r *http.Request) {
	if api.deps.Rain == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "the rain layer is not configured")
		return
	}
	z, zErr := strconv.Atoi(chi.URLParam(r, "z"))
	x, xErr := strconv.Atoi(chi.URLParam(r, "x"))
	y, yErr := strconv.Atoi(chi.URLParam(r, "y"))
	if zErr != nil || xErr != nil || yErr != nil {
		writeError(w, http.StatusBadRequest, "invalid_tile", "z, x and y must be whole numbers")
		return
	}
	body, err := api.deps.Rain.Tile(r.Context(), chi.URLParam(r, "time"), z, x, y)
	if errors.Is(err, rainfall.ErrTileNotFound) {
		writeError(w, http.StatusNotFound, "tile_not_found", "no rain image for that time and place")
		return
	}
	if err != nil {
		api.internalError(w, "rain tile", err)
		return
	}
	header := w.Header()
	header.Set("Content-Type", "image/png")
	header.Set("Cache-Control", "public, max-age=604800, immutable")
	header.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// airCurrent returns the air-quality grid, bare like the weather grid.
func (api *api) airCurrent(w http.ResponseWriter, r *http.Request) {
	if api.deps.Environment == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "air quality is not configured")
		return
	}
	plain, gzipped, err := api.deps.Environment.AirJSON(r.Context())
	if api.weatherError(w, "air quality", err) {
		return
	}
	writeEncoded(w, r, plain, gzipped, "public, max-age=300")
}

// seaCurrent returns the sea-state grid, bare like the weather grid.
func (api *api) seaCurrent(w http.ResponseWriter, r *http.Request) {
	if api.deps.Environment == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "sea conditions are not configured")
		return
	}
	plain, gzipped, err := api.deps.Environment.SeaJSON(r.Context())
	if api.weatherError(w, "sea conditions", err) {
		return
	}
	writeEncoded(w, r, plain, gzipped, "public, max-age=300")
}

func writeCachedGeoJSON(w http.ResponseWriter, body any, cacheControl string) {
	w.Header().Set("Content-Type", "application/geo+json; charset=utf-8")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
