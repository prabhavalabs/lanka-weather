import { describe, expect, it } from "vitest"
import {
  atRisk,
  coverageText,
  reachIdsByLevel,
  riskHeadline,
  riversAtRisk,
  riverSearchNames,
} from "./present"
import type { RiskDocument, RiverRisk } from "./types"

function river(id: string, level: RiverRisk["level"], uplandKm2 = 100): RiverRisk {
  return {
    id,
    name: id,
    nameSi: null,
    aliases: [],
    named: true,
    parentId: null,
    level,
    reasons: [],
    uplandKm2,
    lengthKm: 10,
    bbox: [80, 7, 81, 8],
    reaches: [
      {
        fid: uplandKm2,
        riverId: id,
        name: "Whole river",
        level,
        reasons: [],
        gauged: true,
        lengthKm: 10,
        gauges: [],
        rain: null,
      },
    ],
    gaugedReaches: 1,
    observedAt: null,
  }
}

function document(rivers: RiverRisk[], meta: Partial<RiskDocument["meta"]> = {}): RiskDocument {
  return {
    estimate: true,
    note: "",
    rivers,
    meta: {
      generatedAt: "2026-11-02T12:00:00Z",
      networkVersion: "v",
      networkBuiltAt: null,
      gaugesObservedAt: "2026-11-02T11:00:00Z",
      gaugesCurrent: 38,
      gaugesTotal: 40,
      forecastIssuedAt: null,
      forecastFetchedAt: null,
      rainFetchedAt: null,
      rainStale: false,
      reservoirsFetchedAt: null,
      reservoirLinks: { byPosition: 0, byRiver: 0, unlinked: [] },
      attribution: "",
      seeded: false,
      ...meta,
    },
  }
}

describe("atRisk", () => {
  it("starts at watch; a stretch with no gauge is not at risk, and not safe either", () => {
    expect(atRisk("watch")).toBe(true)
    expect(atRisk("severe")).toBe(true)
    expect(atRisk("normal")).toBe(false)
    expect(atRisk("unknown")).toBe(false)
  })
})

describe("riversAtRisk", () => {
  it("ranks the worst first, then the largest", () => {
    const ranked = riversAtRisk([
      river("calm", "normal"),
      river("nothing-known", "unknown"),
      river("small-high", "high", 100),
      river("large-high", "high", 900),
      river("watch", "watch", 5000),
      river("severe", "severe", 50),
    ])
    expect(ranked.map((item) => item.id)).toEqual(["severe", "large-high", "small-high", "watch"])
  })
})

describe("riskHeadline", () => {
  const now = new Date("2026-11-02T12:00:00Z")

  it("is calm and honest when nothing is above alert", () => {
    const headline = riskHeadline(document([river("a", "normal")]), now)
    expect(headline.calm).toBe(true)
    expect(headline.title).toBe("No river is above its alert level")
    expect(headline.detail).toBe("38 of 40 gauges current · read 1 h ago")
  })

  it("counts what is at risk", () => {
    expect(riskHeadline(document([river("a", "high")]), now).title).toBe("1 river at risk")
    expect(riskHeadline(document([river("a", "high"), river("b", "watch", 2)]), now).title).toBe(
      "2 rivers at risk",
    )
  })

  it("does not invent an age when no gauge has read", () => {
    const headline = riskHeadline(document([], { gaugesObservedAt: null, gaugesCurrent: 0 }), now)
    expect(headline.detail).toBe("0 of 40 gauges current")
  })
})

describe("coverageText", () => {
  it("says plainly what has a gauge", () => {
    expect(coverageText({ gaugedReaches: 0, reaches: river("a", "unknown").reaches })).toBe(
      "No gauge on this river",
    )
    const two = [...river("a", "normal").reaches, ...river("b", "unknown").reaches]
    expect(coverageText({ gaugedReaches: 1, reaches: two })).toBe("1 of 2 stretches have a gauge")
    expect(coverageText({ gaugedReaches: 2, reaches: two })).toBe("Every stretch has a gauge")
  })
})

describe("reachIdsByLevel", () => {
  it("groups stretches for the map", () => {
    const ids = reachIdsByLevel([river("a", "normal", 1), river("b", "severe", 2)])
    expect(ids.normal).toEqual([1])
    expect(ids.severe).toEqual([2])
    expect(ids.watch).toEqual([])
  })
})

describe("riverSearchNames", () => {
  it("lists every name a river goes by", () => {
    expect(
      riverSearchNames({ name: "Malwathu Oya", nameSi: "මල්වතු ඔය", aliases: ["Aruvi Aru"] }),
    ).toEqual(["Malwathu Oya", "මල්වතු ඔය", "Aruvi Aru"])
  })
})
