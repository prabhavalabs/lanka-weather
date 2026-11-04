/**
 * The satellite image's tiles and colours, and words for observed totals.
 * Pure, so it is tested without a browser.
 */
import { formatClock, formatDay } from "@/lib/format"
import { parseHex } from "@/lib/colour"
import { getLocale, translateIn, type Locale } from "@/lib/i18n"
import type { ResolvedTheme } from "@/lib/theme"
import type { ObservedRainMethod, RainStep } from "./types"

/** A tile's column and row at a zoom. */
export function tilesInBounds(
  [west, south, east, north]: [number, number, number, number],
  zoom: number,
): { z: number; x: number; y: number }[] {
  const scale = 2 ** zoom
  const column = (lon: number) => Math.floor(((lon + 180) / 360) * scale)
  const row = (lat: number) => {
    const radians = (lat * Math.PI) / 180
    return Math.floor(
      ((1 - Math.log(Math.tan(radians) + 1 / Math.cos(radians)) / Math.PI) / 2) * scale,
    )
  }
  const epsilon = 1e-6
  const tiles: { z: number; x: number; y: number }[] = []
  for (let y = row(north - epsilon); y <= row(south + epsilon); y += 1) {
    for (let x = column(west + epsilon); x <= column(east - epsilon); x += 1) {
      tiles.push({ z: zoom, x, y })
    }
  }
  return tiles
}

/** The URL of a tile from the layer's template. */
export function tileUrl(template: string, time: string, z: number, x: number, y: number): string {
  return template
    .replace("{time}", time)
    .replace("{z}", String(z))
    .replace("{x}", String(x))
    .replace("{y}", String(y))
}

/**
 * NASA's heaviest rain is drawn nearly black, which a dark map would swallow.
 * The dark theme lifts the image's darkest colours towards grey; the light
 * theme leaves them alone. `raster-brightness-min` does this on the map, and
 * `liftedColour` does the same to the legend, so the two agree.
 */
export function rainImageLift(theme: ResolvedTheme): number {
  return theme === "dark" ? 0.3 : 0
}

/** A colour as `raster-brightness-min` leaves it. */
export function liftedColour(hex: string, lift: number): string {
  const [r, g, b] = parseHex(hex)
  const channel = (value: number) =>
    Math.round(lift * 255 + (value * (255 - lift * 255)) / 255)
      .toString(16)
      .padStart(2, "0")
  return `#${channel(r)}${channel(g)}${channel(b)}`
}

/** "Mon 2 Nov 19:00–19:30": the half hour an image covers, in Sri Lanka time. */
export function stepLabel(
  step: Pick<RainStep, "validAt" | "endsAt">,
  locale: Locale = getLocale(),
): string {
  return translateIn(locale, "rain.step", {
    day: formatDay(step.validAt, locale),
    start: formatClock(step.validAt),
    end: formatClock(step.endsAt),
  })
}

const methodKeys = {
  daily_report: "rain.method.dailyReport",
  sum_3h: "rain.method.sum3h",
  sum_gauge: "rain.method.sumGauge",
} as const satisfies Record<ObservedRainMethod, string>

/** How a station's total was reached, in words; empty for a method we do not know. */
export function methodLabel(method: ObservedRainMethod, locale: Locale = getLocale()): string {
  const key = methodKeys[method] as (typeof methodKeys)[ObservedRainMethod] | undefined
  return key ? translateIn(locale, key) : ""
}
