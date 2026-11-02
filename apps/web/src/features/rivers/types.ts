/** Mirrors internal/rivers in the Go service. */
export type RiverStatus = "normal" | "alert" | "minor_flood" | "major_flood" | "unknown"
export type RiverTrend = "rising" | "falling" | "steady" | "unknown"

export interface StationProperties {
  id: string
  name: string
  basin: string
  unit: "m" | "ft"
  alertLevel: number | null
  minorFloodLevel: number | null
  majorFloodLevel: number | null
  waterLevel: number | null
  rainfallMm: number | null
  observedAt: string | null
  status: RiverStatus
  trend: RiverTrend
  ratePerHour: number | null
  stale: boolean
}

export interface StationFeature {
  type: "Feature"
  id: string
  geometry: { type: "Point"; coordinates: [number, number] }
  properties: StationProperties
}

export interface RiverLevelsCollection {
  type: "FeatureCollection"
  features: StationFeature[]
  meta: {
    source: string
    attribution: string
    fetchedAt: string | null
    generatedAt: string
  }
}

export interface StationHistory {
  station: StationProperties
  readings: { observedAt: string; waterLevel: number; rainfallMm: number | null }[]
}
