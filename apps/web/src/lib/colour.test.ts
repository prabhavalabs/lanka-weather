import { describe, expect, it } from "vitest"
import {
  colourTable,
  parseHex,
  rampFromScale,
  rgbaCss,
  sampleRamp,
  stepExpression,
  stepIndex,
  tableOffset,
  type RampStop,
} from "./colour"
import { rainfallScale, temperatureScale, windScale } from "./tokens"

describe("parseHex", () => {
  it("reads six and eight digit hex", () => {
    expect(parseHex("#ff8000")).toEqual([255, 128, 0, 1])
    expect(parseHex("#00000080")[3]).toBeCloseTo(0.502, 3)
    expect(parseHex(" #0A84D6 ")).toEqual([10, 132, 214, 1])
  })

  it("treats anything else as transparent", () => {
    expect(parseHex("")).toEqual([0, 0, 0, 0])
    expect(parseHex("red")).toEqual([0, 0, 0, 0])
  })

  it("writes CSS rgba()", () => {
    expect(rgbaCss([10, 20, 30, 0.5])).toBe("rgba(10, 20, 30, 0.5)")
  })
})

describe("sampleRamp", () => {
  const ramp: RampStop[] = [
    { value: 0, color: [0, 0, 0, 1] },
    { value: 10, color: [100, 200, 0, 1] },
    { value: 20, color: [100, 200, 200, 0] },
  ]

  it("holds the end colours beyond the ramp", () => {
    expect(sampleRamp(ramp, -5)).toEqual([0, 0, 0, 1])
    expect(sampleRamp(ramp, 99)).toEqual([100, 200, 200, 0])
  })

  it("blends linearly between stops", () => {
    expect(sampleRamp(ramp, 5)).toEqual([50, 100, 0, 1])
    expect(sampleRamp(ramp, 15)).toEqual([100, 200, 100, 0.5])
  })

  it("has nothing for missing values", () => {
    expect(sampleRamp(ramp, Number.NaN)).toEqual([0, 0, 0, 0])
    expect(sampleRamp([], 3)).toEqual([0, 0, 0, 0])
  })
})

describe("colourTable", () => {
  const ramp: RampStop[] = [
    { value: 10, color: [0, 0, 0, 0] },
    { value: 20, color: [255, 255, 255, 1] },
  ]
  const table = colourTable(ramp, 11)

  it("covers the ramp's range with alpha as a byte", () => {
    expect(table.min).toBe(10)
    expect(table.max).toBe(20)
    expect(Array.from(table.rgba.slice(0, 4))).toEqual([0, 0, 0, 0])
    expect(Array.from(table.rgba.slice(40, 44))).toEqual([255, 255, 255, 255])
  })

  it("finds the entry of a value, clamped to the range", () => {
    expect(tableOffset(table, 15)).toBe(20)
    expect(tableOffset(table, 0)).toBe(0)
    expect(tableOffset(table, 1000)).toBe(40)
  })
})

describe("rampFromScale", () => {
  it("pins each colour at its step and opens the first step below the next bound", () => {
    const colours = temperatureScale.map((_, index) => `#0000${String(index).padStart(2, "0")}`)
    const ramp = rampFromScale(temperatureScale, colours)
    expect(ramp.map((stop) => stop.value)).toEqual([18, 21, 24, 27, 30, 33, 36])
    expect(ramp[6]!.color).toEqual([0, 0, 6, 1])
  })
})

describe("stepIndex", () => {
  it("bins by lower bound, open first step included", () => {
    expect(stepIndex(temperatureScale, 5)).toBe(0)
    expect(stepIndex(temperatureScale, 21)).toBe(1)
    expect(stepIndex(temperatureScale, 26.9)).toBe(2)
    expect(stepIndex(temperatureScale, 40)).toBe(6)
    expect(stepIndex(windScale, 118)).toBe(6)
  })

  it("is below the scale under its first bound", () => {
    expect(stepIndex(rainfallScale, 0.4)).toBe(-1)
    expect(stepIndex(rainfallScale, 1)).toBe(0)
    expect(stepIndex(rainfallScale, Number.NaN)).toBe(-1)
  })
})

describe("stepExpression", () => {
  const input = ["feature-state", "value"]

  it("starts from the first colour when the first step is open", () => {
    const colours = temperatureScale.map((_, index) => `c${index}`)
    expect(stepExpression(input, temperatureScale, colours)).toEqual([
      "step",
      input,
      "c0",
      21,
      "c1",
      24,
      "c2",
      27,
      "c3",
      30,
      "c4",
      33,
      "c5",
      36,
      "c6",
    ])
  })

  it("leaves values under a bounded first step in the `below` colour", () => {
    const colours = rainfallScale.map((_, index) => `r${index}`)
    const expression = stepExpression(input, rainfallScale, colours, "none")
    expect(expression.slice(0, 5)).toEqual(["step", input, "none", 1, "r0"])
    expect(expression).toHaveLength(3 + 2 * rainfallScale.length)
  })
})
