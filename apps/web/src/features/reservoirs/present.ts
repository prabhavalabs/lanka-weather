/**
 * How reservoirs read in panels and lists. The status words are the source's;
 * unknown stays unknown: a missing figure is never shown as 0%. Pure, so it is
 * tested without a browser.
 */
import { formatDay, formatInteger, sriLankaTimeZone } from "@/lib/format"
import { translate } from "@/lib/i18n"
import { liveLabels } from "../rivers/present"
import type {
  ReservoirCounts,
  ReservoirFeature,
  ReservoirFlow,
  ReservoirProperties,
  ReservoirStatus,
} from "./types"

export const reservoirTokens: Record<ReservoirStatus, `--${string}`> = {
  spilling: "--reservoir-spilling",
  releasing: "--reservoir-releasing",
  holding: "--reservoir-holding",
  nodata: "--reservoir-nodata",
}

export const reservoirColors: Record<ReservoirStatus, string> = {
  spilling: "var(--reservoir-spilling)",
  releasing: "var(--reservoir-releasing)",
  holding: "var(--reservoir-holding)",
  nodata: "var(--reservoir-nodata)",
}

const statuses: ReservoirStatus[] = ["spilling", "releasing", "holding", "nodata"]

/** The source's status words, in the current language. */
export const reservoirStatusLabels = liveLabels(
  statuses,
  (status) => `terms.reservoirStatus.${status}`,
)

/** What the source's status means, in the current language. */
export const reservoirStatusMeanings = liveLabels(
  statuses,
  (status) => `reservoirs.meaning.${status}`,
)

/**
 * The shapes the map draws, so a status is told apart by shape as well as by
 * colour: a drop for spilling, a falling triangle for releasing, a square for
 * holding and a hollow square for no data. Each is drawn in a 40-pixel box.
 */
export const reservoirShapes = {
  spilling: (() => {
    const points: [number, number][] = [[20, 5]]
    for (let degrees = -35; degrees <= 215; degrees += 25) {
      const radians = (degrees * Math.PI) / 180
      points.push([20 + 12 * Math.cos(radians), 23 + 12 * Math.sin(radians)])
    }
    return points
  })(),
  releasing: [
    [7, 10],
    [33, 10],
    [20, 33],
  ] as [number, number][],
  holding: [
    [10, 10],
    [30, 10],
    [30, 30],
    [10, 30],
  ] as [number, number][],
} satisfies Record<string, [number, number][]>

/** Display order for legends and summaries: what needs attention first. */
export const reservoirStatusOrder: ReservoirStatus[] = [
  "spilling",
  "releasing",
  "holding",
  "nodata",
]

const types = ["major", "medium", "mahaweli"] as const

function typeLabel(type: string): string {
  return (types as readonly string[]).includes(type)
    ? translate(`reservoirs.type.${type as (typeof types)[number]}`)
    : translate("reservoirs.type.generic")
}

/** "Major reservoir · Kurunegala": short enough for one line on a phone. */
export function reservoirSubtitle(reservoir: ReservoirProperties): string {
  return [typeLabel(reservoir.type), reservoir.district]
    .filter(Boolean)
    .join(translate("common.separator"))
}

/** A YYYY-MM-DD day in Sri Lanka as a time at its noon, for date formatting. */
function noon(day: string): string {
  return `${day}T12:00:00+05:30`
}

/** "Mon 2 Nov" */
export function formatReadingDay(day: string): string {
  return formatDay(noon(day))
}

const colomboDay = new Intl.DateTimeFormat("en-CA", {
  timeZone: sriLankaTimeZone,
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
})

/** "today", "yesterday" or "4 days ago", by the calendar in Sri Lanka. */
export function readingDayAge(day: string, now: Date = new Date()): string {
  const today = Date.parse(`${colomboDay.format(now)}T00:00:00Z`)
  const then = Date.parse(`${day}T00:00:00Z`)
  const days = Math.round((today - then) / 86_400_000)
  if (days <= 0) return translate("format.today")
  if (days === 1) return translate("format.yesterday")
  return translate("format.daysAgo", { count: days })
}

/** A figure to at most one decimal, "1,551" or "17.2". */
function figure(value: number): string {
  return formatInteger(value, 1)
}

const flowKinds = ["spill", "outlets", "downstream"] as const

/** "Spill: 1,551 cusecs", or "Spill" when the amount is not known. The unit stays cusecs. */
export function flowShort(flow: ReservoirFlow): string {
  const kind = (flowKinds as readonly string[]).includes(flow.kind)
    ? translate(`reservoirs.flow.${flow.kind as (typeof flowKinds)[number]}`)
    : translate("reservoirs.flow.generic")
  return flow.cusecs === null
    ? kind
    : translate("reservoirs.flow.amount", { kind, amount: figure(flow.cusecs) })
}

/**
 * A caution for flows that are easy to misread. A Mahaweli Authority downstream
 * flow can include water through a power station, and outlets run while the
 * source still calls a reservoir holding.
 */
export function flowNote(flow: ReservoirFlow): string | null {
  switch (flow.kind) {
    case "downstream":
      return translate("reservoirs.flow.noteDownstream")
    case "outlets":
      return translate("reservoirs.flow.noteOutlets")
    default:
      return null
  }
}

/** The storage figure as it should be read, without ever turning unknown into 0%. */
export function storageView(reservoir: ReservoirProperties): {
  value: string
  caption: string
  /** Whether a bar should show it: only a current figure. */
  current: boolean
} {
  const { storagePercent, storageFresh, readingDay } = reservoir
  if (storagePercent === null) {
    return { value: "–", caption: translate("reservoirs.storage.notReported"), current: false }
  }
  if (storageFresh === false || reservoir.stale) {
    return {
      value: figure(storagePercent),
      caption: readingDay
        ? translate("reservoirs.storage.lastKnownOn", { day: formatReadingDay(readingDay) })
        : translate("reservoirs.storage.lastKnown"),
      current: false,
    }
  }
  return {
    value: figure(storagePercent),
    caption: translate("reservoirs.storage.ofCapacity"),
    current: true,
  }
}

/** Spilling or releasing on a reading that is not out of date. */
export function isActive(reservoir: Pick<ReservoirProperties, "status" | "stale">): boolean {
  return !reservoir.stale && (reservoir.status === "spilling" || reservoir.status === "releasing")
}

/** Those spilling or releasing, spilling first, then by name. */
export function activeReservoirs(features: ReservoirFeature[]): ReservoirFeature[] {
  const rank = (status: ReservoirStatus) => (status === "spilling" ? 0 : 1)
  return features
    .filter((feature) => isActive(feature.properties))
    .sort(
      (a, b) =>
        rank(a.properties.status) - rank(b.properties.status) ||
        a.properties.name.localeCompare(b.properties.name),
    )
}

/**
 * Whether the figures say anything about today. A fetch can succeed long after
 * the agencies stopped reporting, so the fetch time alone cannot tell.
 */
export function hasCurrentFigures(features: ReservoirFeature[]): boolean {
  return features.some((feature) => !feature.properties.stale)
}

/** The layer's one line. "None spilling" is said only on current data. */
export function reservoirSummary(
  counts: ReservoirCounts,
  active: number,
  outOfDate: boolean,
): string {
  if (counts.total === 0) return translate("reservoirs.summary.notCollected")
  if (outOfDate) return translate("reservoirs.summary.outOfDate", { count: counts.total })
  if (active === 0) return translate("reservoirs.summary.none")
  return translate("reservoirs.summary.active", { count: active })
}
