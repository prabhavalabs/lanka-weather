package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
)

// RiverRisk serves the river network, the risk estimate and the flood basins.
type RiverRisk interface {
	Risk(ctx context.Context) (*riverrisk.RiskDocument, error)
	River(ctx context.Context, id string) (*riverrisk.RiverDetail, error)
	Basins(ctx context.Context) (*riverrisk.BasinsDocument, error)
	Basin(ctx context.Context, id string) (*riverrisk.BasinStatus, *riverrisk.BasinsMeta, error)
	PastFloods(ctx context.Context) (*riverrisk.PastFloodsDocument, error)
	RiverTile(ctx context.Context, z, x, y int) (regions.Document, error)
	BasinTile(ctx context.Context, z, x, y int) (regions.Document, error)
	PastFloodTile(ctx context.Context, z, x, y int) (regions.Document, error)
}

const (
	// riskCacheControl matches how often the estimate is recomputed.
	riskCacheControl = "public, max-age=60"
	// A rebuild changes the tile URL, so tiles can be cached for a long time.
	versionedTileCacheControl = "public, max-age=604800"
	// Past extents change a few times a year.
	pastCacheControl = "public, max-age=3600"
)

// riverRoutes adds this file's routes.
func (api *api) riverRoutes(r chi.Router) {
	r.Get("/rivers/risk", api.riverRiskAll)
	r.Get("/rivers/{id}", api.riverDetail)
	r.Get("/floods/basins", api.floodBasins)
	r.Get("/floods/basins/{id}", api.floodBasin)
	r.Get("/floods/past", api.pastFloods)
	r.Get("/tiles/rivers/{z}/{x}/{y}.pbf", api.riverTile)
	r.Get("/tiles/river-basins/{z}/{x}/{y}.pbf", api.basinTile)
	r.Get("/tiles/past-floods/{z}/{x}/{y}.pbf", api.pastFloodTile)
}

func (api *api) riverRiskReady(w http.ResponseWriter) bool {
	if api.deps.RiverRisk == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "river risk is not configured")
		return false
	}
	return true
}

// riverRiskAll returns every river and stretch: GET /v1/rivers/risk.
func (api *api) riverRiskAll(w http.ResponseWriter, r *http.Request) {
	if !api.riverRiskReady(w) {
		return
	}
	document, err := api.deps.RiverRisk.Risk(r.Context())
	if err != nil {
		api.internalError(w, "river risk", err)
		return
	}
	if len(document.Rivers) == 0 {
		writeError(w, http.StatusServiceUnavailable, "no_data", "the river network has not been collected yet")
		return
	}
	w.Header().Set("Cache-Control", riskCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: document})
}

// riverDetail returns one river: GET /v1/rivers/{id}.
func (api *api) riverDetail(w http.ResponseWriter, r *http.Request) {
	if !api.riverRiskReady(w) {
		return
	}
	detail, err := api.deps.RiverRisk.River(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, riverrisk.ErrNotFound) {
		writeError(w, http.StatusNotFound, "river_not_found", "unknown river")
		return
	}
	if err != nil {
		api.internalError(w, "river detail", err)
		return
	}
	w.Header().Set("Cache-Control", riskCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: detail})
}

// floodBasins returns every basin's state: GET /v1/floods/basins.
func (api *api) floodBasins(w http.ResponseWriter, r *http.Request) {
	if !api.riverRiskReady(w) {
		return
	}
	document, err := api.deps.RiverRisk.Basins(r.Context())
	if err != nil {
		api.internalError(w, "flood basins", err)
		return
	}
	if len(document.Basins) == 0 {
		writeError(w, http.StatusServiceUnavailable, "no_data", "the river basins have not been collected yet")
		return
	}
	w.Header().Set("Cache-Control", riskCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: document})
}

// floodBasin returns one basin: GET /v1/floods/basins/{id}.
func (api *api) floodBasin(w http.ResponseWriter, r *http.Request) {
	if !api.riverRiskReady(w) {
		return
	}
	basin, meta, err := api.deps.RiverRisk.Basin(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, riverrisk.ErrNotFound) {
		writeError(w, http.StatusNotFound, "basin_not_found", "unknown basin")
		return
	}
	if err != nil {
		api.internalError(w, "flood basin", err)
		return
	}
	w.Header().Set("Cache-Control", riskCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: struct {
		*riverrisk.BasinStatus
		Meta *riverrisk.BasinsMeta `json:"meta"`
	}{basin, meta}})
}

// pastFloods lists the mapped past events: GET /v1/floods/past.
func (api *api) pastFloods(w http.ResponseWriter, r *http.Request) {
	if !api.riverRiskReady(w) {
		return
	}
	document, err := api.deps.RiverRisk.PastFloods(r.Context())
	if err != nil {
		api.internalError(w, "past floods", err)
		return
	}
	w.Header().Set("Cache-Control", pastCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: document})
}

// tileCoordinates reads z/x/y, answering 400 for anything outside the pyramid.
func tileCoordinates(w http.ResponseWriter, r *http.Request) (z, x, y int, ok bool) {
	var zErr, xErr, yErr error
	z, zErr = strconv.Atoi(chi.URLParam(r, "z"))
	x, xErr = strconv.Atoi(chi.URLParam(r, "x"))
	y, yErr = strconv.Atoi(chi.URLParam(r, "y"))
	if zErr != nil || xErr != nil || yErr != nil || z < 0 || z > regions.TileMaxZoom ||
		x < 0 || y < 0 || x >= 1<<z || y >= 1<<z {
		writeError(w, http.StatusBadRequest, "invalid_tile", "tile coordinates must be integers with 0 <= z <= 14 and 0 <= x, y < 2^z")
		return 0, 0, 0, false
	}
	return z, x, y, true
}

type tileRenderer func(ctx context.Context, z, x, y int) (regions.Document, error)

func (api *api) serveTile(w http.ResponseWriter, r *http.Request, what string, render tileRenderer, cacheControl string) {
	z, x, y, ok := tileCoordinates(w, r)
	if !ok {
		return
	}
	document, err := render(r.Context(), z, x, y)
	if err != nil {
		api.internalError(w, what, err)
		return
	}
	writeDocument(w, r, document, tileContentType, cacheControl)
}

func (api *api) riverTile(w http.ResponseWriter, r *http.Request) {
	if api.riverRiskReady(w) {
		api.serveTile(w, r, "river tile", api.deps.RiverRisk.RiverTile, versionedTileCacheControl)
	}
}

func (api *api) basinTile(w http.ResponseWriter, r *http.Request) {
	if api.riverRiskReady(w) {
		api.serveTile(w, r, "basin tile", api.deps.RiverRisk.BasinTile, boundaryCacheControl)
	}
}

func (api *api) pastFloodTile(w http.ResponseWriter, r *http.Request) {
	if api.riverRiskReady(w) {
		api.serveTile(w, r, "past flood tile", api.deps.RiverRisk.PastFloodTile, boundaryCacheControl)
	}
}
