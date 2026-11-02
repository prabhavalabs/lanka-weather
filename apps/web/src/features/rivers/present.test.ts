import { describe, expect, it } from "vitest"
import {
  countByStatus,
  formatAge,
  formatLevel,
  formatRate,
  headroom,
  nextHourLine,
} from "./present"

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

describe("levels with no published unit", () => {
  it("shows the number without a unit", () => {
    expect(formatLevel(0, "unknown")).toBe("0.00")
    expect(formatRate(0.1, "unknown")).toBe("+0.10/h")
  })
  it("keeps units that are published", () => {
    expect(formatLevel(1.76, "m")).toBe("1.76 m")
    expect(formatLevel(1.6, "ft")).toBe("1.6 ft")
    expect(formatRate(-0.2, "m")).toBe("−0.20 m/h")
  })
})

describe("next-hour line", () => {
  const issuedAt = "2026-11-02T18:32:00Z"
  const forecast = {
    level: 1.8,
    low: 1.76,
    high: 1.85,
    unit: "m" as const,
    direction: "rising",
    crossing: null,
    method: "model",
    issuedAt,
    experimental: true,
    attribution: "Sri Lanka Water Levels (waterlevels.lk)",
  }

  it("gives the level, range and issue time", () => {
    const line = nextHourLine(forecast, new Date("2026-11-02T19:00:00Z"))
    expect(line?.value).toBe("1.80 m, range 1.76–1.85")
    expect(line?.issued).toBe("issued 00:02")
  })

  it("writes feet to one decimal", () => {
    const line = nextHourLine(
      { ...forecast, unit: "ft", level: 5.92, low: 5.79, high: 6.06 },
      new Date("2026-11-02T19:00:00Z"),
    )
    expect(line?.value).toBe("5.9 ft, range 5.8–6.1")
  })

  it("leaves out the range of a trend estimate", () => {
    const line = nextHourLine(
      { ...forecast, low: null, high: null, method: "trend" },
      new Date("2026-11-02T19:00:00Z"),
    )
    expect(line?.value).toBe("1.80 m")
  })

  it("hides a missing or old forecast", () => {
    expect(nextHourLine(null)).toBeNull()
    expect(nextHourLine(forecast, new Date("2026-11-02T20:31:00Z"))).not.toBeNull()
    expect(nextHourLine(forecast, new Date("2026-11-02T20:33:00Z"))).toBeNull()
  })
})
