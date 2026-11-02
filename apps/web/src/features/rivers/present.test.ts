import { describe, expect, it } from "vitest"
import { countByStatus, formatAge, formatLevel, formatRate, headroom } from "./present"

describe("formatLevel", () => {
  it("uses two decimals for metres and one for feet", () => {
    expect(formatLevel(2.345, "m")).toBe("2.35 m")
    expect(formatLevel(4.26, "ft")).toBe("4.3 ft")
    expect(formatLevel(null, "m")).toBe("–")
  })
})

describe("formatRate", () => {
  it("shows the sign", () => {
    expect(formatRate(0.126, "m")).toBe("+0.13 m/h")
    expect(formatRate(-0.2, "ft")).toBe("−0.20 ft/h")
    expect(formatRate(0, "m")).toBe("±0.00 m/h")
  })
})

describe("formatAge", () => {
  const now = new Date("2026-11-02T12:00:00Z")
  it("reads naturally", () => {
    expect(formatAge("2026-11-02T11:59:40Z", now)).toBe("just now")
    expect(formatAge("2026-11-02T11:48:00Z", now)).toBe("12 min ago")
    expect(formatAge("2026-11-02T09:00:00Z", now)).toBe("3 h ago")
    expect(formatAge("2026-10-30T12:00:00Z", now)).toBe("3 days ago")
    expect(formatAge(null, now)).toBe("never")
  })
})

describe("headroom", () => {
  const thresholds = { alertLevel: 4, minorFloodLevel: 5, majorFloodLevel: 7 }
  it("measures the distance to the next threshold", () => {
    expect(headroom({ waterLevel: 2, ...thresholds })).toEqual({
      next: "alert",
      remaining: 2,
      fraction: 0.5,
    })
    expect(headroom({ waterLevel: 6, ...thresholds })).toEqual({
      next: "major flood",
      remaining: 1,
      fraction: 0.5,
    })
  })
  it("is null above major flood or without thresholds", () => {
    expect(headroom({ waterLevel: 8, ...thresholds })).toBeNull()
    expect(
      headroom({ waterLevel: 1, alertLevel: null, minorFloodLevel: null, majorFloodLevel: null }),
    ).toBeNull()
  })
})

describe("countByStatus", () => {
  it("counts every status", () => {
    expect(countByStatus([{ status: "alert" }, { status: "alert" }, { status: "normal" }])).toEqual(
      {
        normal: 1,
        alert: 2,
        minor_flood: 0,
        major_flood: 0,
        unknown: 0,
      },
    )
  })
})
