/** Words and shades for earthquakes. Pure, so it is tested without a browser. */
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
export const ageLabels: Record<AgeBand, string> = {
  day: "Last 24 hours",
  week: "Last 7 days",
  month: "Last 30 days",
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
  if (!peak) return [`None of M4.5+ in ${data.meta.windowDays} days`]
  return [
    `${data.features.length} in ${data.meta.windowDays} days`,
    `strongest M${peak.properties.magnitude.toFixed(1)}`,
  ]
}

/** "4,512 km" */
export function formatDistance(km: number): string {
  return `${Math.round(km).toLocaleString("en-GB")} km`
}

export const tsunamiNote =
  "USGS has set its tsunami flag on this event. The flag is an indicator that the event is in an oceanic region and large enough to look at; it is not a warning for Sri Lanka. For tsunami warnings, follow the Disaster Management Centre and the Department of Meteorology."
