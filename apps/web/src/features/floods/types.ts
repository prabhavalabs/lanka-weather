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
  kind: FloodAreaKind
  gaugeId?: string
  severity: FloodSeverity
  issuedAt?: string
  validFrom?: string
  validUntil?: string
  details?: Record<string, unknown>
  fetchedAt?: string
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
    configured: boolean
    fetchedAt: string | null
    generatedAt: string
    gauges: number
    areas: number
  }
}
