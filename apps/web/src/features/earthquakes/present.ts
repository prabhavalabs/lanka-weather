/** Words and shades for earthquakes. Pure, so it is tested without a browser. */
import { formatInteger, formatNumber } from "@/lib/format"
import { translate } from "@/lib/i18n"
import type { EarthquakeCollection } from "./types"

export type AgeBand = "day" | "week" | "month"

const day = 24 * 60 * 60 * 1000

export function ageBand(time: string, now: Date = new Date()): AgeBand {
  const age = now.getTime() - new Date(time).getTime()
  if (age <= day) return "day"
  if (age <= 7 * day) return "week"
  return "month"
}

export const ageOpacity: Record<AgeBand, number> = { day: 0.95, week: 0.65, month: 0.35 }
/** "Last 24 hours", "Last 7 days", "Last 30 days" */
export function ageLabel(band: AgeBand): string {
  return translate(`earthquakes.age.${band}`)
}

/** Circle radius in pixels at the map's middle zooms. */
export function magnitudeRadius(magnitude: number): number {
  return Math.max(3, Math.min(26, 3 + (magnitude - 2.5) * 3.6))
}

/**
 * The layer's features with their age band added, since the age is what shades
 * them and depends on when they are drawn.
 */
export function withAge(data: EarthquakeCollection, now: Date = new Date()): EarthquakeCollection {
  return {
    ...data,
    features: data.features.map((feature) => ({
      ...feature,
      properties: { ...feature.properties, age: ageBand(feature.properties.time, now) },
    })),
  } as EarthquakeCollection
}

export function strongest(data: EarthquakeCollection | undefined) {
  if (!data || data.features.length === 0) return null
  return data.features.reduce((best, feature) =>
    feature.properties.magnitude > best.properties.magnitude ? feature : best,
  )
}

export function earthquakeParts(data: EarthquakeCollection): string[] {
  const peak = strongest(data)
  const days = data.meta.windowDays
  if (!peak) return [translate("earthquakes.summary.none", { days })]
  return [
    translate("earthquakes.summary.count", { count: data.features.length, days }),
    translate("earthquakes.summary.strongest", { magnitude: magnitude(peak.properties.magnitude) }),
  ]
}

/** "6.2": a magnitude as USGS gives it, to one decimal. */
export function magnitude(value: number): string {
  return formatNumber(value, 1)
}

/** "4,512 km" */
export function formatDistance(km: number): string {
  return `${formatInteger(Math.round(km))} km`
}

/** "4,512 km from Sri Lanka" */
export function distanceFromSriLanka(km: number): string {
  return translate("earthquakes.fromSriLanka", { distance: formatDistance(km) })
}

/** What USGS's tsunami flag means, and whom to follow for warnings. */
export function tsunamiNote(): string {
  return translate("earthquakes.tsunami")
}

const reviewStatus = ["reviewed", "automatic", "deleted"] as const

/** USGS's review status in the reader's language; an unknown word stays as USGS gives it. */
export function reviewLabel(status: string): string {
  const key = status.trim().toLowerCase()
  return (reviewStatus as readonly string[]).includes(key)
    ? translate(`earthquakes.status.${key as (typeof reviewStatus)[number]}`)
    : status
}
