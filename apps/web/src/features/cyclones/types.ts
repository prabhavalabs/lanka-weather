/** Mirrors /v1/hazards/cyclones (docs/api/warnings-and-hazards.md). */
import type { Feature, FeatureCollection, Geometry } from "geojson"

export type CycloneStatus = "waiting" | "active" | "none" | "unknown"

export interface CyclonePosition {
  kind: "position"
  eventId: string
  name: string
  alertLevel: string
  windKmh: number | null
  /** GDACS's words, unchanged. */
  severityText: string
  country: string
  startedAt: string
  latestAt: string
  advisorySource: string
  reportUrl: string
  lon: number
  lat: number
  /** To Sri Lanka's nearest coast. */
  distanceKm: number
  stale: boolean
  source: string
  sourceTime: string
  fetchedAt: string
}

export interface CycloneGeometry {
  kind: "track" | "forecast-track" | "track-point" | "wind-buffer" | "uncertainty-cone"
  eventId: string
  name: string
  alertLevel: string
  stage?: string
  label?: string
  windKmh?: number
  band?: string
  time?: string
  forecast?: boolean
}

export type CycloneFeature = Feature<Geometry, CyclonePosition | CycloneGeometry>

export type CycloneCollection = FeatureCollection<Geometry, CyclonePosition | CycloneGeometry> & {
  meta: {
    source: string
    attribution: string
    status: CycloneStatus
    note: string
    fetchedAt: string | null
    generatedAt: string
    stale: boolean
    count: number
  }
}
