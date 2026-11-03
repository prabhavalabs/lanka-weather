/// <reference types="vite-plugin-pwa/client" />
/**
 * Registers the service worker (src/sw.ts) and offers updates. Imported for
 * its side effect from main.tsx.
 *
 * - Production builds register it; the dev server only with VITE_PWA_DEV=1.
 * - A new version waits until the person taps "Reload" on the toast; the
 *   page is never reloaded unasked.
 * - Updates are looked for every hour and whenever the app comes back into
 *   view, at most every few minutes.
 * - It registers after the window's load event; see the end of this file.
 *
 * The push alerts code needs nothing from here but the registration itself:
 * `navigator.serviceWorker.ready`.
 */
import { toast } from "sonner"
import { registerSW } from "virtual:pwa-register"
import { translate } from "@/lib/i18n"
import { stripLaunchSource } from "./launch"

const hour = 60 * 60 * 1000
/** Coming back into view checks again only after this long. */
const minimumGap = 5 * 60 * 1000

export const updateToastId = "pwa-update"

function showUpdate(update: () => Promise<void>) {
  toast(translate("pwa.update.title"), {
    id: updateToastId,
    description: translate("pwa.update.description"),
    duration: Number.POSITIVE_INFINITY,
    action: { label: translate("pwa.update.reload"), onClick: () => void update() },
    cancel: { label: translate("pwa.update.later"), onClick: () => undefined },
  })
}

function watchForUpdates(registration: ServiceWorkerRegistration) {
  let last = Date.now()
  const check = () => {
    if (!navigator.onLine || registration.installing) return
    last = Date.now()
    registration.update().catch(() => {
      // Offline or the server is away; the next check tries again.
    })
  }
  setInterval(check, hour)
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible" && Date.now() - last > minimumGap) check()
  })
}

export function registerServiceWorker() {
  if (typeof window === "undefined" || !("serviceWorker" in navigator)) return
  if (!import.meta.env.PROD && import.meta.env.VITE_PWA_DEV !== "1") return

  const updateServiceWorker = registerSW({
    immediate: true,
    onNeedRefresh: () => showUpdate(() => updateServiceWorker(true)),
    onOfflineReady: () => toast.success(translate("pwa.ready"), { duration: 4000 }),
    onRegisteredSW: (_url, registration) => {
      if (registration) watchForUpdates(registration)
    },
    onRegisterError: (error: unknown) => console.warn("Service worker registration failed", error),
  })
}

if (typeof window !== "undefined") {
  const cleaned = stripLaunchSource(window.location.href)
  if (cleaned) window.history.replaceState(window.history.state, "", cleaned)
  // After the page has loaded: the worker does not compete with the first paint, and the
  // toaster is mounted by then (sonner drops toasts raised before it mounts), so a version
  // already waiting from an earlier visit is still offered.
  if (document.readyState === "complete") registerServiceWorker()
  else window.addEventListener("load", () => registerServiceWorker(), { once: true })
}
