/**
 * This device's copy of the notices it was sent, in IndexedDB. The service
 * worker writes one entry per push (src/sw.ts); the in-app inbox reads them,
 * so a notice can be read again with no connection.
 *
 * Shared by the worker and the page: it must not import React or touch
 * `window`. The store is opened on first use.
 */
import { createStore, delMany, set, type UseStore, values } from "idb-keyval"
import { isLocale, type Locale } from "@/lib/i18n/locales"
import {
  isNoticeId,
  type Notice,
  safePath,
  severities,
  type Severity,
  withoutNotice,
} from "@/features/pwa/push"

/** A notice as the push showed it. */
export interface LocalNotice {
  /** The server's notice id, or "local-…" for a notice that came without one. */
  id: string
  title: string
  body: string
  /** The map state it is about, without `notice=`. */
  url: string
  severity: Severity
  sentAt: string | null
  kind: string | null
  lang: Locale
  /** When this device received it. */
  receivedAt: string
}

const localIdPattern = /^local-[a-z0-9-]{1,60}$/

/** An id the inbox can hold: the server's digits, or one made on this device. */
export function isInboxId(value: unknown): value is string {
  return isNoticeId(value) || (typeof value === "string" && localIdPattern.test(value))
}

/** The copy to keep of a notification the worker is about to show. */
export function localNoticeFrom(notice: Notice, receivedAt: Date, nonce: string): LocalNotice {
  const { data } = notice.options
  const local = nonce
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, "")
    .slice(0, 40)
  return {
    id: data.id ?? `local-${receivedAt.getTime()}${local ? `-${local}` : ""}`,
    title: notice.title,
    body: notice.options.body ?? "",
    url: withoutNotice(data.url),
    severity: data.severity,
    sentAt: data.sentAt,
    kind: data.kind,
    lang: data.lang,
    receivedAt: receivedAt.toISOString(),
  }
}

function isTime(value: unknown): value is string {
  return typeof value === "string" && Number.isFinite(Date.parse(value))
}

/** A stored entry, checked: storage written by an older version may differ. */
export function readLocalNotice(value: unknown): LocalNotice | null {
  if (!value || typeof value !== "object") return null
  const raw = value as Record<string, unknown>
  if (!isInboxId(raw.id) || !isTime(raw.receivedAt)) return null
  if (typeof raw.title !== "string" || typeof raw.body !== "string") return null
  return {
    id: raw.id,
    title: raw.title,
    body: raw.body,
    url: withoutNotice(safePath(raw.url)),
    severity: (severities as readonly unknown[]).includes(raw.severity)
      ? (raw.severity as Severity)
      : "info",
    sentAt: isTime(raw.sentAt) ? raw.sentAt : null,
    kind: typeof raw.kind === "string" ? raw.kind : null,
    lang: isLocale(raw.lang) ? raw.lang : "en",
    receivedAt: raw.receivedAt,
  }
}

let store: UseStore | null = null

function inboxStore(): UseStore {
  store ??= createStore("lanka-weather-inbox", "notices")
  return store
}

export function saveLocalNotice(notice: LocalNotice): Promise<void> {
  return set(notice.id, notice, inboxStore())
}

/** Every stored notice that can be read, in no particular order. */
export async function readLocalNotices(): Promise<LocalNotice[]> {
  const all = await values<unknown>(inboxStore())
  return all.map(readLocalNotice).filter((notice): notice is LocalNotice => notice !== null)
}

export function deleteLocalNotices(ids: string[]): Promise<void> {
  return ids.length === 0 ? Promise.resolve() : delMany(ids, inboxStore())
}
