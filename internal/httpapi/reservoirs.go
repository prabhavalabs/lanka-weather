package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
)

// Reservoirs serves the reservoirs layer.
type Reservoirs interface {
	Layer(ctx context.Context) (reservoirs.FeatureCollection, error)
	Areas(ctx context.Context) (reservoirs.Areas, error)
	History(ctx context.Context, id string, days int) (reservoirs.History, error)
}

const (
	defaultReservoirDays = 60
	maxReservoirDays     = 400
)

// reservoirLayer returns every reservoir, bare GeoJSON like the river layer.
// Reservoirs with no known position have a null geometry.
func (api *api) reservoirLayer(w http.ResponseWriter, r *http.Request) {
	if api.deps.Reservoirs == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "reservoirs are not configured")
		return
	}
	collection, err := api.deps.Reservoirs.Layer(r.Context())
	if err != nil {
		api.internalError(w, "reservoirs", err)
		return
	}
	writeCachedGeoJSON(w, collection, "public, max-age=300")
}

// reservoirAreas returns the polygons of mapped reservoirs.
func (api *api) reservoirAreas(w http.ResponseWriter, r *http.Request) {
	if api.deps.Reservoirs == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "reservoirs are not configured")
		return
	}
	areas, err := api.deps.Reservoirs.Areas(r.Context())
	if err != nil {
		api.internalError(w, "reservoir areas", err)
		return
	}
	writeCachedGeoJSON(w, areas, "public, max-age=300")
}

func (api *api) reservoirReadings(w http.ResponseWriter, r *http.Request) {
	if api.deps.Reservoirs == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "reservoirs are not configured")
		return
	}
	days := defaultReservoirDays
	if raw := r.URL.Query().Get("days"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxReservoirDays {
			writeError(w, http.StatusBadRequest, "invalid_days", "days must be a whole number from 1 to 400")
			return
		}
		days = parsed
	}
	history, err := api.deps.Reservoirs.History(r.Context(), chi.URLParam(r, "id"), days)
	if errors.Is(err, reservoirs.ErrNotFound) {
		writeError(w, http.StatusNotFound, "reservoir_not_found", "unknown reservoir")
		return
	}
	if err != nil {
		api.internalError(w, "reservoir readings", err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, Envelope{Data: history})
}
