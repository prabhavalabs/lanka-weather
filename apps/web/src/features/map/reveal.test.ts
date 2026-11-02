import { describe, expect, it } from "vitest"
import { clearCentreOffset, inClearArea, layoutInsets, sheetPeekHeight } from "./reveal"

describe("layoutInsets", () => {
  it("covers the sidebar and the detail column on wide screens", () => {
    expect(
      layoutInsets({
        desktop: true,
        sidebarCollapsed: false,
        detailOpen: true,
        viewportHeight: 900,
      }),
    ).toEqual({ top: 64, right: 404, bottom: 0, left: 376 })
    expect(
      layoutInsets({
        desktop: true,
        sidebarCollapsed: true,
        detailOpen: false,
        viewportHeight: 900,
      }).left,
    ).toBe(84)
  })

  it("covers the top bar and the sheet on phones", () => {
    expect(
      layoutInsets({
        desktop: false,
        sidebarCollapsed: false,
        detailOpen: true,
        viewportHeight: 844,
      }),
    ).toEqual({ top: 76, right: 0, bottom: 380, left: 0 })
    expect(
      layoutInsets({
        desktop: false,
        sidebarCollapsed: false,
        detailOpen: false,
        viewportHeight: 844,
      }).bottom,
    ).toBe(0)
  })

  it("caps the sheet on tall screens", () => {
    expect(sheetPeekHeight(1400)).toBe(416)
  })
})

describe("clearCentreOffset", () => {
  it("points at the middle of the clear area", () => {
    // 390 × 844 phone: clear from y 76 to 464, centre 270, map centre 422.
    expect(clearCentreOffset(390, 844, { top: 76, right: 0, bottom: 380, left: 0 })).toEqual([
      0, -152,
    ])
    // Desktop: clear from x 376 to 1036 of 1440.
    expect(clearCentreOffset(1440, 900, { top: 64, right: 404, bottom: 0, left: 376 })).toEqual([
      -14, 32,
    ])
  })
})

describe("inClearArea", () => {
  const insets = { top: 76, right: 0, bottom: 380, left: 0 }

  it("rejects points under the sheet or the bar", () => {
    expect(inClearArea({ x: 195, y: 300 }, 390, 844, insets)).toBe(true)
    expect(inClearArea({ x: 195, y: 600 }, 390, 844, insets)).toBe(false)
    expect(inClearArea({ x: 195, y: 90 }, 390, 844, insets)).toBe(false)
    expect(inClearArea({ x: 10, y: 300 }, 390, 844, insets)).toBe(false)
  })
})
