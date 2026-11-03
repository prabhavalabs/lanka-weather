import { describe, expect, it } from "vitest"
import {
  liftedColour,
  methodLabel,
  rainImageLift,
  stepLabel,
  tilesInBounds,
  tileUrl,
} from "./present"

describe("tilesInBounds", () => {
  it("finds the nine collected tiles from their bounds", () => {
    const tiles = tilesInBounds([73.125, 0, 90, 16.636192], 6)
    expect(tiles).toHaveLength(9)
    expect(tiles).toContainEqual({ z: 6, x: 46, y: 30 })
    expect(tiles[0]).toEqual({ z: 6, x: 45, y: 29 })
    expect(tiles.at(-1)).toEqual({ z: 6, x: 47, y: 31 })
  })
})

describe("tileUrl", () => {
  it("fills the template", () => {
    expect(tileUrl("/v1/rain/tiles/{time}/{z}/{x}/{y}.png", "20261102T1330Z", 6, 46, 30)).toBe(
      "/v1/rain/tiles/20261102T1330Z/6/46/30.png",
    )
  })
})

describe("liftedColour", () => {
  it("leaves the image alone in the light theme", () => {
    expect(rainImageLift("light")).toBe(0)
    expect(liftedColour("#3f0000", rainImageLift("light"))).toBe("#3f0000")
  })

  it("lifts the darkest rain off a dark map, as the raster does", () => {
    const lift = rainImageLift("dark")
    expect(lift).toBeGreaterThan(0)
    expect(liftedColour("#000000", lift)).toBe("#4d4d4d")
    expect(liftedColour("#ffffff", lift)).toBe("#ffffff")
  })
})

describe("stepLabel", () => {
  it("shows the half hour in Sri Lanka time", () => {
    expect(stepLabel({ validAt: "2026-11-02T13:30:00Z", endsAt: "2026-11-02T14:00:00Z" })).toBe(
      "Mon 2 Nov 19:00–19:30",
    )
    expect(
      stepLabel({ validAt: "2026-11-02T13:30:00Z", endsAt: "2026-11-02T14:00:00Z" }, "ta"),
    ).toContain("19:00–19:30")
  })
})

describe("methodLabel", () => {
  it("says how a total was reached, in the language", () => {
    expect(methodLabel("sum_gauge")).toBe("Sum of hourly river gauge readings")
    expect(methodLabel("sum_3h", "si")).toContain("කාලගුණ විද්‍යා දෙපාර්තමේන්තුවේ")
  })
})
