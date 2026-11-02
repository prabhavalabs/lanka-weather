import { describe, expect, it } from "vitest"
import { chartGeometry } from "./chart"

const readings = [
  { observedAt: "2026-11-02T10:00:00Z", waterLevel: 1 },
  { observedAt: "2026-11-02T11:00:00Z", waterLevel: 2 },
  { observedAt: "2026-11-02T12:00:00Z", waterLevel: 1.5 },
]

describe("chartGeometry", () => {
  it("spans the plot horizontally from the first to the last reading", () => {
    const geometry = chartGeometry(readings, [], 300, 100)
    expect(geometry.path.startsWith("M0.0,")).toBe(true)
    expect(geometry.last?.x).toBe(300)
  })

  it("keeps thresholds inside the vertical range", () => {
    const geometry = chartGeometry(
      readings,
      [{ label: "Major flood", value: 7, color: "red" }],
      300,
      100,
    )
    expect(geometry.yMax).toBeGreaterThan(7)
    expect(geometry.yMin).toBeLessThan(1)
    const threshold = geometry.thresholds[0]!
    expect(threshold.y).toBeGreaterThan(0)
    expect(threshold.y).toBeLessThan(100)
  })

  it("draws no line for a single reading", () => {
    const geometry = chartGeometry(readings.slice(0, 1), [], 300, 100)
    expect(geometry.path).toBe("")
    expect(geometry.last).not.toBeNull()
  })

  it("handles no data", () => {
    expect(chartGeometry([], [], 300, 100)).toEqual({
      path: "",
      thresholds: [],
      yMax: 1,
      yMin: 0,
      last: null,
    })
  })
})
