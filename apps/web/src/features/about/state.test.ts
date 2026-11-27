import { describe, expect, it } from "vitest"
import { wantsAbout, withAbout } from "./state"

describe("the About page's address", () => {
  it("is read from ?about, alone or beside the map's own state", () => {
    expect(wantsAbout("?about")).toBe(true)
    expect(wantsAbout("?warnings=all&about")).toBe(true)
    expect(wantsAbout("?warnings=all")).toBe(false)
  })

  it("is added and taken away without touching the rest", () => {
    expect(withAbout("https://example.org/?warnings=all#x", true)).toBe("/?warnings=all&about#x")
    expect(withAbout("https://example.org/", true)).toBe("/?about")
    expect(withAbout("https://example.org/?warnings=all&about", false)).toBe("/?warnings=all")
    expect(withAbout("https://example.org/?about", false)).toBe("/")
  })

  it("is left alone when it already says so", () => {
    expect(withAbout("https://example.org/?about", true)).toBeNull()
    expect(withAbout("https://example.org/", false)).toBeNull()
  })
})
