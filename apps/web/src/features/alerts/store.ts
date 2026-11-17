/**
 * Alert state for the whole app: one store, so the toolbar's and the phone
 * bar's buttons agree. Permission is asked for only in `enable`, which runs
 * from a tap, never on load.
 *
 * The service worker (apps/web/src/sw.ts, registered in production builds)
 * shows the notices and calls `/api/push/subscriptions/renew` when the push
 * service replaces a subscription. Here we only subscribe, keep the server's
 * copy of the preferences in step, and unsubscribe.
 */
import { useSyncExternalStore } from "react"
import { ApiError } from "@/lib/api"
import { getLocale, type Locale, type MessageKey } from "@/lib/i18n"
import * as api from "./api"
import {
  type AlertsStatus,
  type Capabilities,
  defaultPreferences,
  isIOS,
  keyBytes,
  type Preferences,
  problem,
  readStoredAlerts,
  sameKey,
  statusOf,
  type StoredAlerts,
  topics,
} from "./present"

const storageKey = "lanka-weather:alerts"
const workerTimeout = 8000
const saveDelay = 700

export type SaveState = "idle" | "saving" | "saved" | "failed"

export interface AlertsState {
  status: AlertsStatus
  busy: boolean
  save: SaveState
  /** A message for the last action, as a catalogue key. */
  message: MessageKey | null
  /** When this device's subscription was made, from the server. */
  since: string | null
  preferences: Preferences
}

type Stored = StoredAlerts

function fresh(): Stored {
  return { preferences: defaultPreferences(), enabled: false, known: [...topics] }
}

/**
 * The stored settings. Default topics added since they were saved are turned
 * on and the list of known topics brought up to date, then written back: the
 * server's copy is replaced with these on every start, so they must not drop
 * a topic the person never saw.
 */
function readStored(): Stored {
  let raw: unknown
  try {
    raw = JSON.parse(localStorage.getItem(storageKey) ?? "null")
  } catch {
    return fresh()
  }
  const read = readStoredAlerts(raw)
  if (raw !== null) writeStored(read)
  return read
}

function writeStored(stored: Stored) {
  try {
    localStorage.setItem(storageKey, JSON.stringify(stored))
  } catch {
    // Storage may be disabled; the server still has the preferences.
  }
}

let stored: Stored = typeof window === "undefined" ? fresh() : readStored()
let state: AlertsState = {
  status: "checking",
  busy: false,
  save: "idle",
  message: null,
  since: null,
  preferences: stored.preferences,
}
let publicKey: string | null = null
let started: Promise<void> | null = null
let saveTimer: ReturnType<typeof setTimeout> | undefined
const listeners = new Set<() => void>()

function set(next: Partial<AlertsState>) {
  state = { ...state, ...next }
  for (const listener of listeners) listener()
}

function remember(next: Partial<Stored>) {
  stored = { ...stored, ...next }
  writeStored(stored)
}

export function capabilities(): Capabilities {
  const push =
    typeof navigator !== "undefined" &&
    "serviceWorker" in navigator &&
    typeof window !== "undefined" &&
    "PushManager" in window &&
    "Notification" in window
  const standalone =
    typeof window !== "undefined" &&
    (window.matchMedia?.("(display-mode: standalone)").matches ||
      (navigator as Navigator & { standalone?: boolean }).standalone === true)
  return {
    push,
    ios: isIOS(navigator.userAgent, navigator.platform, navigator.maxTouchPoints ?? 0),
    installed: standalone,
    permission: "Notification" in window ? Notification.permission : "unsupported",
  }
}

/** The registered service worker, waiting up to a few seconds for one. */
async function worker(wait: boolean): Promise<ServiceWorkerRegistration | null> {
  const existing = await navigator.serviceWorker.getRegistration().catch(() => undefined)
  if (existing || !wait) return existing ?? null
  return Promise.race([
    navigator.serviceWorker.ready,
    new Promise<null>((resolve) => setTimeout(() => resolve(null), workerTimeout)),
  ])
}

async function currentSubscription(): Promise<PushSubscription | null> {
  const registration = await worker(false)
  return (await registration?.pushManager.getSubscription().catch(() => null)) ?? null
}

/** This device's push subscription, or null when it has none. */
export async function subscriptionJSON(): Promise<PushSubscriptionJSON | null> {
  if (typeof navigator === "undefined" || !("serviceWorker" in navigator)) return null
  return (await currentSubscription())?.toJSON() ?? null
}

/** Puts the server's copy in step, subscribing again if it has forgotten us. */
async function sync(subscription: PushSubscription, locale: Locale) {
  const json = subscription.toJSON()
  try {
    const view = await api.updatePreferences(json, state.preferences, locale)
    set({ since: view.createdAt })
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      const view = await api.subscribe(json, state.preferences, locale)
      set({ since: view.createdAt })
      return
    }
    throw error
  }
}

async function check() {
  const device = capabilities()
  let serverEnabled: boolean | null = null
  if (device.push && !(device.ios && !device.installed)) {
    try {
      const key = await api.getPushKey()
      serverEnabled = key.enabled
      publicKey = key.publicKey ?? null
    } catch {
      serverEnabled = null
    }
  }
  let subscribed: boolean | null = null
  let subscription: PushSubscription | null = null
  if (serverEnabled && device.permission !== "denied") {
    subscription = await currentSubscription()
    subscribed = subscription !== null && device.permission === "granted"
  }
  const status = statusOf({ capabilities: device, serverEnabled, subscribed })
  // A failed key request is not "checking" forever: say so and let a retry ask again.
  set({ status: serverEnabled === null && status === "checking" ? "unavailable" : status })
  if (status === "on" && subscription) {
    if (!stored.enabled) remember({ enabled: true })
    sync(subscription, getLocale()).catch(() => set({ save: "failed" }))
  } else if (status === "off" && stored.enabled) {
    remember({ enabled: false })
  }
}

/** Reads the device's state once; `refresh` reads it again. */
export function start(): Promise<void> {
  started ??= check()
  return started
}

export function refresh(): Promise<void> {
  started = check()
  return started
}

/** Turns alerts on. Call it straight from a tap: browsers ask for permission only then. */
export async function enable(locale: Locale) {
  if (state.busy) return
  if (problem(state.preferences)) return
  set({ busy: true, message: null })
  try {
    const permission = await Notification.requestPermission()
    if (permission === "denied") return set({ status: "denied" })
    if (permission !== "granted") return set({ message: "alerts.dismissed" })
    if (!publicKey) await refresh()
    if (!publicKey) return set({ status: "disabled" })
    const registration = await worker(true)
    if (!registration) return set({ status: "unavailable" })
    const key = keyBytes(publicKey)
    let subscription = await registration.pushManager.getSubscription()
    if (subscription && !sameKey(subscription.options.applicationServerKey, key)) {
      // Made with another server key: it cannot receive our notices.
      await subscription.unsubscribe()
      subscription = null
    }
    subscription ??= await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: key,
    })
    const view = await api.subscribe(subscription.toJSON(), state.preferences, locale)
    remember({ enabled: true })
    set({ status: "on", since: view.createdAt, save: "idle" })
  } catch {
    set({ message: "alerts.failed" })
  } finally {
    set({ busy: false })
  }
}

/** Turns alerts off on this device and deletes the server's copy. */
export async function disable() {
  if (state.busy) return
  set({ busy: true, message: null })
  try {
    const subscription = await currentSubscription()
    if (subscription) {
      const json = subscription.toJSON()
      await subscription.unsubscribe().catch(() => false)
      await api.unsubscribe(json).catch((error: unknown) => {
        if (!(error instanceof ApiError && error.status === 404)) throw error
      })
    }
    remember({ enabled: false })
    set({ status: "off", since: null, save: "idle" })
  } catch {
    set({ message: "alerts.failed" })
  } finally {
    set({ busy: false })
  }
}

export async function test() {
  if (state.busy) return
  set({ busy: true, message: null })
  try {
    const subscription = await currentSubscription()
    if (!subscription) return set({ status: "off", message: "alerts.gone" })
    await api.sendTest(subscription.toJSON())
    set({ message: "alerts.testSent" })
  } catch (error) {
    if (error instanceof ApiError && error.status === 429) set({ message: "alerts.testTooSoon" })
    else if (error instanceof ApiError && (error.status === 410 || error.status === 404)) {
      remember({ enabled: false })
      set({ status: "off", message: "alerts.gone" })
    } else set({ message: "alerts.failed" })
  } finally {
    set({ busy: false })
  }
}

/** Saves the preferences here at once and on the server shortly after, when on. */
export function choose(preferences: Preferences, locale: Locale) {
  remember({ preferences })
  set({ preferences })
  if (state.status !== "on" || problem(preferences)) return
  clearTimeout(saveTimer)
  set({ save: "saving" })
  saveTimer = setTimeout(() => {
    void saveNow(locale)
  }, saveDelay)
}

/** Sends the language and preferences now, for example after the language changed. */
export async function saveNow(locale: Locale) {
  if (state.status !== "on" || problem(state.preferences)) return
  set({ save: "saving" })
  try {
    const subscription = await currentSubscription()
    if (!subscription) return set({ status: "off", save: "idle", message: "alerts.gone" })
    await sync(subscription, locale)
    set({ save: "saved" })
  } catch {
    set({ save: "failed" })
  }
}

function subscribeStore(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function useAlerts(): AlertsState {
  return useSyncExternalStore(
    subscribeStore,
    () => state,
    () => state,
  )
}
