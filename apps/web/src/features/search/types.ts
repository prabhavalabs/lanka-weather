/** Mirrors `/v1/places/search` (ADR 0006). */
import type { RegionLevel } from "../location/types"

export type PlaceKind =
  | "city"
  | "town"
  | "suburb"
  | "village"
  | "quarter"
  | "neighbourhood"
  | "hamlet"
  | "locality"
  | "isolated_dwelling"

export interface PlaceSearchResult {
  id: string
  type: "place" | "region"
  kind: PlaceKind | RegionLevel
  name: string
  names: { en: string; si?: string | null; ta?: string | null }
  matchedName: string
  lat: number
  lon: number
  bbox: [number, number, number, number] | null
  population: number | null
  hierarchy: { id: string; level: RegionLevel; name: string }[]
  score: number
}

export interface PlaceSearchResponse {
  query: string
  results: PlaceSearchResult[]
  attribution: string
}

/** What choosing a search result does, with what the map needs to show it. */
export type SearchChoice =
  | { type: "place"; id: string; name: string; kind: PlaceKind; lat: number; lon: number }
  | {
      type: "region"
      id: string
      name: string
      level: RegionLevel
      lat: number
      lon: number
      bbox: [number, number, number, number] | null
    }
  | { type: "station"; id: string; name: string; lat: number; lon: number }
