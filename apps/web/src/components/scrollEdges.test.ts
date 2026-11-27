import { describe, expect, it } from "vitest"
import { edgeMask, overflowEnds } from "./scrollEdges"

describe("overflowEnds", () => {
  it("says which ends have more past them", () => {
    expect(overflowEnds(0, 300, 300)).toEqual({ start: false, end: false })
    expect(overflowEnds(0, 300, 500)).toEqual({ start: false, end: true })
    expect(overflowEnds(100, 300, 500)).toEqual({ start: true, end: true })
    expect(overflowEnds(200, 300, 500)).toEqual({ start: true, end: false })
  })
})

describe("edgeMask", () => {
  it("fades only the ends with more past them", () => {
    expect(edgeMask(false, false)).toBeUndefined()
    expect(edgeMask(false, true)).toBe(
      "linear-gradient(to right, black, black, black calc(100% - 28px), transparent)",
    )
    expect(edgeMask(true, false)).toBe(
      "linear-gradient(to right, transparent, black 28px, black, black)",
    )
  })
})
