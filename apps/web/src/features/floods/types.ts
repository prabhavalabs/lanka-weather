/** Mirrors `/v1/floods/forecast` (docs/api/weather-and-floods.md). */
import type { FloodSeverity, FloodThresholds, FloodTrend } from "../location/types"

export type { FloodSeverity, FloodThresholds, FloodTrend }

export interface FloodGaugeProperties {
  kind: "gauge"
  gaugeId: string
  gaugeSource: string
  siteName: string
  river: string
  qualityVerified: boolean
  severity: FloodSeverity
  trend: FloodTrend
  issuedAt: string
  forecastStart: string | null
  forecastEnd: string | null
  thresholds: FloodThresholds | null
  fetchedAt: string
  stale: boolean
}

export type FloodAreaKind =
  | "notification"
  | "inundation_high"
  | "inundation_medium"
  | "inundation_low"
  | "flash_likely"
  | "flash_highly_likely"
  | "significant_event"

export interface FloodAreaProperties {
  /** The polygon's id, repeated for clicks: vector features keep only properties. */
  id?: string
  kind: FloodAreaKind
  gaugeId?: string
  /** A gauge's severity, or a flash flood's likelihood or an event's size. */
  severity: FloodSeverity | "likely" | "highly_likely" | "moderate" | "minor"
  issuedAt?: string
  validFrom?: string
  validUntil?: string
  details?: Record<string, unknown>
  fetchedAt?: string
  /** The area of a gauge Google has not verified; shown only with Extended coverage. */
  lowerConfidence?: boolean
}

export interface FloodGaugeFeature {
  type: "Feature"
  id: string
  geometry: { type: "Point"; coordinates: [number, number] }
  properties: FloodGaugeProperties
}

export interface FloodAreaFeature {
  type: "Feature"
  id: string
  geometry: { type: "MultiPolygon" | "Polygon"; coordinates: unknown }
  properties: FloodAreaProperties
}

export interface FloodForecastCollection {
  type: "FeatureCollection"
  features: (FloodGaugeFeature | FloodAreaFeature)[]
  meta: {
    source: string
    attribution: string
    region: string
    /** Local test data is in the tables; never true in production. */
    seeded?: boolean
    configured: boolean
    fetchedAt: string | null
    generatedAt: string
    gauges: number
    areas: number
    verifiedGauges?: number
    lowerConfidenceGauges?: number
    flashAreas?: number
  }
}
