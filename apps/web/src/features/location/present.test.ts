import { describe, expect, it } from "vitest"
import type { RegionDetail } from "../regions/types"
import {
  currentFrom,
  daysFrom,
  estimateAlert,
  formatCoordinates,
  hierarchyText,
  hoursFrom,
  localDate,
  placeFor,
  sumNext,
  windFrom,
} from "./present"
import type { PointForecast, Summary } from "./types"

const point: PointForecast = {
  location: { lat: 7.29, lon: 80.63 },
  current: {
    validAt: "2026-11-02T14:45:00Z",
    temperatureC: 22,
    apparentTemperatureC: 26.5,
    relativeHumidityPct: 99.1,
    dewPointC: 21.8,
    precipitationMm: 0.1,
    precipitationProbabilityPct: 81.4,
    cloudCoverPct: 97.1,
    pressureHpa: 1016,
    windSpeedKmh: 2,
    windDirectionDeg: 61,
    windGustsKmh: 7.1,
    weatherCode: 3,
    uvIndex: 0,
    isDay: 0,
  },
  hourly: {
    times: ["2026-11-02T14:00:00Z", "2026-11-02T15:00:00Z", "2026-11-02T16:00:00Z"],
    temperatureC: [22.1, null, 21.6],
    apparentTemperatureC: [26.4, 26.5, 26.4],
    relativeHumidityPct: [99, 99, 99],
    precipitationMm: [0.4, 0.3, null],
    precipitationProbabilityPct: [87.8, 81.6, 70],
    cloudCoverPct: [97, 99, 100],
    windSpeedKmh: [2, 2.3, 3],
    windDirectionDeg: [24, 74, 80],
    windGustsKmh: [7, 7.1, 8],
    weatherCode: [51, 3, 95],
    uvIndex: [0, 0, 0],
    isDay: [0, 0, 0],
  },
  daily: {
    dates: ["2026-11-02", "2026-11-03"],
    temperatureMaxC: [28.6, 27.7],
    temperatureMinC: [21.2, null],
    apparentTemperatureMaxC: [34.1, 33.2],
    precipitationSumMm: [8.4, 31.1],
    precipitationProbabilityMaxPct: [100, 98.8],
    windSpeedMaxKmh: [8.9, 9.7],
    windGustsMaxKmh: [24.5, 27.5],
    windDirectionDominantDeg: [194, 250],
    uvIndexMax: [8, 8],
    weatherCode: [80, 95],
  },
  meta: {
    source: "open-meteo",
    model: "best_match",
    attribution: "Weather data by Open-Meteo.com (CC BY 4.0)",
    grid: { id: "lk-0.2", lon0: 79.4, lat0: 5.6, step: 0.2, nx: 15, ny: 23 },
    interpolation: "bilinear",
    currentFetchedAt: "2026-11-02T14:48:39Z",
    forecastFetchedAt: "2026-11-02T14:49:20Z",
    currentStale: false,
    forecastStale: false,
  },
}

const regions: Summary["regions"] = {
  province: { id: "LK-2", level: "province", name: "Central", nameSi: "මධ්‍යම පළාත" },
  district: { id: "LK-21", level: "district", name: "Kandy" },
  ds: { id: "LK-2130", level: "ds", name: "Gangawata Korale" },
  gn: { id: "LK-2130110", level: "gn", name: "Katukele", nameSi: "කටුකැලේ" },
}

const summary = { regions } as Summary

describe("currentFrom", () => {
  it("maps current conditions, words and the next day's rain", () => {
    expect(currentFrom(point)).toEqual({
      condition: "cloudy",
      summary: "Overcast",
      temperatureC: 22,
      feelsLikeC: 26.5,
      humidityPercent: 99.1,
      rainfallMm: 0.1,
      rainNext24hMm: 0.7,
      rainChancePercent: 81.4,
      cloudCoverPercent: 97.1,
      pressureHpa: 1016,
      uvIndex: 0,
      dewPointC: 21.8,
      observedAt: "2026-11-02T14:45:00Z",
    })
  })

  it("is null without a current temperature", () => {
    expect(currentFrom(null)).toBeNull()
    expect(currentFrom({ ...point, current: null })).toBeNull()
  })
})

describe("hoursFrom and daysFrom", () => {
  it("skips hours without a temperature and fills missing rain with zero", () => {
    const hours = hoursFrom(point)
    expect(hours.map((hour) => hour.time)).toEqual(["2026-11-02T14:00:00Z", "2026-11-02T16:00:00Z"])
    expect(hours[0]).toMatchObject({ condition: "drizzle", rainfallMm: 0.4, rainChancePercent: 88 })
    expect(hours[1]).toMatchObject({
      condition: "thunderstorm",
      rainfallMm: 0,
      windFromDegrees: 80,
    })
  })

  it("keeps days with missing values as null", () => {
    const days = daysFrom(point)
    expect(days).toHaveLength(2)
    expect(days[0]).toMatchObject({ condition: "rain", summary: "Light showers", minC: 21.2 })
    expect(days[1]).toMatchObject({ condition: "thunderstorm", minC: null, rainMm: 31.1 })
  })

  it("reads the wind from the current conditions", () => {
    expect(windFrom(point)).toEqual({ speedKmh: 2, gustKmh: 7.1, fromDegrees: 61 })
    expect(windFrom(null)).toBeNull()
  })
})

describe("sumNext", () => {
  it("adds the hours given, ignoring gaps", () => {
    expect(sumNext([1, null, 2.25], 24)).toBe(3.3)
    expect(sumNext([1, 2, 3], 2)).toBe(3)
    expect(sumNext([null], 24)).toBeNull()
    expect(sumNext(undefined)).toBeNull()
  })
})

describe("placeFor", () => {
  const location = { lat: 7.29, lon: 80.63 }

  it("names a searched place and lists all its regions", () => {
    expect(placeFor({ kind: "point", ...location, name: "Kandy" }, location, summary)).toEqual({
      name: "Kandy",
      area: "Katukele › Gangawata Korale › Kandy District › Central Province",
      latitude: 7.29,
      longitude: 80.63,
    })
  })

  it("names a clicked point after its GN division", () => {
    expect(placeFor({ kind: "point", ...location }, location, summary)).toMatchObject({
      name: "Katukele",
      localName: "කටුකැලේ",
      area: "Gangawata Korale › Kandy District › Central Province",
    })
  })

  it("falls back to coordinates at sea", () => {
    expect(
      placeFor({ kind: "point", lat: 6.5, lon: 79.5 }, { lat: 6.5, lon: 79.5 }, {
        regions: null,
      } as Summary),
    ).toMatchObject({ name: "Off the coast", area: "6.5000° N, 79.5000° E" })
  })

  it("names a region with its level and parents", () => {
    const region = {
      id: "LK-2130",
      level: "ds",
      name: "Gangawata Korale",
      names: { en: "Gangawata Korale", si: "ගඟවටකෝරලේ" },
      hierarchy: [
        { id: "LK-2", level: "province", name: "Central" },
        { id: "LK-21", level: "district", name: "Kandy" },
      ],
    } as RegionDetail
    expect(placeFor({ kind: "region", id: "LK-2130" }, location, null, region)).toMatchObject({
      name: "Gangawata Korale",
      localName: "ගඟවටකෝරලේ",
      area: "DS division · Kandy District › Central Province",
    })
  })
})

describe("hierarchyText", () => {
  it("can leave levels out", () => {
    expect(hierarchyText(regions, ["gn", "ds"])).toBe("Kandy District › Central Province")
    expect(hierarchyText(null)).toBe("")
  })
})

describe("estimateAlert", () => {
  it("labels a derived alert as an estimate with its basis", () => {
    const alert = estimateAlert(
      {
        type: "heavy_rain",
        level: "orange",
        title: "Heavy rain likely",
        detail: "About 117 mm of rain is forecast in the next 24 hours (threshold 100 mm).",
        value: 117,
        threshold: 100,
        unit: "mm",
        validFrom: "2026-11-02T14:00:00Z",
        validUntil: "2026-11-03T14:00:00Z",
        estimate: true,
        basis: "Open-Meteo forecast, interpolated from the weather grid",
      },
      0,
      "Kandy",
      "2026-11-02T14:51:02Z",
    )
    expect(alert).toEqual({
      id: "heavy_rain-0",
      title: "Heavy rain likely",
      severity: "warning",
      estimate: true,
      issuer: "Lanka Weather estimate",
      area: "Kandy",
      issuedAt: "2026-11-02T14:00:00Z",
      validUntil: "2026-11-03T14:00:00Z",
      summary: "About 117 mm of rain is forecast in the next 24 hours (threshold 100 mm).",
      basis: "Open-Meteo forecast, interpolated from the weather grid",
    })
  })

  it("maps yellow and red to watch and emergency", () => {
    const base = { type: "heat", title: "", detail: "", estimate: true, basis: "" } as const
    expect(estimateAlert({ ...base, level: "yellow" }, 1, "", "t").severity).toBe("watch")
    expect(estimateAlert({ ...base, level: "red" }, 1, "", "t")).toMatchObject({
      severity: "emergency",
      issuedAt: "t",
    })
  })
})

describe("localDate and formatCoordinates", () => {
  it("uses Sri Lanka's date", () => {
    expect(localDate(new Date("2026-11-02T19:00:00Z"))).toBe("2026-11-03")
    expect(localDate(new Date("2026-11-02T18:00:00Z"))).toBe("2026-11-02")
  })

  it("writes hemispheres", () => {
    expect(formatCoordinates(-1.5, 80)).toBe("1.5000° S, 80.0000° E")
  })
})
