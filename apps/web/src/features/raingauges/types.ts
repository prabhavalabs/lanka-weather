/** Mirrors `/v1/rain/gauges` (docs/api/rainfall-stations-air-and-sea.md). */
export type RainBand = "unknown" | "none" | "light" | "moderate" | "heavy" | "very_heavy"

export interface RainGaugeProperties {
  id: string
  name: string
  institution: string
  /** The current rolling 24-hour total; null when out of date, so it cannot read as dry. */
  rain24hMm: number | null
  /** The newest figure there is, however old. */
  lastRain24hMm: number | null
  band: RainBand
  observedAt: string | null
  stale: boolean
  source: string
  sourceTime: string
  fetchedAt: string
}

export interface RainGaugeFeature {
  type: "Feature"
  id: string
  geometry: { type: "Point"; coordinates: [number, number] }
  properties: RainGaugeProperties
}

export interface RainGaugeMeta {
  source: string
  attribution: string
  note: string
  fetchedAt: string | null
  sourceTime: string | null
  generatedAt: string
  staleAfterSeconds: number
  total: number
  fresh: number
  stale: number
  wet: number
  institutions: Record<string, number> | null
}

export interface RainGaugeCollection {
  type: "FeatureCollection"
  features: RainGaugeFeature[]
  meta: RainGaugeMeta
}

export interface RainGaugeDetail {
  gauge: RainGaugeFeature
  series: { at: string; rain24hMm: number }[]
  meta: RainGaugeMeta
}

/** The nearest gauge to a place, in `/v1/summary`. */
export interface NearbyRainGauge extends RainGaugeProperties {
  distanceKm: number
  lon: number
  lat: number
  attribution: string
}
