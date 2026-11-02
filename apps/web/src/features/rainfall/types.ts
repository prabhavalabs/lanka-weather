/** Mirrors `/v1/rain/steps` and `/v1/rain/observed` (docs/api/rainfall-stations-air-and-sea.md). */
export interface RainStep {
  /** Goes in the tile URL. */
  time: string
  /** The half hour the image covers. */
  validAt: string
  endsAt: string
  fetchedAt: string
  tiles: number
  emptyTiles: number
}

export interface RainLegendStop {
  rateMmH: number
  color: string
}

export interface RainSteps {
  source: string
  attribution: string
  tilePath: string
  minZoom: number
  maxZoom: number
  tileSize: number
  /** West, south, east, north. */
  bounds: [number, number, number, number]
  /** Oldest first. */
  steps: RainStep[]
  newestValidAt: string | null
  fetchedAt: string | null
  stale: boolean
  estimate: boolean
  legend: { unit: string; stops: RainLegendStop[]; source: string }
  note: string
}

export type ObservedRainKind = "met" | "gauge"
export type ObservedRainMethod = "daily_report" | "sum_3h" | "sum_gauge"

export interface ObservedRainProperties {
  id: string
  /** Opens the station: a Met station, or a river gauge. */
  stationId: string
  kind: ObservedRainKind
  name: string
  rainMm: number
  /** Closes the 24 hours the total covers. */
  windowEnd: string
  method: ObservedRainMethod
  source: string
  stale: boolean
}

export interface ObservedRainCollection {
  type: "FeatureCollection"
  features: {
    type: "Feature"
    id: string
    geometry: { type: "Point"; coordinates: [number, number] }
    properties: ObservedRainProperties
  }[]
  meta: {
    attribution: string
    generatedAt: string
    observed: boolean
    fresh: number
    stale: number
  }
}
