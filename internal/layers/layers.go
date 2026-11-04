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
	// KindGrid layers are values on a regular grid served by the API,
	// drawn as a colour field or contours.
	KindGrid Kind = "grid"
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
		ID:    "rivers",
		Title: "Rivers",
		Description: "Sri Lanka's rivers, with an estimate of each stretch's risk of overflowing from its gauges, " +
			"Google's forecast and forecast rain. Follow official warnings.",
		Category:          CategoryHydrology,
		Kind:              KindVector,
		Status:            StatusAvailable,
		Sources:           []string{"hydrosheds-hydrorivers", "irrigation-river-basins", "irrigation-gauges", "google-flood-forecasting", "open-meteo"},
		RefreshSeconds:    seconds(15 * time.Minute),
		StaleAfterSeconds: seconds(3 * time.Hour),
		Attribution:       "HydroRIVERS (HydroSHEDS), Irrigation Department of Sri Lanka",
		DataPath:          "/v1/rivers/risk",
		TilePath:          "/v1/tiles/rivers/{z}/{x}/{y}.pbf",
		MaxZoom:           14,
	},
	{
		ID:                "river-levels",
		Title:             "River levels",
		Description:       "Water level at gauging stations, compared with alert and flood thresholds.",
		Category:          CategoryHydrology,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"irrigation-gauges", "waterlevels-lk-gauges"},
		RefreshSeconds:    seconds(15 * time.Minute),
		StaleAfterSeconds: seconds(3 * time.Hour),
		Attribution:       "Irrigation Department of Sri Lanka; more gauges: RiverNet (DMC) via Sri Lanka Water Levels (waterlevels.lk)",
		DataPath:          "/v1/rivers/levels",
	},
	{
		ID:                "flood-forecast",
		Title:             "Flood forecast",
		Description:       "Forecast flood status at river gauges, flash-flood areas, significant flood events and inundation probability.",
		Category:          CategoryHydrology,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"google-flood-forecasting"},
		RefreshSeconds:    seconds(1 * time.Hour),
		StaleAfterSeconds: seconds(30 * time.Hour),
		Attribution:       "Google Flood Hub (Flood Forecasting API)",
		DataPath:          "/v1/floods/forecast",
		// Sub-basin and river-basin boundaries, outlined by forecast status.
		TilePath: "/v1/tiles/river-basins/{z}/{x}/{y}.pbf",
		MaxZoom:  14,
	},
	{
		ID:                "past-floods",
		Title:             "Past floods",
		Description:       "Where past floods reached, as mapped by the Irrigation Department. These are past events, not a forecast.",
		Category:          CategoryHydrology,
		Kind:              KindVector,
		Status:            StatusAvailable,
		Sources:           []string{"irrigation-past-floods"},
		RefreshSeconds:    seconds(30 * 24 * time.Hour),
		StaleAfterSeconds: seconds(365 * 24 * time.Hour),
		Attribution:       "Irrigation Department of Sri Lanka",
		DataPath:          "/v1/floods/past",
		TilePath:          "/v1/tiles/past-floods/{z}/{x}/{y}.pbf",
		MaxZoom:           14,
	},
	{
		ID:                "reservoirs",
		Title:             "Reservoirs and tanks",
		Description:       "Storage and spill or release status of reservoirs, updated daily. Not an official warning.",
		Category:          CategoryHydrology,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"waterlevels-lk-reservoirs", "osm-reservoirs"},
		RefreshSeconds:    seconds(6 * time.Hour),
		StaleAfterSeconds: seconds(48 * time.Hour),
		Attribution:       "Irrigation Department and Mahaweli Authority via Sri Lanka Water Levels (waterlevels.lk); positions © OpenStreetMap contributors",
		DataPath:          "/v1/reservoirs",
	},
	{
		ID:                "wind",
		Title:             "Wind",
		Description:       "Animated wind 10 m above ground from the NOAA GFS model, now and every 3 hours to 3 days ahead.",
		Category:          CategoryWeather,
		Kind:              KindWind,
		Status:            StatusAvailable,
		Sources:           []string{"noaa-gfs"},
		RefreshSeconds:    seconds(1 * time.Hour),
		StaleAfterSeconds: seconds(12 * time.Hour),
		Attribution:       "NOAA GFS",
		DataPath:          "/v1/weather/wind",
	},
	{
		ID:                "temperature",
		Title:             "Temperature",
		Description:       "Current temperature, humidity, rain and wind on a 0.2° grid over Sri Lanka, from weather models.",
		Category:          CategoryWeather,
		Kind:              KindGrid,
		Status:            StatusAvailable,
		Sources:           []string{"open-meteo"},
		RefreshSeconds:    seconds(1 * time.Hour),
		StaleAfterSeconds: seconds(3 * time.Hour),
		Attribution:       "Weather data by Open-Meteo.com (CC BY 4.0)",
		DataPath:          "/v1/weather/current",
	},
	{
		ID:    "rainfall",
		Title: "Rainfall",
		Description: "Satellite rain estimate from NASA's IMERG, hours behind real time, " +
			"with rain measured at stations and river gauges over the last 24 hours.",
		Category: CategoryWeather,
		Kind:     KindRaster,
		Status:   StatusAvailable,
		Sources:  []string{"nasa-gibs-imerg", "nuuuwan-weather-lk", "nuuuwan-lk-weather-3h", "irrigation-gauges"},
		// A new image every 30 minutes, published about six hours late, so an
		// image is out of date only some way beyond that lag.
		RefreshSeconds:    seconds(30 * time.Minute),
		StaleAfterSeconds: seconds(10 * time.Hour),
		Attribution:       "NASA GIBS, GPM IMERG; Department of Meteorology; Irrigation Department",
		DataPath:          "/v1/rain/steps",
	},
	{
		ID:    "rain-gauges",
		Title: "Rain gauges",
		Description: "Rain measured over the last 24 hours at about 275 live gauges of NBRO, the Department of " +
			"Meteorology, the Irrigation Department and ACCIMT. Wetter gauges show first; zoom in for the rest.",
		Category:          CategoryWeather,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"hmis-rain-live"},
		RefreshSeconds:    seconds(30 * time.Minute),
		StaleAfterSeconds: seconds(3 * time.Hour),
		Attribution:       "HMIS, via Sri Lanka Water Levels (waterlevels.lk)",
		DataPath:          "/v1/rain/gauges",
	},
	{
		ID:                "weather-stations",
		Title:             "Weather stations",
		Description:       "Observed rainfall, temperature and humidity at Department of Meteorology stations.",
		Category:          CategoryWeather,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"nuuuwan-lk-weather-3h", "nuuuwan-weather-lk"},
		RefreshSeconds:    seconds(1 * time.Hour),
		StaleAfterSeconds: seconds(6 * time.Hour),
		Attribution:       "Department of Meteorology, Sri Lanka",
		DataPath:          "/v1/stations",
	},
	{
		ID:                "air-quality",
		Title:             "Air quality",
		Description:       "US Air Quality Index and PM2.5 from the CAMS atmosphere model, about 40 km across. A model estimate, not a measurement.",
		Category:          CategoryWeather,
		Kind:              KindGrid,
		Status:            StatusAvailable,
		Sources:           []string{"open-meteo-air-quality"},
		RefreshSeconds:    seconds(3 * time.Hour),
		StaleAfterSeconds: seconds(9 * time.Hour),
		Attribution:       "Air quality by Open-Meteo.com (CC BY 4.0), CAMS",
		DataPath:          "/v1/air/current",
	},
	{
		ID:                "sea-conditions",
		Title:             "Sea conditions",
		Description:       "Wave height and swell around the coast, from wave models. Follow official sea-area warnings.",
		Category:          CategoryWeather,
		Kind:              KindGrid,
		Status:            StatusAvailable,
		Sources:           []string{"open-meteo-marine"},
		RefreshSeconds:    seconds(3 * time.Hour),
		StaleAfterSeconds: seconds(9 * time.Hour),
		Attribution:       "Marine data by Open-Meteo.com (CC BY 4.0)",
		DataPath:          "/v1/sea/current",
	},
	{
		ID:                "cyclones",
		Title:             "Cyclones",
		Description:       "Tropical cyclones in the North Indian Ocean: position, track, forecast, wind buffers and alert level.",
		Category:          CategoryHazard,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"gdacs"},
		RefreshSeconds:    seconds(1 * time.Hour),
		StaleAfterSeconds: seconds(6 * time.Hour),
		Attribution:       "GDACS",
		DataPath:          "/v1/hazards/cyclones",
	},
	{
		ID:                "landslide-warnings",
		Title:             "Landslide warnings",
		Description:       "NBRO landslide early-warning level by divisional secretariat division.",
		Category:          CategoryHazard,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"nuuuwan-lk-dmc-landslides", "dmc-reports"},
		RefreshSeconds:    seconds(2 * time.Hour),
		StaleAfterSeconds: seconds(12 * time.Hour),
		Attribution:       "National Building Research Organisation, Disaster Management Centre",
		DataPath:          "/v1/warnings/landslides",
	},
	{
		ID:                "official-warnings",
		Title:             "Official warnings",
		Description:       "Districts with a landslide warning, or named by a warning or advisory in force. Bulletins from the Disaster Management Centre.",
		Category:          CategoryHazard,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"dmc-reports"},
		RefreshSeconds:    seconds(1 * time.Hour),
		StaleAfterSeconds: seconds(6 * time.Hour),
		Attribution:       "Disaster Management Centre",
		DataPath:          "/v1/warnings",
	},
	{
		ID:                "earthquakes",
		Title:             "Earthquakes",
		Description:       "Earthquakes of the last 30 days in the Indian Ocean region.",
		Category:          CategoryHazard,
		Kind:              KindGeoJSON,
		Status:            StatusAvailable,
		Sources:           []string{"usgs-earthquakes"},
		RefreshSeconds:    seconds(15 * time.Minute),
		StaleAfterSeconds: seconds(2 * time.Hour),
		Attribution:       "USGS",
		DataPath:          "/v1/hazards/earthquakes",
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
		case layer.Status == StatusAvailable && layer.Kind != KindRaster && layer.Kind != KindVector && layer.DataPath == "":
			return fmt.Errorf("layer %q: an available layer served by the API needs a data path", layer.ID)
		case layer.Status == StatusAvailable && layer.Kind == KindVector && layer.TilePath == "":
			return fmt.Errorf("layer %q: an available vector layer needs a tile path", layer.ID)
		case layer.MaxZoom < layer.MinZoom:
			return fmt.Errorf("layer %q: maximum zoom is below the minimum", layer.ID)
		}
		seen[layer.ID] = true
	}
	return nil
}
