/**
 * How river risk is named, ranked and coloured. Pure, so it is tested without
 * a browser. Risk levels share the status colours: a river reader learns one
 * scale for gauges, forecasts and stretches of river.
 */
import { formatAge } from "@/lib/format"
import type { RiskDocument, RiskLevel, RiverRisk } from "./types"

export const riskOrder: RiskLevel[] = ["severe", "high", "watch", "normal", "unknown"]

const rank: Record<RiskLevel, number> = { unknown: 0, normal: 1, watch: 2, high: 3, severe: 4 }

export function atRisk(level: RiskLevel): boolean {
  return rank[level] >= rank.watch
}

export const riskLabels: Record<RiskLevel, string> = {
  severe: "Severe",
  high: "High",
  watch: "Watch",
  normal: "Normal",
  unknown: "Not known",
}

/** What a level means, for legends and titles. */
export const riskDescriptions: Record<RiskLevel, string> = {
  severe: "Likely overflowing",
  high: "Overflow risk",
  watch: "Above or nearing alert",
  normal: "Measured, below alert",
  unknown: "No current gauge",
}

/** Theme tokens by level (src/styles/theme.css). */
export const riskTokens: Record<RiskLevel, `--${string}`> = {
  severe: "--status-major-flood",
  high: "--status-minor-flood",
  watch: "--status-alert",
  normal: "--status-normal",
  unknown: "--river-unassessed",
}

export const riskColors: Record<RiskLevel, string> = {
  severe: "var(--status-major-flood)",
  high: "var(--status-minor-flood)",
  watch: "var(--status-alert)",
  normal: "var(--status-normal)",
  unknown: "var(--river-unassessed)",
}

/** Badge variants (components/ui/badge.tsx) by level. */
export const riskBadge = {
  severe: "major_flood",
  high: "minor_flood",
  watch: "alert",
  normal: "normal",
  unknown: "unknown",
} as const satisfies Record<RiskLevel, string>

/** What the map adds to a line's width, by level, so risk reads without colour. */
export const riskExtraWidth: Record<RiskLevel, number> = {
  severe: 3,
  high: 2,
  watch: 1,
  normal: 0,
  unknown: 0,
}

/** Named rivers at risk, worst first, largest first within a level. */
export function riversAtRisk(rivers: RiverRisk[]): RiverRisk[] {
  return rivers
    .filter((river) => atRisk(river.level))
    .sort((a, b) => rank[b.level] - rank[a.level] || b.uplandKm2 - a.uplandKm2)
}

/** Stretch ids by level, for the map's expressions. */
export function reachIdsByLevel(rivers: RiverRisk[]): Record<RiskLevel, number[]> {
  const ids: Record<RiskLevel, number[]> = {
    severe: [],
    high: [],
    watch: [],
    normal: [],
    unknown: [],
  }
  for (const river of rivers) for (const reach of river.reaches) ids[reach.level].push(reach.fid)
  return ids
}

/** "3 of 5 stretches have a gauge", or that the whole river has none. */
export function coverageText(river: Pick<RiverRisk, "gaugedReaches" | "reaches">): string {
  const total = river.reaches.length
  if (river.gaugedReaches === 0) return "No gauge on this river"
  if (river.gaugedReaches === total) return total === 1 ? "Gauged" : "Every stretch has a gauge"
  return `${river.gaugedReaches} of ${total} stretches have a gauge`
}

/**
 * The calm or alarmed line above the ranked list. It never promises safety:
 * it says what gauges read, how many of them count, and how old they are.
 */
export function riskHeadline(document: RiskDocument, now = new Date()) {
  const { gaugesCurrent, gaugesTotal, gaugesObservedAt } = document.meta
  const count = riversAtRisk(document.rivers).length
  const title =
    count === 0
      ? "No river is above its alert level"
      : count === 1
        ? "1 river at risk"
        : `${count} rivers at risk`
  const parts = [`${gaugesCurrent} of ${gaugesTotal} gauges current`]
  if (gaugesObservedAt) parts.push(`read ${formatAge(gaugesObservedAt, now)}`)
  return { title, detail: parts.join(" · "), calm: count === 0 }
}

/** Rain as a whole number of millimetres. */
export function formatRain(value: number): string {
  return `${Math.round(value)} mm`
}

/** A river's other names, for search and the panel. */
export function riverSearchNames(river: Pick<RiverRisk, "name" | "nameSi" | "aliases">): string[] {
  return [river.name, ...(river.nameSi ? [river.nameSi] : []), ...river.aliases]
}
