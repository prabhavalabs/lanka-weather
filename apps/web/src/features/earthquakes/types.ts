/** Mirrors /v1/hazards/earthquakes (docs/api/warnings-and-hazards.md). */
import type { FeatureCollection, Point } from "geojson"

export interface Earthquake {
  id: string
  magnitude: number
  magType: string
  place: string
  time: string
  depthKm: number
  /** To Sri Lanka's nearest coast. */
  distanceKm: number
  url: string
  alert: string
  /** USGS's flag: an indicator, not a warning for Sri Lanka. */
  tsunami: boolean
  significance: number
  status: string
  updatedAt: string
  fetchedAt: string
  source: string
}

export type EarthquakeCollection = FeatureCollection<Point, Earthquake> & {
  meta: {
    source: string
    attribution: string
    windowDays: number
    fetchedAt: string | null
    generatedAt: string
    stale: boolean
    count: number
  }
}
