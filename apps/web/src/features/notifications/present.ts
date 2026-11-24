/**
 * The notification inbox, pure: the server's list (`POST /api/push/inbox`)
 * checked, merged with this device's own copies (./db.ts), and filtered by
 * what the person has read and cleared. Kept free of browser globals so it
 * can be tested.
 */
import { parseSelection, type Selection } from "@/features/location/selection"
import { isNoticeId, safePath, severities, type Severity, withoutNotice } from "@/features/pwa/push"
import type { LocalNotice } from "./db"

export interface Area {
  id: string
  name: string
}

/** One thing a notice gathered: a gauge's level, a landslide warning, a bulletin. */
export interface InboxEvent {
  key: string
  kind: string
  severity: Severity
  title: string
  body: string
  url: string
  detectedAt: string
  issuedAt: string | null
  areas: Area[]
}

/** A notice as the server keeps it, written in the language asked for. */
export interface InboxNotice {
  id: string
  sentAt: string
  kind: string
  severity: Severity
  title: string
  body: string
  url: string
  /** Issued by an official agency, rather than an estimate, a forecast or community data. */
  official: boolean
  source: string
  /** About the whole country rather than districts. */
  national: boolean
  areas: Area[]
  events: InboxEvent[]
}

/** A notice in the list: the server's version when there is one, else the device's copy. */
export interface InboxItem {
  id: string
  title: string
  body: string
  url: string
  severity: Severity
  kind: string | null
  /** When it was sent, else when it arrived: the time the list shows. */
  time: string
  /** The later of sent and received, in ms: what reading and clearing compare with. */
  arrived: number
  local: LocalNotice | null
  server: InboxNotice | null
}

/** What the person has done with the list, kept on this device. */
export interface InboxMarks {
  /** Notices that arrived up to this time (ms) have been seen. */
  seenAt: number | null
  /** "Clear all": notices that arrived up to this time (ms) are gone. */
  clearedBefore: number | null
  /** Notices cleared one by one: id to when (ms). */
  cleared: Record<string, number>
}

const day = 24 * 60 * 60 * 1000
/** The server keeps 30 days; marks are kept a little longer. */
export const keepDays = 30
const markDays = 40
export const maxNotices = 100

export const emptyMarks: InboxMarks = { seenAt: null, clearedBefore: null, cleared: {} }

function text(value: unknown, max: number): string | null {
  if (typeof value !== "string") return null
  const trimmed = value.trim()
  return trimmed === "" ? null : trimmed.slice(0, max)
}

function time(value: unknown): string | null {
  return typeof value === "string" && Number.isFinite(Date.parse(value)) ? value : null
}

function severity(value: unknown): Severity {
  return (severities as readonly unknown[]).includes(value) ? (value as Severity) : "info"
}

function kind(value: unknown): string {
  return typeof value === "string" && /^[a-z][a-z_]{0,31}$/.test(value) ? value : "notice"
}

function areas(value: unknown): Area[] {
  if (!Array.isArray(value)) return []
  const seen = new Set<string>()
  const list: Area[] = []
  for (const item of value) {
    if (!item || typeof item !== "object") continue
    const raw = item as Record<string, unknown>
    const id = text(raw.id, 40)
    const name = text(raw.name, 120)
    if (!id || !name || seen.has(id)) continue
    seen.add(id)
    list.push({ id, name })
  }
  return list
}

function event(value: unknown, index: number): InboxEvent | null {
  if (!value || typeof value !== "object") return null
  const raw = value as Record<string, unknown>
  const title = text(raw.title, 300)
  const body = text(raw.body, 4000)
  if (!title && !body) return null
  const detectedAt = time(raw.detectedAt)
  return {
    key: text(raw.key, 200) ?? `event-${index}`,
    kind: kind(raw.kind),
    severity: severity(raw.severity),
    title: title ?? "",
    body: body ?? "",
    url: withoutNotice(safePath(raw.url)),
    detectedAt: detectedAt ?? "",
    issuedAt: time(raw.issuedAt),
    areas: areas(raw.areas),
  }
}

/** The server's notices, checked: anything malformed is dropped rather than shown half. */
export function parseInbox(value: unknown): InboxNotice[] {
  const list =
    value && typeof value === "object" && !Array.isArray(value)
      ? (value as { notices?: unknown }).notices
      : value
  if (!Array.isArray(list)) return []
  const notices: InboxNotice[] = []
  for (const item of list) {
    if (!item || typeof item !== "object") continue
    const raw = item as Record<string, unknown>
    const sentAt = time(raw.sentAt)
    const title = text(raw.title, 300)
    if (!isNoticeId(raw.id) || !sentAt || !title) continue
    notices.push({
      id: raw.id,
      sentAt,
      kind: kind(raw.kind),
      severity: severity(raw.severity),
      title,
      body: text(raw.body, 4000) ?? "",
      url: withoutNotice(safePath(raw.url)),
      official: raw.official === true,
      source: text(raw.source, 200) ?? "",
      national: raw.national === true,
      areas: areas(raw.areas),
      events: Array.isArray(raw.events)
        ? raw.events.map(event).filter((item): item is InboxEvent => item !== null)
        : [],
    })
  }
  return notices
}

function ms(iso: string | null | undefined): number {
  const value = iso ? Date.parse(iso) : Number.NaN
  return Number.isFinite(value) ? value : 0
}

/**
 * The device's copies and the server's list as one, keyed by id, newest
 * first. The server's wording wins (it has the details and the language
 * asked for); a notice only this device has, or only the server has, is
 * kept as it is.
 */
export function mergeInbox(local: readonly LocalNotice[], server: readonly InboxNotice[]) {
  const byId = new Map<string, InboxItem>()
  for (const notice of local) {
    const sent = notice.sentAt ?? notice.receivedAt
    byId.set(notice.id, {
      id: notice.id,
      title: notice.title,
      body: notice.body,
      url: notice.url,
      severity: notice.severity,
      kind: notice.kind,
      time: sent,
      arrived: Math.max(ms(sent), ms(notice.receivedAt)),
      local: notice,
      server: null,
    })
  }
  for (const notice of server) {
    const mine = byId.get(notice.id)?.local ?? null
    byId.set(notice.id, {
      id: notice.id,
      title: notice.title,
      body: notice.body,
      url: notice.url,
      severity: notice.severity,
      kind: notice.kind,
      time: notice.sentAt,
      arrived: Math.max(ms(notice.sentAt), ms(mine?.receivedAt)),
      local: mine,
      server: notice,
    })
  }
  return [...byId.values()].sort(
    (a, b) => ms(b.time) - ms(a.time) || b.arrived - a.arrived || b.id.localeCompare(a.id),
  )
}

export function isCleared(item: InboxItem, marks: InboxMarks): boolean {
  if (item.id in marks.cleared) return true
  return marks.clearedBefore !== null && item.arrived <= marks.clearedBefore
}

/** What the list shows: not cleared, at most the newest hundred. */
export function visibleItems(items: readonly InboxItem[], marks: InboxMarks): InboxItem[] {
  return items.filter((item) => !isCleared(item, marks)).slice(0, maxNotices)
}

export function isUnread(item: InboxItem, seenAt: number | null): boolean {
  return seenAt === null || item.arrived > seenAt
}

export function unreadCount(items: readonly InboxItem[], seenAt: number | null): number {
  return items.filter((item) => isUnread(item, seenAt)).length
}

/** The badge's text: up to 9, then "9+". */
export function badgeText(count: number): string | null {
  if (count <= 0) return null
  return count > 9 ? "9+" : String(count)
}

/**
 * Everything listed now has been seen. Unchanged (the same object) when it
 * already was, so it can run on every render of the list. Device and server
 * clocks may differ: it never goes back.
 */
export function markSeen(marks: InboxMarks, items: readonly InboxItem[], now: number): InboxMarks {
  const newest = items.reduce((latest, item) => Math.max(latest, item.arrived), 0)
  if (marks.seenAt !== null && newest <= marks.seenAt) return marks
  return { ...marks, seenAt: Math.max(now, newest, marks.seenAt ?? 0) }
}

export function clearOne(marks: InboxMarks, id: string, now: number): InboxMarks {
  return { ...marks, cleared: { ...marks.cleared, [id]: now } }
}

/** Clears every notice that has arrived, however the clocks differ; a later notice still shows. */
export function clearAll(marks: InboxMarks, items: readonly InboxItem[], now: number): InboxMarks {
  const newest = items.reduce((latest, item) => Math.max(latest, item.arrived), 0)
  const before = Math.max(now, newest, marks.clearedBefore ?? 0)
  return { ...markSeen(marks, items, now), clearedBefore: before, cleared: {} }
}

/** Marks read from storage, defensively, without one-by-one clears older than the server keeps. */
export function readMarks(value: unknown, now: number): InboxMarks {
  if (!value || typeof value !== "object") return emptyMarks
  const raw = value as Record<string, unknown>
  const number = (item: unknown) =>
    typeof item === "number" && Number.isFinite(item) && item > 0 ? item : null
  const cleared: Record<string, number> = {}
  if (raw.cleared && typeof raw.cleared === "object") {
    for (const [id, at] of Object.entries(raw.cleared as Record<string, unknown>)) {
      const when = number(at)
      if (when !== null && now - when < markDays * day) cleared[id] = when
    }
  }
  return { seenAt: number(raw.seenAt), clearedBefore: number(raw.clearedBefore), cleared }
}

/** Ids of stored copies to delete: older than the server keeps, or beyond the newest hundred. */
export function expiredLocal(local: readonly LocalNotice[], now: number): string[] {
  const sorted = [...local].sort((a, b) => ms(b.receivedAt) - ms(a.receivedAt))
  return sorted
    .filter((notice, index) => index >= maxNotices || now - ms(notice.receivedAt) > keepDays * day)
    .map((notice) => notice.id)
}

/** The `notice` id in a query string, or null. */
export function noticeParam(search: string): string | null {
  const value = new URLSearchParams(search).get("notice")
  return isNoticeId(value) ? value : null
}

/** The address without its `notice` parameter, or null when it has none. */
export function withoutNoticeParam(href: string): string | null {
  const url = new URL(href)
  if (!url.searchParams.has("notice")) return null
  url.searchParams.delete("notice")
  return `${url.pathname}${url.search.replace(/%2C/g, ",")}${url.hash}`
}

/** Layers to turn on so what a link selects is drawn on the map. */
const selectionLayers: Partial<Record<Selection["kind"], string[]>> = {
  station: ["river-levels"],
  gauge: ["flood-forecast"],
  basin: ["flood-forecast"],
  area: ["flood-forecast"],
  river: ["rivers"],
  past: ["past-floods"],
  landslide: ["landslide-warnings"],
  quake: ["earthquakes"],
  cyclone: ["cyclones"],
  warnings: ["official-warnings"],
  met: ["weather-stations"],
  reservoir: ["reservoirs"],
  reservoirs: ["reservoirs"],
  raingauge: ["rain-gauges"],
}

/** What a notice's map link opens, read the way the app reads its own address. */
export function linkTarget(path: string): { selection: Selection | null; enable: string[] } {
  const url = new URL(safePath(path), "https://lanka-weather.invalid")
  const selection = parseSelection(url.search)
  return { selection, enable: selection ? (selectionLayers[selection.kind] ?? []) : [] }
}

/** Districts a notice concerns: its own list, else those of its events. */
export function noticeAreas(notice: InboxNotice): Area[] {
  if (notice.areas.length > 0) return notice.areas
  const seen = new Map<string, Area>()
  for (const item of notice.events) for (const area of item.areas) seen.set(area.id, area)
  return [...seen.values()]
}

export const severityBadge = {
  info: "chip",
  watch: "alert",
  warning: "minor_flood",
  danger: "major_flood",
} as const satisfies Record<Severity, string>

export const severityColor: Record<Severity, string> = {
  info: "var(--status-unknown)",
  watch: "var(--status-alert)",
  warning: "var(--status-minor-flood)",
  danger: "var(--status-major-flood)",
}
