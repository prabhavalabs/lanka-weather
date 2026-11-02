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
  "unknown",
  "no_flooding",
]

/**
 * Flood Hub's words for Google's severities. Its thresholds are named warning,
 * danger and extreme danger, and ABOVE_NORMAL, SEVERE and EXTREME are forecasts
 * against them. "Warning" here is a forecast, never an official warning.
 */
export const floodSeverityLabels: Record<FloodSeverity, string> = {
  extreme: "Extreme",
  severe: "Danger",
  above_normal: "Warning",
  no_flooding: "Normal",
  unknown: "No data",
}

/** Theme tokens by severity (src/styles/theme.css). */
export const floodSeverityTokens: Record<FloodSeverity, `--${string}`> = {
  extreme: "--flood-extreme",
  severe: "--flood-danger",
  above_normal: "--flood-warning",
  no_flooding: "--flood-normal",
  unknown: "--flood-nodata",
}

export const floodSeverityColors: Record<FloodSeverity, string> = {
  extreme: "var(--flood-extreme)",
  severe: "var(--flood-danger)",
  above_normal: "var(--flood-warning)",
  no_flooding: "var(--flood-normal)",
  unknown: "var(--flood-nodata)",
}

/** Badge variants (components/ui/badge.tsx) by severity. */
export const floodSeverityBadge = {
  extreme: "flood_extreme",
  severe: "flood_danger",
  above_normal: "flood_warning",
  no_flooding: "flood_normal",
  unknown: "flood_nodata",
} as const satisfies Record<FloodSeverity, string>

/** A mark beside the colour, so severity is not carried by colour alone. */
export const floodSeverityGlyphs: Record<FloodSeverity, string> = {
  extreme: "!!!",
  severe: "!!",
  above_normal: "!",
  no_flooding: "",
  unknown: "?",
}

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
  notification: "Riverine flood notification area",
  inundation_high: "Inundation: very high probability",
  inundation_medium: "Inundation: high probability",
  inundation_low: "Inundation: moderate probability",
  flash_likely: "Flash flood: likely",
  flash_highly_likely: "Flash flood: highly likely",
  significant_event: "Significant flood event",
}

/** Flood Hub's words for Google's three inundation levels, most probable first. */
export const inundationWords = {
  inundation_high: { label: "Very high", google: "high" },
  inundation_medium: { label: "High", google: "medium" },
  inundation_low: { label: "Moderate", google: "low" },
} as const

type AreaLike = {
  kind: FloodAreaKind
  severity: string
  details?: Record<string, unknown> | null
}

/**
 * What a flood area says of how likely it is. Google's inundation maps come in
 * two types: its levels mean probability, or else depth.
 */
export function areaLikelihood(area: AreaLike): { label: string; note?: string } {
  switch (area.kind) {
    case "flash_highly_likely":
      return { label: "Highly likely" }
    case "flash_likely":
      return { label: "Likely" }
    case "significant_event":
      return { label: "Event present" }
    case "notification":
      return {
        label: floodSeverityLabels[area.severity as FloodSeverity] ?? "Forecast",
        note: "Google's notification polygon for this gauge's forecast.",
      }
    default: {
      const words = inundationWords[area.kind]
      if (area.details?.mapType === "depth") {
        return {
          label: `${words.google[0]!.toUpperCase()}${words.google.slice(1)} depth`,
          note: "Google gives this map by depth rather than probability.",
        }
      }
      return {
        label: words.label,
        note: `Flood Hub's word for Google's "${words.google}" probability level.`,
      }
    }
  }
}

/** The badge for a flood area, whose severity is not always a gauge's. */
export function areaBadge(area: AreaLike) {
  switch (area.kind) {
    case "flash_highly_likely":
      return "flood_danger" as const
    case "flash_likely":
      return "flood_warning" as const
    case "significant_event":
      return "flood_danger" as const
    default:
      return floodSeverityBadge[area.severity as FloodSeverity] ?? ("chip" as const)
  }
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

/**
 * "10 gauges · none at warning or above" or "10 gauges · 2 at warning or above ·
 * 1 area", of what the map shows.
 */
export function floodSummary(collection: FloodForecastCollection): string {
  const gauges = collection.features.filter(isGauge)
  const counts = countBySeverity(gauges)
  const raised = counts.above_normal + counts.severe + counts.extreme
  const areas = collection.features.length - gauges.length
  const parts = [
    `${gauges.length} ${gauges.length === 1 ? "gauge" : "gauges"}`,
    raised === 0 ? "none at warning or above" : `${raised} at warning or above`,
  ]
  if (areas > 0) parts.push(`${areas} ${areas === 1 ? "area" : "areas"}`)
  return parts.join(" · ")
}
