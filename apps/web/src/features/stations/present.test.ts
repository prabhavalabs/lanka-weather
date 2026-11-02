import { describe, expect, it } from "vitest"
import {
  feedNotice,
  formatRain,
  formatTemperature,
  markerTemperature,
  rainBars,
  rainLabel,
  temperaturePaths,
} from "./present"
import type { MetMeta } from "./types"

const meta = (overrides: Partial<MetMeta> = {}): MetMeta => ({
  attribution: "",
  sourceUrl: "",
  generatedAt: "",
  observed: true,
  threeHourly: {
    source: "a",
    newestObservedAt: "2026-10-29T03:00:00Z",
    fetchedAt: null,
    stale: true,
  },
  daily: { source: "b", newestObservedAt: "2026-10-23T03:00:00Z", fetchedAt: null, stale: true },
  stale: true,
  ...overrides,
})

const now = new Date("2026-11-02T03:00:00Z")

describe("feedNotice", () => {
  it("says the readings are not current, with the age of the newest", () => {
    expect(feedNotice(meta(), now)).toContain("4 days ago")
    expect(feedNotice(meta(), now)).toContain("Not current")
  })

  it("says the daily report has taken over from a stopped 3-hourly feed", () => {
    const daily = {
      source: "b",
      newestObservedAt: "2026-11-02T02:00:00Z",
      fetchedAt: null,
      stale: false,
    }
    expect(feedNotice(meta({ stale: false, daily }), now)).toContain("daily report")
  })

  it("says nothing while the 3-hourly feed is current", () => {
    const threeHourly = {
      source: "a",
      newestObservedAt: "2026-11-02T02:00:00Z",
      fetchedAt: null,
      stale: false,
    }
    expect(feedNotice(meta({ stale: false, threeHourly }), now)).toBeNull()
  })

  it("says when nothing has been collected", () => {
    const none = { source: "a", newestObservedAt: null, fetchedAt: null, stale: true }
    expect(feedNotice(meta({ threeHourly: none, daily: none }), now)).toContain("No Department")
  })
})

describe("what a station shows", () => {
  it("draws the reading, else the daily maximum", () => {
    expect(markerTemperature({ temperatureC: 28.1, tempMaxC: 31 })).toBe(28.1)
    expect(markerTemperature({ temperatureC: null, tempMaxC: 31 })).toBe(31)
    expect(markerTemperature({ temperatureC: null, tempMaxC: null })).toBeNull()
  })

  it("names the rain's period", () => {
    expect(rainLabel({ reading: "3h", rainHours: 3 })).toBe("Rain, 3 h")
    expect(rainLabel({ reading: "daily", rainHours: 24 })).toBe("Rain, 24 h to 08:30")
  })

  it("formats missing values as a dash", () => {
    expect(formatTemperature(null)).toBe("–")
    expect(formatTemperature(16.1)).toBe("16.1 °C")
    expect(formatRain(null)).toBe("–")
    expect(formatRain(2.44)).toBe("2.4 mm")
  })
})

describe("history", () => {
  const readings = [
    { observedAt: "2026-10-28T00:00:00Z", temperatureC: 24, humidityPct: 90, rainMm: 0 },
    { observedAt: "2026-10-28T03:00:00Z", temperatureC: null, humidityPct: 90, rainMm: 2 },
    { observedAt: "2026-10-28T06:00:00Z", temperatureC: 29, humidityPct: 80, rainMm: 4 },
  ]

  it("draws one line through readings three hours apart", () => {
    const shape = temperaturePaths(readings, 100, 100)
    expect(shape.paths).toHaveLength(1)
    expect(shape.dots).toEqual([])
    expect(shape.yMin).toBeLessThan(24)
    expect(shape.yMax).toBeGreaterThan(29)
  })

  it("breaks the line where the source has no reports", () => {
    const shape = temperaturePaths(
      [
        { observedAt: "2026-10-24T00:00:00Z", temperatureC: 24, humidityPct: null, rainMm: null },
        { observedAt: "2026-10-24T03:00:00Z", temperatureC: 25, humidityPct: null, rainMm: null },
        { observedAt: "2026-10-28T06:00:00Z", temperatureC: 29, humidityPct: null, rainMm: null },
      ],
      100,
      100,
    )
    expect(shape.paths).toHaveLength(1)
    expect(shape.dots).toHaveLength(1)
    expect(shape.dots[0]!.x).toBe(100)
  })

  it("has nothing to draw without temperatures", () => {
    expect(temperaturePaths([], 100, 100).paths).toEqual([])
  })

  it("scales rain bars to the wettest reading across the span", () => {
    const { bars, peak } = rainBars(readings)
    expect(peak).toBe(4)
    expect(bars).toEqual([
      { x: 0.5, height: 0.5, mm: 2 },
      { x: 1, height: 1, mm: 4 },
    ])
    expect(rainBars([]).bars).toEqual([])
  })
})
