package httpapi

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/prabhavalabs/lanka-weather/internal/places"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
)

const (
	tileContentType = "application/vnd.mapbox-vector-tile"
	// Boundaries change at most weekly; a day's caching keeps the map fast
	// and a reload visible the next day. Entity tags make revalidation cheap.
	boundaryCacheControl = "public, max-age=86400, stale-while-revalidate=604800"
	searchCacheControl   = "public, max-age=3600"
)

// Sri Lanka with a margin, for point lookups.
const minLon, maxLon, minLat, maxLat = 79.0, 82.5, 5.5, 10.5

// regionCollection returns one level's regions as GeoJSON with simplified
// boundaries: GET /v1/regions?level=province|district|ds|gn&parent=LK-21.
// GN divisions are only served within a district or DS division; for the
// whole country use the vector tiles.
func (api *api) regionCollection(w http.ResponseWriter, r *http.Request) {
	if api.deps.Regions == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "regions are not configured")
		return
	}
	level, ok := regions.ParseLevel(r.URL.Query().Get("level"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_level", "level must be province, district, ds or gn")
		return
	}
	ancestor := ""
	if raw := r.URL.Query().Get("parent"); raw != "" {
		id, parentLevel, ok := regions.ParseID(raw)
		if !ok || parentLevel.Rank() >= level.Rank() {
			writeError(w, http.StatusBadRequest, "invalid_parent", "parent must be the code of a region above the requested level")
			return
		}
		ancestor = id
		if level == regions.LevelGN && parentLevel == regions.LevelProvince {
			writeError(w, http.StatusBadRequest, "invalid_parent", "GN divisions are served within a district or DS division")
			return
		}
	} else if level == regions.LevelGN {
		writeError(w, http.StatusBadRequest, "parent_required", "GN divisions need a district or DS division parent; use the vector tiles for the whole country")
		return
	}
	document, err := api.deps.Regions.Collection(r.Context(), level, ancestor)
	if err != nil {
		api.internalError(w, "region collection", err)
		return
	}
	writeDocument(w, r, document, "application/geo+json; charset=utf-8", boundaryCacheControl)
}

// region returns one region's detail: GET /v1/regions/{id}?geometry=true.
func (api *api) region(w http.ResponseWriter, r *http.Request) {
	if api.deps.Regions == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "regions are not configured")
		return
	}
	id, _, ok := regions.ParseID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_region_id", "region ids look like LK-2, LK-21, LK-2103 or LK-2103005")
		return
	}
	geometry := false
	if raw := r.URL.Query().Get("geometry"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_geometry", "geometry must be true or false")
			return
		}
		geometry = parsed
	}
	detail, err := api.deps.Regions.Region(r.Context(), id, geometry)
	if errors.Is(err, regions.ErrRegionNotFound) {
		writeError(w, http.StatusNotFound, "region_not_found", "unknown region")
		return
	}
	if err != nil {
		api.internalError(w, "region", err)
		return
	}
	w.Header().Set("Cache-Control", boundaryCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: detail})
}

// regionLookup returns the regions containing a point:
// GET /v1/regions/lookup?lat=6.9271&lon=79.8612.
func (api *api) regionLookup(w http.ResponseWriter, r *http.Request) {
	if api.deps.Regions == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "regions are not configured")
		return
	}
	lat, latErr := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, lonErr := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if latErr != nil || lonErr != nil || math.IsNaN(lat) || math.IsNaN(lon) {
		writeError(w, http.StatusBadRequest, "invalid_point", "lat and lon must be decimal degrees")
		return
	}
	if lat < minLat || lat > maxLat || lon < minLon || lon > maxLon {
		writeError(w, http.StatusBadRequest, "out_of_range", "the point is outside Sri Lanka")
		return
	}
	lookup, err := api.deps.Regions.Lookup(r.Context(), lon, lat)
	if err != nil {
		api.internalError(w, "region lookup", err)
		return
	}
	w.Header().Set("Cache-Control", boundaryCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: lookup})
}

// regionTile returns a Mapbox Vector Tile of the boundaries:
// GET /v1/tiles/regions/{z}/{x}/{y}.pbf. A tile with no boundaries is 204.
func (api *api) regionTile(w http.ResponseWriter, r *http.Request) {
	if api.deps.Regions == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "regions are not configured")
		return
	}
	z, zErr := strconv.Atoi(chi.URLParam(r, "z"))
	x, xErr := strconv.Atoi(chi.URLParam(r, "x"))
	y, yErr := strconv.Atoi(chi.URLParam(r, "y"))
	if zErr != nil || xErr != nil || yErr != nil || z < 0 || z > regions.TileMaxZoom ||
		x < 0 || y < 0 || x >= 1<<z || y >= 1<<z {
		writeError(w, http.StatusBadRequest, "invalid_tile", "tile coordinates must be integers with 0 <= z <= 14 and 0 <= x, y < 2^z")
		return
	}
	document, err := api.deps.Regions.Tile(r.Context(), z, x, y)
	if err != nil {
		api.internalError(w, "region tile", err)
		return
	}
	writeDocument(w, r, document, tileContentType, boundaryCacheControl)
}

// searchPlaces finds places and regions by name:
// GET /v1/places/search?q=kandy&limit=10.
func (api *api) searchPlaces(w http.ResponseWriter, r *http.Request) {
	if api.deps.Places == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "search is not configured")
		return
	}
	query := places.NormalizeQuery(r.URL.Query().Get("q"))
	if query == "" || utf8.RuneCountInString(query) > places.MaxQueryLength || !utf8.ValidString(query) {
		writeError(w, http.StatusBadRequest, "invalid_query", "q must be 1 to 100 characters")
		return
	}
	limit := places.DefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > places.MaxLimit {
			writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be a whole number from 1 to 50")
			return
		}
		limit = parsed
	}
	result, err := api.deps.Places.Search(r.Context(), query, limit)
	if err != nil {
		api.internalError(w, "place search", err)
		return
	}
	w.Header().Set("Cache-Control", searchCacheControl)
	writeJSON(w, http.StatusOK, Envelope{Data: result})
}

// writeDocument sends a rendered body with its entity tag, or 304 when the
// client already has it. An empty body is sent as 204.
func writeDocument(w http.ResponseWriter, r *http.Request, document regions.Document, contentType, cacheControl string) {
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("ETag", document.ETag)
	if matches(r.Header.Get("If-None-Match"), document.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if len(document.Body) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document.Body)
}

// matches reports whether an If-None-Match header names the entity tag,
// comparing weakly as RFC 9110 requires for If-None-Match.
func matches(header, etag string) bool {
	if header == "" || etag == "" {
		return false
	}
	want := strings.TrimPrefix(etag, "W/")
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == want {
			return true
		}
	}
	return false
}
