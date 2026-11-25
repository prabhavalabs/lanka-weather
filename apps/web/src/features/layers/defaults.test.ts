import { describe, expect, it } from "vitest"
import { defaultLayers, migrateLayers, warningLayers } from "./defaults"

const oldDefaults = [
  "landslide-warnings",
  "official-warnings",
  "earthquakes",
  "cyclones",
  "rivers",
  "river-levels",
  "flood-forecast",
  "reservoirs",
  "rain-gauges",
  "admin-regions",
]

describe("defaultLayers", () => {
  it("shows every warning and nothing else", () => {
    expect(defaultLayers).toEqual(warningLayers)
    expect(defaultLayers).toContain("official-warnings")
    expect(defaultLayers).not.toContain("rivers")
  })
})

describe("migrateLayers", () => {
  it("moves the untouched old defaults, in any order, to the new ones", () => {
    expect(migrateLayers([...oldDefaults].reverse(), oldDefaults)).toEqual(defaultLayers)
  })

  it("keeps a choice someone made", () => {
    const chosen = ["official-warnings", "landslide-warnings", "earthquakes", "cyclones", "wind"]
    expect(
      migrateLayers(chosen, ["official-warnings", "landslide-warnings", "earthquakes", "cyclones"]),
    ).toEqual(chosen)
  })

  it("does not switch back on a warning someone was offered and turned off", () => {
    expect(migrateLayers(["official-warnings"], ["official-warnings", "cyclones"])).toEqual([
      "official-warnings",
      "landslide-warnings",
      "earthquakes",
    ])
  })

  it("leaves the flood forecast as chosen", () => {
    expect(migrateLayers(["wind"], warningLayers)).toEqual(["wind"])
  })
})
