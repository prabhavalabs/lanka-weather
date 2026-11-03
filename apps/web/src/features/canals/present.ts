/** How canal gauges read. Pure, so it is tested without a browser. */
import { getLocale, translateIn, type Locale } from "@/lib/i18n"
import type { CanalGauge } from "./types"

const stateWords = ["normal", "rising", "higher", "offline", "unknown"] as const

/**
 * The source's state words, as its own page writes them, in the reader's
 * language. An unknown one is shown as it comes (in English).
 */
export function stateLabel(state: string, locale: Locale = getLocale()) {
  return (stateWords as readonly string[]).includes(state)
    ? translateIn(locale, `rivers.canals.state.${state as (typeof stateWords)[number]}`)
    : state
}

const seaTrends: Record<
  string,
  "rising" | "risingSlowly" | "falling" | "fallingSlowly" | "steady"
> = {
  rising: "rising",
  "rising slowly": "risingSlowly",
  falling: "falling",
  "falling slowly": "fallingSlowly",
  steady: "steady",
}

/** The sea's trend in the reader's language; `english` when the word is not one we know. */
export function seaTrendText(trend: string, locale: Locale = getLocale()) {
  const key = seaTrends[trend.trim().toLowerCase()]
  return key
    ? { text: translateIn(locale, `rivers.canals.seaTrend.${key}`), english: false }
    : { text: trend, english: locale !== "en" }
}

/** Badges pair the colour with the word. */
export function stateBadge(state: string) {
  if (state === "higher") return "minor_flood" as const
  if (state === "rising") return "alert" as const
  if (state === "normal") return "normal" as const
  return "chip" as const
}

/**
 * The state to show: a reading too old says nothing about now, whatever word it was
 * left with.
 */
export function shownState(gauge: Pick<CanalGauge, "state" | "stale">) {
  return gauge.stale ? "offline" : gauge.state
}

export interface CanalGroup {
  /** The source's canal name; empty for gauges it puts in no group. */
  name: string
  gauges: CanalGauge[]
}

/**
 * Gauges by canal, raised ones and their groups first; gauges with no group
 * go last, under an empty name the panel shows as "Other gauges".
 */
export function groupCanals(gauges: CanalGauge[]): CanalGroup[] {
  const rank = (gauge: CanalGauge) => {
    const state = shownState(gauge)
    if (state === "higher") return 2
    if (state === "rising") return 1
    return state === "offline" ? -1 : 0
  }
  const groups = new Map<string, CanalGauge[]>()
  for (const gauge of gauges) {
    const name = gauge.group.trim()
    groups.set(name, [...(groups.get(name) ?? []), gauge])
  }
  return [...groups.entries()]
    .map(([name, list]) => ({
      name,
      gauges: [...list].sort((a, b) => rank(b) - rank(a) || a.name.localeCompare(b.name)),
    }))
    .sort((a, b) => {
      if (a.name === "") return 1
      if (b.name === "") return -1
      const raisedDifference = Math.max(...b.gauges.map(rank)) - Math.max(...a.gauges.map(rank))
      return raisedDifference || a.name.localeCompare(b.name)
    })
}

/** "+0.04 m in 3 h", or nothing with no figure. */
export function changeText(change: number | null) {
  if (change === null) return null
  const sign = change > 0 ? "+" : change < 0 ? "−" : ""
  return translateIn(getLocale(), "rivers.canals.change", {
    change: `${sign}${Math.abs(change).toFixed(2)}`,
  })
}
