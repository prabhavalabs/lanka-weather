import {
  cloudScale,
  humidityScale,
  rainRateScale,
  temperatureScale,
  windScale,
  type ScaleStep,
} from "@/lib/tokens"
import type { GridValueKey } from "./types"

/** What the weather field shows. */
export type GridVariable = "temperature" | "precipitation" | "humidity" | "cloud" | "wind"

export interface GridVariableSpec {
  label: string
  /** Short label for a segmented control. */
  short: string
  key: GridValueKey
  unit: string
  scale: ScaleStep[]
  /** Where an open first step ("below 21") starts, below the next bound. */
  openSpan: number
  /** Values under the first step's bound fade to transparent (rain, cloud). */
  fadeBelow: boolean
  decimals: number
}

export const gridVariables: Record<GridVariable, GridVariableSpec> = {
  temperature: {
    label: "Temperature",
    short: "Temp",
    key: "temperatureC",
    unit: "°C",
    scale: temperatureScale,
    openSpan: 3,
    fadeBelow: false,
    decimals: 1,
  },
  precipitation: {
    label: "Rain, this hour",
    short: "Rain",
    key: "precipitationMm",
    unit: "mm",
    scale: rainRateScale,
    openSpan: 0,
    fadeBelow: true,
    decimals: 1,
  },
  humidity: {
    label: "Relative humidity",
    short: "Humid",
    key: "relativeHumidityPct",
    unit: "%",
    scale: humidityScale,
    openSpan: 10,
    fadeBelow: false,
    decimals: 0,
  },
  cloud: {
    label: "Cloud cover",
    short: "Cloud",
    key: "cloudCoverPct",
    unit: "%",
    scale: cloudScale,
    openSpan: 0,
    fadeBelow: false,
    decimals: 0,
  },
  wind: {
    label: "Wind speed",
    short: "Wind",
    key: "windSpeedKmh",
    unit: "km/h",
    scale: windScale,
    openSpan: 5,
    fadeBelow: false,
    decimals: 0,
  },
}

export const gridVariableOrder: GridVariable[] = [
  "temperature",
  "precipitation",
  "humidity",
  "cloud",
  "wind",
]
