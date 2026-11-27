import type { LayerSpecification } from "maplibre-gl"
import { useSyncExternalStore } from "react"

/**
 * Parts of the basemap a reader can switch on and off. The map starts with
 * land, sea and place names only, so the data drawn over it stays readable;
 * the rest of OpenStreetMap's detail is there on request.
 */
export type BasemapGroup = "places" | "roads" | "waterways" | "landcover" | "boundaries"

/** In the order the sidebar lists them. */
export const basemapGroups: readonly BasemapGroup[] = [
  "places",
  "waterways",
  "roads",
  "landcover",
  "boundaries",
]

export const defaultBasemapGroups: readonly BasemapGroup[] = ["places"]

/** Where a basemap layer records its group, to be shown or hidden in place. */
export const basemapGroupKey = "lanka-weather:basemap-group"

/**
 * The group an OpenMapTiles layer belongs to, by its source layer. The sea,
 * lakes and the land itself belong to none: they are always drawn, and the
 * water is what clips the data fills to the coast (slots.ts).
 */
export function basemapGroup(layer: LayerSpecification): BasemapGroup | undefined {
  const source = "source-layer" in layer ? layer["source-layer"] : undefined
  switch (source) {
    case "place":
      return "places"
    case "transportation":
    case "transportation_name":
    case "aeroway":
    case "aerodrome_label":
      return "roads"
    case "waterway":
    case "water_name":
      return "waterways"
    case "landcover":
    case "landuse":
    case "park":
    case "building":
      return "landcover"
    case "boundary":
      return "boundaries"
    default:
      return undefined
  }
}

export function isBasemapGroup(value: unknown): value is BasemapGroup {
  return basemapGroups.includes(value as BasemapGroup)
}

/** Kept in this browser only. */
export const basemapStorageKey = "lanka-weather:basemap"

function readStored(): readonly BasemapGroup[] {
  try {
    const stored = localStorage.getItem(basemapStorageKey)
    if (stored === null) return defaultBasemapGroups
    const parsed: unknown = JSON.parse(stored)
    return Array.isArray(parsed) ? parsed.filter(isBasemapGroup) : defaultBasemapGroups
  } catch {
    return defaultBasemapGroups
  }
}

let shown: readonly BasemapGroup[] =
  typeof window === "undefined" ? defaultBasemapGroups : readStored()
const listeners = new Set<() => void>()

/** The groups on now. Read outside React too, as each basemap style loads. */
export function getBasemapGroups(): readonly BasemapGroup[] {
  return shown
}

export function setBasemapGroup(group: BasemapGroup, on: boolean) {
  const next = basemapGroups.filter((value) => (value === group ? on : shown.includes(value)))
  shown = next
  try {
    localStorage.setItem(basemapStorageKey, JSON.stringify(next))
  } catch {
    // Storage may be full or disabled; the choice then lasts for the session.
  }
  for (const listener of listeners) listener()
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function useBasemapGroups(): readonly BasemapGroup[] {
  return useSyncExternalStore(subscribe, getBasemapGroups, () => defaultBasemapGroups)
}
