/**
 * Air-quality bands and sea-state words, and the pixel work of the sea field:
 * a stepped colour ramp, the fade at its edges and the land cut out of it. Pure,
 * so it is tested without a browser.
 */
import { parseHex, type RampStop } from "@/lib/colour"
import { getLocale, translateIn, type Locale, type MessageKey } from "@/lib/i18n"
import type { ScaleStep } from "@/lib/tokens"
import { gridBounds, mercatorY } from "../weather-grid/field"
import type { GridSpec } from "../location/types"
import type { AqiCategory } from "../location/types"

/** The EPA's AQI bands, in order, with the message key of each one's words. */
export const aqiBands: {
  category: AqiCategory
  key: Extract<MessageKey, `environment.aqi.${string}`>
  token: `--${string}`
}[] = [
  { category: "good", key: "environment.aqi.good", token: "--aqi-1" },
  { category: "moderate", key: "environment.aqi.moderate", token: "--aqi-2" },
  {
    category: "unhealthy_sensitive",
    key: "environment.aqi.unhealthySensitive",
    token: "--aqi-3",
  },
  { category: "unhealthy", key: "environment.aqi.unhealthy", token: "--aqi-4" },
  { category: "very_unhealthy", key: "environment.aqi.veryUnhealthy", token: "--aqi-5" },
  { category: "hazardous", key: "environment.aqi.hazardous", token: "--aqi-6" },
]

const aqiLimits = [50, 100, 150, 200, 300]

/** The band of a US AQI value. */
export function aqiBandOf(value: number) {
  const index = aqiLimits.findIndex((limit) => value <= limit)
  return aqiBands[index === -1 ? aqiBands.length - 1 : index]!
}

/** A band by the API's category. */
export function aqiBand(category: AqiCategory) {
  return aqiBands.find((band) => band.category === category) ?? aqiBands[0]!
}

/** The EPA's words for an AQI band, in the current language: "Good". */
export function aqiLabel(category: AqiCategory, locale: Locale = getLocale()): string {
  return translateIn(locale, aqiBand(category).key)
}

/** The WMO sea states, calmest first, as the API names them. */
export type SeaState =
  "calm" | "smooth" | "slight" | "moderate" | "rough" | "very rough" | "high" | "very high"

const seaStateKeys: Record<SeaState, Extract<MessageKey, `environment.seaState.${string}`>> = {
  calm: "environment.seaState.calm",
  smooth: "environment.seaState.smooth",
  slight: "environment.seaState.slight",
  moderate: "environment.seaState.moderate",
  rough: "environment.seaState.rough",
  "very rough": "environment.seaState.veryRough",
  high: "environment.seaState.high",
  "very high": "environment.seaState.veryHigh",
}

/** The WMO sea state for a wave height in metres. */
export function seaStateCode(heightM: number): SeaState {
  if (heightM < 0.1) return "calm"
  if (heightM < 0.5) return "smooth"
  if (heightM < 1.25) return "slight"
  if (heightM < 2.5) return "moderate"
  if (heightM < 4) return "rough"
  if (heightM < 6) return "very rough"
  if (heightM < 9) return "high"
  return "very high"
}

/** The WMO sea-state word for a wave height in metres, in the current language. */
export function seaStateOf(heightM: number, locale: Locale = getLocale()): string {
  return translateIn(locale, seaStateKeys[seaStateCode(heightM)])
}

/**
 * The API's sea-state word in the current language. A word we do not know
 * falls back to the state of the wave height, so it is never left in English.
 */
export function seaStateLabel(state: string, heightM: number, locale: Locale = getLocale()) {
  const key = seaStateKeys[state.trim().toLowerCase() as SeaState]
  return key ? translateIn(locale, key) : seaStateOf(heightM, locale)
}

/**
 * A ramp whose colours step at the scale's bounds and do not blend, for bands
 * that mean something by themselves: AQI 100 is not half unhealthy.
 */
export function steppedRamp(
  scale: readonly ScaleStep[],
  colours: readonly string[],
  top: number,
): RampStop[] {
  const stops: RampStop[] = []
  scale.forEach((step, index) => {
    const color = parseHex(colours[index] ?? "")
    const from = step.from ?? 0
    const next = scale[index + 1]?.from ?? top
    stops.push({ value: from, color }, { value: next - 1e-6, color })
  })
  return stops
}

/** Fades a painted field's alpha to nothing over `band` pixels at its four edges. */
export function fadeEdges(pixels: Uint8ClampedArray, width: number, height: number, band: number) {
  for (let row = 0; row < height; row += 1) {
    const fromEdgeY = Math.min(row, height - 1 - row)
    for (let column = 0; column < width; column += 1) {
      const fromEdge = Math.min(fromEdgeY, column, width - 1 - column)
      if (fromEdge >= band) continue
      const at = (row * width + column) * 4 + 3
      pixels[at] = pixels[at]! * (fromEdge / band)
    }
  }
  return pixels
}

/** A ring of longitude, latitude pairs as pixels in an image spanning the grid, in Mercator. */
export function ringToPixels(
  ring: readonly (readonly [number, number])[],
  grid: GridSpec,
  width: number,
  height: number,
): [number, number][] {
  const [west, south, east, north] = gridBounds(grid)
  const top = mercatorY(north)
  const bottom = mercatorY(south)
  return ring.map(([lon, lat]) => [
    ((lon - west) / (east - west)) * width,
    ((top - mercatorY(lat)) / (top - bottom)) * height,
  ])
}
