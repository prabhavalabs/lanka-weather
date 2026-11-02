/** Mirrors `/v1/weather/current` (docs/api/weather-and-floods.md). */
import type { GridSpec } from "../location/types"

export type { GridSpec }

export type GridValueKey =
  | "temperatureC"
  | "apparentTemperatureC"
  | "relativeHumidityPct"
  | "dewPointC"
  | "precipitationMm"
  | "precipitationProbabilityPct"
  | "cloudCoverPct"
  | "pressureHpa"
  | "windSpeedKmh"
  | "windDirectionDeg"
  | "windGustsKmh"
  | "weatherCode"
  | "uvIndex"
  | "isDay"

/**
 * Current conditions on the 0.2° grid. The node in column i and row j lies
 * at lon0 + i × step, lat0 + j × step (rows south to north); its value is at
 * index j × nx + i of every array, null where the land mask leaves it out.
 */
export interface CurrentGrid {
  source: string
  model: string
  attribution: string
  validAt: string
  fetchedAt: string
  forecastFetchedAt: string | null
  grid: GridSpec
  units: Partial<Record<GridValueKey, string>>
  values: Partial<Record<GridValueKey, (number | null)[]>>
}
