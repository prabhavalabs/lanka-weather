import {
  cloudScale,
  humidityScale,
  rainRateScale,
  temperatureScale,
  windScale,
  type ScaleStep,
} from "@/lib/tokens"
import { translate } from "@/lib/i18n"
import type { GridValueKey } from "./types"

/** What the weather field shows. */
export type GridVariable = "temperature" | "precipitation" | "humidity" | "cloud" | "wind"

export interface GridVariableSpec {
  /** In the current language: read it while a subscribed component renders. */
  readonly label: string
  /** Short label for a segmented control, in the current language. */
  readonly short: string
  key: GridValueKey
  unit: string
  scale: ScaleStep[]
  /** Where an open first step ("below 21") starts, below the next bound. */
  openSpan: number
  /** Values under the first step's bound fade to transparent (rain, cloud). */
  fadeBelow: boolean
  decimals: number
}

/** Labels are getters, so they read the current language each time. */
export const gridVariables: Record<GridVariable, GridVariableSpec> = {
  temperature: {
    get label() {
      return translate("grid.variable.temperature.label")
    },
    get short() {
      return translate("grid.variable.temperature.short")
    },
    key: "temperatureC",
    unit: "°C",
    scale: temperatureScale,
    openSpan: 3,
    fadeBelow: false,
    decimals: 1,
  },
  precipitation: {
    get label() {
      return translate("grid.variable.precipitation.label")
    },
    get short() {
      return translate("grid.variable.precipitation.short")
    },
    key: "precipitationMm",
    unit: "mm",
    scale: rainRateScale,
    openSpan: 0,
    fadeBelow: true,
    decimals: 1,
  },
  humidity: {
    get label() {
      return translate("grid.variable.humidity.label")
    },
    get short() {
      return translate("grid.variable.humidity.short")
    },
    key: "relativeHumidityPct",
    unit: "%",
    scale: humidityScale,
    openSpan: 10,
    fadeBelow: false,
    decimals: 0,
  },
  cloud: {
    get label() {
      return translate("grid.variable.cloud.label")
    },
    get short() {
      return translate("grid.variable.cloud.short")
    },
    key: "cloudCoverPct",
    unit: "%",
    scale: cloudScale,
    openSpan: 0,
    fadeBelow: false,
    decimals: 0,
  },
  wind: {
    get label() {
      return translate("grid.variable.wind.label")
    },
    get short() {
      return translate("grid.variable.wind.short")
    },
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
