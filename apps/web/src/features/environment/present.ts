/**
 * Air-quality bands and sea-state words, and the pixel work of the sea field:
 * a stepped colour ramp, the fade at its edges and the land cut out of it. Pure,
 * so it is tested without a browser.
 */
import { parseHex, type RampStop } from "@/lib/colour"
import type { ScaleStep } from "@/lib/tokens"
import { gridBounds, mercatorY } from "../weather-grid/field"
import type { GridSpec } from "../location/types"
import type { AqiCategory } from "../location/types"

/** The EPA's words for each AQI band, in order. */
export const aqiBands: { category: AqiCategory; label: string; token: `--${string}` }[] = [
  { category: "good", label: "Good", token: "--aqi-1" },
  { category: "moderate", label: "Moderate", token: "--aqi-2" },
  { category: "unhealthy_sensitive", label: "Unhealthy for sensitive groups", token: "--aqi-3" },
  { category: "unhealthy", label: "Unhealthy", token: "--aqi-4" },
  { category: "very_unhealthy", label: "Very unhealthy", token: "--aqi-5" },
  { category: "hazardous", label: "Hazardous", token: "--aqi-6" },
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

/** The WMO sea-state word for a wave height in metres. */
export function seaStateOf(heightM: number): string {
  if (heightM < 0.1) return "Calm"
  if (heightM < 0.5) return "Smooth"
  if (heightM < 1.25) return "Slight"
  if (heightM < 2.5) return "Moderate"
  if (heightM < 4) return "Rough"
  if (heightM < 6) return "Very rough"
  if (heightM < 9) return "High"
  return "Very high"
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
