import { describe, expect, it } from "vitest"
import type { BasinsDocument } from "../river-risk/types"
import type { FloodForecastCollection, FloodSeverity } from "./types"
import { basinsForView, defaultFloodView, floodsForView } from "./view"

const gauge = (id: string, severity: FloodSeverity, qualityVerified: boolean) =>
  ({
    type: "Feature",
    id,
    geometry: { type: "Point", coordinates: [80, 7] },
    properties: { kind: "gauge", gaugeId: id, severity, qualityVerified },
  }) as unknown as FloodForecastCollection["features"][number]

const area = (id: string, kind: string, lowerConfidence = false) =>
  ({
    type: "Feature",
    id,
    geometry: { type: "MultiPolygon", coordinates: [] },
    properties: { id, kind, severity: "severe", lowerConfidence },
  }) as unknown as FloodForecastCollection["features"][number]

const collection = {
  type: "FeatureCollection",
  features: [
    gauge("calm", "no_flooding", true),
    gauge("raised", "severe", true),
    gauge("calm-unverified", "no_flooding", false),
    gauge("raised-unverified", "severe", false),
    area("likely", "flash_likely"),
    area("highly", "flash_highly_likely"),
    area("event", "significant_event"),
    area("water", "inundation_high"),
    area("water-unverified", "inundation_low", true),
    area("notice", "notification"),
  ],
} as FloodForecastCollection

const ids = (view: typeof defaultFloodView) =>
  floodsForView(collection, view).features.map((feature) => feature.id)

describe("floodsForView", () => {
  it("hides lower-confidence gauges and their areas until Extended coverage is on", () => {
    expect(ids(defaultFloodView)).toEqual([
      "calm",
      "raised",
      "likely",
      "highly",
      "event",
      "water",
      "notice",
    ])
    expect(ids({ ...defaultFloodView, extended: true })).toContain("raised-unverified")
    expect(ids({ ...defaultFloodView, extended: true })).toContain("water-unverified")
  })

  it("has a working toggle for each kind", () => {
    expect(ids({ ...defaultFloodView, normal: false })).not.toContain("calm")
    expect(ids({ ...defaultFloodView, normal: false })).toContain("raised")
    expect(ids({ ...defaultFloodView, flash: false })).not.toContain("highly")
    expect(ids({ ...defaultFloodView, events: false })).not.toContain("event")
    expect(ids({ ...defaultFloodView, inundation: false })).not.toContain("water")
  })
})

describe("basinsForView", () => {
  const document = {
    basins: [
      {
        fid: 1,
        state: "severe",
        verifiedState: "no_flooding",
        onlyUnverified: true,
        gauges: [
          { id: "a", verified: true },
          { id: "b", verified: false },
        ],
        areas: [{ id: "x", lowerConfidence: true }],
      },
    ],
    subbasins: [{ fid: 2, state: "severe", verifiedState: "no_forecast", onlyUnverified: true }],
  } as unknown as BasinsDocument

  it("reads verified evidence alone with Extended coverage off", () => {
    const shown = basinsForView(document, defaultFloodView)
    expect(shown.basins[0]!.state).toBe("no_flooding")
    expect(shown.basins[0]!.onlyUnverified).toBe(false)
    expect(shown.basins[0]!.gauges.map((gauge) => gauge.id)).toEqual(["a"])
    expect(shown.basins[0]!.areas).toEqual([])
    expect(shown.subbasins[0]!.state).toBe("no_forecast")
  })

  it("keeps everything, marked, with it on", () => {
    expect(basinsForView(document, { ...defaultFloodView, extended: true })).toBe(document)
  })
})
