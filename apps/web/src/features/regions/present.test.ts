import { describe, expect, it } from "vitest"
import { ancestorId, districtsOf, regionFid, regionLevelOf, regionVariables } from "./present"
import type { RegionWeather } from "./types"

describe("region codes", () => {
  it("knows each level by its number of digits", () => {
    expect(regionLevelOf("LK-1")).toBe("province")
    expect(regionLevelOf("LK-11")).toBe("district")
    expect(regionLevelOf("LK-1103")).toBe("ds")
    expect(regionLevelOf("LK-1103005")).toBe("gn")
    expect(regionLevelOf("LK-110")).toBeNull()
  })

  it("turns codes into the tiles' numeric feature ids", () => {
    expect(regionFid("LK-1103005")).toBe(1103005)
    expect(regionFid("lk-11")).toBe(11)
  })

  it("finds ancestors from the code", () => {
    expect(ancestorId("LK-1103005", "ds")).toBe("LK-1103")
    expect(ancestorId("LK-1103005", "district")).toBe("LK-11")
    expect(ancestorId("LK-11", "province")).toBe("LK-1")
    expect(ancestorId("LK-11", "ds")).toBeNull()
  })
})

describe("districtsOf", () => {
  it("collects the distinct districts of tile features", () => {
    expect(
      districtsOf([
        { properties: { districtId: "LK-21" } },
        { properties: { districtId: "LK-11" } },
        { properties: { districtId: "LK-21" } },
        { properties: { districtId: "LK-2" } },
        { properties: {} },
        { properties: null },
      ]),
    ).toEqual(["LK-11", "LK-21"])
  })
})

describe("regionVariables", () => {
  const region: RegionWeather = {
    id: "LK-11",
    name: "Colombo",
    method: "cells",
    cells: 6,
    temperatureC: 24.6,
    temperatureMinC: 23,
    temperatureMaxC: 31.7,
    precipitationMm: 0.1,
    precipitationTodayMm: 8,
    windSpeedMaxKmh: 9.4,
    windGustsMaxKmh: 17.6,
  }

  it("reads each variable from a region", () => {
    expect(regionVariables.temperature.value(region)).toBe(24.6)
    expect(regionVariables.rain.value(region)).toBe(8)
    expect(regionVariables.wind.value(region)).toBe(9.4)
  })
})
