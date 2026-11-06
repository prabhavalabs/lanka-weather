/** GET /v1/flood-risk: the combined flood-risk estimate by district (ADR 0019). */

export type FloodRiskFamily =
  | "official"
  | "google_flash"
  | "rain_now"
  | "river"
  | "google_gauge"
  | "rain_forecast"
  | "wet_ground"
  | "reservoir"

export interface FloodRiskSignal {
  family: FloodRiskFamily
  /** 1 notable, 2 significant, 3 severe. */
  points: number
  value: number
  unit: "mm" | "level" | "count"
  window?: "1h" | "3h" | "6h" | "24h"
}

export type FloodRiskVariant = "" | "moderate" | "high" | "very_high"
export type FloodRiskScenario = "flash" | "river" | "both" | "general"

export interface FloodRiskDistrict {
  district: string
  level: 0 | 1 | 2 | 3
  variant: FloodRiskVariant
  score: number
  families: number
  scenario: FloodRiskScenario
  signals: FloodRiskSignal[]
  inputs?: {
    rain: {
      rise1hMm: number
      rise3hMm: number
      gauge: string
      gaugeId: string
      priorMm: number
    } | null
    forecast: { next6hMaxMm: number; next24hMeanMm: number } | null
  }
}

export interface FloodRiskDocument {
  districts: FloodRiskDistrict[]
  meta: { note: string; generatedAt: string; complete: boolean; missing: string[] }
}
