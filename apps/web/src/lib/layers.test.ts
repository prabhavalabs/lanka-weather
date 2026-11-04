import { describe, expect, it } from "vitest"
import { ApiError } from "./api"
import { groupByCategory, isStale, layerQueryOptions, queryTimings, type Layer } from "./layers"

const layer = (overrides: Partial<Layer> = {}): Layer => ({
  id: "river-levels",
  title: "River levels",
  description: "",
  category: "hydrology",
  kind: "geojson",
  status: "planned",
  sources: ["irrigation-gauges"],
  refreshSeconds: 900,
  staleAfterSeconds: 7200,
  attribution: "Irrigation Department",
  ...overrides,
})

describe("queryTimings", () => {
  it("polls at the refresh interval and goes stale at half of it", () => {
    expect(queryTimings(layer())).toEqual({ staleTime: 450_000, refetchInterval: 900_000 })
  })

  it("never polls faster than once a minute", () => {
    expect(queryTimings(layer({ refreshSeconds: 5 })).refetchInterval).toBe(60_000)
  })
})

describe("isStale", () => {
  const now = new Date("2026-11-02T12:00:00Z")

  it("is fresh within the stale limit", () => {
    expect(isStale(layer(), new Date("2026-11-02T10:30:00Z"), now)).toBe(false)
  })

  it("is stale beyond the stale limit", () => {
    expect(isStale(layer(), new Date("2026-11-02T09:59:59Z"), now)).toBe(true)
  })
})

describe("groupByCategory", () => {
  it("orders groups and drops empty ones", () => {
    const groups = groupByCategory([
      layer({ id: "wind", category: "weather" }),
      layer({ id: "cyclones", category: "hazard" }),
    ])
    expect(groups.map((group) => group.category)).toEqual(["hazard", "weather"])
  })
})

describe("layerQueryOptions", () => {
  const notReady = new ApiError(503, "no_data", "weather has not been collected yet")

  it("polls at the catalogue's interval once data has arrived", () => {
    const options = layerQueryOptions(layer({ refreshSeconds: 3600 }))
    expect(options.staleTime).toBe(1_800_000)
    expect(options.refetchInterval({ state: { error: null } })).toBe(3_600_000)
  })

  it("asks every minute while the data is not collected yet", () => {
    const options = layerQueryOptions(layer({ refreshSeconds: 3600 }))
    expect(options.refetchInterval({ state: { error: notReady } })).toBe(60_000)
    expect(
      options.refetchInterval({ state: { error: new ApiError(500, "internal", "boom") } }),
    ).toBe(3_600_000)
  })
})
