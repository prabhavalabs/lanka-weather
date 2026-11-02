import type { LngLatBoundsLike } from "maplibre-gl"

/** Keyless OpenFreeMap vector styles (OpenStreetMap data). */
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
 * Panning limit: wide enough to follow a cyclone approaching from the Bay of
 * Bengal or the Arabian Sea, narrow enough to keep the map about Sri Lanka.
 */
export const panLimit: LngLatBoundsLike = [
  [70, -2],
  [92, 18],
]
