/** Mirrors internal/rivers in the Go service. */
export type RiverStatus = "normal" | "alert" | "minor_flood" | "major_flood" | "unknown"
export type RiverTrend = "rising" | "falling" | "steady" | "unknown"

/** "unknown" when the agency publishes no unit: the level is shown without a verdict. */
export type LevelUnit = "m" | "ft" | "unknown"

/**
 * An experimental forecast of the level about an hour after `issuedAt`, by a
 * third party, in the station's unit. It never changes the station's status.
 */
export interface NextHour {
  level: number
  low: number | null
  high: number | null
  unit: "m" | "ft"
  direction: string | null
  crossing: string | null
  method: string | null
  issuedAt: string
  experimental: boolean
  attribution: string
}

export interface StationProperties {
  id: string
  name: string
  basin: string
  unit: LevelUnit
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
  /** The collector the station comes from. */
  source: string
  /** Who to credit for the reading. */
  attribution: string
  /** Where to read more, when the source has a page for the station. */
  sourceUrl: string | null
  nextHour: NextHour | null
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
