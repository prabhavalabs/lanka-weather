import { describe, expect, it } from "vitest"
import {
  ageBand,
  earthquakeParts,
  formatDistance,
  magnitudeRadius,
  strongest,
  tsunamiNote,
  withAge,
} from "./present"
import type { EarthquakeCollection } from "./types"

const now = new Date("2026-11-02T12:00:00Z")

function collection(magnitudes: number[]): EarthquakeCollection {
  return {
    type: "FeatureCollection",
    meta: {
      source: "s",
      attribution: "a",
      windowDays: 30,
      fetchedAt: null,
      generatedAt: "",
      stale: false,
      count: magnitudes.length,
    },
    features: magnitudes.map((magnitude, index) => ({
      type: "Feature",
      geometry: { type: "Point", coordinates: [90, 5] },
      properties: {
        id: `event-${index}`,
        magnitude,
        magType: "mb",
        place: "somewhere",
        time: "2026-11-02T06:00:00Z",
        depthKm: 10,
        distanceKm: 1200,
        url: "",
        alert: "",
        tsunami: false,
        significance: 1,
        status: "reviewed",
        updatedAt: "",
        fetchedAt: "",
        source: "usgs-earthquakes",
      },
    })),
  }
}

describe("earthquakes", () => {
  it("shades by age", () => {
    expect(ageBand("2026-11-02T06:00:00Z", now)).toBe("day")
    expect(ageBand("2026-10-30T06:00:00Z", now)).toBe("week")
    expect(ageBand("2026-10-10T06:00:00Z", now)).toBe("month")
  })

  it("adds the age to each feature without changing the source", () => {
    const data = collection([5])
    const shown = withAge(data, now)
    expect((shown.features[0]!.properties as unknown as { age: string }).age).toBe("day")
    expect(data.features[0]!.properties).not.toHaveProperty("age")
  })

  it("sizes by magnitude within limits", () => {
    expect(magnitudeRadius(5)).toBeLessThan(magnitudeRadius(7))
    expect(magnitudeRadius(0)).toBe(3)
    expect(magnitudeRadius(10)).toBe(26)
  })

  it("finds the strongest and summarises", () => {
    const data = collection([4.5, 6.2, 5])
    expect(strongest(data)?.properties.magnitude).toBe(6.2)
    expect(strongest(undefined)).toBeNull()
    expect(earthquakeParts(data)).toEqual(["3 in 30 days", "strongest M6.2"])
    expect(earthquakeParts(collection([]))).toEqual(["None of M4.5+ in 30 days"])
  })

  it("groups long distances", () => {
    expect(formatDistance(1605)).toBe("1,605 km")
  })

  it("says the flag is not a warning and points to the DMC", () => {
    expect(tsunamiNote).toContain("not a warning for Sri Lanka")
    expect(tsunamiNote).toContain("Disaster Management Centre")
  })
})
