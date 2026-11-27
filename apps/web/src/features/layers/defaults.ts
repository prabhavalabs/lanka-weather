/**
 * The layers a first visit shows, and how a browser that saved the earlier,
 * busier set is moved on. Pure, so it is tested without a browser.
 */

/**
 * Every warning on the map: official warnings and landslide levels, storms
 * and earthquakes, and Google's flood forecast. The rest (rivers, gauges,
 * reservoirs, the weather) is a switch away in the layer list.
 */
export const warningLayers = [
  "official-warnings",
  "landslide-warnings",
  "flood-forecast",
  "cyclones",
  "earthquakes",
]

export const defaultLayers = [...warningLayers]

/** Bumped when the defaults change; stored once a browser has been moved on. */
export const defaultsVersion = 2

/** The first visit's layers before November 2026. */
const previousDefaults = [
  "landslide-warnings",
  "official-warnings",
  "earthquakes",
  "cyclones",
  "rivers",
  "river-levels",
  "flood-forecast",
  "reservoirs",
  "rain-gauges",
  "admin-regions",
]

function sameSet(a: string[], b: string[]) {
  const left = new Set(a)
  const right = new Set(b)
  return left.size === right.size && [...left].every((id) => right.has(id))
}

/**
 * The layers for a browser that saved its choice under earlier defaults.
 * Someone who never changed the old defaults gets the new ones; someone who
 * chose their own keeps them, with any warning layer they were never
 * offered (`offered`, the earlier one-off list) switched on, once.
 */
export function migrateLayers(current: string[], offered: string[]): string[] {
  if (sameSet(current, previousDefaults)) return [...defaultLayers]
  const unseen = warningLayers.filter((id) => !offered.includes(id) && !current.includes(id))
  // Flood forecast was a default from the start, so never "offered": leave it as chosen.
  const added = unseen.filter((id) => id !== "flood-forecast")
  return added.length === 0 ? current : [...current, ...added]
}
