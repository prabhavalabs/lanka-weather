import { describe, expect, it } from "vitest"
import { overviewAction, sectionTag, shownCounts, shownItems } from "./present"
import type { OverviewItem, OverviewSection } from "./types"

function section(change: Partial<OverviewSection> = {}): OverviewSection {
  return {
    id: "rivers",
    group: "water",
    title: "Rivers at risk",
    summary: "",
    official: false,
    estimate: true,
    available: true,
    stale: false,
    at: null,
    source: "",
    counts: [{ key: "watch", label: "Watch", value: 0 }],
    items: [],
    target: null,
    ...change,
  }
}

const item = (id: string): OverviewItem => ({
  id,
  title: id,
  detail: "",
  tone: "neutral",
  estimate: false,
  stale: false,
  at: null,
  source: "",
  target: null,
})

describe("what a line opens", () => {
  it("opens the river and its stretch, with the layer on", () => {
    expect(overviewAction({ kind: "river", id: "deduru-oya", reach: 41402775 })).toEqual({
      enable: ["rivers"],
      selection: { kind: "river", id: "deduru-oya", reach: 41402775 },
      camera: "fit",
    })
  })

  it("opens a gauge's panel from its rain, with both layers on", () => {
    expect(overviewAction({ kind: "station", id: "thaldena", layer: "rainfall" })).toEqual({
      enable: ["rainfall", "river-levels"],
      selection: { kind: "station", id: "thaldena" },
      camera: "reveal",
    })
  })

  it("opens official warnings of one type", () => {
    expect(overviewAction({ kind: "warnings", id: "weather" }).selection).toEqual({
      kind: "warnings",
      type: "weather",
    })
    expect(overviewAction({ kind: "warnings" }).selection).toEqual({ kind: "warnings" })
  })

  it("colours regions by what the line is about", () => {
    const action = overviewAction({
      kind: "region",
      id: "LK-11",
      layer: "admin-regions",
      fill: "rain",
    })
    expect(action.enable).toEqual(["admin-regions"])
    expect(action.selection).toEqual({ kind: "region", id: "LK-11" })
    expect(action.fill).toBe("rain")
    expect(action.level).toBe("district")
  })

  it("opens a hazard with its layer on", () => {
    expect(overviewAction({ kind: "cyclone", id: "1001" })).toEqual({
      enable: ["cyclones"],
      selection: { kind: "cyclone", id: "1001" },
    })
    expect(overviewAction({ kind: "quake", id: "us1" }).enable).toEqual(["earthquakes"])
    expect(overviewAction({ kind: "landslide", id: "LK-2136" }).enable).toEqual([
      "landslide-warnings",
    ])
    expect(overviewAction({ kind: "reservoirs" }).selection).toEqual({ kind: "reservoirs" })
  })

  it("opens the weather at a place it can answer for, and only moves to one it cannot", () => {
    const near = overviewAction({ kind: "place", layer: "air-quality", lon: 80.2, lat: 7 })
    expect(near.selection?.kind).toBe("point")
    const far = overviewAction({ kind: "place", layer: "sea-conditions", lon: 86, lat: 4 })
    expect(far.selection).toBeUndefined()
    expect(far.reveal).toEqual([86, 4])
    expect(far.enable).toEqual(["sea-conditions"])
  })

  it("only switches a layer on when that is all there is", () => {
    expect(overviewAction({ kind: "layer", layer: "river-levels" })).toEqual({
      enable: ["river-levels"],
      reveal: undefined,
      fill: undefined,
    })
  })
})

describe("how a section reads", () => {
  it("offers the rest of a long list through its target", () => {
    const list = ["a", "b", "c", "d", "e", "f", "g"].map(item)
    const shown = shownItems(section({ id: "reservoirs", items: list }))
    expect(shown.items).toHaveLength(5)
    expect(shown.more).toBe(2)
    expect(shownItems(section({ id: "warnings", items: list })).more).toBe(0)
  })

  it("tallies rivers always, and landslides only when there are levels", () => {
    expect(shownCounts(section())).toHaveLength(1)
    expect(shownCounts(section({ id: "landslides" }))).toEqual([])
    expect(
      shownCounts(
        section({ id: "landslides", counts: [{ key: "level1", label: "Level 1", value: 3 }] }),
      ),
    ).toHaveLength(1)
    expect(shownCounts(section({ id: "reservoirs" }))).toEqual([])
  })

  it("marks what is official and what is an estimate", () => {
    expect(sectionTag(section({ official: true, estimate: false }))).toBe("Official")
    expect(sectionTag(section())).toBe("Estimate")
    expect(sectionTag(section({ estimate: false }))).toBeNull()
  })
})
