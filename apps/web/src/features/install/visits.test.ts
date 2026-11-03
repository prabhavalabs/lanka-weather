import { describe, expect, it } from "vitest"
import { withoutInstallParam } from "./links"
import { emptyRecord, offerRules, recordVisit, shouldOffer, visitGap } from "./visits"

const day = 24 * 60 * 60 * 1000
const start = Date.UTC(2026, 10, 1)

describe("recordVisit", () => {
  it("counts a visit after half an hour away, not on every page load", () => {
    let record = recordVisit(emptyRecord, start)
    record = recordVisit(record, start + 60_000)
    expect(record.visits).toBe(1)
    record = recordVisit(record, start + 60_000 + visitGap)
    expect(record.visits).toBe(2)
  })
})

describe("shouldOffer", () => {
  const second = { ...emptyRecord, visits: 2, lastVisit: start }

  it("waits for the second visit", () => {
    expect(shouldOffer({ ...emptyRecord, visits: 1 }, start)).toBe(false)
    expect(shouldOffer(second, start)).toBe(true)
  })

  it("stays away for two weeks after an offer and stops after three", () => {
    const offered = { ...second, offers: 1, lastOffer: start }
    expect(shouldOffer(offered, start + day)).toBe(false)
    expect(shouldOffer(offered, start + offerRules.offerGap + 1)).toBe(true)
    expect(shouldOffer({ ...offered, offers: 3 }, start + 365 * day)).toBe(false)
  })

  it("respects “Not now” for three months, and never asks once installed", () => {
    const declined = { ...second, declinedAt: start }
    expect(shouldOffer(declined, start + 30 * day)).toBe(false)
    expect(shouldOffer(declined, start + offerRules.declinePause + 1)).toBe(true)
    expect(shouldOffer({ ...second, installed: true }, start)).toBe(false)
  })
})

describe("withoutInstallParam", () => {
  it("removes ?install and keeps the rest", () => {
    expect(withoutInstallParam("https://x.example/?install")).toBe("/")
    expect(withoutInstallParam("https://x.example/?station=hanwella&install=1#map")).toBe(
      "/?station=hanwella#map",
    )
    expect(withoutInstallParam("https://x.example/?station=hanwella")).toBeNull()
  })
})
