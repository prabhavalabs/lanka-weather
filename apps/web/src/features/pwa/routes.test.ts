import { describe, expect, it } from "vitest"
import { cacheLimits, cacheNames, cacheRuleFor, navigationDenylist } from "./routes"

const origin = "https://lankaweather.example"
const rule = (url: string) => cacheRuleFor(new URL(url, origin), origin)

describe("cacheRuleFor", () => {
  it.each([
    ["/v1/tiles/regions/7/95/61.pbf", "api-tiles"],
    ["/v1/tiles/rivers/9/381/245.pbf?v=abc", "api-tiles"],
    ["/v1/tiles/past-floods/8/190/122.pbf", "api-tiles"],
    ["/v1/rain/tiles/20261102T1330Z/6/46/30.png", "rain-tiles"],
    ["https://tiles.openfreemap.org/styles/positron", "basemap-styles"],
    ["https://tiles.openfreemap.org/planet", "basemap-styles"],
    ["https://tiles.openfreemap.org/planet/20261029_001001_pt/7/95/61.pbf", "basemap-tiles"],
    ["https://tiles.openfreemap.org/fonts/Noto%20Sans%20Regular/0-255.pbf", "basemap-assets"],
    ["https://tiles.openfreemap.org/sprites/ofm_f384/ofm.json", "basemap-assets"],
    ["/v1/layers", "api-data"],
    ["/v1/rivers/levels", "api-data"],
    ["/v1/weather/wind?t=2026-11-03T06:00:00.000Z", "api-data"],
    ["/v1/overview", "api-data"],
  ] as const)("%s → %s", (url, expected) => {
    expect(rule(url)).toBe(expected)
  })

  it.each([
    "/v1/places/search?q=kandy",
    "/v1/push/subscriptions",
    "/healthz",
    "/assets/index-abc.js",
    "/",
    "https://example.com/v1/layers",
    "https://tiles.example.org/planet/1/1/1.pbf",
  ])("leaves %s alone", (url) => {
    expect(rule(url)).toBeNull()
  })

  it("bounds every cache by count and age", () => {
    for (const [name, limit] of Object.entries(cacheLimits)) {
      expect(limit.maxEntries, name).toBeGreaterThan(0)
      expect(limit.maxEntries, name).toBeLessThanOrEqual(5000)
      expect(limit.maxAgeSeconds, name).toBeLessThanOrEqual(90 * 24 * 60 * 60)
      expect(cacheNames[name as keyof typeof cacheNames]).toMatch(/^lw-.+-v\d+$/)
    }
  })

  it("keeps the API out of the navigation fallback", () => {
    for (const path of ["/v1/layers", "/api/push/subscriptions/renew", "/healthz"])
      expect(
        navigationDenylist.some((pattern) => pattern.test(path)),
        path,
      ).toBe(true)
    expect(navigationDenylist.some((pattern) => pattern.test("/"))).toBe(false)
  })
})
