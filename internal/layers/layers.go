// Package layers is the catalogue of map layers lanka-weather serves.
//
// The catalogue is the single place where a layer's refresh cadence and
// staleness limit are defined. The API returns it from GET /v1/layers and
// the web application derives its query timings from it, so cadence is
// never duplicated between server and client.
package layers

import (
	"fmt"
	"time"
)

// Category groups layers in the map's layer panel.
type Category string

const (
	CategoryHydrology Category = "hydrology"
	CategoryWeather   Category = "weather"
	CategoryHazard    Category = "hazard"
	CategoryGeography Category = "geography"
)

// Kind is how the web application renders a layer.
type Kind string

const (
	// KindGeoJSON layers are features served by the API.
	KindGeoJSON Kind = "geojson"
	// KindRaster layers are image tiles loaded by the browser.
	KindRaster Kind = "raster"
	// KindVector layers are Mapbox Vector Tiles, served by the API or as
	// PMTiles.
	KindVector Kind = "vector"
	// KindWind is the animated particle overlay drawn from a wind field.
	KindWind Kind = "wind"
)

// Status says whether a layer has data behind it yet.
type Status string

const (
	StatusPlanned   Status = "planned"
	StatusAvailable Status = "available"
)

// Layer describes one map layer.
type Layer struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    Category `json:"category"`
	Kind        Kind     `json:"kind"`
	Status      Status   `json:"status"`
	// Sources are identifiers of entries in docs/data-sources.md.
	Sources []string `json:"sources"`
	// RefreshSeconds is how often new data can be expected.
	RefreshSeconds int `json:"refreshSeconds"`
	// StaleAfterSeconds is the age after which data is shown as stale.
	StaleAfterSeconds int    `json:"staleAfterSeconds"`
	Attribution       string `json:"attribution"`
	// DataPath is the API path serving the layer's data, for available
	// layers whose data comes from the API.
	DataPath string `json:"dataPath,omitempty"`
	// TilePath is the URL template of the layer's vector tiles, with {z},
	// {x} and {y} placeholders, for available vector layers.
	TilePath string `json:"tilePath,omitempty"`
	// MinZoom and MaxZoom bound the zooms the tiles are served at; clients
	// overzoom beyond MaxZoom.
	MinZoom int `json:"minZoom,omitempty"`
	MaxZoom int `json:"maxZoom,omitempty"`
}

func seconds(d time.Duration) int { return int(d / time.Second) }

var catalogue = []Layer{
	{
		ID:                "river-levels",
		Title:             "River levels",
		Description:       "Water level at gauging stations, compared with alert and flood thresholds.",
		Category:          CategoryHydrology,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"irrigation-gauges"},
		RefreshSeconds:    seconds(15 * time.Minute),
		StaleAfterSeconds: seconds(3 * time.Hour),
		Attribution:       "Irrigation Department of Sri Lanka",
		DataPath:          "/v1/rivers/levels",
	},
	{
		ID:                "flood-forecast",
		Title:             "Flood forecast",
		Description:       "Forecast flood status at river gauges and expected inundation areas.",
		Category:          CategoryHydrology,
		Kind:              KindGeoJSON,
		Status:            StatusPlanned,
		Sources:           []string{"google-flood-forecasting"},
		RefreshSeconds:    seconds(6 * time.Hour),
		StaleAfterSeconds: seconds(24 * time.Hour),
		Attribution:       "Google Flood Forecasting",
	},
	{
		ID:                "reservoirs",
		Title:             "Reservoirs and tanks",
		Description:       "Storage level and spill status of major reservoirs.",
		Category:          CategoryHydrology,
		Kind:              KindGeoJSON,
		Status:            StatusPlanned,
		Sources:           []string{"irrigation-reservoirs"},
		RefreshSeconds:    seconds(24 * time.Hour),
		StaleAfterSeconds: seconds(48 * time.Hour),
		Attribution:       "Irrigation Department of Sri Lanka",
	},
	{
		ID:                "wind",
		Title:             "Wind",
		Description:       "Animated wind speed and direction 10 m above ground.",
		Category:          CategoryWeather,
		Kind:              KindWind,
		Status:            StatusPlanned,
		Sources:           []string{"noaa-gfs"},
		RefreshSeconds:    seconds(6 * time.Hour),
		StaleAfterSeconds: seconds(12 * time.Hour),
		Attribution:       "NOAA GFS",
	},
	{
		ID:                "rainfall",
		Title:             "Rainfall",
		Description:       "Satellite precipitation estimate.",
		Category:          CategoryWeather,
		Kind:              KindRaster,
		Status:            StatusPlanned,
		Sources:           []string{"nasa-gibs-imerg"},
		RefreshSeconds:    seconds(30 * time.Minute),
		StaleAfterSeconds: seconds(8 * time.Hour),
		Attribution:       "NASA GIBS, GPM IMERG",
	},
	{
		ID:                "weather-stations",
		Title:             "Weather stations",
		Description:       "Observed rainfall, temperature and humidity at Department of Meteorology stations.",
		Category:          CategoryWeather,
		Kind:              KindGeoJSON,
		Status:            StatusPlanned,
		Sources:           []string{"meteo-observations", "nuuuwan-lk-weather-3h"},
		RefreshSeconds:    seconds(3 * time.Hour),
		StaleAfterSeconds: seconds(6 * time.Hour),
		Attribution:       "Department of Meteorology, Sri Lanka",
	},
	{
		ID:                "cyclones",
		Title:             "Cyclones",
		Description:       "Tropical cyclone positions, forecast tracks and alert levels.",
		Category:          CategoryHazard,
		Kind:              KindGeoJSON,
		Status:            StatusPlanned,
		Sources:           []string{"gdacs"},
		RefreshSeconds:    seconds(30 * time.Minute),
		StaleAfterSeconds: seconds(6 * time.Hour),
		Attribution:       "GDACS",
	},
	{
		ID:                "landslide-warnings",
		Title:             "Landslide warnings",
		Description:       "NBRO landslide early-warning level by divisional secretariat division.",
		Category:          CategoryHazard,
		Kind:              KindGeoJSON,
		Status:            StatusPlanned,
		Sources:           []string{"nbro-landslide", "nuuuwan-lk-dmc-landslides"},
		RefreshSeconds:    seconds(3 * time.Hour),
		StaleAfterSeconds: seconds(12 * time.Hour),
		Attribution:       "National Building Research Organisation, Disaster Management Centre",
	},
	{
		ID:                "official-warnings",
		Title:             "Official warnings",
		Description:       "Department of Meteorology and Disaster Management Centre warnings and advisories.",
		Category:          CategoryHazard,
		Kind:              KindGeoJSON,
		Status:            StatusPlanned,
		Sources:           []string{"dmc-reports", "meteo-warnings"},
		RefreshSeconds:    seconds(1 * time.Hour),
		StaleAfterSeconds: seconds(6 * time.Hour),
		Attribution:       "Department of Meteorology, Disaster Management Centre",
	},
	{
		ID:                "earthquakes",
		Title:             "Earthquakes",
		Description:       "Recent earthquakes in the Indian Ocean region.",
		Category:          CategoryHazard,
		Kind:              KindGeoJSON,
		Status:            StatusPlanned,
		Sources:           []string{"usgs-earthquakes"},
		RefreshSeconds:    seconds(15 * time.Minute),
		StaleAfterSeconds: seconds(2 * time.Hour),
		Attribution:       "USGS",
	},
	{
		ID:    "admin-regions",
		Title: "Administrative regions",
		Description: "Province, district, divisional secretariat and Grama Niladhari division boundaries, " +
			"with names in English, Sinhala and Tamil and 2024 census population.",
		Category: CategoryGeography,
		Kind:     KindVector,
		Status:   StatusAvailable,
		Sources:  []string{"nuuuwan-lk-admin-regions", "nuuuwan-lk-census-2024"},
		// The collector checks weekly for a new upstream version; boundaries
		// change a few times a year at most, so a month without a successful
		// check is when the layer is shown as possibly out of date.
		RefreshSeconds:    seconds(7 * 24 * time.Hour),
		StaleAfterSeconds: seconds(30 * 24 * time.Hour),
		Attribution:       "Survey Department of Sri Lanka and Department of Census and Statistics via OCHA HDX and lk_admin_regions",
		DataPath:          "/v1/regions",
		TilePath:          "/v1/tiles/regions/{z}/{x}/{y}.pbf",
		MinZoom:           0,
		MaxZoom:           14,
	},
}

// All returns a copy of the catalogue in display order.
func All() []Layer {
	out := make([]Layer, len(catalogue))
	for i, layer := range catalogue {
		layer.Sources = append([]string(nil), layer.Sources...)
		out[i] = layer
	}
	return out
}

// ByID returns the layer with the given identifier.
func ByID(id string) (Layer, bool) {
	for _, layer := range All() {
		if layer.ID == id {
			return layer, true
		}
	}
	return Layer{}, false
}

// Validate checks the catalogue's invariants. It is run by tests so a
// malformed entry fails CI rather than reaching the map.
func Validate(list []Layer) error {
	seen := make(map[string]bool, len(list))
	for _, layer := range list {
		switch {
		case layer.ID == "" || layer.Title == "":
			return fmt.Errorf("layer %q: id and title are required", layer.ID)
		case seen[layer.ID]:
			return fmt.Errorf("layer %q: duplicate id", layer.ID)
		case len(layer.Sources) == 0:
			return fmt.Errorf("layer %q: at least one source is required", layer.ID)
		case layer.RefreshSeconds <= 0:
			return fmt.Errorf("layer %q: refresh interval must be positive", layer.ID)
		case layer.StaleAfterSeconds < layer.RefreshSeconds:
			return fmt.Errorf("layer %q: stale limit is shorter than the refresh interval", layer.ID)
		case layer.Attribution == "":
			return fmt.Errorf("layer %q: attribution is required", layer.ID)
		case layer.Status == StatusAvailable && layer.Kind == KindGeoJSON && layer.DataPath == "":
			return fmt.Errorf("layer %q: an available GeoJSON layer needs a data path", layer.ID)
		case layer.Status == StatusAvailable && layer.Kind == KindVector && layer.TilePath == "":
			return fmt.Errorf("layer %q: an available vector layer needs a tile path", layer.ID)
		case layer.MaxZoom < layer.MinZoom:
			return fmt.Errorf("layer %q: maximum zoom is below the minimum", layer.ID)
		}
		seen[layer.ID] = true
	}
	return nil
}
