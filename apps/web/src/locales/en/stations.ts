/** Messages of the stations feature. English is the source: it fixes the keys. */
export const stations = {
  /** Which feed a reading came from. */
  reading: {
    threeHourly: "3-hourly report",
    daily: "Daily report",
    none: "No readings",
  },
  rainLabel: {
    hours: "Rain, {count} h",
    daily: "Rain, 24 h to 08:30",
  },
  /** The warning shown when the Met Department copies are not current. */
  notice: {
    dailyTakeover: "The 3-hourly reports have stopped updating, so stations show the daily report.",
    nothing: "No Department of Meteorology readings have been collected yet.",
    notCurrent:
      "Not current: the newest Department of Meteorology reading is {age}. Our source for these reports has stopped updating.",
  },
  panel: {
    title: "Weather station",
    subtitle: "Met station",
    subtitleWmo: "Met station · WMO {wmo}",
    subtitleRain: "Met rain station",
    close: "Close weather station",
    loading: "Loading station",
    error: "This station could not be loaded.",
    observed: "Observed",
    outOfDate: "Out of date",
    temperature: "Temperature",
    dailyMaximum: "Maximum, daily report",
    observedRow: "Observed",
    observedStale: "{age} · out of date",
    readingRow: "Reading",
    rain24h: "Rain, 24 h",
    humidity: "Humidity",
    maximum: "Maximum",
    minimum: "Minimum",
    elevation: "Elevation",
    position: "Position",
    approximate: "Approximate, from its name",
    history: {
      one: "{count} day to the newest report",
      other: "{count} days to the newest report",
    },
    dailyReports: "Daily reports",
    /** {meteo}, {copy3h} and {copyDaily} are links. */
    source: "Source: {meteo}, through the {copy3h} and {copyDaily} copies.",
    meteo: "Department of Meteorology, Sri Lanka",
  },
  chart: {
    empty: {
      one: "No 3-hourly readings in the {count} day to the newest report.",
      other: "No 3-hourly readings in the {count} days to the newest report.",
    },
    label: {
      one: "Observed temperature from {min} to {max} °C over the {count} day to the newest report",
      other:
        "Observed temperature from {min} to {max} °C over the {count} days to the newest report",
    },
    labelRain: {
      one: "Observed temperature from {min} to {max} °C over the {count} day to the newest report, with rain up to {rain} mm in 3 hours",
      other:
        "Observed temperature from {min} to {max} °C over the {count} days to the newest report, with rain up to {rain} mm in 3 hours",
    },
    range: "{min}–{max} °C",
    rangeRain: "{min}–{max} °C · rain to {rain} mm",
  },
  /** The nearest station in the place panel, and air and sea rows under it. */
  summary: {
    observed: "Observed",
    underKm: "under 1 km",
    km: "{km} km",
    dailyMax: "max",
    rainIn: "{rain} mm in {hours} h",
    airQuality: "Air quality",
    aqiValue: "{band} · AQI {aqi}",
    sea: "Sea",
    seaValue: "{state} · waves {height} m",
    swell: "swell {height} m",
    swellPeriod: "swell {height} m, {period} s",
    swellFrom: "swell {height} m from {direction}",
    swellPeriodFrom: "swell {height} m, {period} s, from {direction}",
    seaNote:
      "Sea values are from a wave model. Follow the Department of Meteorology's sea-area warnings.",
  },
  /** The hover chip over a station on the map. */
  chip: {
    title: "Weather station",
    observed: "Observed: {temperature} °C",
    dailyMaximum: "Observed, daily maximum: {temperature} °C",
    ageStale: "{age} · out of date",
  },
  /** The weather-stations layer in the layer list. */
  layer: {
    observed: "Observed",
    count: { one: "{count} station", other: "{count} stations" },
    legendText: "Measured at Met stations, coloured by temperature. Faded squares are out of date.",
    legendTitle: "Temperature",
    legendTitleObserved: "Temperature, observed",
  },
}
