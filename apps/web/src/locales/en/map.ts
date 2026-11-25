/** Messages of the map feature. English is the source: it fixes the keys. */
export const map = {
  label: "Map of Sri Lanka",
  /** MapLibre's own buttons. */
  controls: {
    zoomIn: "Zoom in",
    zoomOut: "Zoom out",
    resetBearing: "Reset bearing to north",
    resetView: "Show all of Sri Lanka",
    findMyLocation: "Find my location",
    locationNotAvailable: "Location not available",
    toggleAttribution: "Toggle attribution",
    mapFeedback: "Map feedback",
    map: "Map",
  },
  /** The hover chip by the pointer. */
  /** The home place (features/home): saved in this browser only. */
  home: {
    goHome: "Go home: {name}",
    unset: "Set a home place",
    hint: "Open a place on the map, then choose the house in its panel to make it your home place.",
    set: "Make {name} my home place",
    isHome: "{name} is your home place. Remove it",
    saved: "{name} is now your home place. The house button on the map brings you back here.",
    removed: "Home place removed.",
  },
  chip: {
    riverGauge: "River gauge",
    riverGaugeLevel: "{status} · {level}",
    reservoir: "Reservoir",
    reservoirStatus: "Reservoir: {status}",
    floodForecast: "Flood forecast: {severity}",
    noData: "no data",
    lowerConfidence: "Lower confidence",
    floodArea: "Flood area",
    googleForecast: "Google flood forecast",
    stream: "Stream",
    tributary: "Tributary, not assessed",
    risk: "{risk} · Estimate",
    pastFlood: "Past flood",
    pastEvent: "Past event, not a forecast",
    basin: "River basin",
    basinState: "Basin: {state}",
    subbasinState: "Sub-basin: {state}",
    regionValue: "{label}: {value} {unit}",
    wind: "Wind {wind}",
  },
}
