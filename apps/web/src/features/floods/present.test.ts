import { afterEach, describe, expect, it } from "vitest"
import { setLocale } from "@/lib/i18n"
import { basinStateLabels } from "./basins"
import {
  areaLikelihood,
  countBySeverity,
  floodSeverityLabels,
  floodSummary,
  formatWindow,
  gaugeName,
  pastFloodMagnitude,
  pastFloodName,
} from "./present"
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
      "2 gauges · none raised",
    )
    const area = {
      type: "Feature" as const,
      id: "a",
      geometry: { type: "MultiPolygon" as const, coordinates: [] },
      properties: { kind: "notification" as const, severity: "severe" as const },
    }
    expect(floodSummary(collection([gauge("severe"), gauge("above_normal"), area]))).toBe(
      "2 gauges · 2 raised · 1 area",
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

describe("in Sinhala and Tamil", () => {
  afterEach(() => setLocale("en"))

  it("keeps Flood Hub's English words, and reads records in the current language", () => {
    expect(floodSeverityLabels.severe).toBe("Danger")
    expect(floodSeverityLabels.above_normal).toBe("Warning")
    expect(basinStateLabels.no_forecast).toBe("No forecast")
    setLocale("si")
    expect(floodSeverityLabels.severe).toBe("අනතුර")
    expect(basinStateLabels.no_forecast).toBe("පුරෝකථනයක් නැත")
    setLocale("ta")
    expect(floodSeverityLabels.above_normal).toBe("எச்சரிக்கை")
    expect(Object.keys(floodSeverityLabels)).toContain("extreme")
  })

  it("summarises gauges in the language", () => {
    setLocale("ta")
    expect(floodSummary(collection([gauge("no_flooding")]))).toBe(
      "1 அளவீட்டு நிலையம் · எதுவும் உயரவில்லை",
    )
    setLocale("si")
    expect(gaugeName({ gaugeId: "hybas_4121155030" })).toBe("මිනුම් ස්ථානය 4121155030")
  })

  it("names an inundation level, with Google's own word in the note", () => {
    const likelihood = areaLikelihood({ kind: "inundation_high", severity: "" })
    expect(likelihood.label).toBe("Very high")
    expect(likelihood.note).toContain('"high"')
    setLocale("si")
    expect(areaLikelihood({ kind: "flash_likely", severity: "likely" }).label).toBe("ඇතිවිය හැකි")
  })
})

describe("past floods", () => {
  afterEach(() => setLocale("en"))

  const event = { name: "Kelani flood, 2016 (major)", river: "Kelani", year: 2016 }

  it("names an event from its river and year, keeping the river's name", () => {
    expect(pastFloodName({ ...event, magnitude: "major" })).toBe("Kelani flood, 2016 (major)")
    expect(pastFloodName({ ...event, magnitude: null })).toBe("Kelani flood, 2016")
    expect(pastFloodName({ ...event, river: "", magnitude: null })).toBe(event.name)
    setLocale("ta")
    expect(pastFloodName({ ...event, magnitude: null })).toBe("Kelani வெள்ளம், 2016")
  })

  it("labels how it was mapped", () => {
    expect(pastFloodMagnitude("minor")).toBe("Minor")
    expect(pastFloodMagnitude("partial")).toBe("Partial")
    setLocale("si")
    expect(pastFloodMagnitude("major")).toBe("මහ ගංවතුර")
  })
})
