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
  severe: "Danger",
  above_normal: "Warning",
  no_flooding: "Normal",
  no_forecast: "No forecast",
}

/** Theme tokens by state (src/styles/theme.css). */
export const basinTokens: Record<BasinState, `--${string}`> = {
  extreme: "--flood-extreme",
  severe: "--flood-danger",
  above_normal: "--flood-warning",
  no_flooding: "--flood-normal",
  no_forecast: "--basin-outline",
}

export const basinColors: Record<BasinState, string> = {
  extreme: "var(--flood-extreme)",
  severe: "var(--flood-danger)",
  above_normal: "var(--flood-warning)",
  no_flooding: "var(--flood-normal)",
  no_forecast: "var(--basin-outline)",
}

/** Badge variants by state. "No forecast" is grey: nothing is known, not calm. */
export const basinBadge = {
  extreme: "flood_extreme",
  severe: "flood_danger",
  above_normal: "flood_warning",
  no_flooding: "flood_normal",
  no_forecast: "flood_nodata",
} as const satisfies Record<BasinState, string>

/**
 * Tint strength by state. A calm country reads as Flood Hub's does, green dots and
 * no heavy shapes, so a calm basin has barely a tint and a basin with no forecast
 * none at all, so it cannot be mistaken for a calm one. Even at a warning the tint
 * stays faint and the outline dashed: a filled, solid-edged shape on this map always
 * means an area Google published, and a basin is not one. Its state may rest on a
 * single gauge.
 */
export const basinFillOpacity: Record<BasinState, number> = {
  extreme: 0.09,
  severe: 0.07,
  above_normal: 0.05,
  no_flooding: 0.035,
  no_forecast: 0,
}

/** Outlines stay faint until a basin has something to say. */
export const basinLineOpacity: Record<BasinState, number> = {
  extreme: 1,
  severe: 1,
  above_normal: 1,
  no_flooding: 0.4,
  no_forecast: 0.4,
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
 * How each kind of Google polygon is drawn, in Flood Hub's language: a pale fill and
 * a strong outline for flash floods, orange for likely and red for highly likely,
 * and a blue ramp for inundation probability. `rank` orders them, highly likely on top.
 */
export const areaStyle: Record<
  FloodAreaKind,
  { token: `--${string}`; color: string; opacity: number; width: number; rank: number }
> = {
  notification: {
    token: "--flood-warning",
    color: "var(--flood-warning)",
    opacity: 0.1,
    width: 1.25,
    rank: 0,
  },
  significant_event: {
    token: "--flood-event",
    color: "var(--flood-event)",
    opacity: 0.22,
    width: 1.75,
    rank: 1,
  },
  inundation_low: {
    token: "--flood-inundation-low",
    color: "var(--flood-inundation-low)",
    opacity: 0.6,
    width: 0.75,
    rank: 2,
  },
  inundation_medium: {
    token: "--flood-inundation-medium",
    color: "var(--flood-inundation-medium)",
    opacity: 0.65,
    width: 0.75,
    rank: 3,
  },
  inundation_high: {
    token: "--flood-inundation-high",
    color: "var(--flood-inundation-high)",
    opacity: 0.7,
    width: 0.75,
    rank: 4,
  },
  flash_likely: {
    token: "--flood-flash-likely",
    color: "var(--flood-flash-likely)",
    opacity: 0.26,
    width: 1.5,
    rank: 5,
  },
  flash_highly_likely: {
    token: "--flood-flash-highly",
    color: "var(--flood-flash-highly)",
    opacity: 0.3,
    width: 1.75,
    rank: 6,
  },
}

export const areaKinds = Object.keys(areaStyle) as FloodAreaKind[]

/** The most specific of overlapping flood areas: what a click or a hover names. */
export function topArea<T extends { properties: { kind?: unknown } }>(areas: T[]): T | undefined {
  const rank = (area: T) => areaStyle[area.properties.kind as FloodAreaKind]?.rank ?? -1
  return areas.reduce<T | undefined>(
    (best, area) => (best === undefined || rank(area) > rank(best) ? area : best),
    undefined,
  )
}
