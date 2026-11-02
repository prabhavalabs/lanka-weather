import { windScale } from "@/lib/tokens"
import { KMH_PER_MS } from "./field"

/**
 * The particles' colours by speed: the wind legend's seven steps (calm to a
 * very severe cyclonic storm, lib/tokens `windScale`, in km/h), so the
 * particles and the legend agree. Speeds here are in m/s, as the field
 * gives them.
 */

/** Each band's lower bound in m/s. */
export const windBands: readonly number[] = windScale.map((step) => (step.from ?? 0) / KMH_PER_MS)

/** The theme token of each band, in order. */
export const windBandTokens = windScale.map((step) => step.token)

/** The band of a speed in m/s. */
export function windBand(speed: number): number {
  let band = 0
  while (band + 1 < windBands.length && speed >= windBands[band + 1]!) band += 1
  return band
}

/** A band's lower bound in km/h, for a legend: "0", "10", …, "118". */
export function bandLabel(index: number): string {
  return String(Math.round((windBands[index] ?? 0) * KMH_PER_MS))
}
