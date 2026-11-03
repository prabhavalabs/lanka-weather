import { describe, expect, it } from "vitest"
import type { ObservedRainCollection } from "../rainfall/types"
import {
  bandLabel,
  bandMinZoom,
  bandRange,
  countBands,
  gaugesSummary,
  seriesPath,
  shownAt,
  withoutLiveDuplicates,
} from "./present"
import type { RainGaugeCollection } from "./types"

describe("what shows at which zoom", () => {
  it("shows meaningful rain at country zoom and adds the rest as you zoom in", () => {
    const country = 7
    expect(shownAt("very_heavy", country)).toBe(true)
    expect(shownAt("heavy", country)).toBe(true)
    expect(shownAt("moderate", country)).toBe(true)
    expect(shownAt("light", country)).toBe(false)
    expect(shownAt("none", country)).toBe(false)
    expect(shownAt("light", 8.5)).toBe(true)
    expect(shownAt("none", 8.5)).toBe(false)
    expect(shownAt("none", 10)).toBe(true)
  })

  it("never brings an out-of-date gauge in earlier than a dry one", () => {
    expect(bandMinZoom.unknown).toBeGreaterThanOrEqual(bandMinZoom.none)
  })
})

describe("words", () => {
  it("names each band and its range in the language", () => {
    expect(bandLabel("very_heavy")).toBe("Very heavy")
    expect(bandRange("heavy")).toBe("50 to 100 mm")
    expect(bandLabel("heavy", "si")).toBe("තද වැසි")
    expect(bandLabel("none", "ta")).toBe("மழை இல்லை")
  })

  it("sums up the gauges", () => {
    const meta = { fresh: 258, total: 275, wet: 31 } as RainGaugeCollection["meta"]
    expect(gaugesSummary(meta)).toBe("258 of 275 current · 31 with 10 mm or more")
  })
})

describe("countBands", () => {
  it("counts each band", () => {
    expect(
      countBands([{ properties: { band: "light" } }, { properties: { band: "unknown" } }]),
    ).toMatchObject({ light: 1, unknown: 1, heavy: 0 })
  })
})

const point = (id: string, lon: number, lat: number) => ({
  type: "Feature" as const,
  id,
  geometry: { type: "Point" as const, coordinates: [lon, lat] as [number, number] },
  properties: { id } as ObservedRainCollection["features"][number]["properties"],
})

const gauge = (lon: number, lat: number, stale: boolean) =>
  ({
    type: "Feature",
    id: "g",
    geometry: { type: "Point", coordinates: [lon, lat] },
    properties: { stale },
  }) as RainGaugeCollection["features"][number]

describe("withoutLiveDuplicates", () => {
  const observed = {
    features: [point("beside", 80, 7), point("alone", 81, 8)],
  } as ObservedRainCollection

  it("drops an older point a current gauge stands beside, and nothing else", () => {
    const gauges = { features: [gauge(80.001, 7.001, false)] } as RainGaugeCollection
    const left = withoutLiveDuplicates(observed, gauges)
    expect(left?.features.map((feature) => feature.id)).toEqual(["alone"])
  })

  it("keeps a point whose neighbour is out of date", () => {
    const gauges = { features: [gauge(80.001, 7.001, true)] } as RainGaugeCollection
    expect(withoutLiveDuplicates(observed, gauges)?.features).toHaveLength(2)
  })

  it("passes through while either is missing", () => {
    expect(withoutLiveDuplicates(undefined, undefined)).toBeUndefined()
    expect(withoutLiveDuplicates(observed, undefined)).toBe(observed)
  })
})

describe("seriesPath", () => {
  it("scales time and rain to the chart", () => {
    const { points, peak } = seriesPath([
      { at: "2026-11-02T00:00:00Z", rain24hMm: 0 },
      { at: "2026-11-02T01:00:00Z", rain24hMm: 10 },
      { at: "2026-11-02T02:00:00Z", rain24hMm: 5 },
    ])
    expect(peak).toBe(10)
    expect(points[1]).toEqual({ x: 0.5, y: 1 })
  })

  it("is empty with no points", () => {
    expect(seriesPath([]).points).toEqual([])
  })
})
