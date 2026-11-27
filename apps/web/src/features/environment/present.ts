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

/**
 * Each pixel's distance to the nearest land pixel, in pixels, by a two-pass
 * chamfer transform (steps of 1 and √2): close to the true distance and
 * linear in the image size. Infinity where the mask holds no land at all.
 */
export function distanceToLand(land: Uint8Array, width: number, height: number): Float32Array {
  const distance = new Float32Array(width * height)
  for (let index = 0; index < distance.length; index += 1) {
    distance[index] = land[index] ? 0 : Infinity
  }
  const diagonal = Math.SQRT2
  const at = (row: number, column: number) => row * width + column
  for (let row = 0; row < height; row += 1) {
    for (let column = 0; column < width; column += 1) {
      const here = at(row, column)
      let best = distance[here]!
      if (column > 0) best = Math.min(best, distance[here - 1]! + 1)
      if (row > 0) {
        best = Math.min(best, distance[here - width]! + 1)
        if (column > 0) best = Math.min(best, distance[here - width - 1]! + diagonal)
        if (column < width - 1) best = Math.min(best, distance[here - width + 1]! + diagonal)
      }
      distance[here] = best
    }
  }
  for (let row = height - 1; row >= 0; row -= 1) {
    for (let column = width - 1; column >= 0; column -= 1) {
      const here = at(row, column)
      let best = distance[here]!
      if (column < width - 1) best = Math.min(best, distance[here + 1]! + 1)
      if (row < height - 1) {
        best = Math.min(best, distance[here + width]! + 1)
        if (column < width - 1) best = Math.min(best, distance[here + width + 1]! + diagonal)
        if (column > 0) best = Math.min(best, distance[here + width - 1]! + diagonal)
      }
      distance[here] = best
    }
  }
  return distance
}

/**
 * Fades a painted field out with distance from the coast: whole within
 * `inner` pixels of land, gone beyond `outer`, eased between. The sea field
 * then reads as the water around the island, not as the box its grid fills.
 */
export function fadeFromCoast(
  pixels: Uint8ClampedArray,
  distance: Float32Array,
  inner: number,
  outer: number,
) {
  const span = Math.max(outer - inner, 1)
  for (let index = 0; index < distance.length; index += 1) {
    const away = distance[index]!
    if (away <= inner) continue
    const at = index * 4 + 3
    if (away >= outer) {
      pixels[at] = 0
      continue
    }
    const t = (away - inner) / span
    // Smoothstep, so the edge has no visible line.
    pixels[at] = pixels[at]! * (1 - t * t * (3 - 2 * t))
  }
  return pixels
}

/**
 * A grid with its empty cells filled from their filled neighbours, a ring at
 * a time, up to `passes` rings deep. A wave model leaves the cells that touch
 * the coast empty; filled, the sea field reaches the shore, where the land
 * cut-out ends it, instead of stopping a cell out to sea.
 */
export function fillGaps(
  values: readonly (number | null)[],
  nx: number,
  ny: number,
  passes = 2,
): (number | null)[] {
  let current = [...values]
  for (let pass = 0; pass < passes; pass += 1) {
    const next = [...current]
    let filled = false
    for (let j = 0; j < ny; j += 1) {
      for (let i = 0; i < nx; i += 1) {
        if (current[j * nx + i] !== null) continue
        let total = 0
        let count = 0
        for (let dj = -1; dj <= 1; dj += 1) {
          for (let di = -1; di <= 1; di += 1) {
            const x = i + di
            const y = j + dj
            if ((di === 0 && dj === 0) || x < 0 || y < 0 || x >= nx || y >= ny) continue
            const value = current[y * nx + x]
            if (value === null || value === undefined || !Number.isFinite(value)) continue
            total += value
            count += 1
          }
        }
        if (count > 0) {
          next[j * nx + i] = total / count
          filled = true
        }
      }
    }
    current = next
    if (!filled) break
  }
  return current
}
