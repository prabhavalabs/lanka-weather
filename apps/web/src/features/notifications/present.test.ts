import { describe, expect, it } from "vitest"
import { noticeFromPush } from "@/features/pwa/push"
import { formatRecent } from "@/lib/format"
import { isInboxId, type LocalNotice, localNoticeFrom, readLocalNotice } from "./db"
import {
  badgeText,
  clearAll,
  clearOne,
  emptyMarks,
  expiredLocal,
  type InboxNotice,
  isUnread,
  linkTarget,
  markSeen,
  mergeInbox,
  noticeAreas,
  noticeParam,
  parseInbox,
  readMarks,
  unreadCount,
  visibleItems,
  withoutNoticeParam,
} from "./present"

const minute = 60_000
const day = 24 * 60 * minute
const now = Date.parse("2026-11-17T12:00:00Z")
const iso = (ms: number) => new Date(ms).toISOString()

function local(id: string, receivedAt: number, extra: Partial<LocalNotice> = {}): LocalNotice {
  return {
    id,
    title: `Local ${id}`,
    body: "Pushed text",
    url: "/?station=panadugama",
    severity: "warning",
    sentAt: iso(receivedAt - minute),
    kind: "river",
    lang: "en",
    receivedAt: iso(receivedAt),
    ...extra,
  }
}

function server(id: string, sentAt: number, extra: Partial<InboxNotice> = {}): InboxNotice {
  return {
    id,
    sentAt: iso(sentAt),
    kind: "river",
    severity: "danger",
    title: `Server ${id}`,
    body: "Server text",
    url: "/?station=panadugama",
    official: true,
    source: "Irrigation Department",
    national: false,
    areas: [{ id: "LK-31", name: "Galle" }],
    events: [],
    ...extra,
  }
}

const wire = {
  notices: [
    {
      id: "124",
      sentAt: "2026-11-17T10:00:00Z",
      kind: "bulletin",
      severity: "warning",
      title: "Heavy rain warning",
      body: "Over 100 mm expected.",
      url: "/?warnings=weather&notice=124",
      official: true,
      source: "Department of Meteorology",
      national: true,
      areas: [],
      events: [
        {
          key: "bulletin:9",
          kind: "bulletin",
          severity: "warning",
          title: "Heavy rain warning",
          body: "Over 100 mm in Sabaragamuwa. Stay alert.",
          url: "/?warnings=weather",
          detectedAt: "2026-11-17T09:55:00Z",
          issuedAt: "2026-11-17T09:30:00Z",
          areas: [{ id: "LK-91", name: "Ratnapura" }],
        },
        { key: "broken" },
      ],
    },
    { id: "not-a-number", sentAt: "2026-11-17T10:00:00Z", title: "Bad id" },
    { id: "125", sentAt: "yesterday", title: "Bad time" },
    { id: "126", sentAt: "2026-11-17T10:00:00Z", title: "" },
    {
      id: "127",
      sentAt: "2026-11-17T11:00:00Z",
      kind: "Not A Kind",
      severity: "extreme",
      title: "Odd one",
      url: "https://evil.example/",
      areas: [{ id: "LK-11", name: "Colombo" }, { id: "LK-11", name: "Colombo" }, { name: "x" }],
    },
  ],
}

describe("parseInbox", () => {
  it("keeps well-formed notices and drops the rest", () => {
    const notices = parseInbox(wire)
    expect(notices.map((notice) => notice.id)).toEqual(["124", "127"])
    const [first, odd] = notices
    expect(first).toMatchObject({
      official: true,
      national: true,
      source: "Department of Meteorology",
      url: "/?warnings=weather",
    })
    expect(first?.events).toHaveLength(1)
    expect(first?.events[0]).toMatchObject({
      key: "bulletin:9",
      issuedAt: "2026-11-17T09:30:00Z",
      areas: [{ id: "LK-91", name: "Ratnapura" }],
    })
    expect(odd).toMatchObject({
      kind: "notice",
      severity: "info",
      url: "/",
      official: false,
      body: "",
      areas: [{ id: "LK-11", name: "Colombo" }],
      events: [],
    })
  })

  it("reads an array or nothing", () => {
    expect(parseInbox(wire.notices)).toHaveLength(2)
    for (const value of [null, undefined, "x", { notices: "x" }, 5])
      expect(parseInbox(value)).toEqual([])
  })
})

describe("mergeInbox", () => {
  it("merges the device's copies with the server's list by id, newest first", () => {
    const items = mergeInbox(
      [local("1", now - 60 * minute), local("local-1", now - 5 * minute, { sentAt: null })],
      [server("1", now - 61 * minute), server("2", now - 2 * day)],
    )
    expect(items.map((item) => item.id)).toEqual(["local-1", "1", "2"])
    const [mine, both, backfilled] = items
    expect(mine?.server).toBeNull()
    expect(mine?.time).toBe(iso(now - 5 * minute))
    // The server's wording wins; the device's copy stays attached.
    expect(both?.title).toBe("Server 1")
    expect(both?.severity).toBe("danger")
    expect(both?.local?.id).toBe("1")
    expect(both?.arrived).toBe(now - 60 * minute)
    expect(backfilled?.local).toBeNull()
  })
})

describe("clearing and reading", () => {
  const items = mergeInbox(
    [local("10", now - 10 * minute)],
    [server("10", now - 11 * minute), server("11", now - 3 * day), server("12", now - 5 * minute)],
  )

  it("hides a cleared notice whatever the server sends again", () => {
    const marks = clearOne(emptyMarks, "11", now)
    expect(visibleItems(items, marks).map((item) => item.id)).toEqual(["12", "10"])
    const again = mergeInbox([], [server("11", now - 3 * day)])
    expect(visibleItems(again, marks)).toEqual([])
  })

  it("clears everything at once, and a later notice still shows", () => {
    const marks = clearAll(emptyMarks, items, now)
    expect(visibleItems(items, marks)).toEqual([])
    // A server backfill of the same notices brings nothing back.
    expect(visibleItems(mergeInbox([], [server("11", now - 3 * day)]), marks)).toEqual([])
    const later = mergeInbox([local("13", now + 2 * minute)], [server("13", now + minute)])
    expect(visibleItems(later, marks).map((item) => item.id)).toEqual(["13"])
    expect(unreadCount(visibleItems(later, marks), marks.seenAt)).toBe(1)
  })

  it("clears past a server clock that runs ahead", () => {
    const ahead = mergeInbox([], [server("20", now + 5 * minute)])
    expect(visibleItems(ahead, clearAll(emptyMarks, ahead, now))).toEqual([])
  })

  it("counts what arrived after the list was last seen", () => {
    expect(unreadCount(items, null)).toBe(3)
    const seen = markSeen(emptyMarks, items, now)
    expect(seen.seenAt).toBe(now)
    expect(unreadCount(items, seen.seenAt)).toBe(0)
    expect(isUnread(mergeInbox([local("30", now + minute)], [])[0]!, seen.seenAt)).toBe(true)
    // Seeing never goes back in time, and seeing the same list again changes nothing.
    expect(markSeen(seen, items, now - day)).toBe(seen)
    expect(markSeen(seen, items, now + day)).toBe(seen)
    expect(markSeen(seen, [], now + day)).toBe(seen)
    // The first look at an empty list still records a time.
    expect(markSeen(emptyMarks, [], now).seenAt).toBe(now)
  })

  it("shows at most nine on the badge", () => {
    expect(badgeText(0)).toBeNull()
    expect(badgeText(3)).toBe("3")
    expect(badgeText(12)).toBe("9+")
  })

  it("reads stored marks defensively and forgets old one-by-one clears", () => {
    expect(readMarks(null, now)).toEqual(emptyMarks)
    expect(
      readMarks(
        { seenAt: "x", clearedBefore: now, cleared: { a: now - day, b: now - 60 * day, c: "x" } },
        now,
      ),
    ).toEqual({ seenAt: null, clearedBefore: now, cleared: { a: now - day } })
  })
})

describe("expiredLocal", () => {
  it("drops copies older than 30 days and beyond the newest hundred", () => {
    const old = local("1", now - 31 * day)
    const many = Array.from({ length: 101 }, (_, index) =>
      local(String(100 + index), now - index * minute),
    )
    expect(expiredLocal([old, ...many], now).sort()).toEqual(["1", "200"])
  })
})

describe("the device's copy", () => {
  it("is made from the notification the worker shows", () => {
    const notice = noticeFromPush({
      title: "River at major flood",
      body: "Gin Ganga at Baddegama.",
      url: "/?station=baddegama&notice=55",
      severity: "danger",
      sentAt: "2026-11-17T11:59:00Z",
      lang: "si",
      id: "55",
      kind: "river",
    })
    expect(localNoticeFrom(notice, new Date(now), "abc")).toEqual({
      id: "55",
      title: "River at major flood",
      body: "Gin Ganga at Baddegama.",
      url: "/?station=baddegama",
      severity: "danger",
      sentAt: "2026-11-17T11:59:00Z",
      kind: "river",
      lang: "si",
      receivedAt: iso(now),
    })
  })

  it("gets an id of its own when the push had none, the generic notice too", () => {
    const old = localNoticeFrom(noticeFromPush({ title: "Old", body: "x" }), new Date(now), "Ab_9")
    expect(old.id).toBe(`local-${now}-ab9`)
    expect(isInboxId(old.id)).toBe(true)
    const generic = localNoticeFrom(noticeFromPush("garbage{"), new Date(now), "")
    expect(generic.id).toBe(`local-${now}`)
    expect(generic.url).toBe("/")
  })

  it("is read back defensively", () => {
    const copy = local("7", now)
    expect(readLocalNotice(copy)).toEqual(copy)
    expect(readLocalNotice({ ...copy, id: "../x" })).toBeNull()
    expect(readLocalNotice({ ...copy, receivedAt: "soon" })).toBeNull()
    expect(readLocalNotice({ ...copy, severity: "x", url: "//evil", lang: "fr" })).toMatchObject({
      severity: "info",
      url: "/",
      lang: "en",
    })
  })
})

describe("links", () => {
  it("reads the notice id from the address and takes it out again", () => {
    expect(noticeParam("?station=panadugama&notice=123")).toBe("123")
    expect(noticeParam("?notice=abc")).toBeNull()
    expect(noticeParam("")).toBeNull()
    expect(withoutNoticeParam("https://x.example/?station=panadugama&notice=123#map")).toBe(
      "/?station=panadugama#map",
    )
    expect(withoutNoticeParam("https://x.example/?station=panadugama")).toBeNull()
  })

  it("opens what a map link selects, with its layer", () => {
    expect(linkTarget("/?station=panadugama")).toEqual({
      selection: { kind: "station", id: "panadugama" },
      enable: ["river-levels"],
    })
    expect(linkTarget("/?warnings=water_level&notice=124")).toEqual({
      selection: { kind: "warnings", type: "water_level" },
      enable: ["official-warnings"],
    })
    expect(linkTarget("/?landslide=LK-2136").enable).toEqual(["landslide-warnings"])
    expect(linkTarget("https://evil.example/?station=x")).toEqual({ selection: null, enable: [] })
    expect(linkTarget("/")).toEqual({ selection: null, enable: [] })
  })

  it("lists a notice's districts, else its events'", () => {
    expect(noticeAreas(server("1", now)).map((area) => area.name)).toEqual(["Galle"])
    const [parsed] = parseInbox(wire)
    expect(noticeAreas(parsed!).map((area) => area.name)).toEqual(["Ratnapura"])
  })
})

describe("formatRecent", () => {
  it("names today and yesterday in Sri Lanka time", () => {
    const at = new Date("2026-11-17T12:40:00Z") // 18:10 in Colombo
    expect(formatRecent("2026-11-17T12:40:00Z", at, "en")).toBe("Today 18:10")
    expect(formatRecent("2026-11-16T12:40:00Z", at, "en")).toBe("Yesterday 18:10")
    expect(formatRecent("2026-11-16T12:40:00Z", at, "si")).toBe("ඊයේ 18:10")
    // Just after midnight in Colombo is a new day there, while UTC is still on the day before.
    expect(formatRecent("2026-11-16T18:45:00Z", new Date("2026-11-16T19:00:00Z"), "en")).toBe(
      "Today 00:15",
    )
    expect(formatRecent("2026-11-10T12:40:00Z", at, "en")).toContain("18:10")
  })
})
