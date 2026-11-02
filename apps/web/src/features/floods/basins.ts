/**
 * How flood basins are named and tinted. Pure, so it is tested without a
 * browser. They share the flood status colours with gauges and rivers.
 */
import type { BasinState, BasinStatus, SubbasinStatus } from "../river-risk/types"
import type { FloodAreaKind } from "./types"

export const basinStateOrder: BasinState[] = [
  "extreme",
  "severe",
  "above_normal",
  "no_flooding",
  "no_forecast",
]

export const basinStateLabels: Record<BasinState, string> = {
  extreme: "Extreme",
  severe: "Severe",
  above_normal: "Above normal",
  no_flooding: "No flooding",
  no_forecast: "No forecast here",
}

/** Theme tokens by state (src/styles/theme.css). */
export const basinTokens: Record<BasinState, `--${string}`> = {
  extreme: "--status-major-flood",
  severe: "--status-minor-flood",
  above_normal: "--status-alert",
  no_flooding: "--status-normal",
  no_forecast: "--basin-outline",
}

export const basinColors: Record<BasinState, string> = {
  extreme: "var(--status-major-flood)",
  severe: "var(--status-minor-flood)",
  above_normal: "var(--status-alert)",
  no_flooding: "var(--status-normal)",
  no_forecast: "var(--basin-outline)",
}

/** Badge variants by state. "No forecast" is grey: nothing is known, not calm. */
export const basinBadge = {
  extreme: "major_flood",
  severe: "minor_flood",
  above_normal: "alert",
  no_flooding: "normal",
  no_forecast: "unknown",
} as const satisfies Record<BasinState, string>

/**
 * Tint strength by state. A basin with no forecast has no tint at all, so it
 * cannot be mistaken for a calm one, which has a pale tint.
 */
export const basinFillOpacity: Record<BasinState, number> = {
  extreme: 0.46,
  severe: 0.4,
  above_normal: 0.32,
  no_flooding: 0.1,
  no_forecast: 0,
}

function group<T extends { fid: number; state: BasinState }>(items: T[]) {
  const ids: Record<BasinState, number[]> = {
    extreme: [],
    severe: [],
    above_normal: [],
    no_flooding: [],
    no_forecast: [],
  }
  for (const item of items) ids[item.state].push(item.fid)
  return ids
}

export const basinIdsByState = (basins: BasinStatus[]) => group(basins)
export const subbasinIdsByState = (subbasins: SubbasinStatus[]) => group(subbasins)

/** Basins per state. */
export function countBasins(basins: Pick<BasinStatus, "state">[]) {
  const counts: Record<BasinState, number> = {
    extreme: 0,
    severe: 0,
    above_normal: 0,
    no_flooding: 0,
    no_forecast: 0,
  }
  for (const basin of basins) counts[basin.state] += 1
  return counts
}

/** Whether a basin's state rests only on gauges Google has not verified. */
export function anyUnverified(basins: Pick<BasinStatus, "onlyUnverified">[]): boolean {
  return basins.some((basin) => basin.onlyUnverified)
}

/**
 * How each kind of Google polygon is filled: inundation is graded by how likely
 * the water is, and the outline carries the kind where the colour repeats.
 */
export const areaStyle: Record<
  FloodAreaKind,
  { token: `--${string}`; color: string; opacity: number; width: number; rank: number }
> = {
  notification: {
    token: "--status-alert",
    color: "var(--status-alert)",
    opacity: 0.18,
    width: 2,
    rank: 0,
  },
  significant_event: {
    token: "--status-major-flood",
    color: "var(--status-major-flood)",
    opacity: 0.22,
    width: 2.5,
    rank: 1,
  },
  flash_likely: {
    token: "--status-minor-flood",
    color: "var(--status-minor-flood)",
    opacity: 0.38,
    width: 2,
    rank: 2,
  },
  flash_highly_likely: {
    token: "--status-major-flood",
    color: "var(--status-major-flood)",
    opacity: 0.5,
    width: 2.5,
    rank: 3,
  },
  inundation_low: {
    token: "--status-alert",
    color: "var(--status-alert)",
    opacity: 0.42,
    width: 1.5,
    rank: 4,
  },
  inundation_medium: {
    token: "--status-minor-flood",
    color: "var(--status-minor-flood)",
    opacity: 0.55,
    width: 1.75,
    rank: 5,
  },
  inundation_high: {
    token: "--status-major-flood",
    color: "var(--status-major-flood)",
    opacity: 0.68,
    width: 2,
    rank: 6,
  },
}

export const areaKinds = Object.keys(areaStyle) as FloodAreaKind[]
