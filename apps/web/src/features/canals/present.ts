/** How canal gauges read. Pure, so it is tested without a browser. */
import type { CanalGauge } from "./types"

/** The source's state words, as its own page writes them. An unknown one is shown as it comes. */
const stateWords: Record<string, string> = {
  normal: "Normal",
  rising: "Rising",
  higher: "Higher than usual",
  offline: "Not reporting",
}

export function stateLabel(state: string) {
  return stateWords[state] ?? state
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
  name: string
  gauges: CanalGauge[]
}

/** Gauges by canal, raised ones and their groups first; gauges with no group go last. */
export function groupCanals(gauges: CanalGauge[]): CanalGroup[] {
  const rank = (gauge: CanalGauge) => {
    const state = shownState(gauge)
    if (state === "higher") return 2
    if (state === "rising") return 1
    return state === "offline" ? -1 : 0
  }
  const groups = new Map<string, CanalGauge[]>()
  for (const gauge of gauges) {
    const name = gauge.group || "Other gauges"
    groups.set(name, [...(groups.get(name) ?? []), gauge])
  }
  return [...groups.entries()]
    .map(([name, list]) => ({
      name,
      gauges: [...list].sort((a, b) => rank(b) - rank(a) || a.name.localeCompare(b.name)),
    }))
    .sort((a, b) => {
      if (a.name === "Other gauges") return 1
      if (b.name === "Other gauges") return -1
      const raisedDifference = Math.max(...b.gauges.map(rank)) - Math.max(...a.gauges.map(rank))
      return raisedDifference || a.name.localeCompare(b.name)
    })
}

/** "+0.04 m in 3 h", or nothing with no figure. */
export function changeText(change: number | null) {
  if (change === null) return null
  const sign = change > 0 ? "+" : change < 0 ? "−" : ""
  return `${sign}${Math.abs(change).toFixed(2)} m in 3 h`
}
