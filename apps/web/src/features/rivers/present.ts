import { formatClock } from "@/lib/format"
import { translate, type MessageKey } from "@/lib/i18n"
import type { LevelUnit, NextHour, RiverStatus, RiverTrend, StationProperties } from "./types"

export { formatAge } from "@/lib/format"

/**
 * A record of labels read in the current language each time a value is
 * looked up, so `labels[status]` stays a plain lookup at every call site.
 * The component that shows it must subscribe with `useT()` or `useLocale()`.
 */
export function liveLabels<K extends string>(
  keys: readonly K[],
  key: (name: K) => MessageKey,
): Readonly<Record<K, string>> {
  const record = {} as Record<K, string>
  for (const name of keys) {
    Object.defineProperty(record, name, { enumerable: true, get: () => translate(key(name)) })
  }
  return record
}

/**
 * Theme tokens by status, from calm to severe (src/styles/theme.css): blue,
 * yellow, orange, red and grey, distinct in lightness as well as hue so they
 * stay apart for common colour-vision deficiencies. Each has a matching
 * `-foreground` token for text on it.
 */
export const statusTokens: Record<RiverStatus, `--${string}`> = {
  normal: "--status-normal",
  alert: "--status-alert",
  minor_flood: "--status-minor-flood",
  major_flood: "--status-major-flood",
  unknown: "--status-unknown",
}

/** CSS colours by status, for styles and SVG; they follow the theme. */
export const statusColors: Record<RiverStatus, string> = {
  normal: "var(--status-normal)",
  alert: "var(--status-alert)",
  minor_flood: "var(--status-minor-flood)",
  major_flood: "var(--status-major-flood)",
  unknown: "var(--status-unknown)",
}

/** The Irrigation Department's status words, in the current language. */
export const statusLabels = liveLabels<RiverStatus>(
  ["normal", "alert", "minor_flood", "major_flood", "unknown"],
  (status) => `terms.riverStatus.${status}`,
)

/** The thresholds a gauge publishes, lowest first. */
export type Threshold = "alert" | "minor_flood" | "major_flood"

/** "Alert level", "Minor flood level", "Major flood level", in the current language. */
export const thresholdLabels = liveLabels<Threshold>(
  ["alert", "minor_flood", "major_flood"],
  (threshold) => `terms.riverLevel.${threshold}`,
)

/** Display order for legends and summaries, most severe first. */
export const statusOrder: RiverStatus[] = [
  "major_flood",
  "minor_flood",
  "alert",
  "normal",
  "unknown",
]

export const trendLabels = liveLabels<RiverTrend>(
  ["rising", "falling", "steady", "unknown"],
  (trend) => `terms.trend.${trend}`,
)

export const trendArrows: Record<RiverTrend, string> = {
  rising: "↑",
  falling: "↓",
  steady: "→",
  unknown: "",
}

/** A level alone, to two decimals for metres and one for feet. */
export function levelNumber(value: number, unit: LevelUnit): string {
  return value.toFixed(unit === "ft" ? 1 : 2)
}

/** A level with its unit, or with none when the agency publishes none. */
export function formatLevel(value: number | null, unit: LevelUnit): string {
  if (value === null) return "–"
  const number = levelNumber(value, unit)
  return unit === "unknown" ? number : `${number} ${unit}`
}

/** A rate of change per hour with an explicit sign. */
export function formatRate(value: number | null, unit: LevelUnit): string {
  if (value === null) return "–"
  const sign = value > 0 ? "+" : value < 0 ? "−" : "±"
  const number = `${sign}${Math.abs(value).toFixed(2)}`
  return unit === "unknown" ? `${number}/h` : `${number} ${unit}/h`
}

/** How old a next-hour forecast may be and still be shown. */
export const nextHourMaxAgeMs = 2 * 60 * 60 * 1000

/**
 * The experimental next-hour forecast as one line, or null when there is none
 * or it is too old, for example from a page left open or a persisted cache.
 * Worded so it cannot be read as the station's official status.
 */
export function nextHourLine(
  forecast: NextHour | null,
  now: Date = new Date(),
): { value: string; issued: string } | null {
  if (!forecast) return null
  const age = now.getTime() - new Date(forecast.issuedAt).getTime()
  if (!Number.isFinite(age) || age > nextHourMaxAgeMs) return null
  const level = formatLevel(forecast.level, forecast.unit)
  const value =
    forecast.low !== null && forecast.high !== null
      ? translate("rivers.nextHour.range", {
          level,
          low: levelNumber(forecast.low, forecast.unit),
          high: levelNumber(forecast.high, forecast.unit),
        })
      : level
  return {
    value,
    issued: translate("rivers.nextHour.issued", { time: formatClock(forecast.issuedAt) }),
  }
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
  const steps: [number, number, Threshold][] = [
    [0, alertLevel, "alert"],
    [alertLevel, minorFloodLevel, "minor_flood"],
    [minorFloodLevel, majorFloodLevel, "major_flood"],
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

/**
 * A gauge's published thresholds as chart lines and rows, lowest first,
 * named in the current language.
 */
export function stationThresholds(
  station: Pick<StationProperties, "alertLevel" | "minorFloodLevel" | "majorFloodLevel">,
) {
  const levels: [Threshold, number | null][] = [
    ["alert", station.alertLevel],
    ["minor_flood", station.minorFloodLevel],
    ["major_flood", station.majorFloodLevel],
  ]
  return levels.flatMap(([threshold, value]) =>
    value === null
      ? []
      : [{ label: thresholdLabels[threshold], value, color: statusColors[threshold] }],
  )
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
