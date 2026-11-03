import { afterEach, describe, expect, it } from "vitest"
import { setLocale } from "@/lib/i18n"
import {
  activeReservoirs,
  hasCurrentFigures,
  flowNote,
  flowShort,
  formatReadingDay,
  isActive,
  readingDayAge,
  reservoirStatusLabels,
  reservoirSubtitle,
  reservoirSummary,
  storageView,
} from "./present"
import type { ReservoirFeature, ReservoirProperties } from "./types"

function reservoir(overrides: Partial<ReservoirProperties> = {}): ReservoirProperties {
  return {
    id: "x",
    name: "X",
    district: null,
    river: null,
    type: "major",
    status: "holding",
    storagePercent: 44.6,
    storageFresh: true,
    flow: null,
    readingDay: "2026-11-02",
    agency: "Irrigation Department",
    source: "waterlevels-lk-reservoirs",
    sourceUrl: null,
    fetchedAt: "2026-11-02T19:45:00Z",
    stale: false,
    mapped: true,
    osmName: null,
    osmKind: null,
    altNames: null,
    ...overrides,
  }
}

function feature(overrides: Partial<ReservoirProperties>): ReservoirFeature {
  const properties = reservoir(overrides)
  return { type: "Feature", id: properties.id, geometry: null, properties }
}

describe("storageView", () => {
  it("shows a current figure, with a bar", () => {
    expect(storageView(reservoir())).toEqual({
      value: "44.6",
      caption: "Of capacity.",
      current: true,
    })
  })

  it("never shows unknown storage as zero", () => {
    const view = storageView(reservoir({ status: "nodata", storagePercent: null }))
    expect(view.value).toBe("–")
    expect(view.current).toBe(false)
    expect(view.caption).toBe("Storage is not reported.")
  })

  it("keeps a true zero", () => {
    expect(storageView(reservoir({ storagePercent: 0 })).value).toBe("0")
  })

  it("says an old figure is old", () => {
    const view = storageView(
      reservoir({ storagePercent: 68.4, storageFresh: false, readingDay: "2026-10-29" }),
    )
    expect(view.value).toBe("68.4")
    expect(view.current).toBe(false)
    expect(view.caption).toContain("Last known figure from")
    expect(view.caption).toContain("29 Oct")
  })

  it("does not call a figure current on a stale reading", () => {
    expect(storageView(reservoir({ stale: true })).current).toBe(false)
  })
})

describe("flows", () => {
  it("reads each kind with its amount", () => {
    expect(flowShort({ kind: "spill", cusecs: 1551, agency: "x" })).toBe("Spill: 1,551 cusecs")
    expect(flowShort({ kind: "outlets", cusecs: 9, agency: "x" })).toBe("Through outlets: 9 cusecs")
    expect(flowShort({ kind: "spill", cusecs: 17.22, agency: "x" })).toBe("Spill: 17.2 cusecs")
    expect(flowShort({ kind: "new-kind", cusecs: 3, agency: "x" })).toBe("Flow: 3 cusecs")
  })

  it("does not invent an amount", () => {
    expect(flowShort({ kind: "spill", cusecs: null, agency: "x" })).toBe("Spill")
  })

  it("warns that a downstream flow is not necessarily a flood release", () => {
    expect(flowNote({ kind: "downstream", cusecs: 690, agency: "Mahaweli Authority" })).toContain(
      "power-station",
    )
    expect(flowNote({ kind: "outlets", cusecs: 1400, agency: "x" })).toContain(
      "not count as spilling",
    )
    expect(flowNote({ kind: "spill", cusecs: 1, agency: "x" })).toBeNull()
  })
})

describe("reading days", () => {
  const now = new Date("2026-11-02T19:39:00Z") // 01:09 on 3 November in Sri Lanka

  it("counts by the Sri Lankan calendar", () => {
    expect(readingDayAge("2026-11-03", now)).toBe("today")
    expect(readingDayAge("2026-11-02", now)).toBe("yesterday")
    expect(readingDayAge("2026-10-29", now)).toBe("5 days ago")
  })

  it("formats a day without moving it across time zones", () => {
    expect(formatReadingDay("2026-11-02")).toBe("Mon 2 Nov")
  })
})

describe("active reservoirs", () => {
  it("is spilling or releasing, on a current reading", () => {
    expect(isActive({ status: "spilling", stale: false })).toBe(true)
    expect(isActive({ status: "releasing", stale: false })).toBe(true)
    expect(isActive({ status: "holding", stale: false })).toBe(false)
    expect(isActive({ status: "nodata", stale: false })).toBe(false)
    expect(isActive({ status: "spilling", stale: true })).toBe(false)
  })

  it("lists spilling first, then by name", () => {
    const ids = activeReservoirs([
      feature({ id: "b", name: "B", status: "releasing" }),
      feature({ id: "d", name: "D", status: "spilling" }),
      feature({ id: "c", name: "C", status: "spilling" }),
      feature({ id: "a", name: "A", status: "holding" }),
      feature({ id: "e", name: "E", status: "spilling", stale: true }),
    ]).map((item) => item.id)
    expect(ids).toEqual(["c", "d", "b"])
  })
})

describe("reservoirSummary", () => {
  const counts = { total: 142, mapped: 115, spilling: 10, releasing: 2, holding: 76, nodata: 54 }

  it("counts those needing attention", () => {
    expect(reservoirSummary(counts, 12, false)).toBe("12 spilling or releasing")
  })

  it("is calm when there are none", () => {
    expect(reservoirSummary(counts, 0, false)).toBe("None spilling or releasing")
  })

  it("does not claim calm on old data", () => {
    expect(reservoirSummary(counts, 0, true)).toBe("142 reservoirs · figures out of date")
  })

  it("does not claim calm before anything is collected", () => {
    const none = { total: 0, mapped: 0, spilling: 0, releasing: 0, holding: 0, nodata: 0 }
    expect(reservoirSummary(none, 0, false)).toBe("Not collected yet")
  })

  it("needs at least one current figure to speak about today", () => {
    expect(hasCurrentFigures([feature({ stale: true }), feature({ stale: false })])).toBe(true)
    expect(hasCurrentFigures([feature({ stale: true })])).toBe(false)
    expect(hasCurrentFigures([])).toBe(false)
  })
})

it("writes a subtitle from what is known", () => {
  expect(
    reservoirSubtitle(reservoir({ district: "Kurunegala", river: "Deduru Oya", type: "major" })),
  ).toBe("Major reservoir · Kurunegala")
  expect(reservoirSubtitle(reservoir({ type: "mahaweli" }))).toBe("Mahaweli reservoir")
})

describe("in Sinhala and Tamil", () => {
  afterEach(() => setLocale("en"))
  const now = new Date("2026-11-02T19:39:00Z")

  it("uses the Sri Lankan words for spilling and keeps cusecs", () => {
    setLocale("si")
    expect(reservoirStatusLabels.spilling).toBe("වාන් දමමින්")
    expect(flowShort({ kind: "spill", cusecs: 1551, agency: "x" })).toBe("වාන් දැමීම: 1,551 cusecs")
    setLocale("ta")
    expect(flowShort({ kind: "spill", cusecs: null, agency: "x" })).toBe("வான் பாய்தல்")
  })

  it("counts reading days in the language", () => {
    setLocale("si")
    expect(readingDayAge("2026-11-03", now)).toBe("අද")
    expect(readingDayAge("2026-11-02", now)).toBe("ඊයේ")
    setLocale("ta")
    expect(readingDayAge("2026-10-29", now)).toBe("5 நாட்களுக்கு முன்")
  })

  it("says the summary in the language", () => {
    setLocale("ta")
    const counts = { total: 1, mapped: 1, spilling: 0, releasing: 0, holding: 1, nodata: 0 }
    expect(reservoirSummary(counts, 0, true)).toBe("1 நீர்த்தேக்கம் · தரவு காலாவதியானது")
  })
})
