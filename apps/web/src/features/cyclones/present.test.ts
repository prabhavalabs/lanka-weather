import { describe, expect, it } from "vitest"
import { boundsWithSriLanka, cycloneSummary, evaluationNote, isCalm, positions } from "./present"
import type { CycloneCollection, CyclonePosition } from "./types"

const system = {
  kind: "position",
  eventId: "1001346",
  name: "MURASU-26",
  alertLevel: "Red",
  windKmh: 120,
  severityText: "Hurricane/Typhoon > 74 mph (maximum wind speed of 120 km/h)",
  country: "India",
  startedAt: "",
  latestAt: "",
  advisorySource: "JTWC",
  reportUrl: "",
  lon: 80.7,
  lat: 14.7,
  distanceKm: 545,
  stale: false,
  source: "gdacs",
  sourceTime: "",
  fetchedAt: "",
} satisfies CyclonePosition

const meta = {
  source: "gdacs",
  attribution: "a",
  fetchedAt: null,
  generatedAt: "",
  stale: false,
  count: 0,
}

function collection(
  features: CycloneCollection["features"],
  note = "",
  status: CycloneCollection["meta"]["status"] = "none",
): CycloneCollection {
  return { type: "FeatureCollection", features, meta: { ...meta, status, note } }
}

const position: CycloneCollection["features"][number] = {
  type: "Feature",
  geometry: { type: "Point", coordinates: [80.7, 14.7] },
  properties: system,
}
const track: CycloneCollection["features"][number] = {
  type: "Feature",
  geometry: {
    type: "LineString",
    coordinates: [
      [80, 14],
      [80.7, 14.7],
    ],
  },
  properties: { kind: "track", eventId: "1001346", name: "MURASU-26", alertLevel: "Red" },
}

describe("cyclones", () => {
  it("finds the systems among their tracks", () => {
    expect(positions(collection([track, position]))).toEqual([system])
    expect(positions(undefined)).toEqual([])
  })

  it("names active systems", () => {
    expect(cycloneSummary(collection([position, track]))).toEqual(["MURASU-26 active"])
  })

  it("says the calm state in the API's words", () => {
    const note = "No active cyclone in the North Indian Ocean"
    expect(cycloneSummary(collection([], note))).toEqual([note])
  })

  it("only calls the feed calm when the feed is fresh", () => {
    expect(isCalm("none")).toBe(true)
    expect(isCalm("unknown")).toBe(false)
    expect(isCalm("waiting")).toBe(false)
    expect(isCalm("active")).toBe(false)
  })

  it("fits the system and Sri Lanka together", () => {
    const [west, south, east, north] = boundsWithSriLanka(system)
    expect(west).toBeLessThan(79.52)
    expect(south).toBeLessThan(5.85)
    expect(east).toBeGreaterThan(81.95)
    expect(north).toBeGreaterThan(14.7)
  })

  it("keeps JTWC and IMD under evaluation in the words shown", () => {
    expect(evaluationNote).toContain("under evaluation")
  })
})
