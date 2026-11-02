/** Mirrors internal/layers.Layer in the Go service. */
export type LayerCategory = "hydrology" | "weather" | "hazard" | "geography"
export type LayerKind = "geojson" | "raster" | "vector" | "wind"
export type LayerStatus = "planned" | "available"

export interface Layer {
  id: string
  title: string
  description: string
  category: LayerCategory
  kind: LayerKind
  status: LayerStatus
  sources: string[]
  refreshSeconds: number
  staleAfterSeconds: number
  attribution: string
  /** API path serving the layer's data, for layers whose data comes from the API. */
  dataPath?: string
}

export const categoryOrder: LayerCategory[] = ["hazard", "hydrology", "weather", "geography"]

export const categoryLabels: Record<LayerCategory, string> = {
  hazard: "Hazards and warnings",
  hydrology: "Rivers and floods",
  weather: "Weather",
  geography: "Geography",
}

/** Floor for refetching so a misconfigured layer cannot hammer the API. */
const minimumRefreshMs = 60_000

/**
 * Query timings for a layer's data, derived from the catalogue so cadence is
 * defined once, on the server. Data becomes stale (refetched on focus or
 * mount) after half the refresh interval and is polled at the interval.
 */
export function queryTimings(layer: Pick<Layer, "refreshSeconds">) {
  const refreshMs = Math.max(layer.refreshSeconds * 1000, minimumRefreshMs)
  return { staleTime: Math.round(refreshMs / 2), refetchInterval: refreshMs }
}

/** Whether data observed at `observedAt` is too old to be shown as current. */
export function isStale(
  layer: Pick<Layer, "staleAfterSeconds">,
  observedAt: Date,
  now: Date = new Date(),
): boolean {
  return now.getTime() - observedAt.getTime() > layer.staleAfterSeconds * 1000
}

/** Layers grouped by category in display order, omitting empty groups. */
export function groupByCategory(layers: Layer[]) {
  return categoryOrder
    .map((category) => ({
      category,
      label: categoryLabels[category],
      layers: layers.filter((layer) => layer.category === category),
    }))
    .filter((group) => group.layers.length > 0)
}
