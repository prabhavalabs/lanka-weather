import { describe, expect, it } from "vitest"
import { changeText, groupCanals, shownState, stateLabel } from "./present"
import type { CanalGauge } from "./types"

const gauge = (name: string, group: string, state: string, stale = false): CanalGauge => ({
  id: name,
  name,
  group,
  state,
  stale,
  levelM: stale ? null : 1,
  observedAt: null,
  change3hM: null,
  last24h: [],
})

describe("stateLabel", () => {
  it("keeps the source's words, and an unknown word as it comes", () => {
    expect(stateLabel("higher")).toBe("Higher than usual")
    expect(stateLabel("offline")).toBe("Not reporting")
    expect(stateLabel("surging")).toBe("surging")
  })
})

describe("shownState", () => {
  it("does not let an old reading keep its word", () => {
    expect(shownState(gauge("a", "", "normal", true))).toBe("offline")
    expect(shownState(gauge("a", "", "higher", false))).toBe("higher")
  })
})

describe("groupCanals", () => {
  it("puts raised groups first and gauges with no group last", () => {
    const groups = groupCanals([
      gauge("Quiet", "North Canal", "normal"),
      gauge("Stray", "", "normal"),
      gauge("High", "Kelani River", "higher"),
      gauge("Silent", "Kelani River", "offline"),
      gauge("Calm", "Kelani River", "normal"),
    ])
    expect(groups.map((group) => group.name)).toEqual([
      "Kelani River",
      "North Canal",
      "Other gauges",
    ])
    expect(groups[0]!.gauges.map((item) => item.name)).toEqual(["High", "Calm", "Silent"])
  })
})

describe("changeText", () => {
  it("signs a change and leaves out a missing one", () => {
    expect(changeText(0.04)).toBe("+0.04 m in 3 h")
    expect(changeText(-0.042)).toBe("−0.04 m in 3 h")
    expect(changeText(null)).toBeNull()
  })
})
