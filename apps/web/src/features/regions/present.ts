/**
 * Region codes, levels and the choropleth's variables. Codes are the
 * Department of Census and Statistics' (ADR 0006): one digit for a
 * province, two for a district, four for a DS division and seven for a GN
 * division, each starting with its parent's digits.
 */
import { rainfallScale, temperatureScale, windScale, type ScaleStep } from "@/lib/tokens"
import type { RegionLevel, RegionWeather } from "./types"

export const regionLevels: RegionLevel[] = ["province", "district", "ds", "gn"]

const digitsByLevel: Record<RegionLevel, number> = { province: 1, district: 2, ds: 4, gn: 7 }

/** Lowest zoom at which the vector tiles carry each level (ADR 0006). */
export const levelMinZoom: Record<RegionLevel, number> = { province: 0, district: 6, ds: 8, gn: 10 }

function digits(id: string): string {
  return id.toUpperCase().replace(/^LK/, "").replace(/-/g, "")
}

/** The level of a region code, or null if it is not one. */
export function regionLevelOf(id: string): RegionLevel | null {
  const length = digits(id).length
  return regionLevels.find((level) => digitsByLevel[level] === length) ?? null
}

/** A region's numeric feature id in the vector tiles: "LK-1103005" is 1103005. */
export function regionFid(id: string): number {
  return Number(digits(id))
}

/** The code of the region containing `id` at a coarser level: ("LK-1103005", "ds") is "LK-1103". */
export function ancestorId(id: string, level: RegionLevel): string | null {
  const code = digits(id)
  const length = digitsByLevel[level]
  if (code.length <= length) return null
  return `LK-${code.slice(0, length)}`
}

/** What the choropleth colours regions by. */
export type RegionVariable = "temperature" | "rain" | "wind"

export interface RegionVariableSpec {
  label: string
  /** Short label for a segmented control. */
  short: string
  unit: string
  scale: ScaleStep[]
  value: (region: RegionWeather) => number | null
  /** Decimals when shown. */
  decimals: number
}

export const regionVariables: Record<RegionVariable, RegionVariableSpec> = {
  temperature: {
    label: "Temperature now",
    short: "Temp",
    unit: "°C",
    scale: temperatureScale,
    value: (region) => region.temperatureC,
    decimals: 1,
  },
  rain: {
    label: "Rain today",
    short: "Rain",
    unit: "mm",
    scale: rainfallScale,
    value: (region) => region.precipitationTodayMm,
    decimals: 1,
  },
  wind: {
    label: "Strongest wind now",
    short: "Wind",
    unit: "km/h",
    scale: windScale,
    value: (region) => region.windSpeedMaxKmh,
    decimals: 0,
  },
}

/**
 * Which GN divisions' weather to load: those of the districts the map shows.
 * A district's GN divisions are about 130 KB (11 KB gzipped); the whole
 * country's are 3.2 MB, so they are loaded district by district as the map
 * moves, and only at the zooms that draw GN divisions.
 */
export function districtsOf(features: { properties?: Record<string, unknown> | null }[]): string[] {
  const ids = new Set<string>()
  for (const feature of features) {
    const district = feature.properties?.districtId
    if (typeof district === "string" && regionLevelOf(district) === "district") ids.add(district)
  }
  return [...ids].sort()
}
