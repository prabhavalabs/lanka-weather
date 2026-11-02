import { describe, expect, it } from "vitest"
import {
  bannerLead,
  bulletinAlert,
  bulletinBadge,
  formatReportDate,
  highlightLines,
  landslideAlert,
  landslideNote,
  landslideSummary,
  levelsShown,
  officialSlot,
  pruneDismissed,
  visibleBanner,
} from "./present"
import type { BannerItem, Bulletin, LandslideHere, LandslideMeta, OfficialAt } from "./types"

const meta: LandslideMeta = {
  source: "s",
  attribution: "a",
  status: "active",
  note: "Report of 31 Oct 2026, valid until 1 Nov 2026 16:00.",
  stale: false,
  reportDate: "2026-10-31",
  issuedAt: "2026-10-31T10:30:00Z",
  validUntil: "2026-11-01T10:30:00Z",
  validityStated: true,
  messageId: "LEWM",
  pdfUrl: "https://www.dmc.gov.lk/a.pdf",
  sourcePageUrl: "https://www.dmc.gov.lk/p",
  counts: { "1": 3, "2": 1, "3": 0 },
  unresolved: [],
  fetchedAt: "2026-10-31T12:00:00Z",
  upstreamAt: "2026-10-31T12:00:00Z",
  generatedAt: "2026-10-31T12:00:00Z",
}

const bulletin: Bulletin = {
  id: "doc",
  type: "weather",
  typeLabel: "Weather",
  title: "Warning for strong winds and rough seas",
  severity: "warning",
  colour: "Red",
  messageId: null,
  issuedAt: "2026-11-02T05:00:00Z",
  validFrom: null,
  validUntil: "2026-11-03T05:00:00Z",
  current: true,
  pdfUrl: "https://www.dmc.gov.lk/w.pdf",
  sourcePageUrl: "https://www.dmc.gov.lk/p",
  language: "en",
  text: "The Severe Cyclone Murasu is very likely to move northwestwards.",
  textStatus: "clean",
  districts: [{ id: "LK-42", name: "Mannar" }],
  source: "dmc-reports",
  sourceTime: "2026-11-02T05:00:00Z",
  fetchedAt: "2026-11-02T06:00:00Z",
}

const here: LandslideHere = {
  status: "active",
  note: meta.note,
  stale: false,
  level: null,
  dsId: "LK-2136",
  ds: "Yatinuwara",
  district: "Kandy",
  reportDate: "2026-10-31",
  issuedAt: "2026-10-31T10:30:00Z",
  validUntil: "2026-11-01T10:30:00Z",
  pdfUrl: "https://www.dmc.gov.lk/a.pdf",
}

describe("official words", () => {
  it("keeps the issuer's title and colour on a bulletin's badge", () => {
    expect(bulletinBadge(bulletin)).toEqual({ variant: "major_flood", label: "Warning · Red" })
    expect(bulletinBadge({ severity: "advisory", colour: "Amber" }).label).toBe("Advisory · Amber")
    expect(bulletinBadge({ severity: "routine", colour: null }).label).toBe("Report")
    expect(bulletinBadge({ severity: "withdrawal", colour: null }).label).toBe("Withdrawal")
  })

  it("carries the bulletin's words and link into the alert list unchanged", () => {
    const alert = bulletinAlert(bulletin, "Mentioned in bulletin: Mannar")
    expect(alert.title).toBe(bulletin.title)
    expect(alert.summary).toBe(bulletin.text)
    expect(alert.excerpt).toBe(true)
    expect(alert.estimate).toBeUndefined()
    expect(alert.link?.href).toBe(bulletin.pdfUrl)
    expect(alert.badgeLabel).toBe("Warning · Red")
  })

  it("does not call a bulletin without clean text an excerpt", () => {
    expect(bulletinAlert({ ...bulletin, text: null }, "Sri Lanka").excerpt).toBe(false)
  })

  it("formats a report's date without moving it across midnight", () => {
    expect(formatReportDate("2026-10-31")).toBe("31 Oct 2026")
  })
})

describe("landslide status", () => {
  it("shows levels only for a report that can still apply", () => {
    expect(levelsShown("active")).toBe(true)
    expect(levelsShown("lapsed")).toBe(true)
    expect(levelsShown("pending")).toBe(true)
    expect(levelsShown("none")).toBe(false)
    expect(levelsShown("unknown")).toBe(false)
    expect(levelsShown("waiting")).toBe(false)
  })

  it("summarises levels, and never calls no report calm when the feed is old", () => {
    expect(landslideSummary(meta, 4)).toBe("Level 2: 1 · Level 1: 3")
    expect(landslideSummary({ ...meta, status: "none" }, 0)).toBe(
      "None active as far as we know · last 31 Oct 2026",
    )
    expect(landslideSummary({ ...meta, status: "unknown" }, 0)).toBe(
      "Cannot say · feed out of date",
    )
  })

  it("names the level for a division in the report", () => {
    const alert = landslideAlert({
      ...here,
      level: { level: 2, label: "Level 2 (Amber)", colour: "Amber", action: "Alert" },
    })
    expect(alert?.badgeLabel).toBe("Level 2 (Amber) · Alert")
    expect(alert?.severity).toBe("warning")
    expect(alert?.area).toBe("Yatinuwara, Kandy District")
    expect(alert?.link?.href).toBe(here.pdfUrl)
  })

  it("does not read a division left out of a report as safe", () => {
    expect(landslideAlert(here)).toBeNull()
    expect(landslideNote(here)).toBe(
      "Yatinuwara is not named in the landslide report of 31 Oct 2026. That does not mean it is safe.",
    )
  })

  it("passes on what the API says when no report is in force", () => {
    const note = "Last report 31 Oct 2026; none currently active as far as we know."
    expect(landslideNote({ ...here, status: "none", note })).toBe(note)
  })
})

describe("official slot", () => {
  const official: OfficialAt = {
    status: "available",
    landslide: {
      ...here,
      level: { level: 3, label: "Level 3 (Red)", colour: "Red", action: "Evacuate" },
    },
    warnings: [bulletin, { ...bulletin, id: "report", severity: "routine", current: false }],
    general: [
      { ...bulletin, id: "water", title: "Water Level", severity: "routine", current: false },
    ],
    note: "note",
    stale: false,
    fetchedAt: null,
  }

  it("counts warnings in force and leaves routine bulletins for reading", () => {
    const slot = officialSlot(official, "Kandy")
    expect(slot.alerts.map((alert) => alert.id)).toEqual(["landslide-LK-2136", "doc"])
    expect(slot.bulletins.map((alert) => alert.id)).toEqual(["report", "water"])
  })

  it("labels a district match as a mention, nothing stronger", () => {
    const slot = officialSlot(official, "Kandy")
    expect(slot.alerts[1]?.area).toBe("Mentioned in bulletin: Kandy")
    expect(slot.bulletins[1]?.area).toBe("Sri Lanka")
  })

  it("copes with no official data", () => {
    expect(officialSlot(undefined)).toEqual({ alerts: [], bulletins: [], note: undefined })
  })
})

describe("banner", () => {
  const item = (key: string, colour: string | null): BannerItem => ({
    key,
    kind: "bulletin",
    title: key,
    detail: "",
    colour,
    issuedAt: null,
    pdfUrl: null,
  })

  it("hides what was dismissed and leads with the most serious", () => {
    const items = [item("amber", "Amber"), item("red", "Red"), item("plain", null)]
    expect(bannerLead(items)?.key).toBe("red")
    expect(visibleBanner(items, ["red"]).map((entry) => entry.key)).toEqual(["amber", "plain"])
    expect(bannerLead([])).toBeUndefined()
  })

  it("forgets dismissals of what is no longer announced, so a new warning shows", () => {
    expect(pruneDismissed(["red", "old"], [item("red", "Red")])).toEqual(["red"])
  })
})

describe("highlights", () => {
  it("says why a district is marked", () => {
    expect(
      highlightLines({
        id: "LK-21",
        name: "Kandy",
        landslideLevel: 2,
        mentioned: [
          { id: "a", title: "Flood warning" },
          { id: "b", title: "Advisory" },
        ],
      }),
    ).toEqual([
      "Landslide warning: Level 2 (Amber)",
      "Mentioned in bulletin: Flood warning",
      "and 1 more",
    ])
  })
})
