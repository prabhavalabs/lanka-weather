import { describe, expect, it } from "vitest"
import {
  browserFamily,
  type Capabilities,
  defaultPreferences,
  isIOS,
  keyBytes,
  originalTopics,
  problem,
  readPreferences,
  readStoredAlerts,
  sameKey,
  shortDistrictName,
  statusOf,
  topics,
} from "./present"

const device: Capabilities = { push: true, ios: false, installed: false, permission: "default" }

describe("statusOf", () => {
  it("asks iPhone and iPad tabs to install first", () => {
    expect(
      statusOf({
        capabilities: { ...device, push: false, ios: true },
        serverEnabled: null,
        subscribed: null,
      }),
    ).toBe("ios-install")
    expect(
      statusOf({
        capabilities: { ...device, ios: true, installed: true },
        serverEnabled: true,
        subscribed: false,
      }),
    ).toBe("off")
  })

  it("says what is missing, in order", () => {
    expect(
      statusOf({ capabilities: { ...device, push: false }, serverEnabled: true, subscribed: null }),
    ).toBe("unsupported")
    expect(statusOf({ capabilities: device, serverEnabled: null, subscribed: null })).toBe(
      "checking",
    )
    expect(statusOf({ capabilities: device, serverEnabled: false, subscribed: null })).toBe(
      "disabled",
    )
    expect(
      statusOf({
        capabilities: { ...device, permission: "denied" },
        serverEnabled: true,
        subscribed: null,
      }),
    ).toBe("denied")
    expect(
      statusOf({
        capabilities: device,
        serverEnabled: true,
        subscribed: false,
        workerMissing: true,
      }),
    ).toBe("unavailable")
    expect(
      statusOf({
        capabilities: { ...device, permission: "granted" },
        serverEnabled: true,
        subscribed: true,
      }),
    ).toBe("on")
  })
})

describe("preferences", () => {
  it("defaults to the official and serious topics for the whole country", () => {
    const preferences = defaultPreferences()
    expect(preferences.topics).toEqual([
      "official",
      "landslide",
      "river",
      "heavy_rain",
      "flood_risk",
      "flood_forecast",
      "cyclone",
      "earthquake",
    ])
    expect(preferences.topics).not.toContain("landslide_level1")
    expect(preferences.topics).not.toContain("reservoir")
    expect(preferences.minSeverity).toBe("warning")
    expect(preferences.areas.country).toBe(true)
  })

  it("reads stored preferences defensively", () => {
    expect(readPreferences(null)).toEqual(defaultPreferences())
    const read = readPreferences({
      topics: ["river", "gossip", "official"],
      minSeverity: "extreme",
      areas: {
        country: false,
        districts: ["LK-91", "Colombo"],
        near: { lat: 6.68279, lon: 80.39951, radiusKm: 7 },
      },
    })
    expect(read.topics).toEqual(["official", "river"])
    expect(read.minSeverity).toBe("warning")
    expect(read.areas.districts).toEqual(["LK-91"])
    expect(read.areas.near).toEqual({ lat: 6.68, lon: 80.4, radiusKm: 10 })
    expect(
      readPreferences({ areas: { near: { lat: 13, lon: 80, radiusKm: 10 } } }).areas.near,
    ).toBeNull()
  })

  it("needs a topic and an area", () => {
    const preferences = defaultPreferences()
    expect(problem(preferences)).toBeNull()
    expect(problem({ ...preferences, topics: [] })).toBe("topics")
    expect(problem({ ...preferences, areas: { country: false, districts: [], near: null } })).toBe(
      "areas",
    )
    expect(
      problem({ ...preferences, areas: { country: false, districts: ["LK-11"], near: null } }),
    ).toBeNull()
  })
})

describe("readStoredAlerts", () => {
  const before = {
    preferences: {
      topics: ["official", "landslide", "river", "flood_forecast", "cyclone", "earthquake"],
      minSeverity: "warning",
      areas: { country: true, districts: [], near: null },
    },
    enabled: true,
  }

  it("turns on default topics added since settings without a known list were saved", () => {
    const read = readStoredAlerts(before)
    expect(read.preferences.topics).toEqual([
      "official",
      "landslide",
      "river",
      "heavy_rain",
      "flood_risk",
      "flood_forecast",
      "cyclone",
      "earthquake",
    ])
    expect(read.enabled).toBe(true)
    expect(read.known).toEqual([...topics])
  })

  it("leaves off an original topic the person turned off", () => {
    const read = readStoredAlerts({
      ...before,
      preferences: { ...before.preferences, topics: ["official"] },
    })
    expect(read.preferences.topics).toEqual(["official", "heavy_rain", "flood_risk"])
    // Opt-in topics are never added.
    expect(read.preferences.topics).not.toContain("reservoir")
    expect(read.preferences.topics).not.toContain("landslide_level1")
  })

  it("keeps a new topic off once the person has seen it and turned it off", () => {
    const repaired = readStoredAlerts(before)
    const chosen = {
      ...repaired,
      preferences: {
        ...repaired.preferences,
        topics: repaired.preferences.topics.filter((topic) => topic !== "heavy_rain"),
      },
    }
    const again = readStoredAlerts(JSON.parse(JSON.stringify(chosen)))
    expect(again.preferences.topics).not.toContain("heavy_rain")
    expect(again.preferences.topics).toContain("flood_risk")
  })

  it("adds only the topics missing from a known list", () => {
    const read = readStoredAlerts({
      ...before,
      known: [...originalTopics, "heavy_rain"],
    })
    expect(read.preferences.topics).toContain("flood_risk")
    expect(read.preferences.topics).not.toContain("heavy_rain")
  })

  it("starts with the defaults when nothing is stored", () => {
    for (const value of [null, undefined, "x", 3]) {
      const read = readStoredAlerts(value)
      expect(read.preferences).toEqual(defaultPreferences())
      expect(read.enabled).toBe(false)
      expect(read.known).toEqual([...topics])
    }
  })
})

describe("devices", () => {
  it("knows an iPad that calls itself a Mac", () => {
    expect(isIOS("Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X)", "iPhone", 5)).toBe(true)
    expect(isIOS("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", "MacIntel", 5)).toBe(true)
    expect(isIOS("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", "MacIntel", 0)).toBe(false)
  })

  it("picks the instructions for the browser", () => {
    expect(browserFamily("Mozilla/5.0 (Linux; Android 14) Chrome/130.0 Mobile Safari/537.36")).toBe(
      "android",
    )
    expect(browserFamily("Mozilla/5.0 (Macintosh) Gecko/20100101 Firefox/132.0")).toBe("firefox")
    expect(
      browserFamily("Mozilla/5.0 (Windows NT 10.0) Chrome/130.0 Safari/537.36 Edg/130.0"),
    ).toBe("chrome")
    expect(browserFamily("Mozilla/5.0 (Macintosh) Version/18.1 Safari/605.1.15")).toBe("safari")
  })

  it("decodes the server key and compares it", () => {
    const key = keyBytes(
      "BP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A8",
    )
    expect(key.length).toBe(65)
    expect(key[0]).toBe(4)
    expect(sameKey(key.buffer, key)).toBe(true)
    expect(sameKey(new Uint8Array(65).buffer, key)).toBe(false)
    expect(sameKey(null, key)).toBe(true)
  })
})

describe("shortDistrictName", () => {
  it("drops the cut word district", () => {
    expect(shortDistrictName("අනුරාධපුර දිස්ත්‍")).toBe("අනුරාධපුර")
    expect(shortDistrictName("அம்பாந்தோட்டை மாவ")).toBe("அம்பாந்தோட்டை")
    expect(shortDistrictName("Ratnapura")).toBe("Ratnapura")
  })
})
