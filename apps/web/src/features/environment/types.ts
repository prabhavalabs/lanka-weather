/** Mirrors `/v1/air/current` and `/v1/sea/current` (docs/api/rainfall-stations-air-and-sea.md). */
import type { GridSpec } from "../location/types"

export interface EnvironmentGrid<Key extends string> {
  source: string
  model: string
  attribution: string
  validAt: string
  fetchedAt: string
  grid: GridSpec
  units: Partial<Record<Key, string>>
  values: Partial<Record<Key, (number | null)[]>>
  /** Sri Lanka's land, simplified, as closed longitude, latitude rings; for the sea only. */
  land?: [number, number][][]
  note: string
}

export type AirKey = "usAqi" | "pm25"
export type SeaKey =
  | "waveHeightM"
  | "waveDirectionDeg"
  | "wavePeriodS"
  | "swellHeightM"
  | "swellDirectionDeg"
  | "swellPeriodS"

export type AirGrid = EnvironmentGrid<AirKey>
export type SeaGrid = EnvironmentGrid<SeaKey>
