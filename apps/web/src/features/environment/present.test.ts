import { describe, expect, it } from "vitest"
import { sampleRamp } from "@/lib/colour"
import { aqiBand, aqiBandOf, fadeEdges, ringToPixels, seaStateOf, steppedRamp } from "./present"

const grid = { id: "g", lon0: 79, lat0: 5, step: 0.5, nx: 8, ny: 12 }

describe("aqiBandOf", () => {
  it("follows the EPA's bands", () => {
    expect(aqiBandOf(0).label).toBe("Good")
    expect(aqiBandOf(50).label).toBe("Good")
    expect(aqiBandOf(51).label).toBe("Moderate")
    expect(aqiBandOf(150).category).toBe("unhealthy_sensitive")
    expect(aqiBandOf(151).label).toBe("Unhealthy")
    expect(aqiBandOf(300).label).toBe("Very unhealthy")
    expect(aqiBandOf(301).label).toBe("Hazardous")
    expect(aqiBand("moderate").token).toBe("--aqi-2")
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
