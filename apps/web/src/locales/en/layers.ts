/** The layer list (features/layers) and the layer catalogue's words (lib/layers). */
export const layers = {
  category: {
    hazard: "Hazards and warnings",
    hydrology: "Rivers and floods",
    weather: "Weather",
    geography: "Geography",
  },
  status: {
    waiting: "Data is still loading · collected shortly",
    error: "Unavailable · retrying shortly",
  },
  /** The age at the end of a layer's one-line status. */
  age: {
    updated: "updated {age}",
    read: "read {age}",
    run: "run {age}",
    image: "image {age}",
    newest: "newest {age}",
    issued: "issued {age}",
  },
  title: {
    rivers: "Rivers",
    "river-levels": "River levels",
    "flood-forecast": "Flood forecast",
    "past-floods": "Past floods",
    reservoirs: "Reservoirs and tanks",
    wind: "Wind",
    temperature: "Temperature",
    rainfall: "Rainfall",
    "rain-gauges": "Rain gauges",
    "weather-stations": "Weather stations",
    "air-quality": "Air quality",
    "sea-conditions": "Sea conditions",
    cyclones: "Cyclones",
    "landslide-warnings": "Landslide warnings",
    "official-warnings": "Official warnings",
    earthquakes: "Earthquakes",
    "admin-regions": "Administrative regions",
  },
  description: {
    rivers:
      "Sri Lanka's rivers, with an estimate of each stretch's risk of overflowing from its gauges, Google's forecast and forecast rain. Follow official warnings.",
    "river-levels": "Water level at gauging stations, compared with alert and flood thresholds.",
    "flood-forecast":
      "Forecast flood status at river gauges, flash-flood areas, significant flood events and inundation probability.",
    "past-floods":
      "Where past floods reached, as mapped by the Irrigation Department. These are past events, not a forecast.",
    reservoirs:
      "Storage and spill or release status of reservoirs, updated daily. Not an official warning.",
    wind: "Animated wind 10 m above ground from the NOAA GFS model, now and every 3 hours to 3 days ahead.",
    temperature:
      "Current temperature, humidity, rain and wind on a 0.2° grid over Sri Lanka, from weather models.",
    rainfall:
      "Satellite rain estimate from NASA's IMERG, hours behind real time, with rain measured at stations and river gauges over the last 24 hours.",
    "rain-gauges":
      "Rain measured over the last 24 hours at about 275 live gauges of NBRO, the Department of Meteorology, the Irrigation Department and ACCIMT. Wetter gauges show first; zoom in for the rest.",
    "weather-stations":
      "Observed rainfall, temperature and humidity at Department of Meteorology stations.",
    "air-quality":
      "US Air Quality Index and PM2.5 from the CAMS atmosphere model, about 40 km across. A model estimate, not a measurement.",
    "sea-conditions":
      "Wave height and swell around the coast, from wave models. Follow official sea-area warnings.",
    cyclones:
      "Tropical cyclones in the North Indian Ocean: position, track, forecast, wind buffers and alert level.",
    "landslide-warnings": "NBRO landslide early-warning level by divisional secretariat division.",
    "official-warnings":
      "Districts with a landslide warning, or named by a warning or advisory in force. Bulletins from the Disaster Management Centre.",
    earthquakes: "Earthquakes of the last 30 days in the Indian Ocean region.",
    "admin-regions":
      "Province, district, divisional secretariat and Grama Niladhari division boundaries, with names in English, Sinhala and Tamil and 2024 census population.",
  },
  panel: {
    loading: "Loading layers",
    failed: "Layers are unavailable: {message}",
    showDetailsOf: "Show details of {title}",
    hideDetailsOf: "Hide details of {title}",
    soon: "Soon",
  },
}
