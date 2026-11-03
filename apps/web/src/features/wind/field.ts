/**
 * The wind field the map animates (ADR 0007): NOAA GFS's wind 10 m above
 * ground on a regular latitude/longitude grid, as `/v1/weather/wind` sends
 * it, and how to read it at any point. Ported from TerraGuard, whose
 * format the API keeps. Pure, so it is tested without a browser.
 */
import { compassPoint as formatCompassPoint } from "@/lib/format"
import { getLocale, translateIn, type Locale } from "@/lib/i18n"

/** The API's answer, as far as the map needs it. */
export type WindResponse = {
  source?: string
  model?: string
  runAt: string
  forecastHour: number
  validAt: string
  firstValidAt?: string
  lastValidAt: string
  stepHours: number
  grid: { nx: number; ny: number; lon0: number; lat0: number; dlon: number; dlat: number }
  scale: number
  u: string
  v: string
  attribution: string
  fetchedAt?: string
}

/** A decoded field: value j × nx + i lies at lat0 + j × dlat, lon0 + i × dlon, in m/s. */
export type WindField = {
  runAt: string
  forecastHour: number
  validAt: string
  lastValidAt: string
  attribution: string
  nx: number
  ny: number
  lon0: number
  lat0: number
  dlon: number
  dlat: number
  /** Whether the grid goes round the world, so columns wrap. */
  global: boolean
  u: Float32Array
  v: Float32Array
}

/** Little-endian int16s from base64, times scale. */
function component(encoded: string, scale: number, count: number): Float32Array {
  const binary = atob(encoded)
  if (binary.length !== count * 2)
    throw new Error(`Wind field has ${binary.length / 2} values for ${count} points`)
  const values = new Float32Array(count)
  for (let index = 0; index < count; index += 1) {
    const raw = binary.charCodeAt(2 * index) | (binary.charCodeAt(2 * index + 1) << 8)
    values[index] = ((raw << 16) >> 16) * scale
  }
  return values
}

export function decodeWind(response: WindResponse): WindField {
  const { nx, ny, lon0, lat0, dlon, dlat } = response.grid
  const count = nx * ny
  return {
    runAt: response.runAt,
    forecastHour: response.forecastHour,
    validAt: response.validAt,
    lastValidAt: response.lastValidAt,
    attribution: response.attribution,
    nx,
    ny,
    lon0,
    lat0,
    dlon,
    dlat,
    global: Math.abs(nx * dlon) >= 359.999,
    u: component(response.u, response.scale, count),
    v: component(response.v, response.scale, count),
  }
}

/**
 * The wind at a point, bilinear between the four grid points around it,
 * written into `out` as [u, v] in m/s. Longitudes wrap on a global grid, so
 * the antimeridian and 0°/360° are seamless. False outside the grid.
 */
export function sampleInto(
  field: WindField,
  lon: number,
  lat: number,
  out: [number, number] | Float32Array,
): boolean {
  const row = (lat - field.lat0) / field.dlat
  if (!(row >= 0 && row <= field.ny - 1)) return false
  let column = (lon - field.lon0) / field.dlon
  if (field.global) column = ((column % field.nx) + field.nx) % field.nx
  else if (!(column >= 0 && column <= field.nx - 1)) return false
  const x0 = Math.floor(column)
  const y0 = Math.floor(row)
  const x1 = field.global ? (x0 + 1) % field.nx : Math.min(x0 + 1, field.nx - 1)
  const y1 = Math.min(y0 + 1, field.ny - 1)
  const tx = column - x0
  const ty = row - y0
  const a = y0 * field.nx
  const b = y1 * field.nx
  const blend = (values: Float32Array) =>
    (values[a + x0]! * (1 - tx) + values[a + x1]! * tx) * (1 - ty) +
    (values[b + x0]! * (1 - tx) + values[b + x1]! * tx) * ty
  out[0] = blend(field.u)
  out[1] = blend(field.v)
  return true
}

/** The wind at a point as [u, v] in m/s, or null outside the grid. */
export function sampleWind(field: WindField, lon: number, lat: number): [number, number] | null {
  const out: [number, number] = [0, 0]
  return sampleInto(field, lon, lat, out) ? out : null
}

/** Where the wind blows from, in degrees clockwise from north, as forecasters give it. */
export function windFrom(u: number, v: number): number {
  const degrees = (Math.atan2(-u, -v) * 180) / Math.PI
  return (degrees + 360) % 360
}

/** One of the 16 compass points, in the current language ("NE" in English). */
export function compassPoint(degrees: number, locale: Locale = getLocale()): string {
  return formatCompassPoint(degrees, 16, locale)
}

export const KMH_PER_MS = 3.6
export const KNOTS_PER_MS = 3600 / 1852

/** Below this the wind has no meaningful direction. */
const CALM_MS = 0.5

/** "35 km/h (19 kt) from the NE", or "Calm", in the current language. */
export function windText(u: number, v: number, locale: Locale = getLocale()): string {
  const speed = Math.hypot(u, v)
  if (speed < CALM_MS) return translateIn(locale, "grid.wind.calm")
  return translateIn(locale, "grid.wind.text", {
    kmh: Math.round(speed * KMH_PER_MS),
    knots: Math.round(speed * KNOTS_PER_MS),
    direction: compassPoint(windFrom(u, v), locale),
  })
}
