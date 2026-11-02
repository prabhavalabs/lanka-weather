/** Mirrors the location endpoints of the Go service (docs/api/weather-and-floods.md). */
import type { NearbyRiverRisk } from "../river-risk/types"
import type { ReservoirFlow, ReservoirStatus } from "../reservoirs/types"
import type { LevelUnit, RiverStatus, RiverTrend } from "../rivers/types"
import type { OfficialAt } from "../warnings/types"

export type RegionLevel = "province" | "district" | "ds" | "gn"

export interface RegionRef {
  id: string
  level: RegionLevel
  name: string
  nameSi?: string | null
  nameTa?: string | null
}

export interface GridSpec {
  id: string
  lon0: number
  lat0: number
  step: number
  nx: number
  ny: number
}

/** Current conditions at a point; null where the source gave no value. */
export interface PointCurrent {
  validAt: string
  temperatureC: number | null
  apparentTemperatureC: number | null
  relativeHumidityPct: number | null
  dewPointC: number | null
  precipitationMm: number | null
  precipitationProbabilityPct: number | null
  cloudCoverPct: number | null
  pressureHpa: number | null
  windSpeedKmh: number | null
  windDirectionDeg: number | null
  windGustsKmh: number | null
  weatherCode: number | null
  uvIndex: number | null
  isDay: number | null
}

export interface PointHourly {
  times: string[]
  temperatureC: (number | null)[]
  apparentTemperatureC: (number | null)[]
  relativeHumidityPct: (number | null)[]
  precipitationMm: (number | null)[]
  precipitationProbabilityPct: (number | null)[]
  cloudCoverPct: (number | null)[]
  windSpeedKmh: (number | null)[]
  windDirectionDeg: (number | null)[]
  windGustsKmh: (number | null)[]
  weatherCode: (number | null)[]
  uvIndex: (number | null)[]
  isDay: (number | null)[]
}

export interface PointDaily {
  dates: string[]
  temperatureMaxC: (number | null)[]
  temperatureMinC: (number | null)[]
  apparentTemperatureMaxC: (number | null)[]
  precipitationSumMm: (number | null)[]
  precipitationProbabilityMaxPct: (number | null)[]
  windSpeedMaxKmh: (number | null)[]
  windGustsMaxKmh: (number | null)[]
  windDirectionDominantDeg: (number | null)[]
  uvIndexMax: (number | null)[]
  weatherCode: (number | null)[]
}

export interface PointMeta {
  source: string
  model: string
  attribution: string
  grid: GridSpec
  interpolation: string
  currentFetchedAt: string | null
  forecastFetchedAt: string | null
  currentStale: boolean
  forecastStale: boolean
}

/** `/v1/weather/point`: 72 hours and 7 days; `/v1/summary` cuts it to 24 hours and 3 days. */
export interface PointForecast {
  location: { lat: number; lon: number }
  current: PointCurrent | null
  hourly: PointHourly | null
  daily: PointDaily | null
  meta: PointMeta
}

export interface NearbyRiver {
  id: string
  name: string
  basin: string
  distanceKm: number
  status: RiverStatus
  trend: RiverTrend
  waterLevel: number | null
  unit: LevelUnit
  alertLevel: number | null
  minorFloodLevel: number | null
  majorFloodLevel: number | null
  observedAt: string | null
  stale: boolean
  lon: number
  lat: number
  source: string
  attribution: string
}

/** A mapped reservoir near the place, spilling or releasing on a current reading. */
export interface NearbyReservoir {
  id: string
  name: string
  distanceKm: number
  status: ReservoirStatus
  storagePercent: number | null
  flow: ReservoirFlow | null
  readingDay: string | null
  lon: number
  lat: number
}

export type FloodSeverity = "no_flooding" | "above_normal" | "severe" | "extreme" | "unknown"
export type FloodTrend = "rise" | "fall" | "no_change" | "unknown"

export interface FloodThresholds {
  warning: number | null
  danger: number | null
  extremeDanger: number | null
  unit: string
}

export interface NearbyFloodGauge {
  gaugeId: string
  siteName?: string
  river?: string
  severity: FloodSeverity
  trend: FloodTrend
  distanceKm?: number
  issuedAt?: string
  forecastStart?: string
  forecastEnd?: string
  thresholds?: FloodThresholds
  qualityVerified?: boolean
  stale?: boolean
  lon?: number
  lat?: number
}

export interface NearbyFloodArea {
  id?: string
  kind: string
  severity: FloodSeverity
  gaugeId?: string
  validFrom?: string
  validUntil?: string
}

export type AlertType = "heavy_rain" | "strong_wind" | "heat" | "river_level" | "flood_forecast"
export type AlertLevel = "yellow" | "orange" | "red"

/** A derived alert: an estimate made by lanka-weather, never an official warning. */
export interface DerivedAlert {
  type: AlertType
  level: AlertLevel
  title: string
  detail: string
  value?: number | null
  threshold?: number | null
  unit?: string | null
  validFrom?: string | null
  validUntil?: string | null
  estimate: true
  basis: string
}

export interface Summary {
  location: { lat: number; lon: number }
  generatedAt: string
  regions: {
    province: RegionRef | null
    district: RegionRef | null
    ds: RegionRef | null
    gn: RegionRef | null
  } | null
  weather: PointForecast | null
  rivers: NearbyRiver[] | null
  /** Rivers near the place, with the estimated risk of their nearest stretch. */
  nearbyRivers?: NearbyRiverRisk[] | null
  floodForecast: { gauges: NearbyFloodGauge[]; areas: NearbyFloodArea[] } | null
  reservoirs: NearbyReservoir[] | null
  alerts: DerivedAlert[]
  official: OfficialAt
  /** The nearest station's latest reading: measured, not modelled. */
  observed?: NearbyObservation | null
  /** Model estimate here; null outside its grid. */
  airQuality?: AirQualityPoint | null
  /** Model sea state on the water or at a coast; null inland. */
  sea?: SeaPoint | null
  unavailable: string[]
}

export interface NearbyObservation {
  id: string
  name: string
  reading: "3h" | "daily" | "none"
  observedAt: string | null
  temperatureC: number | null
  humidityPct: number | null
  rainMm: number | null
  rainHours: number
  rain24hMm: number | null
  tempMinC: number | null
  tempMaxC: number | null
  stale: boolean
  distanceKm: number
  lon: number
  lat: number
  attribution: string
}

export type AqiCategory =
  "good" | "moderate" | "unhealthy_sensitive" | "unhealthy" | "very_unhealthy" | "hazardous"

export interface AirQualityPoint {
  usAqi: number
  category: AqiCategory
  pm25: number | null
  validAt: string
  fetchedAt: string
  stale: boolean
  estimate: boolean
  attribution: string
}

export interface SeaPoint {
  waveHeightM: number
  wavePeriodS: number | null
  waveDirectionDeg: number | null
  swellHeightM: number | null
  swellPeriodS: number | null
  swellDirectionDeg: number | null
  state: string
  distanceKm: number
  validAt: string
  fetchedAt: string
  stale: boolean
  estimate: boolean
  attribution: string
  note: string
}
