/**
 * How Google flood forecasts are named and coloured. Severities share the
 * river-level status colours, from calm to severe, so a map reader learns
 * one scale for both.
 */
import { formatDay } from "@/lib/format"
import type {
  FloodAreaKind,
  FloodForecastCollection,
  FloodGaugeFeature,
  FloodSeverity,
  FloodTrend,
} from "./types"

export const floodSeverityOrder: FloodSeverity[] = [
  "extreme",
  "severe",
  "above_normal",
  "no_flooding",
  "unknown",
]

export const floodSeverityLabels: Record<FloodSeverity, string> = {
  extreme: "Extreme",
  severe: "Severe",
  above_normal: "Above normal",
  no_flooding: "No flooding",
  unknown: "Unknown",
}

/** Theme tokens by severity (src/styles/theme.css). */
export const floodSeverityTokens: Record<FloodSeverity, `--${string}`> = {
  extreme: "--status-major-flood",
  severe: "--status-minor-flood",
  above_normal: "--status-alert",
  no_flooding: "--status-normal",
  unknown: "--status-unknown",
}

export const floodSeverityColors: Record<FloodSeverity, string> = {
  extreme: "var(--status-major-flood)",
  severe: "var(--status-minor-flood)",
  above_normal: "var(--status-alert)",
  no_flooding: "var(--status-normal)",
  unknown: "var(--status-unknown)",
}

/** Badge variants (components/ui/badge.tsx) by severity. */
export const floodSeverityBadge = {
  extreme: "major_flood",
  severe: "minor_flood",
  above_normal: "alert",
  no_flooding: "normal",
  unknown: "unknown",
} as const satisfies Record<FloodSeverity, string>

export const floodTrendLabels: Record<FloodTrend, string> = {
  rise: "Rising",
  fall: "Falling",
  no_change: "Steady",
  unknown: "Trend unknown",
}

export const floodTrendArrows: Record<FloodTrend, string> = {
  rise: "↑",
  fall: "↓",
  no_change: "→",
  unknown: "",
}

export const floodAreaLabels: Record<FloodAreaKind, string> = {
  notification: "Flood notification area",
  inundation_high: "Inundation likely (high)",
  inundation_medium: "Inundation possible (medium)",
  inundation_low: "Inundation possible (low)",
  flash_likely: "Flash flood likely",
  flash_highly_likely: "Flash flood highly likely",
  significant_event: "Significant flood event",
}

/** A gauge's name: its site or river, else its short id ("Gauge 4121155030"). */
export function gaugeName(gauge: { siteName?: string; river?: string; gaugeId: string }): string {
  if (gauge.siteName) return gauge.siteName
  if (gauge.river) return `${gauge.river} gauge`
  const tail = gauge.gaugeId.replace(/^hybas_/i, "")
  return `Gauge ${tail}`
}

/** "Sat 7 Nov → Sun 8 Nov", or one day when the window is a day or less. */
export function formatWindow(start: string | null | undefined, end: string | null | undefined) {
  if (!start) return "–"
  if (!end) return formatDay(start)
  const hours = (Date.parse(end) - Date.parse(start)) / 3_600_000
  // Forecast windows are whole UTC days; Sri Lanka's day starts 5.5 h later.
  return hours <= 24 ? formatDay(start) : `${formatDay(start)} → ${formatDay(end)}`
}

export function isGauge(
  feature: FloodForecastCollection["features"][number],
): feature is FloodGaugeFeature {
  return feature.properties.kind === "gauge"
}

/** Gauges per severity. */
export function countBySeverity(gauges: { properties: { severity: FloodSeverity } }[]) {
  const counts: Record<FloodSeverity, number> = {
    extreme: 0,
    severe: 0,
    above_normal: 0,
    no_flooding: 0,
    unknown: 0,
  }
  for (const gauge of gauges) counts[gauge.properties.severity] += 1
  return counts
}

/** "10 gauges · none above normal" or "10 gauges · 2 above normal · 1 area". */
export function floodSummary(collection: FloodForecastCollection): string {
  const gauges = collection.features.filter(isGauge)
  const counts = countBySeverity(gauges)
  const raised = counts.above_normal + counts.severe + counts.extreme
  const areas = collection.features.length - gauges.length
  const parts = [
    `${gauges.length} ${gauges.length === 1 ? "gauge" : "gauges"}`,
    raised === 0 ? "none above normal" : `${raised} above normal`,
  ]
  if (areas > 0) parts.push(`${areas} ${areas === 1 ? "area" : "areas"}`)
  return parts.join(" · ")
}
