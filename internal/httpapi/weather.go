package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// Point forecasts span 72 hours and 7 days.
const (
	pointHours = 72
	pointDays  = 7
)

// weatherCurrent returns the grid of current values for map rendering, bare
// rather than in the envelope, like other map data.
func (api *api) weatherCurrent(w http.ResponseWriter, r *http.Request) {
	if api.deps.Weather == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "weather is not configured")
		return
	}
	plain, gzipped, err := api.deps.Weather.CurrentGridJSON(r.Context())
	if api.weatherError(w, "weather current", err) {
		return
	}
	writeEncoded(w, r, plain, gzipped, "public, max-age=300")
}

// weatherPoint returns current conditions, a 72-hour hourly and a 7-day
// daily forecast interpolated at a point.
func (api *api) weatherPoint(w http.ResponseWriter, r *http.Request) {
	if api.deps.Weather == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "weather is not configured")
		return
	}
	lon, lat, ok := coordinates(w, r)
	if !ok {
		return
	}
	point, err := api.deps.Weather.Point(r.Context(), lon, lat, pointHours, pointDays)
	if api.weatherError(w, "weather point", err) {
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, Envelope{Data: point})
}

// weatherWind returns the wind field valid nearest t (RFC 3339, default
// now), in the format the particle renderer reads, bare.
func (api *api) weatherWind(w http.ResponseWriter, r *http.Request) {
	if api.deps.Weather == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "weather is not configured")
		return
	}
	at := api.now()
	if raw := r.URL.Query().Get("t"); raw != "" && raw != "now" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_time", "t must be an RFC 3339 time such as 2026-11-02T12:00:00Z, or now")
			return
		}
		at = parsed
	}
	plain, gzipped, err := api.deps.Weather.WindJSON(r.Context(), at)
	if errors.Is(err, weather.ErrNoWind) {
		writeError(w, http.StatusNotFound, "no_wind_field", "no stored wind forecast is valid at that time")
		return
	}
	if api.weatherError(w, "weather wind", err) {
		return
	}
	writeEncoded(w, r, plain, gzipped, "public, max-age=600")
}

// weatherRegions returns current weather aggregated per administrative
// region of a level.
func (api *api) weatherRegions(w http.ResponseWriter, r *http.Request) {
	if api.deps.Weather == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "weather is not configured")
		return
	}
	level := r.URL.Query().Get("level")
	if !weather.ValidLevel(level) {
		writeError(w, http.StatusBadRequest, "invalid_level", "level must be one of province, district, ds or gn")
		return
	}
	regions, err := api.deps.Weather.RegionWeather(r.Context(), level)
	if errors.Is(err, weather.ErrRegionsUnavailable) {
		writeError(w, http.StatusServiceUnavailable, "not_available", "administrative regions are not loaded yet")
		return
	}
	if api.weatherError(w, "weather regions", err) {
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, Envelope{Data: regions})
}

// floodForecast returns Google's flood forecast for Sri Lanka as GeoJSON,
// bare like the river-levels layer.
func (api *api) floodForecast(w http.ResponseWriter, r *http.Request) {
	if api.deps.Floods == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "flood forecasts are not configured")
		return
	}
	collection, err := api.deps.Floods.Layer(r.Context())
	if err != nil {
		api.internalError(w, "flood forecast", err)
		return
	}
	w.Header().Set("Content-Type", "application/geo+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(collection)
}

// summary returns everything known about one place.
func (api *api) summary(w http.ResponseWriter, r *http.Request) {
	if api.deps.Summary == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "summaries are not configured")
		return
	}
	lon, lat, ok := coordinates(w, r)
	if !ok {
		return
	}
	result, err := api.deps.Summary.Summary(r.Context(), lon, lat)
	if err != nil {
		api.internalError(w, "summary", err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=120")
	writeJSON(w, http.StatusOK, Envelope{Data: result})
}

// weatherError writes the response for a weather error and reports whether
// there was one.
func (api *api) weatherError(w http.ResponseWriter, what string, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, weather.ErrOutsideGrid):
		writeError(w, http.StatusNotFound, "outside_grid", "the point is outside the Sri Lanka weather grid")
	case errors.Is(err, weather.ErrNoData):
		writeError(w, http.StatusServiceUnavailable, "no_data", "weather has not been collected yet")
	default:
		api.internalError(w, what, err)
	}
	return true
}

// coordinates reads and checks the lat and lon query parameters.
func coordinates(w http.ResponseWriter, r *http.Request) (lon, lat float64, ok bool) {
	query := r.URL.Query()
	lat, latErr := strconv.ParseFloat(query.Get("lat"), 64)
	lon, lonErr := strconv.ParseFloat(query.Get("lon"), 64)
	if latErr != nil || lonErr != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		writeError(w, http.StatusBadRequest, "invalid_coordinates", "lat and lon must be decimal degrees (lat -90 to 90, lon -180 to 180)")
		return 0, 0, false
	}
	return lon, lat, true
}

// writeEncoded writes a JSON body encoded once, gzipped when the client
// accepts it.
func writeEncoded(w http.ResponseWriter, r *http.Request, plain, gzipped []byte, cacheControl string) {
	header := w.Header()
	header.Set("Content-Type", "application/json; charset=utf-8")
	header.Set("Cache-Control", cacheControl)
	header.Add("Vary", "Accept-Encoding")
	if gzipped != nil && acceptsGzip(r.Header.Get("Accept-Encoding")) {
		header.Set("Content-Encoding", "gzip")
		header.Set("Content-Length", strconv.Itoa(len(gzipped)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(gzipped)
		return
	}
	header.Set("Content-Length", strconv.Itoa(len(plain)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(plain)
}

// acceptsGzip reads an Accept-Encoding header: gzip, unless its quality is 0.
func acceptsGzip(header string) bool {
	for part := range strings.SplitSeq(header, ",") {
		name, parameters, _ := strings.Cut(part, ";")
		if !strings.EqualFold(strings.TrimSpace(name), "gzip") {
			continue
		}
		quality, given := strings.CutPrefix(strings.ReplaceAll(parameters, " ", ""), "q=")
		value, err := strconv.ParseFloat(quality, 64)
		return !given || (err == nil && value > 0)
	}
	return false
}

func (api *api) now() time.Time {
	if api.deps.Now != nil {
		return api.deps.Now()
	}
	return time.Now()
}
