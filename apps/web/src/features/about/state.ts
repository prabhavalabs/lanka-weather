/**
 * Whether the About page is open, kept in the address (`/?about`) so the page
 * can be linked to and the back button closes it. The rest of the address,
 * the map's own state, is left as it is.
 */
import { useSyncExternalStore } from "react"

export const aboutParam = "about"

/** Whether an address's query asks for the About page. */
export function wantsAbout(search: string): boolean {
  return new URLSearchParams(search).has(aboutParam)
}

/** The address with the About page added or taken away; null when it already is so. */
export function withAbout(href: string, open: boolean): string | null {
  const url = new URL(href)
  if (url.searchParams.has(aboutParam) === open) return null
  if (open) url.searchParams.set(aboutParam, "")
  else url.searchParams.delete(aboutParam)
  // `?about=` reads better as `?about`.
  return `${url.pathname}${url.search.replace(/([?&]about)=(?=&|$)/, "$1")}${url.hash}`
}

const listeners = new Set<() => void>()
const browser = typeof window !== "undefined"
let open = browser && wantsAbout(window.location.search)
/** Whether this page added the history entry the About page sits on. */
let pushed = false

function emit() {
  for (const listener of listeners) listener()
}

if (browser) {
  window.addEventListener("popstate", () => {
    const next = wantsAbout(window.location.search)
    pushed = false
    if (next !== open) {
      open = next
      emit()
    }
  })
}

export function openAbout() {
  if (open) return
  const next = withAbout(window.location.href, true)
  if (next !== null) {
    window.history.pushState(window.history.state, "", next)
    pushed = true
  }
  open = true
  emit()
}

export function closeAbout() {
  if (!open) return
  if (pushed) {
    // Back to the entry the page was opened from; popstate closes it.
    window.history.back()
    return
  }
  const next = withAbout(window.location.href, false)
  if (next !== null) window.history.replaceState(window.history.state, "", next)
  open = false
  emit()
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function useAboutOpen(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => open,
    () => false,
  )
}
