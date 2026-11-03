/** Messages of the rain feature. English is the source: it fixes the keys. */
export const rain = {
  /** The half hour a satellite image covers: "Mon 2 Nov 19:00–19:30". */
  step: "{day} {start}–{end}",
  /** How a station's 24-hour total was reached. */
  method: {
    dailyReport: "Department of Meteorology daily report, 24 h to 08:30",
    sum3h: "Sum of 3-hourly Department of Meteorology readings",
    sumGauge: "Sum of hourly river gauge readings",
  },
  timeline: {
    section: "Satellite rain image time",
    play: "Play the rain images",
    pause: "Pause the rain images",
    thumb: "Time of the satellite rain image",
    thumbValue: "{time}, {age}",
    note: "Estimate · NASA IMERG · lags by hours",
    outOfDate: "Out of date",
    scale: "Satellite image colours, from {from} to {to} {unit} and above",
  },
  /** The rainfall layer in the layer list. */
  layer: {
    satellite: "Satellite estimate",
    points: { one: "{count} measured point", other: "{count} measured points" },
    satelliteText: "NASA IMERG, not a measurement. Images lag by hours.",
    satelliteTextNewest: "NASA IMERG, not a measurement. Images lag by hours; the newest is {age}.",
    measuredTitle: "Measured, last 24 hours",
    measuredText:
      "Circles are river gauges and squares Met stations. Hollow points are too old to be current; open one for its date. Where a live rain gauge stands beside one, only the live gauge shows while that layer is on.",
    measuredCounts: "{fresh} current, {stale} old.",
    forecastTitle: "Forecast",
    forecastText: "Rain ahead is in the weather field and the place panel. It is model data.",
    showForecast: "Show forecast rain",
  },
  /** Live rain gauges. */
  gauges: {
    band: {
      veryHeavy: "Very heavy",
      heavy: "Heavy",
      moderate: "Moderate",
      light: "Light",
      none: "No rain",
      unknown: "Out of date",
    },
    range: {
      veryHeavy: "100 mm or more",
      heavy: "50 to 100 mm",
      moderate: "10 to 50 mm",
      light: "under 10 mm",
      none: "0 mm",
      unknown: "newest reading over 3 h old",
    },
    summary: "{fresh} of {total} current · {wet} with 10 mm or more",
    legendTitle: "Rain gauges",
    legendUnit: "last 24 h",
    layerText:
      "Rolling 24-hour totals, measured. Zoom in for light rain, then for dry gauges; a hollow drop has not reported for over 3 hours, which does not mean it has been dry.",
  },
  panel: {
    title: "Rain gauge",
    subtitle: "Rain gauge · {institution}",
    close: "Close rain gauge",
    error: "This rain gauge could not be read just now.",
    unit: "mm in 24 h",
    silent:
      "This gauge has not reported for over 3 hours, so we cannot say how much rain has fallen. That does not mean it has been dry.",
    lastFigure: "Its last figure was {rain} mm, {time}.",
    measured: "Measured",
    read: "Read",
    readValue: "{time} · {age}",
    noReading: "No reading",
    runBy: "Run by",
    runByNamed: "Run by {institution}.",
    rolling: "Rolling 24-hour total, up to 48 hours",
    rollingNote:
      "Each point is the sum of the 24 hours before it, so it says nothing about any one hour.",
    chartLabel: "Rolling 24-hour rain from {low} to {peak} mm",
    upTo: "up to {peak} mm",
  },
  /** The hover chip over rain gauges and measured totals on the map. */
  chip: {
    gauge: "Rain gauge",
    rainfall: "Rainfall",
    total: "Rain, 24 h: {rain}",
    totalBand: "Rain, 24 h: {rain} · {band}",
    noTotal: "Out of date: no current total",
    runBy: "{institution} · {age}",
    ending: "Ending {age}",
    endingStale: "Ending {age} · out of date",
  },
}
