import { describe, expect, it } from "vitest"
import { firstHit, hitOrder, orderedHits, type HitQuery } from "./hits"

const order = hitOrder({
  points: ["gauges", "quakes"],
  rivers: ["river-lines"],
  water: ["reservoir-water"],
  areas: ["flood-areas", "landslides", "official"],
  past: ["past-floods"],
  basins: ["basins"],
  regions: ["regions"],
})

/** A pointer over the given layers, whatever order they are drawn in. */
function under(...layers: string[]): HitQuery<string> {
  return (asked) => layers.filter((layer) => asked.includes(layer))
}

describe("what a click or hover lands on", () => {
  it.each([
    ["a gauge in a landslide area", ["landslides", "gauges"], "gauges"],
    ["an earthquake over everything", ["regions", "basins", "quakes"], "quakes"],
    ["a river under a landslide area", ["landslides", "river-lines"], "river-lines"],
    ["a river under a flood area", ["flood-areas", "river-lines"], "river-lines"],
    ["a river under an official district", ["official", "river-lines"], "river-lines"],
    [
      "a reservoir's water in a landslide area",
      ["landslides", "reservoir-water"],
      "reservoir-water",
    ],
    ["a river through a reservoir", ["reservoir-water", "river-lines"], "river-lines"],
    ["a flood area over a landslide area", ["landslides", "flood-areas"], "flood-areas"],
    ["a landslide area inside an official district", ["official", "landslides"], "landslides"],
    ["a landslide area over a past flood", ["past-floods", "landslides"], "landslides"],
    ["a past flood over a basin", ["basins", "past-floods"], "past-floods"],
    ["a basin over a region", ["regions", "basins"], "basins"],
    ["a region alone", ["regions"], "regions"],
  ])("%s", (_, layers, want) => {
    expect(firstHit(order, under(...layers))?.feature).toBe(want)
  })

  it("finds nothing over nothing", () => {
    expect(firstHit(order, under())).toBeUndefined()
  })

  it("is the same list for a click that passes over a hit it cannot use", () => {
    const hits = [...orderedHits(order, under("regions", "basins", "landslides"))]
    expect(hits.map((hit) => hit.feature)).toEqual(["landslides", "basins", "regions"])
    expect(hits.map((hit) => hit.group)).toEqual(["area", "basin", "region"])
  })

  it("asks for points and lines with slack, and areas exactly", () => {
    const asked: [string[], boolean][] = []
    firstHit(order, (layers, near) => {
      asked.push([layers, near])
      return []
    })
    expect(asked[0]).toEqual([["gauges", "quakes"], true])
    expect(asked[1]).toEqual([["river-lines"], true])
    expect(asked.slice(2).every(([, near]) => !near)).toBe(true)
  })

  it("stops asking once a hover has what it needs", () => {
    let queries = 0
    firstHit(order, () => {
      queries++
      return ["gauges"]
    })
    expect(queries).toBe(1)
  })

  it("skips a group with no layers", () => {
    const empty = hitOrder({
      points: [],
      rivers: [],
      water: [],
      areas: [],
      past: [],
      basins: [],
      regions: ["regions"],
    })
    const asked: string[][] = []
    firstHit(empty, (layers) => {
      asked.push(layers)
      return []
    })
    expect(asked).toEqual([["regions"]])
  })
})
