/** Mirrors `/v1/stations` and `/v1/stations/{id}` (docs/api/rainfall-stations-air-and-sea.md). */
export type ReadingKind = "3h" | "daily" | "none"

export interface MetStationProperties {
  id: string
  name: string
  wmoId: string | null
  elevationM: number | null
  /** The position is geocoded from the station's name. */
  positionApproximate: boolean
  reading: ReadingKind
  observedAt: string | null
  /** An instantaneous reading, from the 3-hourly feed only. */
  temperatureC: number | null
  humidityPct: number | null
  /** Rain in the `rainHours` ending at `observedAt`. */
  rainMm: number | null
  rainHours: number
  rain24hMm: number | null
  tempMinC: number | null
  tempMaxC: number | null
  reportsTemperature: boolean
  stale: boolean
}

export interface MetFeed {
  source: string
  newestObservedAt: string | null
  fetchedAt: string | null
  stale: boolean
}

export interface MetMeta {
  attribution: string
  sourceUrl: string
  generatedAt: string
  observed: boolean
  threeHourly: MetFeed
  daily: MetFeed
  stale: boolean
}

export interface MetStationFeature {
  type: "Feature"
  id: string
  geometry: { type: "Point"; coordinates: [number, number] }
  properties: MetStationProperties
}

export interface MetStationCollection {
  type: "FeatureCollection"
  features: MetStationFeature[]
  meta: MetMeta
}

export interface ThreeHourlyReading {
  observedAt: string
  temperatureC: number | null
  humidityPct: number | null
  rainMm: number | null
}

export interface DailyReading {
  date: string
  observedAt: string
  rainMm: number | null
  tempMinC: number | null
  tempMaxC: number | null
}

export interface MetStationDetail {
  station: MetStationFeature
  threeHourly: ThreeHourlyReading[]
  daily: DailyReading[]
  meta: MetMeta
}
