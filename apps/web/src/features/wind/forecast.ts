/**
 * The wind layer's forecast times (ADR 0007): every 3 hours, from the step
 * nearest now to three days ahead, no further than the newest model run
 * reaches. Ported from TerraGuard. Pure, so it is tested without a browser.
 */

const HOUR = 3_600_000
export const STEP_MS = 3 * HOUR
/** The API keeps forecast hours 0 to 72 of each run. */
export const AHEAD_MS = 72 * HOUR

/** The forecast step nearest a time: steps fall on 00, 03, …, 21 UTC. */
export function nearestStep(now: number): number {
  return Math.round(now / STEP_MS) * STEP_MS
}

/**
 * The times offered, as epoch milliseconds, the first nearest now. Until the
 * first field says how far its run reaches, only now is offered.
 */
export function forecastSteps(now: number, lastValidAt: string | null | undefined): number[] {
  const first = nearestStep(now)
  const reach = lastValidAt ? Date.parse(lastValidAt) : Number.NaN
  const last = Math.min(first + AHEAD_MS, Number.isNaN(reach) ? first : reach)
  const steps: number[] = []
  for (let time = first; time <= last; time += STEP_MS) steps.push(time)
  return steps.length ? steps : [first]
}

/** The time `offset` steps after now's, kept within the steps offered. */
export function stepAt(steps: number[], offset: number): number {
  return steps[Math.max(0, Math.min(steps.length - 1, Math.round(offset)))]!
}

/** "Now", "+3 h", "+2 d 6 h": how far a step is from now's. */
export function leadLabel(step: number, now: number): string {
  const hours = Math.round((step - nearestStep(now)) / HOUR)
  if (hours <= 0) return "Now"
  const days = Math.floor(hours / 24)
  const rest = hours % 24
  if (!days) return `+${hours} h`
  return rest ? `+${days} d ${rest} h` : `+${days} d`
}
