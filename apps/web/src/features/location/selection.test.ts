import { describe, expect, it } from "vitest"
import { normaliseRegionId, parseSelection, sameSelection, selectionSearch } from "./selection"

describe("parseSelection", () => {
  it("reads a point, rounded, with an optional name", () => {
    expect(parseSelection("?at=7.293121,80.635036&name=Kandy")).toEqual({
      kind: "point",
      lat: 7.2931,
      lon: 80.635,
      name: "Kandy",
    })
    expect(parseSelection("?at=6.9271, 79.8612")).toEqual({
      kind: "point",
      lat: 6.9271,
      lon: 79.8612,
    })
  })

  it("ignores points that are malformed or outside Sri Lanka's box", () => {
    expect(parseSelection("?at=7.29")).toBeNull()
    expect(parseSelection("?at=abc,80")).toBeNull()
    expect(parseSelection("?at=51.5,-0.12")).toBeNull()
    expect(parseSelection("?at=12,80")).toBeNull()
  })

  it("reads regions in any of the API's spellings", () => {
    expect(parseSelection("?region=lk21")).toEqual({ kind: "region", id: "LK-21" })
    expect(parseSelection("?region=LK-1103005")).toEqual({ kind: "region", id: "LK-1103005" })
    expect(parseSelection("?region=LK-123")).toBeNull()
  })

  it("reads stations and gauges with safe ids only", () => {
    expect(parseSelection("?station=nagalagam-street")).toEqual({
      kind: "station",
      id: "nagalagam-street",
    })
    expect(parseSelection("?gauge=hybas_4121155030")).toEqual({
      kind: "gauge",
      id: "hybas_4121155030",
    })
    expect(parseSelection("?station=<script>")).toBeNull()
  })

  it("prefers a point when several are given, and is null when none is", () => {
    expect(parseSelection("?region=LK-21&at=7,80")?.kind).toBe("point")
    expect(parseSelection("")).toBeNull()
    expect(parseSelection("?kit")).toBeNull()
  })

  it("caps the name's length", () => {
    const selection = parseSelection(`?at=7,80&name=${"x".repeat(200)}`)
    expect(selection?.kind === "point" && selection.name?.length).toBe(80)
  })
})

describe("selectionSearch", () => {
  it("writes each kind and round-trips through parseSelection", () => {
    const point = { kind: "point", lat: 7.293121, lon: 80.635036, name: "Kandy" } as const
    expect(selectionSearch(point)).toBe("?at=7.2931,80.6350&name=Kandy")
    expect(parseSelection(selectionSearch(point))).toEqual({ ...point, lat: 7.2931, lon: 80.635 })
    expect(selectionSearch({ kind: "region", id: "LK-21" })).toBe("?region=LK-21")
    expect(selectionSearch({ kind: "gauge", id: "hybas_1" })).toBe("?gauge=hybas_1")
  })

  it("keeps unrelated parameters and drops the old selection", () => {
    expect(selectionSearch({ kind: "station", id: "hanwella" }, "?kit&at=7,80&name=x")).toBe(
      "?kit=&station=hanwella",
    )
    expect(selectionSearch(null, "?region=LK-21")).toBe("")
  })
})

describe("normaliseRegionId", () => {
  it("accepts the four code lengths", () => {
    expect(normaliseRegionId("LK-1")).toBe("LK-1")
    expect(normaliseRegionId("LK-1-1-03")).toBe("LK-1103")
    expect(normaliseRegionId("LK-12345")).toBeNull()
    expect(normaliseRegionId("Kandy")).toBeNull()
  })
})

describe("sameSelection", () => {
  it("compares points by rounded position and name", () => {
    expect(
      sameSelection(
        { kind: "point", lat: 7.29312, lon: 80.63501 },
        { kind: "point", lat: 7.29314, lon: 80.63499 },
      ),
    ).toBe(true)
    expect(sameSelection({ kind: "region", id: "LK-1" }, { kind: "station", id: "LK-1" })).toBe(
      false,
    )
    expect(sameSelection(null, null)).toBe(true)
  })
})

describe("met station selection", () => {
  it("round-trips through the URL", () => {
    const selection = parseSelection("?met=nuwara-eliya")
    expect(selection).toEqual({ kind: "met", id: "nuwara-eliya" })
    expect(selectionSearch(selection, "?gauge=x")).toBe("?met=nuwara-eliya")
    expect(sameSelection(selection, { kind: "met", id: "nuwara-eliya" })).toBe(true)
    expect(sameSelection(selection, { kind: "station", id: "nuwara-eliya" })).toBe(false)
  })
})
