/** Messages of the environment feature. English is the source: it fixes the keys. */
export const environment = {
  /** The US EPA's words for each AQI band. */
  aqi: {
    good: "Good",
    moderate: "Moderate",
    unhealthySensitive: "Unhealthy for sensitive groups",
    unhealthy: "Unhealthy",
    veryUnhealthy: "Very unhealthy",
    hazardous: "Hazardous",
  },
  /** The WMO sea-state words for wave height. */
  seaState: {
    calm: "Calm",
    smooth: "Smooth",
    slight: "Slight",
    moderate: "Moderate",
    rough: "Rough",
    veryRough: "Very rough",
    high: "High",
    veryHigh: "Very high",
  },
  air: {
    usAqi: "US AQI",
    pm25: "PM2.5",
    legendTitle: "US Air Quality Index",
    pm25Title: "PM2.5 · Estimate",
    toggle: "Air quality shows",
    note: "Estimate from the CAMS atmosphere model, about 40 km across. It does not measure the air at any place.",
    credit:
      "Air quality by Open-Meteo.com (CC BY 4.0), generated using Copernicus Atmosphere Monitoring Service information.",
    hoverAqi: "Air quality (estimate): {band}, AQI {aqi}",
    hoverPm25: "PM2.5 (estimate): {value} µg/m³",
  },
  sea: {
    model: "Model",
    waveHeight: "wave height",
    legendTitle: "Wave height · Model",
    note: "Wave and swell model values on a 0.5° grid (about 55 km), not observations. Follow the Department of Meteorology's sea-area warnings.",
    credit: "Marine data by Open-Meteo.com (CC BY 4.0).",
    hover: "Waves (model): {height} m · {state}",
  },
}
