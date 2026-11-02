import { describe, expect, it } from "vitest"
import { forecastSteps, leadLabel, nearestStep, stepAt, STEP_MS } from "./forecast"

const at = (iso: string) => Date.parse(iso)

describe("nearestStep", () => {
  it("rounds to the 3-hourly steps in UTC", () => {
    expect(nearestStep(at("2026-10-19T17:28:00Z"))).toBe(at("2026-10-19T18:00:00Z"))
    expect(nearestStep(at("2026-10-19T16:29:59Z"))).toBe(at("2026-10-19T15:00:00Z"))
    expect(nearestStep(at("2026-10-19T22:30:00Z"))).toBe(at("2026-10-20T00:00:00Z"))
  })
})

describe("forecastSteps", () => {
  it("runs from now's step to three days ahead, within the run's reach", () => {
    const now = at("2026-11-02T15:25:00Z")
    // The 06 UTC run reaches 72 hours past its run time: 63 hours past now's step.
    const steps = forecastSteps(now, "2026-11-05T06:00:00Z")
    expect(steps[0]).toBe(at("2026-11-02T15:00:00Z"))
    expect(steps.at(-1)).toBe(at("2026-11-05T06:00:00Z"))
    expect(steps).toHaveLength(22)
    for (let index = 1; index < steps.length; index += 1) {
      expect(steps[index]! - steps[index - 1]!).toBe(STEP_MS)
    }
  })

  it("cuts a run that reaches further at three days", () => {
    const now = at("2026-11-02T15:25:00Z")
    expect(forecastSteps(now, "2026-11-30T00:00:00Z")).toHaveLength(25)
  })

  it("offers only now until a field says how far its run reaches", () => {
    const now = at("2026-10-19T17:28:00Z")
    expect(forecastSteps(now, null)).toEqual([at("2026-10-19T18:00:00Z")])
    expect(forecastSteps(now, "2026-10-19T12:00:00Z")).toEqual([at("2026-10-19T18:00:00Z")])
  })
})

describe("stepAt", () => {
  it("keeps an offset within the steps", () => {
    const steps = forecastSteps(at("2026-10-19T17:28:00Z"), "2026-10-20T03:00:00Z")
    expect(steps).toHaveLength(4)
    expect(stepAt(steps, 0)).toBe(steps[0])
    expect(stepAt(steps, 2)).toBe(at("2026-10-20T00:00:00Z"))
    expect(stepAt(steps, 40)).toBe(steps[3])
    expect(stepAt(steps, -1)).toBe(steps[0])
  })
})

describe("leadLabel", () => {
  it("says how far ahead a step is", () => {
    const now = at("2026-10-19T17:28:00Z")
    expect(leadLabel(at("2026-10-19T18:00:00Z"), now)).toBe("Now")
    expect(leadLabel(at("2026-10-19T21:00:00Z"), now)).toBe("+3 h")
    expect(leadLabel(at("2026-10-21T00:00:00Z"), now)).toBe("+1 d 6 h")
    expect(leadLabel(at("2026-10-22T18:00:00Z"), now)).toBe("+3 d")
  })
})
