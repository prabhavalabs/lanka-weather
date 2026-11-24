import { describe, expect, it } from "vitest"
import { pwa as en } from "@/locales/en/pwa"
import { pwa as si } from "@/locales/si/pwa"
import {
  genericTag,
  noticeBadge,
  noticeFromPush,
  noticeIcon,
  noticeUrl,
  parsePushPayload,
  pickLocale,
  renewBody,
  safePath,
  withoutNotice,
} from "./push"

const full = {
  title: "Flood warning: Kelani Ganga",
  body: "Major flood level at Hanwella. Move to higher ground.",
  url: "/?station=hanwella&notice=123",
  tag: "warning-123",
  severity: "danger",
  sentAt: "2026-11-03T04:30:00Z",
  lang: "en",
  id: "123",
  kind: "river",
}

describe("parsePushPayload", () => {
  it("reads a complete payload, as JSON text or an object", () => {
    const text = parsePushPayload(JSON.stringify(full))
    expect(text).toEqual(full)
    expect(parsePushPayload(full)).toEqual(full)
  })

  it("reads the notice id and kind, and drops malformed ones", () => {
    expect(
      parsePushPayload({ title: "A", id: "98765432109876543210", kind: "flash_flood" }),
    ).toMatchObject({
      id: "98765432109876543210",
      kind: "flash_flood",
    })
    for (const id of ["", "12a", "-1", "1".repeat(21), 123, null, " 12"]) {
      expect(parsePushPayload({ title: "A", id }), String(id)).not.toHaveProperty("id")
    }
    for (const kind of ["", "River", "flood-risk", "x".repeat(40), 7, "rain fall", "_rain"]) {
      expect(parsePushPayload({ title: "A", kind }), String(kind)).not.toHaveProperty("kind")
    }
  })

  it("reads a payload without id or kind, as older servers send", () => {
    const old: Record<string, unknown> = { ...full }
    delete old.id
    delete old.kind
    const payload = parsePushPayload(old)
    expect(payload).toEqual(old)
    expect(noticeFromPush(old).options.data).toMatchObject({ id: null, kind: null })
  })

  it("refuses what is not a JSON object or has no title or body", () => {
    for (const data of [null, undefined, "", "not json", "42", "[]", "null", { url: "/" }, 7])
      expect(parsePushPayload(data), String(data)).toBeNull()
    expect(parsePushPayload({ title: "   ", body: "" })).toBeNull()
  })

  it("fills in what is missing or wrong", () => {
    expect(
      parsePushPayload({
        body: "Heavy rain expected",
        url: "https://evil.example/",
        severity: "apocalyptic",
        sentAt: "yesterday",
        lang: "fr",
        requireInteraction: "yes",
        tag: 5,
      }),
    ).toEqual({
      title: "",
      body: "Heavy rain expected",
      url: "/",
      tag: "",
      severity: "info",
      sentAt: "",
    })
  })

  it("tidies white space and caps long text", () => {
    const payload = parsePushPayload({ title: `  A \n  title  `, body: "x".repeat(5000) })
    expect(payload?.title).toBe("A title")
    expect(payload?.body.length).toBe(600)
  })
})

describe("safePath", () => {
  it.each([
    ["/?warning=123", "/?warning=123"],
    ["/", "/"],
    ["  /?station=hanwella  ", "/?station=hanwella"],
    ["https://evil.example/", "/"],
    ["//evil.example/", "/"],
    ["/\\evil.example", "/"],
    ["javascript:alert(1)", "/"],
    ["?warning=1", "/"],
    ["/\nfoo", "/"],
    [42, "/"],
  ] as const)("%s → %s", (value, expected) => {
    expect(safePath(value)).toBe(expected)
  })
})

describe("noticeFromPush", () => {
  it("shows a full payload with the app's icon, badge and tag", () => {
    const notice = noticeFromPush(JSON.stringify(full))
    expect(notice.title).toBe(full.title)
    expect(notice.options).toMatchObject({
      body: full.body,
      icon: noticeIcon,
      badge: noticeBadge,
      tag: "warning-123",
      renotify: true,
      requireInteraction: true,
      lang: "en-GB",
      timestamp: Date.parse(full.sentAt),
      data: {
        url: "/?station=hanwella&notice=123",
        severity: "danger",
        sentAt: full.sentAt,
        parsed: true,
        id: "123",
        kind: "river",
        lang: "en",
      },
    })
    expect(notice.options.vibrate).toBeDefined()
  })

  it("asks for interaction only for danger, or when the server asks", () => {
    const watch = noticeFromPush({ ...full, severity: "watch" })
    expect(watch.options.requireInteraction).toBe(false)
    expect(watch.options.vibrate).toBeUndefined()
    const asked = noticeFromPush({ ...full, severity: "info", requireInteraction: true })
    expect(asked.options.requireInteraction).toBe(true)
  })

  it("still shows a generic notice for a malformed message", () => {
    for (const data of [null, "garbage{", "{}", JSON.stringify({ severity: "danger" })]) {
      const notice = noticeFromPush(data)
      expect(notice.title).toBe(en.push.genericTitle)
      expect(notice.options.body).toBe(en.push.genericBody)
      expect(notice.options.tag).toBe(genericTag)
      expect(notice.options.data).toEqual({
        url: "/",
        severity: "info",
        sentAt: null,
        parsed: false,
        id: null,
        kind: null,
        lang: "en",
      })
    }
  })

  it("words the generic notice in the device's language", () => {
    const notice = noticeFromPush(null, "si")
    expect(notice.options.body).toBe(si.push.genericBody)
    expect(notice.options.lang).toBe("si-LK")
  })

  it("gives a body without a title the app's name", () => {
    const notice = noticeFromPush({ body: "Rain soon", lang: "si" }, "en")
    expect(notice.title).toBe(si.push.genericTitle)
    expect(notice.options.body).toBe("Rain soon")
    expect(notice.options.tag).toBe(genericTag)
    expect(notice.options.lang).toBe("si-LK")
  })
})

describe("noticeUrl", () => {
  const origin = "https://lankaweather.example"
  it("opens the notice's path on this site, else the map", () => {
    expect(noticeUrl({ url: "/?warning=9" }, origin)).toBe(`${origin}/?warning=9`)
    // The tapped notice opens its detail on top of the map state.
    expect(noticeUrl({ url: "/?station=panadugama&notice=123" }, origin)).toBe(
      `${origin}/?station=panadugama&notice=123`,
    )
    expect(noticeUrl({ url: "https://evil.example" }, origin)).toBe(`${origin}/`)
    expect(noticeUrl(null, origin)).toBe(`${origin}/`)
  })
})

describe("withoutNotice", () => {
  it("keeps the map state and drops the notice", () => {
    expect(withoutNotice("/?station=panadugama&notice=123")).toBe("/?station=panadugama")
    expect(withoutNotice("/?notice=124&warnings=water_level")).toBe("/?warnings=water_level")
    expect(withoutNotice("/?notice=1")).toBe("/")
    expect(withoutNotice("/?at=7.2931,80.6350")).toBe("/?at=7.2931,80.6350")
    expect(withoutNotice("https://evil.example/?notice=1")).toBe("/")
  })
})

describe("pickLocale", () => {
  it("takes the first language we have", () => {
    expect(pickLocale(["fr-FR", "ta-LK", "si"])).toBe("ta")
    expect(pickLocale([])).toBe("en")
  })
})

describe("renewBody", () => {
  it("is the shape the push alerts API expects", () => {
    const subscription = { endpoint: "https://push.example/new", keys: { p256dh: "k", auth: "a" } }
    expect(renewBody("https://push.example/old", subscription)).toEqual({
      oldEndpoint: "https://push.example/old",
      subscription,
    })
    expect(renewBody(undefined, subscription).oldEndpoint).toBeNull()
  })
})
