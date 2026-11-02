/**
 * Colour ramps for map data: continuous ramps for the weather field (an
 * image the app paints itself) and stepped bins for choropleths (MapLibre
 * expressions). Colours come from the theme tokens (src/styles/theme.css),
 * so callers read them with readTokens and pass them in; nothing here
 * touches the DOM.
 */
import type { ScaleStep } from "./tokens"

/** Red, green and blue from 0 to 255, and alpha from 0 to 1. */
export type Rgba = readonly [number, number, number, number]

const transparent: Rgba = [0, 0, 0, 0]

/** "#rrggbb" or "#rrggbbaa" as RGBA; anything else is transparent. */
export function parseHex(color: string): Rgba {
  const match = /^#([0-9a-f]{6})([0-9a-f]{2})?$/i.exec(color.trim())
  if (!match) return transparent
  const value = parseInt(match[1]!, 16)
  const alpha = match[2] === undefined ? 1 : parseInt(match[2], 16) / 255
  return [(value >> 16) & 255, (value >> 8) & 255, value & 255, alpha]
}

/** CSS rgba() for a colour, e.g. for MapLibre paint properties. */
export function rgbaCss([r, g, b, a]: Rgba): string {
  return `rgba(${Math.round(r)}, ${Math.round(g)}, ${Math.round(b)}, ${Number(a.toFixed(3))})`
}

/** A colour pinned to a value; between stops colours blend linearly. */
export interface RampStop {
  value: number
  color: Rgba
}

/**
 * The colour of a value on a continuous ramp, blended between the two stops
 * around it and held at the end colours beyond the first and last stop.
 */
export function sampleRamp(ramp: readonly RampStop[], value: number): Rgba {
  const first = ramp[0]
  if (!first || !Number.isFinite(value)) return transparent
  if (value <= first.value) return first.color
  for (let index = 1; index < ramp.length; index += 1) {
    const upper = ramp[index]!
    if (value <= upper.value) {
      const lower = ramp[index - 1]!
      const span = upper.value - lower.value
      const t = span > 0 ? (value - lower.value) / span : 1
      return [
        lower.color[0] + (upper.color[0] - lower.color[0]) * t,
        lower.color[1] + (upper.color[1] - lower.color[1]) * t,
        lower.color[2] + (upper.color[2] - lower.color[2]) * t,
        lower.color[3] + (upper.color[3] - lower.color[3]) * t,
      ]
    }
  }
  return ramp[ramp.length - 1]!.color
}

/**
 * A lookup table of `size` RGBA entries (alpha 0–255) covering the ramp's
 * range, so painting thousands of pixels needs no blending per pixel.
 */
export interface ColourTable {
  min: number
  max: number
  rgba: Uint8ClampedArray
}

export function colourTable(ramp: readonly RampStop[], size = 256): ColourTable {
  const min = ramp[0]?.value ?? 0
  const max = ramp[ramp.length - 1]?.value ?? 1
  const rgba = new Uint8ClampedArray(size * 4)
  for (let index = 0; index < size; index += 1) {
    const value = min + ((max - min) * index) / (size - 1)
    const [r, g, b, a] = sampleRamp(ramp, value)
    rgba.set([r, g, b, a * 255], index * 4)
  }
  return { min, max, rgba }
}

/** The table entry for a value: its offset into `table.rgba`. */
export function tableOffset(table: ColourTable, value: number): number {
  const size = table.rgba.length / 4
  const span = table.max - table.min
  const t = span > 0 ? (value - table.min) / span : 0
  const index = Math.round(Math.min(1, Math.max(0, t)) * (size - 1))
  return index * 4
}

/**
 * A continuous ramp through a stepped scale's colours. Each colour sits at
 * its step's lower bound; an open first step ("below 21") sits `openSpan`
 * under the next bound, as the legends label it.
 */
export function rampFromScale(
  scale: readonly ScaleStep[],
  colours: readonly string[],
  openSpan = 3,
): RampStop[] {
  return scale.map((step, index) => ({
    value: step.from ?? (scale[index + 1]?.from ?? 0) - openSpan,
    color: parseHex(colours[index] ?? ""),
  }))
}

/**
 * The step a value falls in: the last whose lower bound it reaches, or -1
 * below the first bounded step (for example under 1 mm of rain).
 */
export function stepIndex(scale: readonly ScaleStep[], value: number): number {
  if (!Number.isFinite(value)) return -1
  let found = -1
  scale.forEach((step, index) => {
    if (step.from === null || value >= step.from) found = index
  })
  return found
}

/**
 * A MapLibre `step` expression colouring `input` by a scale: values below
 * the first bounded step take `below`. With an open first step the first
 * colour is the default.
 */
export function stepExpression(
  input: unknown,
  scale: readonly ScaleStep[],
  colours: readonly string[],
  below = "rgba(0, 0, 0, 0)",
): unknown[] {
  const openFirst = scale[0]?.from === null
  const expression: unknown[] = ["step", input, openFirst ? colours[0] : below]
  scale.forEach((step, index) => {
    if (step.from === null) return
    expression.push(step.from, colours[index])
  })
  return expression
}
