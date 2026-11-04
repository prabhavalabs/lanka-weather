package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

const (
	// Warnings change hourly at most, and must not linger in a cache.
	warningsCacheControl = "public, max-age=60"
	maxWarningsLimit     = 200
)

// warningsOverview returns bulletins, what is in force, the banner and the
// districts to highlight: GET /v1/warnings?type=&district=&limit=.
func (api *api) warningsOverview(w http.ResponseWriter, r *http.Request) {
	if api.deps.Warnings == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "official warnings are not configured")
		return
	}
	query := r.URL.Query()
	filter := warnings.Filter{Type: query.Get("type")}
	if filter.Type != "" && !warnings.ValidType(filter.Type) {
		writeError(w, http.StatusBadRequest, "invalid_type", "type must be landslide, water_level, weather or situation")
		return
	}
	if raw := query.Get("district"); raw != "" {
		id, level, ok := regions.ParseID(raw)
		if !ok || level != regions.LevelDistrict {
			writeError(w, http.StatusBadRequest, "invalid_district", "district must be a district code such as LK-21")
			return
		}
		filter.District = id
	}
	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxWarningsLimit {
			writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be a whole number from 1 to 200")
			return
		}
		filter.Limit = limit
	}
	overview, err := api.deps.Warnings.Overview(r.Context(), filter)
	if err != nil {
		api.internalError(w, "warnings overview", err)
		return
	}
	w.Header().Set("Cache-Control", warningsCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: overview})
}

// landslideWarnings returns the landslide-warnings layer as bare GeoJSON.
func (api *api) landslideWarnings(w http.ResponseWriter, r *http.Request) {
	if api.deps.Warnings == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "official warnings are not configured")
		return
	}
	layer, err := api.deps.Warnings.Landslides(r.Context())
	if err != nil {
		api.internalError(w, "landslide warnings", err)
		return
	}
	writeGeoJSON(w, layer)
}

// landslideDivision returns one DS division's level and recent history.
func (api *api) landslideDivision(w http.ResponseWriter, r *http.Request) {
	if api.deps.Warnings == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "official warnings are not configured")
		return
	}
	division, err := api.deps.Warnings.Division(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, warnings.ErrDivisionNotFound) {
		writeError(w, http.StatusNotFound, "division_not_found", "unknown DS division")
		return
	}
	if err != nil {
		api.internalError(w, "landslide division", err)
		return
	}
	w.Header().Set("Cache-Control", warningsCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: division})
}

// earthquakes returns the earthquakes layer as bare GeoJSON.
func (api *api) earthquakes(w http.ResponseWriter, r *http.Request) {
	if api.deps.Hazards == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "hazards are not configured")
		return
	}
	layer, err := api.deps.Hazards.Earthquakes(r.Context())
	if err != nil {
		api.internalError(w, "earthquakes", err)
		return
	}
	writeGeoJSON(w, layer)
}

// cyclones returns the cyclones layer as bare GeoJSON.
func (api *api) cyclones(w http.ResponseWriter, r *http.Request) {
	if api.deps.Hazards == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "hazards are not configured")
		return
	}
	layer, err := api.deps.Hazards.Cyclones(r.Context())
	if err != nil {
		api.internalError(w, "cyclones", err)
		return
	}
	writeGeoJSON(w, layer)
}

func writeGeoJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/geo+json; charset=utf-8")
	w.Header().Set("Cache-Control", warningsCacheControl)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
