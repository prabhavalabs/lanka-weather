/**
 * Design tokens for code that cannot use CSS classes, such as MapLibre paint
 * properties. The values live in src/styles/theme.css; read them after the
 * theme class is set (MapLibre layers re-read them on every style load).
 */

/** The current value of a CSS custom property on <html>, e.g. "--status-alert". */
export function readToken(name: `--${string}`, element?: Element): string {
  if (typeof document === "undefined") return ""
  return getComputedStyle(element ?? document.documentElement)
    .getPropertyValue(name)
    .trim()
}

/** Reads several tokens at once, keyed by the given names. */
export function readTokens<K extends string>(names: Record<K, `--${string}`>): Record<K, string> {
  const style = typeof document === "undefined" ? null : getComputedStyle(document.documentElement)
  const values = {} as Record<K, string>
  for (const key of Object.keys(names) as K[]) {
    values[key] = style ? style.getPropertyValue(names[key]).trim() : ""
  }
  return values
}

/** A colour scale step: the lower bound it starts at and the token that colours it. */
export interface ScaleStep {
  /** Inclusive lower bound, or null for "anything below the next step". */
  from: number | null
  token: `--${string}`
}

/** Air temperature in °C. */
export const temperatureScale: ScaleStep[] = [
  { from: null, token: "--temp-1" },
  { from: 21, token: "--temp-2" },
  { from: 24, token: "--temp-3" },
  { from: 27, token: "--temp-4" },
  { from: 30, token: "--temp-5" },
  { from: 33, token: "--temp-6" },
  { from: 36, token: "--temp-7" },
]

/** Rainfall over 24 hours in mm. */
export const rainfallScale: ScaleStep[] = [
  { from: 1, token: "--rain-1" },
  { from: 10, token: "--rain-2" },
  { from: 25, token: "--rain-3" },
  { from: 50, token: "--rain-4" },
  { from: 100, token: "--rain-5" },
  { from: 150, token: "--rain-6" },
]

/** Rain in one hour in mm, for the current-conditions field. */
export const rainRateScale: ScaleStep[] = [
  { from: 0.1, token: "--rain-1" },
  { from: 0.5, token: "--rain-2" },
  { from: 1, token: "--rain-3" },
  { from: 2.5, token: "--rain-4" },
  { from: 5, token: "--rain-5" },
  { from: 10, token: "--rain-6" },
]

/** Relative humidity in %, dry to saturated. */
export const humidityScale: ScaleStep[] = [
  { from: null, token: "--humidity-1" },
  { from: 60, token: "--humidity-2" },
  { from: 70, token: "--humidity-3" },
  { from: 80, token: "--humidity-4" },
  { from: 90, token: "--humidity-5" },
]

/** Cloud cover in %: a veil that thickens, transparent when clear. */
export const cloudScale: ScaleStep[] = [
  { from: 0, token: "--cloud-1" },
  { from: 30, token: "--cloud-2" },
  { from: 60, token: "--cloud-3" },
  { from: 90, token: "--cloud-4" },
]

/** Wind speed in km/h; from 118 km/h a cyclone is "very severe" on the IMD scale. */
export const windScale: ScaleStep[] = [
  { from: null, token: "--wind-1" },
  { from: 10, token: "--wind-2" },
  { from: 20, token: "--wind-3" },
  { from: 30, token: "--wind-4" },
  { from: 50, token: "--wind-5" },
  { from: 70, token: "--wind-6" },
  { from: 118, token: "--wind-7" },
]

/** The token for a value on a scale (the last step whose lower bound it reaches), or null below the scale. */
export function scaleToken(scale: ScaleStep[], value: number): `--${string}` | null {
  let token: `--${string}` | null = null
  for (const step of scale) if (step.from === null || value >= step.from) token = step.token
  return token
}
