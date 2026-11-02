/**
 * How the rain gauges read and which show at which zoom. Pure, so it is tested
 * without a browser.
 */
import type { ObservedRainCollection } from "../rainfall/types"
import type { RainBand, RainGaugeCollection } from "./types"

export const bandLabels: Record<RainBand, string> = {
  very_heavy: "Very heavy",
  heavy: "Heavy",
  moderate: "Moderate",
  light: "Light",
  none: "No rain",
  unknown: "Out of date",
}

export const bandRanges: Record<RainBand, string> = {
  very_heavy: "100 mm or more",
  heavy: "50 to 100 mm",
  moderate: "10 to 50 mm",
  light: "under 10 mm",
  none: "0 mm",
  unknown: "newest reading over 3 h old",
}

export const bandOrder: RainBand[] = ["very_heavy", "heavy", "moderate", "light", "none", "unknown"]

/** Theme tokens by band: the rainfall colours, one to a band so the key is exact. */
export const bandTokens: Record<RainBand, `--${string}`> = {
  very_heavy: "--rain-6",
  heavy: "--rain-4",
  moderate: "--rain-3",
  light: "--rain-1",
  none: "--status-unknown",
  unknown: "--status-unknown",
}

/** Badge variants by band, which always pair the colour with a word. */
export const bandBadge = {
  very_heavy: "major_flood",
  heavy: "minor_flood",
  moderate: "normal",
  light: "chip",
  none: "chip",
  unknown: "chip",
} as const satisfies Record<RainBand, string>

/**
 * The zoom a band first shows at, so the gauges do not bury the map: a meaningful
 * total at any zoom, light rain as the country fills the screen and gives way, and
 * dry and out-of-date gauges small and quiet only when close.
 */
export const bandMinZoom: Record<RainBand, number> = {
  very_heavy: 0,
  heavy: 0,
  moderate: 0,
  light: 8,
  none: 9,
  unknown: 9,
}

export function shownAt(band: RainBand, zoom: number) {
  return zoom >= bandMinZoom[band]
}

/** Counts by band. */
export function countBands(features: { properties: { band: RainBand } }[]) {
  const counts: Record<RainBand, number> = {
    very_heavy: 0,
    heavy: 0,
    moderate: 0,
    light: 0,
    none: 0,
    unknown: 0,
  }
  for (const feature of features) counts[feature.properties.band] += 1
  return counts
}

/** "258 of 275 current · 31 with 10 mm or more". */
export function gaugesSummary(meta: RainGaugeCollection["meta"]) {
  return `${meta.fresh} of ${meta.total} current · ${meta.wet} with 10 mm or more`
}

function kilometresBetween([lon1, lat1]: [number, number], [lon2, lat2]: [number, number]) {
  const toRadians = Math.PI / 180
  const dLat = (lat2 - lat1) * toRadians
  const dLon = (lon2 - lon1) * toRadians
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(lat1 * toRadians) * Math.cos(lat2 * toRadians) * Math.sin(dLon / 2) ** 2
  return 2 * 6371 * Math.asin(Math.min(1, Math.sqrt(a)))
}

/** How near a gauge must lie to another layer's point to be taken for the same place. */
export const sameSiteKm = 0.5

/**
 * The older rainfall points that a current live gauge already speaks for: with both
 * layers on, a river gauge and the device beside it would otherwise show the same
 * rain twice, often a little differently. A point nothing live stands beside stays.
 */
export function withoutLiveDuplicates(
  observed: ObservedRainCollection | undefined,
  gauges: RainGaugeCollection | undefined,
): ObservedRainCollection | undefined {
  if (!observed || !gauges) return observed
  const live = gauges.features.filter((feature) => !feature.properties.stale)
  return {
    ...observed,
    features: observed.features.filter(
      (point) =>
        !live.some(
          (gauge) =>
            kilometresBetween(point.geometry.coordinates, gauge.geometry.coordinates) <= sameSiteKm,
        ),
    ),
  }
}

/** The chart's line for a gauge's rolling total: x and y in 0–1, y up. */
export function seriesPath(series: { at: string; rain24hMm: number }[]) {
  if (series.length === 0) return { points: [], peak: 0, low: 0 }
  const times = series.map((point) => Date.parse(point.at))
  const start = Math.min(...times)
  const span = Math.max(Math.max(...times) - start, 1)
  const peak = Math.max(...series.map((point) => point.rain24hMm))
  const low = Math.min(...series.map((point) => point.rain24hMm))
  const top = Math.max(peak, 1)
  return {
    points: series.map((point, index) => ({
      x: (times[index]! - start) / span,
      y: point.rain24hMm / top,
    })),
    peak,
    low,
  }
}
