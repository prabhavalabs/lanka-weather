import { afterEach, describe, expect, it } from "vitest"
import { setLocale, translateIn } from "@/lib/i18n"
import {
  atRisk,
  coverageText,
  reachIdsByLevel,
  reasonBasis,
  reasonText,
  rebuildReason,
  riskLabels,
  riverName,
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

describe("riverName", () => {
  it("uses the Sinhala name where the data has one, else English", () => {
    const named = { name: "Malwathu Oya", nameSi: "මල්වතු ඔය" }
    expect(riverName(named, "en")).toBe("Malwathu Oya")
    expect(riverName(named, "si")).toBe("මල්වතු ඔය")
    expect(riverName(named, "ta")).toBe("Malwathu Oya")
    expect(riverName({ name: "Kelani Ganga", nameSi: null }, "si")).toBe("Kelani Ganga")
  })
})

describe("in Sinhala and Tamil", () => {
  afterEach(() => setLocale("en"))

  it("words the headline, coverage and levels", () => {
    setLocale("si")
    const headline = riskHeadline(document([river("a", "high")]), new Date("2026-11-02T12:00:00Z"))
    expect(headline.title).toBe("අවදානමේ ඇති ගංගා 1")
    expect(headline.detail).toContain("මිනුම් ස්ථාන 40න් 38ක්")
    expect(coverageText({ gaugedReaches: 0, reaches: [] })).toBe("මෙම ගඟේ මිනුම් ස්ථානයක් නැත")
    setLocale("ta")
    expect(riskLabels.severe).toBe("கடும் அபாயம்")
  })

  it("never names our estimate with the Irrigation Department's status words", () => {
    for (const locale of ["en", "si", "ta"] as const) {
      expect(translateIn(locale, "rivers.risk.level.watch")).not.toBe(
        translateIn(locale, "terms.riverStatus.alert"),
      )
    }
  })
})

describe("reasonText", () => {
  const reason = (kind: string, text: string) => ({ kind, text })

  it("keeps the server's sentence in English", () => {
    const text = "Kalu Ganga at Ratnapura is at 5.20 m, above its alert level of 5.00 m."
    expect(reasonText(reason("gauge", text), "en")).toEqual({ text, english: false })
  })

  it("rebuilds the sentences it knows, which read the same in English", () => {
    const sentences: [string, string][] = [
      ["coverage", "No gauge on this stretch, so its level is not known."],
      [
        "coverage",
        "No gauge on this stretch, so its level is not known. The next gauge downstream is Hanwella.",
      ],
      ["rain", "The rain forecast is out of date, so it is not used."],
      [
        "rain",
        "Heavy rain is forecast over the catchment: 120 mm in the next 24 hours and 210 mm in 72 hours on average.",
      ],
      ["gauge", "Hanwella publishes no alert level to compare with."],
      ["gauge", "Hanwella is at 6.10 m, above its minor flood level of 6.00 m."],
      [
        "gauge",
        "Hanwella is at 4.80 m, below its alert level of 5.00 m. It is within 0.20 m of that level.",
      ],
      [
        "gauge",
        "Hanwella is at 5.50 m, above its alert level of 5.00 m. Rising 0.20 m an hour, it would reach minor flood level in about 3 hours.",
      ],
    ]
    for (const [kind, text] of sentences) {
      // Rebuilt in English, each matches the server's sentence exactly.
      expect(rebuildReason(reason(kind, text), "en")).toBe(text)
      const parts = reasonText(reason(kind, text), "si")
      expect(parts.english, text).toBe(false)
      expect(parts.text, text).not.toBe(text)
    }
    expect(reasonText(reason("gauge", sentences[5]![1]), "si").text).toBe(
      "Hanwella හි ජල මට්ටම 6.10 m කි; එය 6.00 m වන සුළු ගංවතුර මට්ටමට ඉහළින් ය.",
    )
    expect(reasonText(reason("gauge", sentences[7]![1]), "ta").text).toContain(
      "ஏறத்தாழ 3 மணித்தியாலத்தில்",
    )
  })

  it("keeps an unknown sentence in English, marked as such", () => {
    const text = "Victoria is spilling 1,200 cusecs directly upstream."
    expect(reasonText(reason("reservoir", text), "ta")).toEqual({ text, english: true })
    expect(reasonBasis("reservoir", "ta")).toBe("மேல்நோக்கிய நீர்த்தேக்கம் ஒன்றின்படி")
    expect(reasonBasis("surprise", "en")).toBe("Estimate")
  })
})
