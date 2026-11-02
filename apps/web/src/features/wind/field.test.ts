import { describe, expect, it } from "vitest"
import {
  compassPoint,
  decodeWind,
  sampleWind,
  windFrom,
  windText,
  type WindResponse,
} from "./field"

/** Tenths of m/s as the API encodes them: little-endian int16, base64. */
function encode(values: number[]) {
  const bytes = new Uint8Array(new Int16Array(values.map((value) => Math.round(value * 10))).buffer)
  return btoa(String.fromCharCode(...bytes))
}

/** A 4 × 3 global grid at 90° steps, from 90° N southwards and 0° E eastwards, as GFS lays out its grid. */
function response(
  u: number[],
  v: number[],
  grid: Partial<WindResponse["grid"]> = {},
): WindResponse {
  return {
    runAt: "2026-10-19T12:00:00Z",
    forecastHour: 6,
    validAt: "2026-10-19T18:00:00Z",
    lastValidAt: "2026-10-22T12:00:00Z",
    stepHours: 3,
    grid: { nx: 4, ny: 3, lon0: 0, lat0: 90, dlon: 90, dlat: -90, ...grid },
    scale: 0.1,
    u: encode(u),
    v: encode(v),
    attribution: "NOAA GFS",
  }
}

//          0°E  90°E 180° 270°E
const u = [0, 0, 0, 0, /* 90° N */ 10, 20, 30, 40, /* equator */ -5, -5, -5, -5 /* 90° S */]
const v = [1, 1, 1, 1, 0, 0, 0, -12.5, 2, 2, 2, 2]

describe("decodeWind", () => {
  it("reads little-endian tenths of m/s, negatives included", () => {
    const field = decodeWind(response(u, v))
    expect(field.global).toBe(true)
    expect(Array.from(field.u)).toEqual(u.map((value) => Math.fround(value)))
    expect(field.v[7]).toBe(-12.5)
    expect(() => decodeWind({ ...response(u, v), u: encode([1, 2, 3]) })).toThrow(
      /3 values for 12 points/,
    )
  })

  it("reads the API's regional field over Sri Lanka", () => {
    const field = decodeWind(
      response([1, 2, 3, 4], [0, 0, 0, 0], {
        nx: 2,
        ny: 2,
        lon0: 70,
        lat0: 18,
        dlon: 22,
        dlat: -20,
      }),
    )
    expect(field.global).toBe(false)
    expect(sampleWind(field, 81, 8)).toEqual([2.5, 0])
  })
})

describe("sampleWind", () => {
  it("is exact on grid points and bilinear between them", () => {
    const field = decodeWind(response(u, v))
    expect(sampleWind(field, 90, 0)).toEqual([20, 0])
    expect(sampleWind(field, 0, 90)).toEqual([0, 1])
    // Half way between 0° E and 90° E on the equator, and a quarter of the way south to 90° S.
    expect(sampleWind(field, 45, 0)).toEqual([15, 0])
    const [east, north] = sampleWind(field, 45, -22.5)!
    expect(east).toBeCloseTo(15 * 0.75 - 5 * 0.25, 6)
    expect(north).toBeCloseTo(0.5, 6)
  })

  it("wraps longitude across the antimeridian and 360°", () => {
    const field = decodeWind(response(u, v))
    // 315° E (−45°) is half way from 270° E back round to 0° E.
    expect(sampleWind(field, -45, 0)).toEqual([25, -6.25])
    expect(sampleWind(field, 315, 0)).toEqual(sampleWind(field, -45, 0))
    expect(sampleWind(field, 180, 0)).toEqual(sampleWind(field, -180, 0))
    expect(sampleWind(field, 450, 0)).toEqual([20, 0])
  })

  it("has nothing outside a regional grid or beyond the poles", () => {
    const regional = decodeWind(response(u, v, { dlon: 10, lon0: 100 }))
    expect(regional.global).toBe(false)
    expect(sampleWind(regional, 110, 0)).toEqual([20, 0])
    expect(sampleWind(regional, 99, 0)).toBeNull()
    expect(sampleWind(regional, 131, 0)).toBeNull()
    expect(sampleWind(decodeWind(response(u, v)), 0, 91)).toBeNull()
  })
})

describe("direction", () => {
  it("is where the wind comes from, on 16 points", () => {
    expect(windFrom(0, -10)).toBe(0)
    expect(windFrom(-10, 0)).toBe(90)
    expect(windFrom(0, 10)).toBe(180)
    expect(windFrom(10, 0)).toBe(270)
    expect(compassPoint(windFrom(-5, -5))).toBe("NE")
    expect(compassPoint(350)).toBe("N")
    expect(compassPoint(-22.5)).toBe("NNW")
    expect(windText(-7, -7)).toBe("36 km/h (19 kt) from the NE")
    expect(windText(0.2, 0.1)).toBe("Calm")
  })
})
