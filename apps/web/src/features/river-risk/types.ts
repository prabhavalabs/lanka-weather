/** Mirrors internal/riverrisk in the Go service (docs/api/rivers-and-basins.md). */
import type { FloodSeverity, FloodTrend } from "../floods/types"
import type { RegionRef } from "../location/types"
import type { RiverStatus } from "../rivers/types"

export type RiskLevel = "normal" | "watch" | "high" | "severe" | "unknown"

export interface RiskReason {
  level: RiskLevel
  /** gauge, forecast, rain, reservoir or coverage. */
  kind: string
  text: string
}

export interface ReachGauge {
  kind: "station" | "google"
  id: string
  name: string
  /** next is named but not used as evidence: the first gauge below an ungauged stretch. */
  role: "upstream" | "downstream" | "next"
  waterLevel?: number
  unit?: "m" | "ft"
  status?: string
  trend?: string
  observedAt?: string
  stale: boolean
}

export interface RainSummary {
  next24hMm: number
  next72hMm: number
  catchmentKm2: number
  stale: boolean
}

export interface ReachRisk {
  fid: number
  riverId: string
  name: string
  level: RiskLevel
  reasons: RiskReason[]
  gauged: boolean
  lengthKm: number
  gauges: ReachGauge[]
  rain: RainSummary | null
}

export interface RiverRisk {
  id: string
  name: string
  nameSi: string | null
  aliases: string[]
  named: boolean
  parentId: string | null
  level: RiskLevel
  reasons: RiskReason[]
  uplandKm2: number
  lengthKm: number
  /** [west, south, east, north] */
  bbox: [number, number, number, number]
  reaches: ReachRisk[]
  gaugedReaches: number
  observedAt: string | null
}

export interface RiskMeta {
  generatedAt: string
  networkVersion: string
  networkBuiltAt: string | null
  gaugesObservedAt: string | null
  gaugesCurrent: number
  gaugesTotal: number
  forecastIssuedAt: string | null
  forecastFetchedAt: string | null
  rainFetchedAt: string | null
  rainStale: boolean
  reservoirsFetchedAt: string | null
  /** How many reservoirs are placed on the network, and why the others are not. */
  reservoirLinks: {
    byPosition: number
    byRiver: number
    unlinked: { id: string; name: string; reason: string; detail: string }[]
  }
  attribution: string
  /** Local test data is in the tables; never true in production. */
  seeded: boolean
}

export interface RiskDocument {
  estimate: boolean
  note: string
  rivers: RiverRisk[]
  meta: RiskMeta
}

export interface RiverGauge {
  id: string
  name: string
  basin: string
  reachFid: number
  unit: "m" | "ft"
  status: RiverStatus
  trend: string
  waterLevel: number | null
  alertLevel: number | null
  minorFloodLevel: number | null
  majorFloodLevel: number | null
  observedAt: string | null
  stale: boolean
  readings: { observedAt: string; waterLevel: number; rainfallMm: number | null }[]
}

export interface RiverForecast {
  gaugeId: string
  reachFid: number
  severity: FloodSeverity
  trend: FloodTrend
  issuedAt: string
  forecastStart: string | null
  forecastEnd: string | null
  stale: boolean
}

export interface AffectedStretch {
  reachFid: number
  reachName: string
  level: RiskLevel
  districts: RegionRef[]
  dsDivisions: RegionRef[]
}

export interface RiverRef {
  id: string
  name: string
  level: RiskLevel
}

export interface RiverDetail extends RiverRisk {
  estimate: boolean
  note: string
  parent: RiverRef | null
  tributaries: RiverRef[]
  gauges: RiverGauge[]
  forecasts: RiverForecast[]
  rain: RainSummary | null
  affected: AffectedStretch[]
  meta: RiskMeta
}

export interface NearbyRiverRisk {
  id: string
  name: string
  named: boolean
  level: RiskLevel
  reachName: string
  reason: string
  distanceKm: number
  estimate: boolean
}

export type BasinState = "no_forecast" | "no_flooding" | "above_normal" | "severe" | "extreme"

export interface BasinGauge {
  kind: "station" | "google"
  id: string
  name: string
  /** Empty when the gauge says nothing: out of date, or no thresholds. */
  state: BasinState | ""
  status: string
  trend: string
  waterLevel?: number
  unit?: string
  observedAt: string | null
  stale: boolean
  verified: boolean
}

export interface BasinArea {
  id: string
  kind: string
  state: BasinState
  lowerConfidence?: boolean
}

export interface BasinStatus {
  fid: number
  id: string
  name: string
  named: boolean
  state: BasinState
  areaKm2: number
  riverId: string | null
  bbox: [number, number, number, number]
  labelPoint: [number, number]
  gauges: BasinGauge[]
  areas: BasinArea[]
  onlyUnverified: boolean
  /** What verified evidence alone says: drawn while Extended coverage is off. */
  verifiedState?: BasinState
  observedAt: string | null
}

export interface SubbasinStatus {
  fid: number
  state: BasinState
  onlyUnverified: boolean
  verifiedState?: BasinState
}

export interface BasinsDocument {
  basins: BasinStatus[]
  subbasins: SubbasinStatus[]
  meta: {
    generatedAt: string
    attribution: string
    boundarySource: string
    subbasinSource: string
    boundaryTime: string | null
    gaugesObservedAt: string | null
    forecastIssuedAt: string | null
    forecastFetchedAt: string | null
    includesUnverified: boolean
    seeded: boolean
  }
}

export interface PastFlood {
  fid: number
  id: string
  name: string
  sourceName: string
  river: string
  year: number
  /** A day, when the source names one. */
  date: string | null
  magnitude: string | null
  areaKm2: number
  districts: string[]
  bbox: [number, number, number, number]
  service: string
}

export interface PastFloodsDocument {
  note: string
  events: PastFlood[]
  meta: {
    source: string
    attribution: string
    sourceTime: string | null
    fetchedAt: string | null
  }
}
