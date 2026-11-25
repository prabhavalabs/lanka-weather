import { describe, expect, it } from "vitest"
import { homeFor, homeSelection, isHome, parseHome } from "./home"

const kandy = { lat: 7.2906, lon: 80.6337 }

describe("parseHome", () => {
  it("reads a saved place and rounds it as the address does", () => {
    expect(parseHome({ lat: 7.290612, lon: 80.633721, name: " Kandy " })).toEqual({
      lat: 7.2906,
      lon: 80.6337,
      name: "Kandy",
    })
  })

  it("keeps a region home's id", () => {
    expect(parseHome({ ...kandy, name: "Kandy District", region: "LK-21" })?.region).toBe("LK-21")
  })

  it("refuses what is missing, malformed or outside Sri Lanka", () => {
    expect(parseHome(null)).toBeNull()
    expect(parseHome("Kandy")).toBeNull()
    expect(parseHome({ lat: "7.29", lon: 80.63, name: "Kandy" })).toBeNull()
    expect(parseHome({ lat: Number.NaN, lon: 80.63, name: "Kandy" })).toBeNull()
    expect(parseHome({ lat: 51.5, lon: -0.12, name: "London" })).toBeNull()
  })
})

describe("homeFor and homeSelection", () => {
  it("brings a point home back as that point, with its name", () => {
    const home = homeFor({ kind: "point", ...kandy }, kandy, "Kandy")!
    expect(homeSelection(home)).toEqual({ kind: "point", ...kandy, name: "Kandy" })
  })

  it("brings a region home back as the region", () => {
    const home = homeFor({ kind: "region", id: "LK-21" }, kandy, "Kandy District")!
    expect(homeSelection(home)).toEqual({ kind: "region", id: "LK-21" })
  })
})

describe("isHome", () => {
  const home = parseHome({ ...kandy, name: "Kandy" })

  it("matches the saved point and nothing else", () => {
    expect(isHome(home, { kind: "point", ...kandy }, kandy)).toBe(true)
    expect(isHome(home, { kind: "point", lat: 6.9, lon: 79.86 }, { lat: 6.9, lon: 79.86 })).toBe(
      false,
    )
    expect(isHome(null, { kind: "point", ...kandy }, kandy)).toBe(false)
  })

  it("matches a region home by its id, not by its label point", () => {
    const region = parseHome({ ...kandy, name: "Kandy District", region: "LK-21" })
    expect(isHome(region, { kind: "region", id: "LK-21" }, kandy)).toBe(true)
    expect(isHome(region, { kind: "point", ...kandy }, kandy)).toBe(false)
  })
})
