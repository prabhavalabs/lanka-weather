/**
 * Reading a push message and turning it into a notification. Pure, so the
 * service worker (src/sw.ts) and the tests share it; it must not import
 * React or anything that touches `window`.
 *
 * The payload the server sends, as JSON in the push message data:
 *
 *   {
 *     "title": string,
 *     "body": string,         // up to about 300 characters
 *     "url": string,          // a same-origin path to open, with the map state and
 *                             // the notice, e.g. "/?station=panadugama&notice=123"
 *     "tag": string,          // replaces an earlier notice with the same tag
 *     "severity": "info" | "watch" | "warning" | "danger",
 *     "sentAt": ISO-8601 string,
 *     "requireInteraction"?: boolean,
 *     "lang"?: "en" | "si" | "ta",
 *     "id"?: string,          // the notice's id, digits, e.g. "123": unique per notice
 *                             // per device; the in-app inbox (/api/push/inbox) uses it
 *     "kind"?: string         // what it is about: bulletin, landslide, river,
 *                             // flood_forecast, flash_flood, cyclone, earthquake,
 *                             // reservoir, rain, flood_risk or test
 *   }
 *
 * The service worker also keeps a copy of every notice it shows in
 * IndexedDB (features/notifications/db.ts), for the in-app inbox.
 *
 * A malformed or empty message still shows a generic notice: browsers
 * penalise push messages that show nothing, and a warning service should
 * rather say "open the map" than stay silent.
 */
import { isLocale, locales, type Locale } from "@/lib/i18n/locales"
import { pwa as en } from "@/locales/en/pwa"
import { pwa as si } from "@/locales/si/pwa"
import { pwa as ta } from "@/locales/ta/pwa"

export const severities = ["info", "watch", "warning", "danger"] as const
export type Severity = (typeof severities)[number]

export interface PushPayload {
  title: string
  body: string
  url: string
  tag: string
  severity: Severity
  sentAt: string
  requireInteraction?: boolean
  lang?: Locale
  /** The notice's id on the server, digits only. */
  id?: string
  /** What the notice is about, e.g. "river" or "flash_flood". */
  kind?: string
}

/** What the service worker hands to `registration.showNotification`. */
export interface Notice {
  title: string
  options: NoticeOptions
}

/** NotificationOptions with the members TypeScript's DOM library leaves out. */
export interface NoticeOptions extends NotificationOptions {
  badge: string
  icon: string
  tag: string
  renotify: boolean
  requireInteraction: boolean
  timestamp?: number
  vibrate?: number[]
  data: NoticeData
}

/** Kept on the notification for the click handler. */
export interface NoticeData {
  url: string
  severity: Severity
  sentAt: string | null
  /** False when the message could not be read and this is the generic notice. */
  parsed: boolean
  /** The notice's id on the server; null for an old payload or the generic notice. */
  id: string | null
  kind: string | null
  /** The language the notice is written in. */
  lang: Locale
}

export const noticeIcon = "/icon-192.png"
/** Monochrome (white on transparent): Android draws it in the status bar. */
export const noticeBadge = "/badge-96.png"
/** Tag of the generic notice, so repeated unreadable messages do not pile up. */
export const genericTag = "lanka-weather"

const maxTitle = 120
const maxBody = 600
const maxTag = 100

const texts = { en, si, ta }

function clean(value: unknown, max: number): string | null {
  if (typeof value !== "string") return null
  const trimmed = value.replace(/\s+/g, " ").trim()
  return trimmed === "" ? null : trimmed.slice(0, max)
}

/**
 * A path on this site, or "/" for anything else. Absolute URLs, protocol-
 * relative `//host` and backslash tricks are refused, so a notification can
 * never open another site.
 */
export function safePath(value: unknown): string {
  if (typeof value !== "string") return "/"
  const path = value.trim()
  if (!path.startsWith("/") || path.startsWith("//") || path.includes("\\")) return "/"
  for (const char of path) {
    const code = char.charCodeAt(0)
    if (code < 0x20 || code === 0x7f) return "/"
  }
  return path.slice(0, 2000)
}

const idPattern = /^[0-9]{1,20}$/
const kindPattern = /^[a-z][a-z_]{0,31}$/

/** A notice id: 1 to 20 digits. */
export function isNoticeId(value: unknown): value is string {
  return typeof value === "string" && idPattern.test(value)
}

function isKind(value: unknown): value is string {
  return typeof value === "string" && kindPattern.test(value)
}

function isSeverity(value: unknown): value is Severity {
  return typeof value === "string" && (severities as readonly string[]).includes(value)
}

/** The payload, or null when the message is not JSON or has no title or body. */
export function parsePushPayload(data: unknown): PushPayload | null {
  let value = data
  if (typeof value === "string") {
    try {
      value = JSON.parse(value)
    } catch {
      return null
    }
  }
  if (typeof value !== "object" || value === null || Array.isArray(value)) return null
  const raw = value as Record<string, unknown>
  const title = clean(raw.title, maxTitle)
  const body = clean(raw.body, maxBody)
  if (title === null && body === null) return null
  const sentAt =
    typeof raw.sentAt === "string" && Number.isFinite(Date.parse(raw.sentAt)) ? raw.sentAt : ""
  return {
    title: title ?? "",
    body: body ?? "",
    url: safePath(raw.url),
    tag: clean(raw.tag, maxTag) ?? "",
    severity: isSeverity(raw.severity) ? raw.severity : "info",
    sentAt,
    ...(typeof raw.requireInteraction === "boolean"
      ? { requireInteraction: raw.requireInteraction }
      : {}),
    ...(isLocale(raw.lang) ? { lang: raw.lang } : {}),
    ...(isNoticeId(raw.id) ? { id: raw.id } : {}),
    ...(isKind(raw.kind) ? { kind: raw.kind } : {}),
  }
}

/** The first of the device's languages we have, for a message that names none. */
export function pickLocale(languages: readonly string[]): Locale {
  for (const language of languages) {
    const base = language.toLowerCase().split("-")[0]
    if (isLocale(base)) return base
  }
  return "en"
}

/**
 * The notification for a push message's text. `fallbackLocale` words the
 * generic notice and fills in a missing title.
 */
export function noticeFromPush(data: unknown, fallbackLocale: Locale = "en"): Notice {
  const payload = parsePushPayload(data)
  const locale = payload?.lang ?? fallbackLocale
  const words = texts[locale].push
  if (!payload) {
    return {
      title: words.genericTitle,
      options: {
        body: words.genericBody,
        icon: noticeIcon,
        badge: noticeBadge,
        tag: genericTag,
        renotify: true,
        requireInteraction: false,
        lang: locales[locale].tag,
        data: {
          url: "/",
          severity: "info",
          sentAt: null,
          parsed: false,
          id: null,
          kind: null,
          lang: locale,
        },
      },
    }
  }
  const timestamp = payload.sentAt ? Date.parse(payload.sentAt) : undefined
  const urgent = payload.severity === "danger" || payload.severity === "warning"
  return {
    title: payload.title || words.genericTitle,
    options: {
      body: payload.body,
      icon: noticeIcon,
      badge: noticeBadge,
      tag: payload.tag || genericTag,
      // A replaced notice (same tag) still alerts: a warning that got worse must be heard.
      renotify: true,
      requireInteraction: payload.severity === "danger" || payload.requireInteraction === true,
      lang: locales[locale].tag,
      ...(timestamp !== undefined ? { timestamp } : {}),
      ...(urgent ? { vibrate: [200, 100, 200] } : {}),
      data: {
        url: payload.url,
        severity: payload.severity,
        sentAt: payload.sentAt || null,
        parsed: true,
        id: payload.id ?? null,
        kind: payload.kind ?? null,
        lang: locale,
      },
    },
  }
}

/** The URL a clicked notification opens: its path on this site, else the map. */
export function noticeUrl(data: unknown, origin: string): string {
  const path =
    typeof data === "object" && data !== null ? safePath((data as { url?: unknown }).url) : "/"
  return new URL(path, origin).href
}

/** A same-origin path without its `notice` parameter: the map state alone. */
export function withoutNotice(path: string): string {
  const url = new URL(safePath(path), "https://lanka-weather.invalid")
  if (!url.searchParams.has("notice")) return safePath(path)
  url.searchParams.delete("notice")
  return `${url.pathname}${url.search.replace(/%2C/g, ",")}${url.hash}`
}

/** The message the service worker posts to open windows when a notice arrives. */
export const inboxMessage = "lanka-weather:inbox"

/** Where the service worker reports a renewed subscription (the push alerts API). */
export const renewSubscriptionPath = "/api/push/subscriptions/renew"

export interface RenewBody {
  oldEndpoint: string | null
  subscription: PushSubscriptionJSON
}

/** The body of the renew request, as the push alerts API expects it. */
export function renewBody(
  oldEndpoint: string | null | undefined,
  subscription: PushSubscriptionJSON,
): RenewBody {
  return { oldEndpoint: oldEndpoint ?? null, subscription }
}
