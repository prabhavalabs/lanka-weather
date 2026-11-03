import { afterEach, describe, expect, it } from "vitest"
import { setLocale } from "@/lib/i18n"
import { conditionFromWmo, conditionLabel } from "./conditions"
import { estimateLabel, severityLabel } from "./severity"

describe("conditionFromWmo", () => {
  it("maps codes to conditions and words", () => {
    expect(conditionFromWmo(0, 1)).toEqual({ condition: "clear", summary: "Clear" })
    expect(conditionFromWmo(2, 1)).toEqual({ condition: "partly-cloudy", summary: "Partly cloudy" })
    expect(conditionFromWmo(3, 0)).toEqual({ condition: "cloudy", summary: "Overcast" })
    expect(conditionFromWmo(45)).toEqual({ condition: "fog", summary: "Fog" })
    expect(conditionFromWmo(53).condition).toBe("drizzle")
    expect(conditionFromWmo(61).condition).toBe("rain")
    expect(conditionFromWmo(80)).toEqual({ condition: "rain", summary: "Light showers" })
    expect(conditionFromWmo(82)).toEqual({ condition: "heavy-rain", summary: "Violent showers" })
    expect(conditionFromWmo(95).condition).toBe("thunderstorm")
    expect(conditionFromWmo(99).summary).toBe("Thunderstorm with hail")
  })

  it("uses the night sky for clear and mainly clear nights", () => {
    expect(conditionFromWmo(0, 0).condition).toBe("clear-night")
    expect(conditionFromWmo(1, 0).condition).toBe("clear-night")
    expect(conditionFromWmo(1, 1).condition).toBe("partly-cloudy")
  })

  it("calls dry weather in a strong wind windy, but not rain", () => {
    expect(conditionFromWmo(1, 1, 45)).toEqual({
      condition: "windy",
      summary: "Mainly clear, windy",
    })
    expect(conditionFromWmo(63, 1, 60).condition).toBe("rain")
  })

  it("copes with a missing code", () => {
    expect(conditionFromWmo(null, 1)).toEqual({ condition: "partly-cloudy", summary: "Unknown" })
  })
})

describe("in Sinhala and Tamil", () => {
  afterEach(() => setLocale("en"))

  it("words the conditions as the Department of Meteorology does", () => {
    setLocale("si")
    expect(conditionFromWmo(95).summary).toBe("ගිගුරුම් සහිත වැසි")
    expect(conditionLabel("rain")).toBe("වැසි")
    expect(conditionFromWmo(1, 1, 45).summary).toBe("බොහෝ දුරට පැහැදිලි, තද සුළං")
    setLocale("ta")
    expect(conditionFromWmo(95).summary).toBe("இடியுடன் கூடிய மழை")
    expect(conditionLabel("rain")).toBe("மழை")
  })

  it("keeps an estimate's level apart from the official words", () => {
    setLocale("si")
    expect(estimateLabel("warning")).toBe("පෙර සූදානමක් කරගන්න")
    expect(estimateLabel("warning")).not.toBe(severityLabel("warning"))
    setLocale("en")
    expect(estimateLabel("emergency")).toBe("Take action")
    expect(severityLabel("emergency")).toBe("Emergency")
  })
})
