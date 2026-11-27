import { describe, expect, it } from "vitest"
import { sampleRamp } from "@/lib/colour"
import {
  aqiBand,
  aqiBandOf,
  aqiLabel,
  distanceToLand,
  fadeEdges,
  fadeFromCoast,
  fillGaps,
  ringToPixels,
  seaStateLabel,
  seaStateOf,
  steppedRamp,
} from "./present"

const grid = { id: "g", lon0: 79, lat0: 5, step: 0.5, nx: 8, ny: 12 }

describe("aqiBandOf", () => {
  it("follows the EPA's bands", () => {
    const label = (value: number) => aqiLabel(aqiBandOf(value).category)
    expect(label(0)).toBe("Good")
    expect(label(50)).toBe("Good")
    expect(label(51)).toBe("Moderate")
    expect(aqiBandOf(150).category).toBe("unhealthy_sensitive")
    expect(label(151)).toBe("Unhealthy")
    expect(label(300)).toBe("Very unhealthy")
    expect(label(301)).toBe("Hazardous")
    expect(aqiBand("moderate").token).toBe("--aqi-2")
  })

  it("names the bands in Sinhala and Tamil", () => {
    expect(aqiLabel("good", "si")).toBe("හොඳයි")
    expect(aqiLabel("hazardous", "ta")).toBe("அபாயகரமானது")
  })
})

describe("seaStateOf", () => {
  it("uses the WMO words", () => {
    expect(seaStateOf(0.05)).toBe("Calm")
    expect(seaStateOf(0.3)).toBe("Smooth")
    expect(seaStateOf(1.0)).toBe("Slight")
    expect(seaStateOf(2)).toBe("Moderate")
    expect(seaStateOf(3)).toBe("Rough")
    expect(seaStateOf(5)).toBe("Very rough")
    expect(seaStateOf(7)).toBe("High")
    expect(seaStateOf(10)).toBe("Very high")
    expect(seaStateOf(3, "si")).toBe("රළු")
  })

  it("reads the API's word, falling back to the wave height", () => {
    expect(seaStateLabel("very rough", 5)).toBe("Very rough")
    expect(seaStateLabel("slight", 1, "ta")).toBe("சிறிதளவு கொந்தளிப்பு")
    expect(seaStateLabel("choppy", 0.3)).toBe("Smooth")
  })
})

describe("steppedRamp", () => {
  it("holds each band's colour to its bound, with no blending", () => {
    const ramp = steppedRamp(
      [
        { from: 0, token: "--a" },
        { from: 51, token: "--b" },
      ],
      ["#000000", "#ffffff"],
      100,
    )
    expect(sampleRamp(ramp, 50.9)[0]).toBe(0)
    expect(sampleRamp(ramp, 51)[0]).toBe(255)
    expect(sampleRamp(ramp, 99)[0]).toBe(255)
  })
})

describe("fadeEdges", () => {
  it("fades only the outer band", () => {
    const pixels = new Uint8ClampedArray(5 * 5 * 4).fill(255)
    fadeEdges(pixels, 5, 5, 2)
    const alpha = (x: number, y: number) => pixels[(y * 5 + x) * 4 + 3]
    expect(alpha(0, 2)).toBe(0)
    expect(alpha(1, 2)).toBeCloseTo(128, -1)
    expect(alpha(2, 2)).toBe(255)
  })
})

describe("ringToPixels", () => {
  it("maps corners to the image's corners", () => {
    const pixels = ringToPixels(
      [
        [79, 10.5],
        [82.5, 5],
      ],
      grid,
      100,
      200,
    )
    expect(pixels[0]![0]).toBeCloseTo(0)
    expect(pixels[0]![1]).toBeCloseTo(0)
    expect(pixels[1]![0]).toBeCloseTo(100)
    expect(pixels[1]![1]).toBeCloseTo(200)
  })
})

describe("distanceToLand", () => {
  it("measures each pixel's distance to the nearest land", () => {
    // One land pixel in the middle of a 5 × 5 image.
    const land = new Uint8Array(25)
    land[12] = 1
    const distance = distanceToLand(land, 5, 5)
    expect(distance[12]).toBe(0)
    expect(distance[11]).toBe(1)
    expect(distance[6]).toBeCloseTo(Math.SQRT2)
    expect(distance[2]).toBe(2)
    expect(distance[0]).toBeCloseTo(2 * Math.SQRT2)
  })

  it("is infinite everywhere without land", () => {
    expect([...distanceToLand(new Uint8Array(4), 2, 2)]).toEqual([
      Infinity,
      Infinity,
      Infinity,
      Infinity,
    ])
  })
})

describe("fadeFromCoast", () => {
  it("keeps the field near the coast, eases it out and drops it far off", () => {
    const pixels = new Uint8ClampedArray(4 * 4).fill(200)
    fadeFromCoast(pixels, Float32Array.from([0, 10, 20, 40]), 10, 30)
    expect(pixels[3]).toBe(200)
    expect(pixels[7]).toBe(200)
    expect(pixels[11]).toBe(100)
    expect(pixels[15]).toBe(0)
  })
})

describe("fillGaps", () => {
  it("fills empty cells from their neighbours, ring by ring", () => {
    // A 3 × 1 strip with one value at the end.
    expect(fillGaps([null, null, 2], 3, 1, 1)).toEqual([null, 2, 2])
    expect(fillGaps([null, null, 2], 3, 1, 2)).toEqual([2, 2, 2])
  })

  it("averages the filled neighbours, and leaves an empty grid alone", () => {
    expect(fillGaps([1, null, 3], 3, 1)).toEqual([1, 2, 3])
    expect(fillGaps([null, null], 2, 1)).toEqual([null, null])
  })
})
