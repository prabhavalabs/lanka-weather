import { describe, expect, it } from "vitest"
import { slotPlan } from "./slots"

describe("slotPlan", () => {
  it("puts fills under the first line and overlays under the trailing labels (positron)", () => {
    const plan = slotPlan([
      { id: "background", type: "background" },
      { id: "park", type: "fill", "source-layer": "park" },
      { id: "water", type: "fill", "source-layer": "water" },
      { id: "landuse_residential", type: "fill", "source-layer": "landuse" },
      { id: "waterway", type: "line", "source-layer": "waterway" },
      { id: "building", type: "fill", "source-layer": "building" },
      { id: "boundary_2", type: "line", "source-layer": "boundary" },
      { id: "water_name", type: "symbol", "source-layer": "water_name" },
      { id: "label_city", type: "symbol", "source-layer": "place" },
    ])
    expect(plan).toEqual({
      fillsBefore: "waterway",
      overlaysBefore: "water_name",
      waterId: "water",
    })
  })

  it("skips labels mixed in among the roads (dark)", () => {
    const plan = slotPlan([
      { id: "background", type: "background" },
      { id: "water", type: "fill", "source-layer": "water" },
      { id: "waterway", type: "line", "source-layer": "waterway" },
      { id: "water_name", type: "symbol", "source-layer": "water_name" },
      { id: "highway_minor", type: "line", "source-layer": "transportation" },
      { id: "boundary_state", type: "line", "source-layer": "boundary" },
      { id: "place_village", type: "symbol", "source-layer": "place" },
    ])
    expect(plan.fillsBefore).toBe("waterway")
    expect(plan.overlaysBefore).toBe("place_village")
  })

  it("puts everything on top of a style with no labels", () => {
    const plan = slotPlan([{ id: "background", type: "background" }])
    expect(plan).toEqual({ fillsBefore: undefined, overlaysBefore: undefined, waterId: undefined })
  })
})
