import { describe, expect, it } from "vitest"
import {
  anyUnverified,
  areaKinds,
  areaStyle,
  basinFillOpacity,
  basinIdsByState,
  countBasins,
  subbasinIdsByState,
  topArea,
} from "./basins"

describe("basin tints", () => {
  it("leaves a basin with no forecast untinted, so it cannot read as calm", () => {
    expect(basinFillOpacity.no_forecast).toBe(0)
    expect(basinFillOpacity.no_flooding).toBeGreaterThan(0)
    expect(basinFillOpacity.above_normal).toBeGreaterThan(basinFillOpacity.no_flooding)
    expect(basinFillOpacity.severe).toBeGreaterThan(basinFillOpacity.above_normal)
    expect(basinFillOpacity.extreme).toBeGreaterThan(basinFillOpacity.severe)
  })
})

describe("grouping", () => {
  it("groups feature ids by state", () => {
    const ids = basinIdsByState([
      { fid: 1, state: "extreme" },
      { fid: 2, state: "no_forecast" },
      { fid: 3, state: "extreme" },
    ] as never)
    expect(ids.extreme).toEqual([1, 3])
    expect(ids.no_forecast).toEqual([2])
    expect(ids.severe).toEqual([])
    expect(
      subbasinIdsByState([{ fid: 4121599890, state: "severe", onlyUnverified: false }]).severe,
    ).toEqual([4121599890])
  })

  it("counts basins", () => {
    const counts = countBasins([{ state: "severe" }, { state: "severe" }, { state: "no_forecast" }])
    expect(counts.severe).toBe(2)
    expect(counts.no_forecast).toBe(1)
    expect(counts.extreme).toBe(0)
  })

  it("notices basins resting only on unverified gauges", () => {
    expect(anyUnverified([{ onlyUnverified: false }])).toBe(false)
    expect(anyUnverified([{ onlyUnverified: false }, { onlyUnverified: true }])).toBe(true)
  })
})

describe("flood area styles", () => {
  it("grades inundation by how likely the water is", () => {
    expect(areaStyle.inundation_high.opacity).toBeGreaterThan(areaStyle.inundation_medium.opacity)
    expect(areaStyle.inundation_medium.opacity).toBeGreaterThan(areaStyle.inundation_low.opacity)
  })

  it("draws the likeliest water on top", () => {
    expect(areaStyle.inundation_high.rank).toBeGreaterThan(areaStyle.inundation_medium.rank)
    expect(areaStyle.inundation_medium.rank).toBeGreaterThan(areaStyle.inundation_low.rank)
    expect(areaStyle.inundation_low.rank).toBeGreaterThan(areaStyle.notification.rank)
  })

  it("covers every kind Google publishes", () => {
    expect(areaKinds).toHaveLength(7)
    for (const kind of areaKinds) expect(areaStyle[kind].width).toBeGreaterThan(0)
  })
})

describe("flash-flood areas", () => {
  it("draws highly likely over likely, and both over inundation", () => {
    expect(areaStyle.flash_highly_likely.rank).toBeGreaterThan(areaStyle.flash_likely.rank)
    expect(areaStyle.flash_likely.rank).toBeGreaterThan(areaStyle.inundation_high.rank)
    expect(areaStyle.inundation_high.rank).toBeGreaterThan(areaStyle.inundation_low.rank)
  })

  it("names the topmost of overlapping areas", () => {
    const areas = [
      { id: 1, properties: { kind: "flash_likely" } },
      { id: 2, properties: { kind: "flash_highly_likely" } },
      { id: 3, properties: { kind: "inundation_low" } },
    ]
    expect(topArea(areas)?.id).toBe(2)
    expect(topArea([])).toBeUndefined()
  })
})
