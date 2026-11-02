import { describe, expect, it } from "vitest"
import { conditionFromWmo } from "./conditions"

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
