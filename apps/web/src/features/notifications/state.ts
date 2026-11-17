/**
 * The inbox's state on this device, shared by the toolbar's and the phone
 * bar's bells:
 *
 * - marks: what has been seen and cleared, in localStorage, so a sync with
 *   the server never brings a cleared notice back;
 * - the panel: open or not, which tab, which notice.
 *
 * `/?notice=123` (the link a tapped notification opens) opens that notice
 * on top of the map state the rest of the address sets. Closing it takes
 * `notice` out of the address, so a reload does not open it again.
 */
import { useSyncExternalStore } from "react"
import { emptyMarks, type InboxMarks, noticeParam, readMarks, withoutNoticeParam } from "./present"

const storageKey = "lanka-weather:inbox"

export type InboxTab = "list" | "settings"

export interface InboxView {
  open: boolean
  tab: InboxTab
  /** The notice shown in detail, or null for the list. */
  noticeId: string | null
  /** When the panel opened, what had been seen: notices after it are marked new while it is open. */
  seenBefore: number | null
}

function loadMarks(): InboxMarks {
  try {
    return readMarks(JSON.parse(localStorage.getItem(storageKey) ?? "null"), Date.now())
  } catch {
    return emptyMarks
  }
}

const browser = typeof window !== "undefined"
let marks: InboxMarks = browser ? loadMarks() : emptyMarks
const linked = browser ? noticeParam(window.location.search) : null
let view: InboxView = {
  open: linked !== null,
  tab: "list",
  noticeId: linked,
  seenBefore: marks.seenAt,
}
const listeners = new Set<() => void>()

function emit() {
  for (const listener of listeners) listener()
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function setMarks(update: (current: InboxMarks) => InboxMarks) {
  const next = update(marks)
  if (next === marks) return
  marks = next
  try {
    localStorage.setItem(storageKey, JSON.stringify(marks))
  } catch {
    // Storage may be disabled; the marks then last for the session.
  }
  emit()
}

/** Takes `notice` out of the address, keeping the map state. */
function dropNoticeParam() {
  if (!browser) return
  const cleaned = withoutNoticeParam(window.location.href)
  if (cleaned !== null) window.history.replaceState(window.history.state, "", cleaned)
}

function setView(next: Partial<InboxView>) {
  const leaving = view.noticeId !== null && next.noticeId === null
  view = { ...view, ...next }
  if (leaving) dropNoticeParam()
  emit()
}

export function openInbox(options: { tab?: InboxTab; noticeId?: string | null } = {}) {
  setView({
    open: true,
    tab: options.tab ?? "list",
    noticeId: options.noticeId ?? null,
    seenBefore: view.open ? view.seenBefore : marks.seenAt,
  })
}

export function closeInbox() {
  setView({ open: false, noticeId: null })
}

export function showTab(tab: InboxTab) {
  setView({ tab, noticeId: null })
}

export function showNotice(id: string | null) {
  setView({ tab: "list", noticeId: id })
}

export function useInboxMarks(): InboxMarks {
  return useSyncExternalStore(
    subscribe,
    () => marks,
    () => marks,
  )
}

export function useInboxView(): InboxView {
  return useSyncExternalStore(
    subscribe,
    () => view,
    () => view,
  )
}
