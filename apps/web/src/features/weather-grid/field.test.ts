import { describe, expect, it } from "vitest"
import { colourTable } from "@/lib/colour"
import {
  countValues,
  gridBounds,
  imageSize,
  latitudeOf,
  mercatorY,
  paintField,
  rasterise,
  sampleGrid,
} from "./field"
import type { GridSpec } from "./types"

// 3 × 2 nodes at 1° steps from 80° E, 6° N; rows south to north.
const grid: GridSpec = { id: "test", lon0: 80, lat0: 6, step: 1, nx: 3, ny: 2 }
//            80E  81E  82E
const values = [10, 20, null, /* 6° N */ 30, 40, 50 /* 7° N */]

describe("gridBounds and countValues", () => {
  it("spans the outermost nodes", () => {
    expect(gridBounds(grid)).toEqual([80, 6, 82, 7])
    expect(gridBounds({ id: "lk", lon0: 79.4, lat0: 5.6, step: 0.2, nx: 15, ny: 23 })).toEqual([
      79.4,
      5.6,
      79.4 + 14 * 0.2,
      5.6 + 22 * 0.2,
    ])
  })

  it("counts nodes with values", () => {
    expect(countValues(values)).toBe(5)
    expect(countValues(undefined)).toBe(0)
  })
})

describe("sampleGrid", () => {
  it("is exact on nodes and bilinear between them", () => {
    expect(sampleGrid(grid, values, 80, 6)).toBe(10)
    expect(sampleGrid(grid, values, 81, 7)).toBe(40)
    expect(sampleGrid(grid, values, 80.5, 6.5)).toBe(25)
    expect(sampleGrid(grid, values, 80.25, 6)).toBe(12.5)
  })

  it("reweights around nodes without a value", () => {
    // The cell 81–82° E lacks its south-east node: the other three share the weight.
    expect(sampleGrid(grid, values, 81.5, 6.5)).toBeCloseTo((20 + 40 + 50) / 3, 6)
    // On the edge shared with the full cell the two cells agree.
    expect(sampleGrid(grid, values, 81, 6.5)).toBe(30)
  })

  it("has nothing off the grid or where all four nodes are empty", () => {
    expect(sampleGrid(grid, values, 79.9, 6.5)).toBeNull()
    expect(sampleGrid(grid, values, 81, 7.01)).toBeNull()
    expect(sampleGrid(grid, [null, null, null, null, null, null], 80.5, 6.5)).toBeNull()
  })
})

describe("mercator", () => {
  it("round-trips latitudes", () => {
    for (const lat of [0, 5.6, 10, -33.9, 60])
      expect(latitudeOf(mercatorY(lat))).toBeCloseTo(lat, 9)
  })
})

describe("rasterise", () => {
  it("samples pixel centres from north to south", () => {
    const field = rasterise(grid, values, 2, 2)
    // Top-left pixel centre: 80.5° E, just north of 6.75° N in Mercator.
    expect(field[0]).toBeGreaterThan(30)
    expect(field[0]).toBeLessThan(40)
    // Bottom-right pixel: 81.5° E near 6.25° N, beside the empty node.
    expect(field[3]).toBeGreaterThan(20)
    // Rows are north first.
    expect(field[0]!).toBeGreaterThan(field[2]!)
  })

  it("leaves NaN where there is no value", () => {
    const field = rasterise(grid, [null, null, null, null, null, null], 3, 3)
    expect(Array.from(field).every(Number.isNaN)).toBe(true)
  })
})

describe("paintField", () => {
  it("colours values from the table and leaves gaps transparent", () => {
    const table = colourTable(
      [
        { value: 0, color: [0, 0, 255, 1] },
        { value: 100, color: [255, 0, 0, 0.5] },
      ],
      101,
    )
    const out = paintField(new Float32Array([0, 100, Number.NaN]), table, new Uint8ClampedArray(12))
    expect(Array.from(out.slice(0, 4))).toEqual([0, 0, 255, 255])
    expect(Array.from(out.slice(4, 8))).toEqual([255, 0, 0, 128])
    expect(out[11]).toBe(0)
  })
})

describe("imageSize", () => {
  it("keeps pixels about square for Sri Lanka's grid", () => {
    const [width, height] = imageSize({
      id: "lk",
      lon0: 79.4,
      lat0: 5.6,
      step: 0.2,
      nx: 15,
      ny: 23,
    })
    expect(width).toBe(280)
    // 4.4° of latitude near 8° N is a little taller than 4.4° of longitude.
    expect(height).toBeGreaterThan(440)
    expect(height).toBeLessThan(446)
  })
})
