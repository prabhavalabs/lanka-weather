import { describe, expect, it } from "vitest"
import { warningFocus } from "./focus"
import type { Bulletin, DistrictHighlight } from "./types"

const bulletin = (type: Bulletin["type"], ids: string[]) =>
  ({ type, districts: ids.map((id) => ({ id, name: id })) }) as unknown as Bulletin

const district = (id: string, landslideLevel = 0): DistrictHighlight => ({
  id,
  name: id,
  landslideLevel,
  mentioned: [],
})

const data = {
  inForce: [bulletin("weather", ["LK-11", "LK-12"]), bulletin("water_level", ["LK-12", "LK-31"])],
  districts: [district("LK-11"), district("LK-21", 2), district("LK-92", 1)],
}

describe("warningFocus", () => {
  it("keeps a chosen district, whatever the data", () => {
    expect(warningFocus({ district: "LK-21" }, undefined)).toEqual(["LK-21"])
  })

  it("keeps the districts one kind of warning names", () => {
    expect(warningFocus({ type: "weather" }, data)).toEqual(["LK-11", "LK-12"])
    expect(warningFocus({ type: "water_level" }, data)).toEqual(["LK-12", "LK-31"])
  })

  it("adds the districts with a landslide level for landslide warnings", () => {
    expect(warningFocus({ type: "landslide" }, data)).toEqual(["LK-21", "LK-92"])
  })

  it("keeps every highlighted district for all warnings", () => {
    expect(warningFocus({}, data)).toEqual(["LK-11", "LK-21", "LK-92"])
  })

  it("focuses on nothing for a warning for the whole island", () => {
    expect(
      warningFocus({ type: "weather" }, { inForce: [bulletin("weather", [])], districts: [] }),
    ).toEqual([])
    expect(warningFocus({}, undefined)).toEqual([])
  })
})
