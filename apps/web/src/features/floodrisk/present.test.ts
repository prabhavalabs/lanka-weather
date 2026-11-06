import { describe, expect, it } from "vitest"
import { districtRisk, riskBadge, signalValue } from "./present"
import type { FloodRiskDocument, FloodRiskSignal } from "./types"

const flash: FloodRiskSignal = { family: "google_flash", points: 2, value: 1, unit: "level" }
const rain: FloodRiskSignal = {
  family: "rain_now",
  points: 2,
  value: 45.2,
  unit: "mm",
  window: "1h",
}

const document: FloodRiskDocument = {
  districts: [
    {
      district: "LK-11",
      level: 2,
      variant: "high",
      score: 4,
      families: 2,
      scenario: "flash",
      signals: [flash, rain],
    },
  ],
  meta: { note: "", generatedAt: "2026-11-06T08:30:00Z", complete: true, missing: [] },
}

describe("flood risk", () => {
  it("finds a district, and says nothing is known of one it does not list", () => {
    expect(districtRisk(document, "LK-11")?.variant).toBe("high")
    expect(districtRisk(document, "LK-92")).toMatchObject({ level: 0, signals: [] })
    expect(districtRisk(undefined, "LK-11")).toBeNull()
  })

  it("never shows level 0 in the colour of normal", () => {
    expect(riskBadge[0]).toBe("outline")
  })

  it("writes each signal with its window and figures", () => {
    expect(signalValue(flash)).toEqual({ key: "floodRisk.value.flash.likely" })
    expect(signalValue(rain)).toEqual({ key: "floodRisk.value.rain1h", params: { mm: 45 } })
    expect(
      signalValue({ family: "rain_forecast", points: 1, value: 80, unit: "mm", window: "24h" }),
    ).toEqual({ key: "floodRisk.value.forecast24h", params: { mm: 80 } })
    expect(signalValue({ family: "river", points: 2, value: 2, unit: "level" })).toEqual({
      key: "floodRisk.value.river.minor_flood",
    })
  })
})
