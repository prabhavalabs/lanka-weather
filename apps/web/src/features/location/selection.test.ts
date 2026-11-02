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

describe("reservoir selections", () => {
  it("reads and writes a reservoir", () => {
    const selection = parseSelection("?reservoir=minneriya")
    expect(selection).toEqual({ kind: "reservoir", id: "minneriya" })
    expect(selectionSearch(selection)).toBe("?reservoir=minneriya")
  })

  it("reads and writes the list of those spilling or releasing", () => {
    expect(parseSelection("?reservoirs=active")).toEqual({ kind: "reservoirs" })
    expect(selectionSearch({ kind: "reservoirs" })).toBe("?reservoirs=active")
  })

  it("clears the keys it owns when the selection changes", () => {
    expect(selectionSearch(null, "?reservoir=minneriya&kit")).toBe("?kit=")
    expect(selectionSearch({ kind: "station", id: "pugoda" }, "?reservoirs=active")).toBe(
      "?station=pugoda",
    )
  })

  it("compares them", () => {
    expect(sameSelection({ kind: "reservoirs" }, { kind: "reservoirs" })).toBe(true)
    expect(sameSelection({ kind: "reservoir", id: "a" }, { kind: "reservoir", id: "a" })).toBe(true)
    expect(sameSelection({ kind: "reservoir", id: "a" }, { kind: "reservoir", id: "b" })).toBe(
      false,
    )
    expect(sameSelection({ kind: "reservoirs" }, { kind: "reservoir", id: "a" })).toBe(false)
  })

  it("ignores a malformed identifier", () => {
    expect(parseSelection("?reservoir=%3Cscript%3E")).toBeNull()
  })
})
