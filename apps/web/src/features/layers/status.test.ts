import { describe, expect, it } from "vitest"
import { ApiError } from "@/lib/api"
import { count, layerStatus } from "./status"

const layer = { staleAfterSeconds: 3 * 3600 }
const now = new Date("2026-11-02T15:00:00Z")

describe("layerStatus", () => {
  it("says what the data holds and how old it is", () => {
    expect(
      layerStatus({ layer, parts: ["214 grid points"], observedAt: "2026-11-02T14:48:00Z", now }),
    ).toEqual({ state: "ready", summary: "214 grid points · updated 12 min ago", stale: false })
  })

  it("marks data older than the layer's limit as stale", () => {
    expect(
      layerStatus({ layer, parts: ["214 grid points"], observedAt: "2026-11-02T10:00:00Z", now }),
    ).toMatchObject({ summary: "214 grid points · updated 5 h ago", stale: true })
  })

  it("waits calmly while data is not collected yet", () => {
    expect(
      layerStatus({
        layer,
        parts: [],
        observedAt: null,
        error: new ApiError(503, "no_data", "weather has not been collected yet"),
      }),
    ).toEqual({
      state: "waiting",
      summary: "Data is still loading · collected shortly",
      stale: false,
    })
  })

  it("reports loading and failures without data", () => {
    expect(layerStatus({ layer, parts: [], observedAt: null, pending: true }).state).toBe("loading")
    expect(
      layerStatus({ layer, parts: [], observedAt: null, error: new Error("offline") }).state,
    ).toBe("error")
  })

  it("keeps showing data when a later refresh fails", () => {
    expect(
      layerStatus({
        layer,
        parts: ["10 gauges"],
        observedAt: "2026-11-02T14:00:00Z",
        error: new Error("offline"),
        now,
      }).state,
    ).toBe("ready")
  })
})

describe("count", () => {
  it("pluralises", () => {
    expect(count(1, "gauge")).toBe("1 gauge")
    expect(count(25, "district")).toBe("25 districts")
    expect(count(340, "DS division")).toBe("340 DS divisions")
  })
})

describe("layerStatus age label", () => {
  it("can call the age a model run's", () => {
    expect(
      layerStatus({
        layer: { staleAfterSeconds: 12 * 3600 },
        parts: ["NOAA GFS, 10 m"],
        observedAt: "2026-11-02T06:00:00Z",
        ageLabel: "run",
        now: new Date("2026-11-02T15:00:00Z"),
      }).summary,
    ).toBe("NOAA GFS, 10 m · run 9 h ago")
  })
})
