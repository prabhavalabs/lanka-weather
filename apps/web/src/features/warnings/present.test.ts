import { afterEach, describe, expect, it } from "vitest"
import { setLocale, translateIn } from "@/lib/i18n"
import {
  bannerLead,
  bannerView,
  bulletinAlert,
  bulletinBadge,
  bulletinEntries,
  editionTitle,
  formatReportDate,
  groupEditions,
  highlightLines,
  landslideAlert,
  landslideNote,
  landslideStatusNote,
  landslideSummary,
  levelsShown,
  officialSlot,
  presentEditions,
  pruneDismissed,
  scriptLang,
  titleStem,
  visibleBanner,
} from "./present"
import type { BannerItem, Bulletin, LandslideHere, LandslideMeta, OfficialAt } from "./types"

const meta: LandslideMeta = {
  source: "s",
  attribution: "a",
  status: "active",
  note: "Report of 31 Oct 2026, valid until 1 Nov 2026 16:00.",
  limit: "A level may already have been withdrawn.",
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

describe("landslide notes in the reader's language", () => {
  afterEach(() => setLocale("en"))

  it("says in English what the server says, from the status and dates", () => {
    expect(landslideStatusNote(meta, meta.fetchedAt)).toEqual({ text: meta.note })
    expect(landslideStatusNote({ ...meta, status: "none", note: "x" }, meta.fetchedAt).text).toBe(
      "Last report 31 Oct 2026; none currently active as far as we know.",
    )
  })

  it("never calls an old feed calm, in any language", () => {
    const unknown = { ...meta, status: "unknown" as const }
    expect(landslideStatusNote(unknown, null).text).toContain("cannot say")
    expect(landslideStatusNote(unknown, meta.fetchedAt, "ta").text).toContain("கூற முடியாது")
  })

  it("keeps the server's words, marked English, when the facts are missing", () => {
    expect(landslideStatusNote({ ...meta, reportDate: null }, null, "si")).toEqual({
      text: meta.note,
      lang: "en",
    })
  })

  it("writes the note and the level in Sinhala", () => {
    setLocale("si")
    expect(landslideStatusNote(meta, meta.fetchedAt).text).toContain("වලංගුයි")
    expect(landslideNote(here)).toContain("Yatinuwara")
    expect(landslideNote(here)).toContain("ආරක්ෂිත")
    const alert = landslideAlert({
      ...here,
      level: { level: 3, label: "Level 3 (Red)", colour: "Red", action: "Evacuate" },
    })
    // The level and colour words are the shared official terms.
    expect(alert?.badgeLabel).toBe(
      `${translateIn("si", "terms.landslideLevel.l3")} · ${translateIn("si", "terms.landslideAction.l3")}`,
    )
    expect(bulletinBadge(bulletin).label).toBe(
      `${translateIn("si", "terms.bulletinSeverity.warning")} · ${translateIn("si", "terms.colour.red")}`,
    )
  })
})

describe("editions", () => {
  // As the DMC published them on 3 Nov 2026: the Tamil edition came 25 minutes
  // after the English one, with no usable text.
  const english: Bulletin = {
    ...bulletin,
    id: "en-1120",
    colour: "Amber",
    messageId: "08 MSA /26/11/03/01",
    issuedAt: "2026-11-03T05:50:00Z",
    pdfUrl: "https://www.dmc.gov.lk/en.pdf",
  }
  const tamil: Bulletin = {
    ...bulletin,
    id: "ta-1145",
    title: "Warning for strong winds and rough seas (Tamil Language)",
    language: "ta",
    colour: null,
    messageId: null,
    validUntil: null,
    text: null,
    textStatus: "unusable",
    districts: [],
    issuedAt: "2026-11-03T06:15:00Z",
    pdfUrl: "https://www.dmc.gov.lk/ta.pdf",
  }
  const report: Bulletin = {
    ...bulletin,
    id: "report-ta",
    type: "weather",
    severity: "routine",
    title: "Weather report (Tamil Language)",
    language: "ta",
    text: null,
    districts: [],
    issuedAt: "2026-11-01T09:30:00Z",
  }

  it("reduces titles to what their editions share", () => {
    expect(titleStem(tamil.title)).toBe(titleStem(english.title))
    expect(titleStem("Weather Forecast - Tamil")).toBe("weather forecast")
    expect(editionTitle(tamil)).toBe("Warning for strong winds and rough seas")
    expect(editionTitle({ title: "Weather Forecast - Tamil", language: "ta" })).toBe(
      "Weather Forecast",
    )
    expect(editionTitle(english)).toBe(english.title)
    expect(scriptLang("කාලගුණ")).toBe("si")
    expect(scriptLang("வானிலை")).toBe("ta")
    expect(scriptLang(english.title)).toBe("en")
  })

  it("pairs the Tamil edition with the English one and leaves others alone", () => {
    const groups = groupEditions([tamil, english, report])
    expect(groups.map((group) => group.map((item) => item.id))).toEqual([
      ["ta-1145", "en-1120"],
      ["report-ta"],
    ])
  })

  it("pairs numbered updates with their own editions, nearest first", () => {
    const later = { ...english, id: "en-0930", issuedAt: "2026-11-01T09:30:00Z" }
    const earlier = { ...english, id: "en-0700", issuedAt: "2026-11-01T07:00:00Z" }
    const laterTa = { ...tamil, id: "ta-0930", issuedAt: "2026-11-01T09:30:00Z" }
    const earlierTa = { ...tamil, id: "ta-0700", issuedAt: "2026-11-01T07:00:00Z" }
    const groups = groupEditions([later, laterTa, earlier, earlierTa])
    expect(groups.map((group) => group.map((item) => item.id).sort())).toEqual([
      ["en-0930", "ta-0930"],
      ["en-0700", "ta-0700"],
    ])
  })

  it("does not pair editions issued far apart", () => {
    const old = { ...tamil, issuedAt: "2026-11-02T05:50:00Z" }
    expect(groupEditions([english, old])).toHaveLength(2)
  })

  it("shows a Tamil reader the Tamil edition, with the English text marked as such", () => {
    const entry = presentEditions([tamil, english], "ta")
    expect(entry.shown.id).toBe("ta-1145")
    expect(entry.title).toBe("Warning for strong winds and rough seas")
    expect(entry.others.map((item) => item.id)).toEqual(["en-1120"])
    expect(entry.note).toBeNull()
    expect(entry.text).toBe(english.text)
    expect(entry.textFromEnglish).toBe(true)
    // The colour, message and period are read from the English text.
    expect(entry.merged.colour).toBe("Amber")
    expect(entry.merged.messageId).toBe(english.messageId)
    expect(entry.merged.validUntil).toBe(english.validUntil)
    expect(entry.merged.districts).toEqual(english.districts)
  })

  it("tells a Sinhala reader when there is no Sinhala edition", () => {
    const entry = presentEditions([tamil, english], "si")
    expect(entry.shown.id).toBe("en-1120")
    expect(entry.note).toBe("missingSi")
    expect(entry.others.map((item) => item.id)).toEqual(["ta-1145"])
    expect(entry.textFromEnglish).toBe(false)
    expect(presentEditions([english], "ta").note).toBe("onlyEn")
  })

  it("shows an English reader the English edition, and a Tamil-only one as published", () => {
    expect(presentEditions([tamil, english], "en").shown.id).toBe("en-1120")
    expect(presentEditions([tamil, english], "en").note).toBeNull()
    expect(presentEditions([report], "en").note).toBe("onlyTa")
    expect(presentEditions([report], "ta").note).toBeNull()
  })

  it("finds a filtered-out edition in the full list and hides no bulletin", () => {
    const entries = bulletinEntries([english, report], [tamil, english, report], "ta")
    expect(entries.map((entry) => entry.shown.id)).toEqual(["ta-1145", "report-ta"])
    const all = bulletinEntries([tamil, english, report], [], "en")
    expect(all.flatMap((entry) => entry.editions).length).toBe(3)
  })

  it("announces the reader's edition in the banner", () => {
    const item: BannerItem = {
      key: "bulletin:en-1120",
      kind: "bulletin",
      title: english.title,
      detail: "Weather",
      colour: "Amber",
      issuedAt: english.issuedAt,
      pdfUrl: english.pdfUrl,
    }
    const [tamilView] = bannerView([item], [tamil, english], "ta")
    expect(tamilView?.title).toBe("Warning for strong winds and rough seas")
    expect(tamilView?.pdfUrl).toBe(tamil.pdfUrl)
    expect(tamilView?.detail).toBe(translateIn("ta", "terms.bulletinType.weather"))
    expect(tamilView?.edition).toBe("ta")
    expect(tamilView?.key).toBe(item.key)
    const [englishView] = bannerView([item], [tamil, english], "en")
    expect(englishView?.title).toBe(english.title)
    expect(englishView?.edition).toBeUndefined()
  })

  it("writes the landslide banner from its counts", () => {
    const item: BannerItem = {
      key: "landslide:2026-10-31:2:1",
      kind: "landslide",
      title: "Landslide early warning",
      detail: "Level 3 (Red): 2 DS divisions; Level 2 (Amber): 1",
      level: 3,
      colour: "Red",
      issuedAt: null,
      pdfUrl: null,
    }
    expect(bannerView([item], [], "en")[0]?.detail).toBe(item.detail)
    expect(bannerView([{ ...item, key: "landslide:2026-10-31:0:1" }], [], "en")[0]?.detail).toBe(
      "Level 2 (Amber): 1 DS division",
    )
    expect(bannerView([item], [], "si")[0]?.title).toBe("නායයෑම් පිළිබඳ පූර්ව අනතුරු ඇඟවීම")
  })
})
