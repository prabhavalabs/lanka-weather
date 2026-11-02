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

describe("river, basin, flood area and past flood selections", () => {
  it("round-trips through the URL", () => {
    for (const [query, selection] of [
      ["?river=kelani-ganga", { kind: "river", id: "kelani-ganga" }],
      [
        "?river=kelani-ganga&reach=41405515",
        { kind: "river", id: "kelani-ganga", reach: 41405515 },
      ],
      ["?basin=mahaweli-ganga", { kind: "basin", id: "mahaweli-ganga" }],
      [
        "?area=seed-kelani-ganga-flood-2016-major",
        { kind: "area", id: "seed-kelani-ganga-flood-2016-major" },
      ],
      ["?past=yan-oya-flood-2016", { kind: "past", id: "yan-oya-flood-2016" }],
    ] as const) {
      expect(parseSelection(query)).toEqual(selection)
      expect(selectionSearch(selection)).toBe(query)
    }
  })

  it("ignores a stretch that is not a number", () => {
    expect(parseSelection("?river=kelani-ganga&reach=abc")).toEqual({
      kind: "river",
      id: "kelani-ganga",
    })
    expect(parseSelection("?river=kelani-ganga&reach=-4")).toEqual({
      kind: "river",
      id: "kelani-ganga",
    })
  })

  it("rejects malformed ids", () => {
    expect(parseSelection("?river=<script>")).toBeNull()
    expect(parseSelection("?basin=a b c")).toBeNull()
  })

  it("clears every selection key when closing", () => {
    expect(selectionSearch(null, "?river=kelani-ganga&reach=4&kit")).toBe("?kit=")
  })

  it("tells stretches of one river apart", () => {
    expect(
      sameSelection({ kind: "river", id: "a", reach: 1 }, { kind: "river", id: "a", reach: 2 }),
    ).toBe(false)
    expect(sameSelection({ kind: "river", id: "a" }, { kind: "basin", id: "a" })).toBe(false)
  })
})

describe("hazard selections", () => {
  it("reads and writes each kind", () => {
    const cases = [
      ["?landslide=LK-2136", { kind: "landslide", id: "LK-2136" }],
      ["?quake=us7000tty7", { kind: "quake", id: "us7000tty7" }],
      ["?cyclone=1001346", { kind: "cyclone", id: "1001346" }],
      ["?warnings=all", { kind: "warnings" }],
      ["?warnings=water_level", { kind: "warnings", type: "water_level" }],
      ["?warnings=all&district=LK-21", { kind: "warnings", district: "LK-21" }],
    ] as const
    for (const [search, selection] of cases) {
      expect(parseSelection(search)).toEqual(selection)
      expect(selectionSearch(selection)).toBe(search)
    }
  })

  it("ignores what is malformed", () => {
    expect(parseSelection("?landslide=javascript:alert(1)")).toBeNull()
    expect(parseSelection("?quake=<script>")).toBeNull()
    expect(parseSelection("?warnings=rumours")).toEqual({ kind: "warnings" })
    expect(parseSelection("?warnings=all&district=kandy")).toEqual({ kind: "warnings" })
  })

  it("tells warning filters apart", () => {
    expect(sameSelection({ kind: "warnings" }, { kind: "warnings" })).toBe(true)
    expect(sameSelection({ kind: "warnings", type: "weather" }, { kind: "warnings" })).toBe(false)
    expect(
      sameSelection({ kind: "landslide", id: "LK-2136" }, { kind: "quake", id: "LK-2136" }),
    ).toBe(false)
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
