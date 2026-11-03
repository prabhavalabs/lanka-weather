/** Messages of the rivers feature. English is the source: it fixes the keys. */
export const rivers = {
  /** River gauges (features/rivers). Statuses and levels are in `terms`. */
  legend: {
    title: "River levels",
    unit: "status",
  },
  nextHour: {
    range: "{level}, range {low}–{high}",
    issued: "issued {time}",
  },
  station: {
    subtitle: "{basin} · river gauge",
    close: "Close station details",
    source: "Source: {attribution}.",
    sourceLink: "This gauge at the source",
    official: "Official flood warnings come from the Irrigation Department and the DMC.",
    waterLevel: "Water level",
    below: {
      alert: "{amount} below alert level",
      minor_flood: "{amount} below minor flood level",
      major_flood: "{amount} below major flood level",
    },
    towards: {
      alert: "Level between thresholds, towards alert level",
      minor_flood: "Level between thresholds, towards minor flood level",
      major_flood: "Level between thresholds, towards major flood level",
    },
    noReading: "This gauge has no reading at the moment.",
    noUnit:
      "No unit or flood thresholds are published for this gauge, so there is no flood verdict.",
    noThresholds: "This station publishes no flood thresholds.",
    aboveMajor: "Above major flood level",
    nextHour: "Next hour (experimental, by Sri Lanka Water Levels): {value}, {issued}",
    observed: "Observed",
    observedStale: "{age} · out of date",
    trend: "Trend",
    rainfall: "Rainfall at gauge",
    history: "Last {hours} hours",
    loadingHistory: "Loading history",
    historyUnavailable: "History is unavailable.",
    noUnitShort: "(no unit)",
  },
  chart: {
    empty: "No readings in the last {hours} hours.",
    label: "Water level over the last {hours} hours, in {unit}",
  },

  /** River risk (features/river-risk): our estimate, never an official warning. */
  risk: {
    level: {
      severe: "Severe",
      high: "High",
      watch: "Watch",
      normal: "Normal",
      unknown: "Not known",
    },
    meaning: {
      severe: "Likely overflowing",
      high: "Overflow risk",
      watch: "Above or nearing alert",
      normal: "Measured, below alert",
      unknown: "No current gauge",
    },
    legend: {
      title: "River risk",
      severe: "likely overflowing",
      high: "overflow risk",
      watch: "at or nearing alert",
      normal: "gauge below alert",
      unknown: "no current gauge",
    },
    headline: {
      calm: "No river is above its alert level",
      count: { one: "{count} river at risk", other: "{count} rivers at risk" },
      gauges: "{current} of {total} gauges current",
    },
    coverage: {
      none: "No gauge on this river",
      whole: "Gauged",
      every: "Every stretch has a gauge",
      some: "{gauged} of {total} stretches have a gauge",
    },
    /** What a reason rests on, when its sentence is only in English. */
    basis: {
      gauge: "From a gauge reading",
      forecast: "From Google's flood forecast",
      rain: "From forecast rain over the catchment",
      reservoir: "From a reservoir upstream",
      coverage: "Not measured",
      generic: "Estimate",
    },
    list: {
      title: "Rivers at risk",
      footnote:
        "Estimated from gauges, Google's forecast and forecast rain. Follow official warnings from the Irrigation Department, the Department of Meteorology and the DMC.",
    },
    /** The note every estimate carries (riverrisk.Note in the Go service). */
    note: "Estimate from gauge readings, Google flood forecasts, forecast rain and reported reservoir spills. Thresholds are the Irrigation Department's. Follow official warnings from the Irrigation Department, the Department of Meteorology and the Disaster Management Centre.",
    panel: {
      title: "River",
      close: "Close river details",
      loadError: "This river could not be loaded.",
      loading: "Loading river",
      testData: "Test data",
      footer: "Sources: {attribution}. Gauges read {gauges}.",
      footerRain: "Sources: {attribution}. Gauges read {gauges}, rain forecast fetched {rain}.",
      stretches: "Stretches · {coverage}",
      stretchUnknown: "Nothing is known about this stretch.",
      areas: "Areas along stretches at risk",
      districts: "Districts: {names}",
      dsDivisions: "DS divisions: {names}",
      readAge: "Read {age}",
      gaugeLine: "{level} · read {age}",
      gaugeLineStale: "{level} · read {age} · out of date",
      otherGauges: "Other gauges",
      forecastTitle: "Google flood forecast on this river",
      forecastIssued: "Issued {time}. Source: Google Flood Hub.",
      rainTitle: "Forecast rain over the catchment",
      next24h: "Next 24 hours",
      next72h: "Next 72 hours",
      catchment: "Catchment",
      rainNote: "An average over the whole catchment, from weather models.",
      rainStale: "This forecast is out of date, so it is not used.",
      rainCredit: "Weather data by Open-Meteo.com.",
      flowsInto: "Flows into",
      tributary: "Tributary",
      connected: "Connected rivers",
    },
    /** The server's reasons, rebuilt from their known sentences. */
    reason: {
      noGauge: "No gauge on this stretch, so its level is not known.",
      noCurrent: "No current reading from its gauges, so its level is not known.",
      noGaugeNext:
        "No gauge on this stretch, so its level is not known. The next gauge downstream is {gauge}.",
      noAlertLevel: "{gauge} publishes no alert level to compare with.",
      above: {
        alert: "{gauge} is at {level}, above its alert level of {threshold}.",
        minor_flood: "{gauge} is at {level}, above its minor flood level of {threshold}.",
        major_flood: "{gauge} is at {level}, above its major flood level of {threshold}.",
      },
      below: "{gauge} is at {level}, below its alert level of {threshold}.",
      near: "It is within {gap} of that level.",
      rising: {
        alert: "Rising {rate} an hour, it would reach alert level in {time}.",
        minor_flood: "Rising {rate} an hour, it would reach minor flood level in {time}.",
        major_flood: "Rising {rate} an hour, it would reach major flood level in {time}.",
      },
      lessThanHour: "less than an hour",
      aboutHour: "about an hour",
      aboutHours: "about {count} hours",
      forecastStale: "A Google flood forecast here is out of date, so it is not used.",
      rainStale: "The rain forecast is out of date, so it is not used.",
      rainHeavy:
        "Heavy rain is forecast over the catchment: {day} mm in the next 24 hours and {days} mm in 72 hours on average.",
      rainFairly:
        "Fairly heavy rain is forecast over the catchment: {day} mm in the next 24 hours and {days} mm in 72 hours on average.",
      rainAverage:
        "Forecast rain over the catchment is {day} mm in the next 24 hours and {days} mm in 72 hours on average.",
    },
  },

  /** Colombo's canal gauges (features/canals): the source's own words. */
  canals: {
    state: {
      normal: "Normal",
      rising: "Rising",
      higher: "Higher than usual",
      offline: "Not reporting",
      unknown: "State not given",
    },
    seaTrend: {
      rising: "rising",
      risingSlowly: "rising slowly",
      falling: "falling",
      fallingSlowly: "falling slowly",
      steady: "steady",
    },
    change: "{change} m in 3 h",
    otherGauges: "Other gauges",
    title: "Colombo canals",
    subtitle: "{reporting} of {total} gauges reporting",
    subtitleEmpty: "Canal gauges",
    close: "Close Colombo canals",
    through: "Through {via}",
    context: "Context, not a flood warning.",
    error: "The canal levels could not be read just now.",
    allStale: "No gauge has a current reading, so we cannot say how the canals stand.",
    read: "Read",
    raised: "Higher than usual or rising",
    sea: "Sea at the Kelani mouth",
    seaValue: "{level} m, {trend}",
    tide: {
      high: "Next predicted high tide",
      low: "Next predicted low tide",
      generic: "Next predicted tide",
    },
    lastRead: "last read {age}",
    noReading: "No reading",
    footnote:
      "The source gives no positions for these gauges, so they are listed rather than drawn on the map. Levels are metres above mean sea level.",
  },
}
