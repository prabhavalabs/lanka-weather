/** Mirrors the region endpoints of the Go service (ADR 0006, docs/api/weather-and-floods.md). */
import type { RegionLevel } from "../location/types"

export type { RegionLevel }

export interface RegionDetail {
  id: string
  level: RegionLevel
  name: string
  names: { en: string; si?: string | null; ta?: string | null }
  parentId: string | null
  hierarchy: { id: string; level: RegionLevel; name: string; nameSi?: string | null }[]
  childCount: number
  centroid: [number, number]
  labelPoint: [number, number]
  bbox: [number, number, number, number]
  areaKm2: number
  population: number | null
  populationYear: number | null
  source: { name: string; attribution: string; fetchedAt: string; sourceTime: string }
}

/** One region's weather from `/v1/weather/regions`. */
export interface RegionWeather {
  id: string
  name: string
  method: "cells" | "point"
  cells: number
  temperatureC: number | null
  temperatureMinC: number | null
  temperatureMaxC: number | null
  precipitationMm: number | null
  precipitationTodayMm: number | null
  windSpeedMaxKmh: number | null
  windGustsMaxKmh: number | null
}

export interface RegionWeatherResponse {
  level: RegionLevel
  parent?: string
  regions: RegionWeather[]
  validAt: string
  fetchedAt: string
  forecastFetchedAt: string | null
  source: string
  attribution: string
}
