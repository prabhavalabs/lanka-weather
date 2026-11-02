import { isNotReady } from "./api"

/** Mirrors internal/layers.Layer in the Go service. */
export type LayerCategory = "hydrology" | "weather" | "hazard" | "geography"
/**
 * How a layer is drawn: GeoJSON features, raster image tiles, vector tiles,
 * the animated wind particles, or values on a regular grid (colour field).
 */
export type LayerKind = "geojson" | "raster" | "vector" | "wind" | "grid"
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
  /** URL template of a vector layer's tiles, with {z}, {x} and {y} placeholders. */
  tilePath?: string
  /** Zooms the tiles are served at; clients overzoom beyond maxZoom. */
  minZoom?: number
  maxZoom?: number
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

/** How often to ask again while a layer's data has not been collected yet. */
export const notReadyPollMs = 60_000

/**
 * Query options for a layer's data: the catalogue's timings, except that
 * while the API answers "not collected yet" (503 no_data) the query asks
 * again every minute, so data appears soon after the collectors run.
 */
export function layerQueryOptions(layer: Pick<Layer, "refreshSeconds">) {
  const { staleTime, refetchInterval } = queryTimings(layer)
  return {
    staleTime,
    refetchInterval: (query: { state: { error: unknown } }) =>
      isNotReady(query.state.error) ? notReadyPollMs : refetchInterval,
  }
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
