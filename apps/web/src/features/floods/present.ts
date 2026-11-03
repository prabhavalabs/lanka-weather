/**
 * How Google flood forecasts are named and coloured. Severities share the
 * river-level status colours, from calm to severe, so a map reader learns
 * one scale for both.
 */
import { translate } from "@/lib/i18n"
import { formatDay } from "@/lib/format"
import type {
  FloodAreaKind,
  FloodForecastCollection,
  FloodGaugeFeature,
  FloodSeverity,
  FloodTrend,
} from "./types"

/**
 * A record whose values are read in the current language each time they are
 * looked up, so callers can keep indexing it while a subscribed component renders.
 */
export function labelRecord<K extends string>(
  keys: readonly K[],
  label: (key: K) => string,
): Readonly<Record<K, string>> {
  const record = {} as Record<K, string>
  for (const key of keys) {
    Object.defineProperty(record, key, { enumerable: true, get: () => label(key) })
  }
  return record
}

export const floodSeverityOrder: FloodSeverity[] = [
  "extreme",
  "severe",
  "above_normal",
  "unknown",
  "no_flooding",
]

function isSeverity(value: string): value is FloodSeverity {
  return (floodSeverityOrder as string[]).includes(value)
}

/**
 * Flood Hub's words for Google's severities, in the current language. Its
 * thresholds are named warning, danger and extreme danger, and ABOVE_NORMAL,
 * SEVERE and EXTREME are forecasts against them. "Warning" here is a forecast,
 * never an official warning.
 */
export function floodSeverityLabel(severity: FloodSeverity): string {
  return translate(`floods.severity.${severity}`)
}

/** `floodSeverityLabel` as a record, read in the current language. */
export const floodSeverityLabels = labelRecord(floodSeverityOrder, floodSeverityLabel)

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

const trendTerms = {
  rise: "terms.trend.rising",
  fall: "terms.trend.falling",
  no_change: "terms.trend.steady",
  unknown: "terms.trend.unknown",
} as const satisfies Record<FloodTrend, string>

/** A forecast's trend, in the shared official words. */
export function floodTrendLabel(trend: FloodTrend): string {
  return translate(trendTerms[trend] ?? trendTerms.unknown)
}

/** `floodTrendLabel` as a record, read in the current language. */
export const floodTrendLabels = labelRecord(
  Object.keys(trendTerms) as FloodTrend[],
  floodTrendLabel,
)

export const floodTrendArrows: Record<FloodTrend, string> = {
  rise: "↑",
  fall: "↓",
  no_change: "→",
  unknown: "",
}

const floodAreaKinds: FloodAreaKind[] = [
  "notification",
  "inundation_high",
  "inundation_medium",
  "inundation_low",
  "flash_likely",
  "flash_highly_likely",
  "significant_event",
]

/** A flood area's kind in full, e.g. "Flash flood: likely"; "Flood area" for one unknown. */
export function floodAreaLabel(kind: string): string {
  return (floodAreaKinds as string[]).includes(kind)
    ? translate(`floods.area.${kind as FloodAreaKind}`)
    : translate("floods.area.fallback")
}

/** `floodAreaLabel` as a record, read in the current language. */
export const floodAreaLabels = labelRecord(floodAreaKinds, floodAreaLabel)

/** Google's own names for its three inundation levels, most probable first. */
export const inundationWords = {
  inundation_high: { google: "high" },
  inundation_medium: { google: "medium" },
  inundation_low: { google: "low" },
} as const

export type InundationKind = keyof typeof inundationWords

/** Flood Hub's word for an inundation level: very high, high or moderate. */
export function inundationLabel(kind: InundationKind): string {
  return translate(`floods.inundation.${kind}`)
}

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
      return { label: translate("floods.likelihood.highlyLikely") }
    case "flash_likely":
      return { label: translate("floods.likelihood.likely") }
    case "significant_event":
      return { label: translate("floods.likelihood.eventPresent") }
    case "notification":
      return {
        label: isSeverity(area.severity)
          ? floodSeverityLabel(area.severity)
          : translate("floods.likelihood.forecast"),
        note: translate("floods.likelihood.notificationNote"),
      }
    default: {
      const kind = area.kind
      if (!(kind in inundationWords)) return { label: translate("floods.likelihood.forecast") }
      if (area.details?.mapType === "depth") {
        return {
          label: translate(`floods.likelihood.depth.${kind}`),
          note: translate("floods.likelihood.depthNote"),
        }
      }
      return {
        label: inundationLabel(kind),
        note: translate("floods.likelihood.probabilityNote", {
          google: inundationWords[kind].google,
        }),
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
  if (gauge.river) return translate("floods.gaugeName.river", { river: gauge.river })
  const tail = gauge.gaugeId.replace(/^hybas_/i, "")
  return translate("floods.gaugeName.id", { id: tail })
}

const magnitudes = ["critical", "major", "minor"] as const
type Magnitude = (typeof magnitudes)[number]

function isMagnitude(value: string | null): value is Magnitude {
  return value !== null && (magnitudes as readonly string[]).includes(value)
}

/** How a past flood was mapped (critical, major, minor), as a label; else as published. */
export function pastFloodMagnitude(magnitude: string): string {
  const value = magnitude.toLowerCase()
  if (isMagnitude(value)) return translate(`floods.past.magnitudeLabel.${value}`)
  return magnitude[0]!.toUpperCase() + magnitude.slice(1)
}

/**
 * A past flood's name in the current language, "Kelani flood, 2016 (major)", from
 * its river (a proper name, as published) and year. Falls back to the server's name.
 */
export function pastFloodName(event: {
  name: string
  river: string
  year: number
  magnitude: string | null
}): string {
  if (!event.river || !event.year) return event.name
  const values = { river: event.river, year: String(event.year) }
  const magnitude = event.magnitude?.toLowerCase() ?? null
  if (isMagnitude(magnitude)) {
    return translate("floods.past.nameWithMagnitude", {
      ...values,
      magnitude: translate(`floods.past.magnitude.${magnitude}`),
    })
  }
  return translate("floods.past.name", values)
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
 * "10 gauges · none raised" or "10 gauges · 2 raised · 1 area", of what the map
 * shows, in the current language. Raised means at warning or above.
 */
export function floodSummary(collection: FloodForecastCollection): string {
  const gauges = collection.features.filter(isGauge)
  const counts = countBySeverity(gauges)
  const raised = counts.above_normal + counts.severe + counts.extreme
  const areas = collection.features.length - gauges.length
  const parts = [
    translate("floods.summary.gauges", { count: gauges.length }),
    raised === 0
      ? translate("floods.summary.noneRaised")
      : translate("floods.summary.raised", { count: raised }),
  ]
  if (areas > 0) parts.push(translate("floods.summary.areas", { count: areas }))
  return parts.join(translate("common.separator"))
}
