import type { LngLatBoundsLike } from "maplibre-gl"

/**
 * Keyless OpenFreeMap vector styles (OpenStreetMap data). Land, water and
 * labels are retuned to the theme tokens as each style loads (basemap.ts).
 */
export const mapStyles = {
  light: "https://tiles.openfreemap.org/styles/positron",
  dark: "https://tiles.openfreemap.org/styles/dark",
} as const

export type MapTheme = keyof typeof mapStyles

/** Sri Lanka's mainland and near-shore islands, west-south-east-north. */
export const sriLankaBounds: LngLatBoundsLike = [
  [79.52, 5.85],
  [81.95, 9.9],
]

/**
 * Panning limit: wide enough for the earthquakes and cyclones drawn, the
 * Indian Ocean from 60° to 105° E, and narrow enough to keep the map about
 * Sri Lanka.
 */
export const panLimit: LngLatBoundsLike = [
  [55, -20],
  [110, 30],
]

/** Far enough out to see that region at once on a laptop. */
export const minimumZoom = 3
