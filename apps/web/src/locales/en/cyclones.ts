/** Messages of the cyclones feature. English is the source: it fixes the keys. */
export const cyclones = {
  title: "Cyclones",
  cyclone: "Cyclone",
  newTab: "(opens in a new tab)",
  /** "GDACS Red alert"; the colour is GDACS's. */
  alert: "GDACS {level} alert",
  summary: {
    one: "{name} active",
    many: { one: "{count} active: {names}", other: "{count} active: {names}" },
  },
  /** A system in the layer list: "MURASU-26 · 545 km from Sri Lanka". */
  listItem: "{name} · {distance}",
  checked: "checked {age}",
  missing: "This system is no longer listed as active.",
  /** What can be said about the feed (hazards.Cyclones). */
  status: {
    waiting: "Cyclones have not been collected yet.",
    activeStale: "We have not been able to check GDACS recently, so this may be out of date.",
    unknown:
      "We have not been able to check GDACS recently, so we cannot say whether a cyclone is active. Check with the Department of Meteorology.",
    none: "No active cyclone in the North Indian Ocean",
  },
  evaluation:
    "JTWC and IMD RSMC New Delhi advisories remain under evaluation as direct sources. Track data here is GDACS's, which draws on JTWC.",
  legend: {
    track: "Track so far",
    forecast: "Forecast",
    buffer: "Wind buffer",
    uncertainty: "Uncertainty",
  },
  panel: {
    subtitle: "Tropical cyclone · North Indian Ocean",
    close: "Close cyclone details",
    footer:
      "Source: Global Disaster Alert and Coordination System (GDACS), gdacs.org. Fetched {age}.",
    fromCoast: "From Sri Lanka's coast",
    maxWind: "Maximum wind",
    position: "Position",
    /** "14.7° N, 80.7° E" */
    positionValue: "{lat}° {ns}, {lon}° {ew}",
    advisory: "Advisory",
    firstSeen: "First seen",
    lists: "GDACS lists",
    trackFrom: "Track from",
    report: "GDACS report",
    disclaimer:
      "GDACS's alert rates likely impact on the countries it lists; it is not a warning for Sri Lanka. Official warnings come from the Department of Meteorology and the DMC.",
  },
}
