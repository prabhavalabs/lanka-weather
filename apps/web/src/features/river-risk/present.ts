/**
 * How river risk is named, ranked and coloured. Pure, so it is tested without
 * a browser. Risk levels share the status colours: a river reader learns one
 * scale for gauges, forecasts and stretches of river.
 */
import { formatAge } from "@/lib/format"
import {
  getLocale,
  localName,
  translate,
  translateIn,
  type Locale,
  type MessageKey,
} from "@/lib/i18n"
import { liveLabels, type Threshold } from "../rivers/present"
import type { RiskDocument, RiskLevel, RiskReason, RiverRisk } from "./types"

export const riskOrder: RiskLevel[] = ["severe", "high", "watch", "normal", "unknown"]

const rank: Record<RiskLevel, number> = { unknown: 0, normal: 1, watch: 2, high: 3, severe: 4 }

export function atRisk(level: RiskLevel): boolean {
  return rank[level] >= rank.watch
}

/**
 * Our estimated levels, in the current language. They are worded as risk,
 * not as the Irrigation Department's gauge statuses, and are always shown
 * marked as an estimate.
 */
export const riskLabels = liveLabels<RiskLevel>(riskOrder, (level) => `rivers.risk.level.${level}`)

/** What a level means, for legends and titles. */
export const riskDescriptions = liveLabels<RiskLevel>(
  riskOrder,
  (level) => `rivers.risk.meaning.${level}`,
)

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
  if (river.gaugedReaches === 0) return translate("rivers.risk.coverage.none")
  if (river.gaugedReaches === total) {
    return translate(total === 1 ? "rivers.risk.coverage.whole" : "rivers.risk.coverage.every")
  }
  return translate("rivers.risk.coverage.some", { gauged: river.gaugedReaches, total })
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
      ? translate("rivers.risk.headline.calm")
      : translate("rivers.risk.headline.count", { count })
  const parts = [
    translate("rivers.risk.headline.gauges", { current: gaugesCurrent, total: gaugesTotal }),
  ]
  if (gaugesObservedAt) {
    parts.push(translate("common.readAgo", { age: formatAge(gaugesObservedAt, now) }))
  }
  return { title, detail: parts.join(translate("common.separator")), calm: count === 0 }
}

/** Rain as a whole number of millimetres. */
export function formatRain(value: number): string {
  return `${Math.round(value)} mm`
}

/** A river's name in the language: Sinhala where the data has it, else English. */
export function riverName(river: Pick<RiverRisk, "name" | "nameSi">, locale = getLocale()) {
  return localName({ name: river.name, nameSi: river.nameSi }, locale)
}

/** A river's other names, for search and the panel. */
export function riverSearchNames(river: Pick<RiverRisk, "name" | "nameSi" | "aliases">): string[] {
  return [river.name, ...(river.nameSi ? [river.nameSi] : []), ...river.aliases]
}

const reasonKinds = ["gauge", "forecast", "rain", "reservoir", "coverage"] as const

/** What a reason rests on, e.g. "From a gauge reading", for a reason kept in English. */
export function reasonBasis(kind: string, locale: Locale = getLocale()): string {
  const known = (reasonKinds as readonly string[]).includes(kind)
  return translateIn(
    locale,
    `rivers.risk.basis.${known ? (kind as (typeof reasonKinds)[number]) : "generic"}`,
  )
}

const thresholdWords: Record<string, Threshold> = {
  alert: "alert",
  "minor flood": "minor_flood",
  "major flood": "major_flood",
}

/** "about 3 hours", the server's rounding of a time to a threshold. */
function hoursText(words: string, locale: Locale): string | null {
  if (words === "less than an hour") return translateIn(locale, "rivers.risk.reason.lessThanHour")
  if (words === "about an hour") return translateIn(locale, "rivers.risk.reason.aboutHour")
  const hours = /^about (\d+) hours$/.exec(words)
  if (!hours) return null
  return translateIn(locale, "rivers.risk.reason.aboutHours", { count: Number(hours[1]) })
}

/** A gauge reason: its level against a threshold, then whether it is near or rising. */
function gaugeReason(text: string, locale: Locale): string | null {
  const base =
    /^(.+) is at (.+?), (above|below) its (alert|minor flood|major flood) level of (.+?)\.(?: (.+))?$/.exec(
      text,
    )
  if (!base) return null
  const [, gauge, level, side, name, threshold, rest] = base as unknown as string[]
  const values = { gauge: gauge!, level: level!, threshold: threshold! }
  let sentence: string
  if (side === "below") {
    if (name !== "alert") return null
    sentence = translateIn(locale, "rivers.risk.reason.below", values)
  } else {
    sentence = translateIn(locale, `rivers.risk.reason.above.${thresholdWords[name!]!}`, values)
  }
  const more = (rest ?? "").trim()
  if (more === "") return sentence
  const near = /^It is within (.+) of that level\.$/.exec(more)
  if (near)
    return `${sentence} ${translateIn(locale, "rivers.risk.reason.near", { gap: near[1]! })}`
  const rising =
    /^Rising (.+) an hour, it would reach (alert|minor flood|major flood) level in (.+)\.$/.exec(
      more,
    )
  const time = rising ? hoursText(rising[3]!, locale) : null
  if (!rising || time === null) return null
  const next = translateIn(locale, `rivers.risk.reason.rising.${thresholdWords[rising[2]!]!}`, {
    rate: rising[1]!,
    time,
  })
  return `${sentence} ${next}`
}

const fixedReasons: Record<string, MessageKey> = {
  "No gauge on this stretch, so its level is not known.": "rivers.risk.reason.noGauge",
  "No current reading from its gauges, so its level is not known.": "rivers.risk.reason.noCurrent",
  "A Google flood forecast here is out of date, so it is not used.":
    "rivers.risk.reason.forecastStale",
  "The rain forecast is out of date, so it is not used.": "rivers.risk.reason.rainStale",
}

const rainReasons: [RegExp, MessageKey][] = [
  [/^Heavy rain is forecast over the catchment: /, "rivers.risk.reason.rainHeavy"],
  [/^Fairly heavy rain is forecast over the catchment: /, "rivers.risk.reason.rainFairly"],
  [/^Forecast rain over the catchment is /, "rivers.risk.reason.rainAverage"],
]

/**
 * A reason in the reader's language. The server writes reasons in English;
 * the sentences it is known to write are rebuilt from their parts, and any
 * other is kept as it came, marked `english` so it is shown with `lang="en"`.
 */
export function reasonText(
  reason: Pick<RiskReason, "kind" | "text">,
  locale: Locale = getLocale(),
): { text: string; english: boolean } {
  const { text } = reason
  if (locale === "en") return { text, english: false }
  const translated = rebuildReason(reason, locale)
  return translated === null ? { text, english: true } : { text: translated, english: false }
}

/**
 * A known server sentence rebuilt from its parts in a language, or null.
 * Rebuilt in English it gives the server's sentence back, which the tests
 * check so the two stay in step.
 */
export function rebuildReason(
  reason: Pick<RiskReason, "kind" | "text">,
  locale: Locale,
): string | null {
  const { text } = reason
  {
    const fixed = fixedReasons[text]
    if (fixed) return translateIn(locale, fixed)
    const next =
      /^No gauge on this stretch, so its level is not known\. The next gauge downstream is (.+)\.$/.exec(
        text,
      )
    if (next) return translateIn(locale, "rivers.risk.reason.noGaugeNext", { gauge: next[1]! })
    const noAlert = /^(.+) publishes no alert level to compare with\.$/.exec(text)
    if (noAlert)
      return translateIn(locale, "rivers.risk.reason.noAlertLevel", { gauge: noAlert[1]! })
    for (const [start, key] of rainReasons) {
      if (!start.test(text)) continue
      const amounts = /(\d+) mm in the next 24 hours and (\d+) mm in 72 hours on average\.$/.exec(
        text,
      )
      return amounts ? translateIn(locale, key, { day: amounts[1]!, days: amounts[2]! }) : null
    }
    return reason.kind === "gauge" ? gaugeReason(text, locale) : null
  }
}
