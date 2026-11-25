import { describe, expect, it } from "vitest"
import type { MultiPolygon, Polygon } from "geojson"
import { asFocusGeometry, focusBounds, focusVeil, padBounds, signedArea } from "./focus"

const square = (x: number, y: number, size = 1) => [
  [x, y],
  [x + size, y],
  [x + size, y + size],
  [x, y + size],
  [x, y],
]

describe("focusVeil", () => {
  it("cuts the shape out of the world, the hole wound clockwise", () => {
    const polygon: Polygon = { type: "Polygon", coordinates: [square(80, 7)] }
    const veil = focusVeil(polygon)!
    const [outer, hole] = veil.geometry.coordinates
    expect(signedArea(outer!)).toBeGreaterThan(0)
    expect(signedArea(hole!)).toBeLessThan(0)
    expect(veil.geometry.coordinates).toHaveLength(2)
  })

  it("rewinds a hole that arrives clockwise already, and keeps it", () => {
    const ring = [...square(80, 7)].reverse()
    const veil = focusVeil({ type: "Polygon", coordinates: [ring] })!
    expect(veil.geometry.coordinates[1]).toEqual(ring)
  })

  it("cuts one hole per part of a multipolygon and leaves the parts' own holes veiled", () => {
    const multi: MultiPolygon = {
      type: "MultiPolygon",
      coordinates: [[square(80, 7, 2), square(80.5, 7.5, 0.5)], [square(81, 8)]],
    }
    const rings = focusVeil(multi)!.geometry.coordinates
    expect(rings).toHaveLength(3)
    for (const hole of rings.slice(1)) expect(signedArea(hole)).toBeLessThan(0)
  })

  it("draws nothing for a shape with no usable ring", () => {
    expect(focusVeil({ type: "MultiPolygon", coordinates: [] })).toBeNull()
    expect(focusVeil({ type: "Polygon", coordinates: [[[80, 7]]] })).toBeNull()
  })
})

describe("asFocusGeometry", () => {
  it("keeps areas and refuses points", () => {
    expect(asFocusGeometry({ type: "Polygon", coordinates: [square(80, 7)] })).not.toBeNull()
    expect(asFocusGeometry({ type: "Point", coordinates: [80, 7] })).toBeNull()
    expect(asFocusGeometry(undefined)).toBeNull()
  })
})

describe("focusBounds", () => {
  it("spans every part", () => {
    expect(
      focusBounds({ type: "MultiPolygon", coordinates: [[square(80, 7)], [square(81, 8, 0.5)]] }),
    ).toEqual([80, 7, 81.5, 8.5])
    expect(focusBounds({ type: "MultiPolygon", coordinates: [] })).toBeNull()
  })
})

describe("padBounds", () => {
  it("grows each side by the share of the span", () => {
    expect(padBounds([80, 7, 81, 8], 0.25)).toEqual([79.75, 6.75, 81.25, 8.25])
  })
})
