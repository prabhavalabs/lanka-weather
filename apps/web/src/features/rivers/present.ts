import type { RiverStatus, RiverTrend, StationProperties } from "./types"

/** Colours by status, from calm to severe; chosen to stay distinct for common colour-vision deficiencies. */
export const statusColors: Record<RiverStatus, string> = {
  normal: "#0ea5e9",
  alert: "#facc15",
  minor_flood: "#f97316",
  major_flood: "#dc2626",
  unknown: "#94a3b8",
}

export const statusLabels: Record<RiverStatus, string> = {
  normal: "Normal",
  alert: "Alert",
  minor_flood: "Minor flood",
  major_flood: "Major flood",
  unknown: "No thresholds",
}

/** Display order for legends and summaries, most severe first. */
export const statusOrder: RiverStatus[] = [
  "major_flood",
  "minor_flood",
  "alert",
  "normal",
  "unknown",
]

export const trendLabels: Record<RiverTrend, string> = {
  rising: "Rising",
  falling: "Falling",
  steady: "Steady",
  unknown: "Trend unknown",
}

export const trendArrows: Record<RiverTrend, string> = {
  rising: "↑",
  falling: "↓",
  steady: "→",
  unknown: "",
}

/** A level with its unit, to two decimals for metres and one for feet. */
export function formatLevel(value: number | null, unit: "m" | "ft"): string {
  if (value === null) return "–"
  return `${value.toFixed(unit === "ft" ? 1 : 2)} ${unit}`
}

/** A rate of change per hour with an explicit sign. */
export function formatRate(value: number | null, unit: "m" | "ft"): string {
  if (value === null) return "–"
  const sign = value > 0 ? "+" : value < 0 ? "−" : "±"
  return `${sign}${Math.abs(value).toFixed(2)} ${unit}/h`
}

/** How long ago a time was, in words: "just now", "12 min ago", "3 h ago", "2 days ago". */
export function formatAge(iso: string | null, now: Date = new Date()): string {
  if (!iso) return "never"
  const minutes = Math.round((now.getTime() - new Date(iso).getTime()) / 60_000)
  if (minutes < 1) return "just now"
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.round(minutes / 60)
  if (hours < 48) return `${hours} h ago`
  return `${Math.round(hours / 24)} days ago`
}

/**
 * How far the level is towards the next threshold it has not reached, from 0
 * (at the previous threshold, or the river bed for the first) to 1. Null when
 * the station has no thresholds or is above major flood.
 */
export function headroom(
  station: Pick<
    StationProperties,
    "waterLevel" | "alertLevel" | "minorFloodLevel" | "majorFloodLevel"
  >,
) {
  const { waterLevel, alertLevel, minorFloodLevel, majorFloodLevel } = station
  if (
    waterLevel === null ||
    alertLevel === null ||
    minorFloodLevel === null ||
    majorFloodLevel === null
  ) {
    return null
  }
  const steps: [number, number, string][] = [
    [0, alertLevel, "alert"],
    [alertLevel, minorFloodLevel, "minor flood"],
    [minorFloodLevel, majorFloodLevel, "major flood"],
  ]
  for (const [from, to, next] of steps) {
    if (waterLevel < to) {
      return {
        next,
        remaining: to - waterLevel,
        fraction: Math.max(0, (waterLevel - from) / (to - from)),
      }
    }
  }
  return null
}

/** Count of stations per status. */
export function countByStatus(stations: Pick<StationProperties, "status">[]) {
  const counts: Record<RiverStatus, number> = {
    normal: 0,
    alert: 0,
    minor_flood: 0,
    major_flood: 0,
    unknown: 0,
  }
  for (const station of stations) counts[station.status] += 1
  return counts
}
