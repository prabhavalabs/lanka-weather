/** Messages of the floods feature. English is the source: it fixes the keys. */
export const floods = {
  /**
   * Flood Hub's words for Google's gauge severities (ADR 0015). "Warning" here is
   * Google's forecast, never an official warning.
   */
  severity: {
    extreme: "Extreme",
    severe: "Danger",
    above_normal: "Warning",
    no_flooding: "Normal",
    unknown: "No data",
  },
  /** A river basin's state: the worst forecast or gauge in it. */
  basinState: {
    extreme: "Extreme",
    severe: "Danger",
    above_normal: "Warning",
    no_flooding: "Normal",
    no_forecast: "No forecast",
  },
  /** Each kind of Google flood polygon, in full. */
  area: {
    notification: "Riverine flood notification area",
    inundation_high: "Inundation: very high probability",
    inundation_medium: "Inundation: high probability",
    inundation_low: "Inundation: moderate probability",
    flash_likely: "Flash flood: likely",
    flash_highly_likely: "Flash flood: highly likely",
    significant_event: "Significant flood event",
    fallback: "Flood area",
  },
  /** The kind alone, without its level. */
  areaKind: {
    notification: "Riverine flood notification",
    inundation: "Inundation",
    flash: "Flash flood",
    significant_event: "Significant flood event",
  },
  /** Flood Hub's words for Google's three inundation probability levels. */
  inundation: {
    inundation_high: "Very high",
    inundation_medium: "High",
    inundation_low: "Moderate",
  },
  /** How likely a flood area is. */
  likelihood: {
    highlyLikely: "Highly likely",
    likely: "Likely",
    eventPresent: "Event present",
    forecast: "Forecast",
    notificationNote: "Google's notification polygon for this gauge's forecast.",
    /** Google's levels when its map gives depth rather than probability. */
    depth: {
      inundation_high: "High depth",
      inundation_medium: "Medium depth",
      inundation_low: "Low depth",
    },
    depthNote: "Google gives this map by depth rather than probability.",
    /** `{google}` is Google's own level, as its API names it: high, medium or low. */
    probabilityNote: "Flood Hub's word for Google's \"{google}\" probability level.",
  },
  /** A gauge with no site name. */
  gaugeName: {
    river: "{river} gauge",
    id: "Gauge {id}",
  },
  /** The layer's one-line summary: "10 gauges · 2 raised · 1 area". */
  summary: {
    gauges: { one: "{count} gauge", other: "{count} gauges" },
    noneRaised: "none raised",
    raised: "{count} raised",
    areas: { one: "{count} area", other: "{count} areas" },
  },
  /** Every Google forecast panel says what it is, and who issues warnings. */
  forecastFooter:
    "Google Flood Hub (Flood Forecasting API). A forecast, not an official warning. Official flood warnings come from the Irrigation Department and the DMC.",
  lowerConfidence: "Lower confidence",
  testData: "Test data",
  legend: {
    title: "Flood forecast",
    riverine: "Riverine floods",
    expectedSeverity: "Expected severity",
    showNormal: "Show gauges forecast normal",
    trend: "Forecast trend",
    flash: "Flash floods",
    within24h: "Within 24 h",
    showFlash: "Show flash-flood areas",
    noFlash: "No flash-flood areas for Sri Lanka right now",
    extended: "Extended coverage",
    extendedCount: "Extended coverage ({count})",
    extendedNote: "Lower-confidence gauges only",
    showExtended: "Show lower-confidence gauges",
    extendedHelp:
      "Hollow dots. Google has not verified these models. They do not raise river risk, alerts or the overview.",
    events: "Significant flood events",
    inundation: "Inundation probability",
    basins: "River basins",
    basinsNote: "Dashed: the worst forecast or gauge in each",
    basinsHelp:
      "A basin is not a flood area: solid shapes are the areas Google publishes. No forecast means nothing is known there, not that it is calm.",
    basinsUnverified: "Some rest only on lower-confidence gauges; each basin's panel says so.",
    credit: "Google Flood Hub (Flood Forecasting API). Forecasts, not official warnings.",
  },
  gauge: {
    subtitle: "Flood forecast gauge · {source}",
    close: "Close gauge details",
    lowerConfidenceHelp:
      "Google has not quality-verified this gauge's model, and asks that it be used with caution. Flood Hub calls this Extended coverage. It does not raise river risk, alerts or the overview here.",
    forecastFor: "Forecast for",
    quality: "Quality",
    verified: "Verified by Google",
    notVerified: "Lower confidence, not verified",
    thresholds: "Gauge model thresholds",
    thresholdsHelp:
      "Severity is Google's forecast of the flow against these levels in the forecast window.",
    areas: "Forecast flood areas",
  },
  areaPanel: {
    subtitle: "Google flood forecast",
    close: "Close flood area details",
    kind: "Kind",
    likelihood: "Likelihood",
    from: "Forecast from",
    until: "Forecast until",
    read: "Read",
    openGauge: "Open its forecast gauge",
  },
  basin: {
    river: "River basin",
    coast: "Coast outside any named basin",
    close: "Close basin details",
    footer:
      "Boundary: {attribution}. Official flood warnings come from the Irrigation Department and the DMC.",
    unverifiedOnly: "Unverified gauges only",
    noForecast:
      "No current gauge or forecast covers this basin, so nothing is known about flooding here. That does not mean it is safe.",
    calm: "No gauge or forecast in this basin is above normal.",
    worst: "The worst gauge or forecast in this basin reads {state}.",
    newest: "Newest reading {age}",
    gauges: "Gauges and forecasts here",
    areas: "Google flood areas touching this basin",
    notVerified: "not verified by Google",
    notUsed: "Not used",
    openRiver: "Open the river",
    weatherHere: "Weather at this place",
  },
  /** Mapped past floods: past events, never a forecast. */
  past: {
    title: "Past flood",
    legend: "Past events, not a current forecast. {attribution}.",
    note: "Past flood extents mapped by the Irrigation Department. These show where water reached in past events, not a current forecast.",
    loadingList: "Loading past floods",
    close: "Close past flood details",
    missing: "This event is not in the mapped floods.",
    source: "Source: {attribution}.",
    pastEvent: "Past event",
    notForecast: "Not a forecast",
    when: "When",
    mappedAs: "Mapped as",
    extent: "Mapped extent",
    districts: "Districts",
    /** `{name}` is the source's own title, shown as published. */
    asPublished: "As published: {name}",
    /** An event's name, from its river (a proper name) and year. */
    name: "{river} flood, {year}",
    nameWithMagnitude: "{river} flood, {year} ({magnitude})",
    magnitude: {
      critical: "critical",
      major: "major",
      minor: "minor",
    },
    magnitudeLabel: {
      critical: "Critical",
      major: "Major",
      minor: "Minor",
    },
    /** For the layer list and hover chips. */
    listSummary: {
      one: "{count} mapped event · past, not a forecast",
      other: "{count} mapped events · past, not a forecast",
    },
    hover: "Past event, not a forecast",
  },
  /** Hover chips on the map. */
  hover: {
    forecast: "Flood forecast: {severity}",
    basin: "Basin: {state}",
    subbasin: "Sub-basin: {state}",
  },
}
