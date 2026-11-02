import { describe, expect, it } from "vitest"
import type { StationFeature } from "../rivers/types"
import { addRecent, maxRecent, searchEntry, stationEntries, validRecent } from "./present"
import type { PlaceSearchResult } from "./types"

const kandyCity: PlaceSearchResult = {
  id: "osm-node-2908772560",
  type: "place",
  kind: "city",
  name: "Kandy",
  names: { en: "Kandy", si: "මහනුවර", ta: "கண்டி" },
  matchedName: "Kandy",
  lat: 7.293121,
  lon: 80.635036,
  bbox: null,
  population: 125000,
  hierarchy: [
    { id: "LK-2", level: "province", name: "Central" },
    { id: "LK-21", level: "district", name: "Kandy" },
    { id: "LK-2130", level: "ds", name: "Gangawata Korale" },
    { id: "LK-2130120", level: "gn", name: "Mahanuwara" },
  ],
  score: 1.029,
}

const kandyDistrict: PlaceSearchResult = {
  id: "LK-21",
  type: "region",
  kind: "district",
  name: "Kandy",
  names: { en: "Kandy", si: "මහනුවර දිස්ත්‍රික" },
  matchedName: "Kandy",
  lat: 7.334296,
  lon: 80.86275,
  bbox: [80.421171, 6.939991, 81.01643, 7.490084],
  population: 1461895,
  hierarchy: [{ id: "LK-2", level: "province", name: "Central" }],
  score: 0.975,
}

describe("searchEntry", () => {
  it("shows a place with its kind, DS division and district", () => {
    expect(searchEntry(kandyCity)).toMatchObject({
      id: "osm-node-2908772560",
      kind: "place",
      title: "Kandy",
      localName: "මහනුවර",
      subtitle: "City · Gangawata Korale, Kandy District",
      choice: { type: "place", kind: "city", lat: 7.293121, lon: 80.635036, name: "Kandy" },
    })
  })

  it("shows a region by its level, with its bounding box to fit", () => {
    expect(searchEntry(kandyDistrict)).toMatchObject({
      kind: "area",
      title: "Kandy District",
      subtitle: "District · Central Province",
      choice: { type: "region", id: "LK-21", level: "district", bbox: kandyDistrict.bbox },
    })
  })

  it("leaves out a local name equal to the title", () => {
    expect(
      searchEntry({ ...kandyCity, names: { en: "kandy", si: "Kandy" } }).localName,
    ).toBeUndefined()
  })
})

describe("stationEntries", () => {
  const station = (id: string, name: string, basin: string): StationFeature => ({
    type: "Feature",
    id,
    geometry: { type: "Point", coordinates: [80.59, 7.27] },
    properties: {
      id,
      name,
      basin,
      unit: "m",
      alertLevel: 5,
      minorFloodLevel: 7,
      majorFloodLevel: 9,
      waterLevel: 3.641,
      rainfallMm: null,
      observedAt: null,
      status: "normal",
      trend: "rising",
      ratePerHour: null,
      stale: false,
    },
  })
  const stations = [
    station("peradeniya", "Peradeniya", "Mahaweli Ganga"),
    station("nagalagam-street", "Nagalagam Street", "Kelani Ganga"),
  ]

  it("matches the start of a word of the name or basin", () => {
    expect(stationEntries(stations, "pera").map((entry) => entry.title)).toEqual(["Peradeniya"])
    expect(stationEntries(stations, "street").map((entry) => entry.title)).toEqual([
      "Nagalagam Street",
    ])
    expect(stationEntries(stations, "kelani")[0]).toMatchObject({
      kind: "station",
      subtitle: "Kelani Ganga · river gauge",
      meta: "3.64 m",
      choice: { type: "station", id: "nagalagam-street", lat: 7.27, lon: 80.59 },
    })
  })

  it("needs two letters", () => {
    expect(stationEntries(stations, "p")).toEqual([])
  })
})

describe("recent searches", () => {
  const entry = searchEntry(kandyCity)

  it("puts the latest first without duplicates and keeps a few", () => {
    const district = searchEntry(kandyDistrict)
    expect(addRecent([district, entry], entry).map((item) => item.id)).toEqual([
      entry.id,
      district.id,
    ])
    let recent: (typeof entry)[] = []
    for (let index = 0; index < 10; index += 1)
      recent = addRecent(recent, { ...entry, id: `${index}` })
    expect(recent).toHaveLength(maxRecent)
    expect(recent[0]!.id).toBe("9")
  })

  it("drops malformed stored entries", () => {
    expect(validRecent("nope")).toEqual([])
    expect(validRecent([entry, { id: 1 }, { ...entry, choice: { lat: "x" } }])).toEqual([entry])
  })
})
