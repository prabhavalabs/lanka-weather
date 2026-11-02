import { describe, expect, it } from "vitest"
import { countBySeverity, floodSummary, formatWindow, gaugeName } from "./present"
import type { FloodForecastCollection, FloodSeverity } from "./types"

const gauge = (severity: FloodSeverity) => ({
  type: "Feature" as const,
  id: `g-${severity}`,
  geometry: { type: "Point" as const, coordinates: [80, 7] as [number, number] },
  properties: {
    kind: "gauge" as const,
    gaugeId: `hybas_${severity}`,
    gaugeSource: "HYBAS",
    siteName: "",
    river: "",
    qualityVerified: true,
    severity,
    trend: "rise" as const,
    issuedAt: "2026-11-02T13:23:42Z",
    forecastStart: "2026-11-07T00:00:00Z",
    forecastEnd: "2026-11-08T00:00:00Z",
    thresholds: null,
    fetchedAt: "2026-11-02T14:50:47Z",
    stale: false,
  },
})

const collection = (features: FloodForecastCollection["features"]): FloodForecastCollection => ({
  type: "FeatureCollection",
  features,
  meta: {
    source: "google-flood-forecasting",
    attribution: "",
    region: "LK",
    configured: true,
    fetchedAt: null,
    generatedAt: "",
    gauges: 0,
    areas: 0,
  },
})

describe("gaugeName", () => {
  it("prefers the site, then the river, then the short id", () => {
    expect(gaugeName({ gaugeId: "hybas_1", siteName: "Hanwella", river: "Kelani" })).toBe(
      "Hanwella",
    )
    expect(gaugeName({ gaugeId: "hybas_1", siteName: "", river: "Kelani" })).toBe("Kelani gauge")
    expect(gaugeName({ gaugeId: "hybas_4121155030" })).toBe("Gauge 4121155030")
  })
})

describe("formatWindow", () => {
  it("shows a one-day window as one day", () => {
    expect(formatWindow("2026-11-07T00:00:00Z", "2026-11-08T00:00:00Z")).toBe("Sat 7 Nov")
    expect(formatWindow("2026-11-07T00:00:00Z", "2026-11-09T00:00:00Z")).toBe(
      "Sat 7 Nov → Mon 9 Nov",
    )
    expect(formatWindow(null, null)).toBe("–")
  })
})

describe("floodSummary", () => {
  it("counts gauges above normal and areas", () => {
    expect(floodSummary(collection([gauge("no_flooding"), gauge("no_flooding")]))).toBe(
      "2 gauges · none at warning or above",
    )
    const area = {
      type: "Feature" as const,
      id: "a",
      geometry: { type: "MultiPolygon" as const, coordinates: [] },
      properties: { kind: "notification" as const, severity: "severe" as const },
    }
    expect(floodSummary(collection([gauge("severe"), gauge("above_normal"), area]))).toBe(
      "2 gauges · 2 at warning or above · 1 area",
    )
  })

  it("counts by severity", () => {
    expect(countBySeverity([gauge("extreme"), gauge("extreme"), gauge("unknown")])).toMatchObject({
      extreme: 2,
      unknown: 1,
      severe: 0,
    })
  })
})
