/**
 * The weather grid as a smooth field: values between nodes by bilinear
 * interpolation, and the field drawn into an image whose rows follow Web
 * Mercator, so it lines up with the map at every latitude. Pure, so it is
 * tested without a browser.
 */
import { tableOffset, type ColourTable } from "@/lib/colour"
import type { GridSpec } from "./types"

const RADIANS = Math.PI / 180

/** The grid's extent through its outermost nodes: [west, south, east, north]. */
export function gridBounds(grid: GridSpec): [number, number, number, number] {
  return [
    grid.lon0,
    grid.lat0,
    grid.lon0 + (grid.nx - 1) * grid.step,
    grid.lat0 + (grid.ny - 1) * grid.step,
  ]
}

/** How many nodes carry a value. */
export function countValues(values: readonly (number | null)[] | undefined): number {
  if (!values) return 0
  let count = 0
  for (const value of values) if (value !== null && Number.isFinite(value)) count += 1
  return count
}

/**
 * The value at a point, bilinear between the four nodes around it. Nodes
 * without a value (open sea) are left out and the others reweighted, so the
 * field runs on smoothly to the coast; null when none of the four has one
 * or the point is off the grid.
 */
export function sampleGrid(
  grid: GridSpec,
  values: readonly (number | null)[],
  lon: number,
  lat: number,
): number | null {
  const fx = (lon - grid.lon0) / grid.step
  const fy = (lat - grid.lat0) / grid.step
  if (!(fx >= 0 && fx <= grid.nx - 1 && fy >= 0 && fy <= grid.ny - 1)) return null
  const i0 = Math.min(Math.floor(fx), grid.nx - 2)
  const j0 = Math.min(Math.floor(fy), grid.ny - 2)
  const tx = fx - i0
  const ty = fy - j0
  let total = 0
  let weight = 0
  const add = (i: number, j: number, w: number) => {
    if (w <= 0) return
    const value = values[j * grid.nx + i]
    if (value === null || value === undefined || !Number.isFinite(value)) return
    total += value * w
    weight += w
  }
  add(i0, j0, (1 - tx) * (1 - ty))
  add(i0 + 1, j0, tx * (1 - ty))
  add(i0, j0 + 1, (1 - tx) * ty)
  add(i0 + 1, j0 + 1, tx * ty)
  return weight > 0 ? total / weight : null
}

export function mercatorY(lat: number): number {
  return Math.log(Math.tan(Math.PI / 4 + (lat * RADIANS) / 2))
}

export function latitudeOf(y: number): number {
  return (2 * Math.atan(Math.exp(y)) - Math.PI / 2) / RADIANS
}

/**
 * The field sampled at the centres of a `width` × `height` image spanning
 * the grid's bounds, top row north. Rows are spaced evenly in Web Mercator,
 * as the map stretches an image between its corners. NaN where there is no
 * value.
 */
export function rasterise(
  grid: GridSpec,
  values: readonly (number | null)[],
  width: number,
  height: number,
): Float32Array {
  const [west, south, east, north] = gridBounds(grid)
  const top = mercatorY(north)
  const bottom = mercatorY(south)
  const out = new Float32Array(width * height)
  for (let row = 0; row < height; row += 1) {
    const lat = latitudeOf(top - ((row + 0.5) / height) * (top - bottom))
    for (let column = 0; column < width; column += 1) {
      const lon = west + ((column + 0.5) / width) * (east - west)
      out[row * width + column] = sampleGrid(grid, values, lon, lat) ?? Number.NaN
    }
  }
  return out
}

/** Colours a rasterised field into RGBA pixels; NaN is transparent. */
export function paintField(field: Float32Array, table: ColourTable, out: Uint8ClampedArray) {
  for (let index = 0; index < field.length; index += 1) {
    const value = field[index]!
    const at = index * 4
    if (Number.isNaN(value)) {
      out[at + 3] = 0
      continue
    }
    const offset = tableOffset(table, value)
    out[at] = table.rgba[offset]!
    out[at + 1] = table.rgba[offset + 1]!
    out[at + 2] = table.rgba[offset + 2]!
    out[at + 3] = table.rgba[offset + 3]!
  }
  return out
}

/** Image size for a grid: `pixelsPerDegree` across, rows to keep pixels square in Mercator. */
export function imageSize(grid: GridSpec, pixelsPerDegree = 100): [number, number] {
  const [west, south, east, north] = gridBounds(grid)
  const width = Math.max(2, Math.round((east - west) * pixelsPerDegree))
  const height = Math.max(
    2,
    Math.round(((mercatorY(north) - mercatorY(south)) / ((east - west) * RADIANS)) * width),
  )
  return [width, height]
}
