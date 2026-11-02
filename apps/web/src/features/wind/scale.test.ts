import { describe, expect, it } from "vitest"
import { bandLabel, windBand, windBands, windBandTokens } from "./scale"

describe("wind bands", () => {
  it("rise with speed and follow the legend's km/h steps", () => {
    for (let index = 1; index < windBands.length; index += 1) {
      expect(windBands[index]).toBeGreaterThan(windBands[index - 1]!)
    }
    expect(windBand(0)).toBe(0)
    expect(windBand(2.7)).toBe(0)
    expect(windBand(10 / 3.6)).toBe(1)
    expect(windBand(29.9 / 3.6)).toBe(2)
    expect(windBand(30 / 3.6)).toBe(3)
    expect(windBand(117.9 / 3.6)).toBe(5)
    expect(windBand(118 / 3.6)).toBe(6)
    expect(windBand(80)).toBe(6)
    expect(windBand(Number.NaN)).toBe(0)
  })

  it("each have a theme token and a legend label", () => {
    expect(windBandTokens).toEqual([
      "--wind-1",
      "--wind-2",
      "--wind-3",
      "--wind-4",
      "--wind-5",
      "--wind-6",
      "--wind-7",
    ])
    expect(windBands.map((_, index) => bandLabel(index))).toEqual([
      "0",
      "10",
      "20",
      "30",
      "50",
      "70",
      "118",
    ])
  })
})
