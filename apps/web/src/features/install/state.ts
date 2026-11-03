/**
 * The install guide's shared state: whether it is open and why, the
 * browser's deferred install prompt, and whether the app was just
 * installed. A tiny external store, so any code (the push alerts flow, the
 * sidebar, a shared link) can open the guide without passing props.
 *
 * Loaded at start-up (the sidebar imports it), so `beforeinstallprompt`,
 * which Chrome fires once early on, is not missed.
 */
import { useSyncExternalStore } from "react"
import { toast } from "sonner"
import { translate } from "@/lib/i18n"
import { markInstalled } from "./visits"

/** Why the guide was opened; "alerts" leads with the iPhone alerts note. */
export type GuideReason = "menu" | "link" | "prompt" | "alerts"

/** Chrome, Edge and Samsung Internet's event; not in TypeScript's DOM library. */
export interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>
  readonly userChoice: Promise<{ outcome: "accepted" | "dismissed"; platform: string }>
}

export interface InstallState {
  open: boolean
  reason: GuideReason
  /** The browser can install in one tap (`beforeinstallprompt` fired). */
  canPrompt: boolean
  /** `appinstalled` fired during this visit. */
  installed: boolean
  /** The push alerts code registered a way to open its settings. */
  hasAlertSettings: boolean
}

/** Event that opens the guide, for code that would rather not import this module. */
export const openInstallGuideEvent = "lanka-weather:open-install-guide"

let deferred: BeforeInstallPromptEvent | null = null
let alertSettings: (() => void) | null = null
let state: InstallState = {
  open: false,
  reason: "menu",
  canPrompt: false,
  installed: false,
  hasAlertSettings: false,
}
const listeners = new Set<() => void>()

function update(next: Partial<InstallState>) {
  state = { ...state, ...next }
  for (const listener of listeners) listener()
}

/**
 * Opens the install guide. The push alerts code calls it with
 * `{ reason: "alerts" }` when someone on iPhone taps "Get alerts" in a
 * browser tab, where Apple allows no web push.
 */
export function openInstallGuide(options: { reason?: GuideReason } = {}) {
  update({ open: true, reason: options.reason ?? "menu" })
}

export function closeInstallGuide() {
  update({ open: false })
}

/**
 * Lets the guide's "installed" view link to the alert settings. Returns a
 * function that removes the link again.
 */
export function registerAlertSettingsOpener(open: () => void): () => void {
  alertSettings = open
  update({ hasAlertSettings: true })
  return () => {
    if (alertSettings !== open) return
    alertSettings = null
    update({ hasAlertSettings: false })
  }
}

export function openAlertSettings() {
  if (!alertSettings) return
  closeInstallGuide()
  alertSettings()
}

/** Shows the browser's own install dialog, when it offered one. */
export async function promptInstall(): Promise<"accepted" | "dismissed" | "unavailable"> {
  const event = deferred
  if (!event) return "unavailable"
  // The event can be used once.
  deferred = null
  update({ canPrompt: false })
  try {
    await event.prompt()
    const choice = await event.userChoice
    return choice.outcome
  } catch {
    return "unavailable"
  }
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function getInstallState(): InstallState {
  return state
}

export function useInstallState(): InstallState {
  return useSyncExternalStore(subscribe, getInstallState, getInstallState)
}

if (typeof window !== "undefined") {
  window.addEventListener("beforeinstallprompt", (event) => {
    // Our guide and prompt take the place of the browser's banner, at a calmer moment.
    event.preventDefault()
    deferred = event as BeforeInstallPromptEvent
    update({ canPrompt: true })
  })
  window.addEventListener("appinstalled", () => {
    deferred = null
    markInstalled()
    update({ canPrompt: false, installed: true, open: false })
    toast.success(translate("install.thanks.title"), {
      id: "install-thanks",
      description: translate("install.thanks.body"),
    })
  })
  window.addEventListener(openInstallGuideEvent, (event) => {
    const reason = (event as CustomEvent<{ reason?: GuideReason } | undefined>).detail?.reason
    openInstallGuide(reason ? { reason } : {})
  })
}
