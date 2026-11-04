/** Mirrors internal/reservoirs in the Go service (docs/api/reservoirs.md). */
import type { MultiPolygon, Polygon } from "geojson"

export type ReservoirStatus = "spilling" | "releasing" | "holding" | "nodata"

/** Water leaving a reservoir. A null flow means unknown, never zero. */
export interface ReservoirFlow {
  /** spill, outlets or downstream, as the source writes it. */
  kind: string
  cusecs: number | null
  agency: string | null
  day?: string | null
}

export interface ReservoirProperties {
  id: string
  name: string
  district: string | null
  river: string | null
  type: string
  status: ReservoirStatus
  /** The last figure the source has: `storageFresh` says whether it is current. */
  storagePercent: number | null
  storageFresh: boolean | null
  flow: ReservoirFlow | null
  /** The day in Sri Lanka the figures were read (YYYY-MM-DD). */
  readingDay: string | null
  agency: string | null
  source: string
  sourceUrl: string | null
  fetchedAt: string
  stale: boolean
  mapped: boolean
  osmName: string | null
  osmKind: "water" | "dam" | null
  /** Other spellings of the name, for search. */
  altNames: string[] | null
}

export interface ReservoirFeature {
  type: "Feature"
  id: string
  /** Null when the reservoir has no known position: listed, never guessed on the map. */
  geometry: { type: "Point"; coordinates: [number, number] } | null
  properties: ReservoirProperties
}

export interface ReservoirCounts {
  total: number
  mapped: number
  spilling: number
  releasing: number
  holding: number
  nodata: number
}

export interface ReservoirsCollection {
  type: "FeatureCollection"
  features: ReservoirFeature[]
  meta: {
    source: string
    mapSource: string
    attribution: string
    note: string
    fetchedAt: string | null
    sourceTime: string | null
    mapTime: string | null
    generatedAt: string
    counts: ReservoirCounts
  }
}

export interface ReservoirAreas {
  type: "FeatureCollection"
  features: {
    type: "Feature"
    id: string
    geometry: Polygon | MultiPolygon
    properties: { id: string; name: string; status: ReservoirStatus; stale: boolean }
  }[]
}

export interface ReservoirReading {
  day: string
  status: ReservoirStatus
  storagePercent: number | null
  storageFresh: boolean | null
  flow: ReservoirFlow | null
  agency: string | null
}

export interface ReservoirHistory {
  reservoir: ReservoirProperties
  note: string
  readings: ReservoirReading[]
}
