import { describe, expect, it } from "vitest"
import { polygonDistance, polygonSdf, sdfAlpha, type Point } from "./sdf"

const square: Point[] = [
  [0, 0],
  [10, 0],
  [10, 10],
  [0, 10],
]

describe("polygonDistance", () => {
  it("is negative inside, zero on the edge and positive outside", () => {
    expect(polygonDistance(5, 5, square)).toBe(-5)
    expect(polygonDistance(10, 5, square)).toBe(0)
    expect(polygonDistance(13, 14, square)).toBe(5)
  })
})

describe("sdfAlpha", () => {
  it("puts the edge at 0.75 and fades outward", () => {
    expect(sdfAlpha(0)).toBe(191)
    expect(sdfAlpha(-2)).toBe(255)
    expect(sdfAlpha(6)).toBe(0)
    expect(sdfAlpha(2)).toBe(128)
  })
})

describe("polygonSdf", () => {
  it("draws a white image whose alpha is the distance field", () => {
    const image = polygonSdf(
      12,
      square.map(([x, y]) => [x + 1, y + 1] as Point),
    )
    expect(image.width).toBe(12)
    expect(image.data).toHaveLength(12 * 12 * 4)
    const alphaAt = (x: number, y: number) => image.data[(y * 12 + x) * 4 + 3]
    expect(alphaAt(6, 6)).toBe(255)
    expect(alphaAt(0, 6)!).toBeLessThan(191)
    expect(image.data[0]).toBe(255)
  })
})
