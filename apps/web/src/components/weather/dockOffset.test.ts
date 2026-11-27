import { describe, expect, it } from "vitest"
import { clampOffset } from "./dockOffset"

const view = { width: 1000, height: 800 }

describe("clampOffset", () => {
  it("leaves a dock that is on screen where it is", () => {
    const box = { left: 100, top: 500, right: 400, bottom: 700 }
    expect(clampOffset({ x: 20, y: -30 }, box, view)).toEqual({ x: 20, y: -30 })
  })

  it("pulls a dock back inside the window, keeping a margin", () => {
    expect(
      clampOffset({ x: -200, y: 0 }, { left: -50, top: 500, right: 250, bottom: 700 }, view),
    ).toEqual({ x: -142, y: 0 })
    expect(
      clampOffset({ x: 0, y: 100 }, { left: 100, top: 650, right: 400, bottom: 850 }, view),
    ).toEqual({ x: 0, y: 42 })
  })
})
