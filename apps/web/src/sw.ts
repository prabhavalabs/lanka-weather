/**
 * The service worker (ADR 0017), built by vite-plugin-pwa's injectManifest
 * strategy into /sw.js.
 *
 * - Precaches the app shell (HTML, scripts, styles, fonts, icons), so the
 *   app opens with no connection, and answers navigations with it.
 * - Caches map tiles and API data at runtime, with bounded storage
 *   (features/pwa/routes.ts).
 * - Shows push alerts and opens the app at the alert when one is tapped.
 *
 * Updates wait: a new worker is installed in the background and takes over
 * only when the page asks (the "Reload" toast posts SKIP_WAITING), so the
 * app is never reloaded under someone reading a warning.
 */
import { clientsClaim } from "workbox-core"
import { CacheableResponsePlugin } from "workbox-cacheable-response"
import { ExpirationPlugin } from "workbox-expiration"
import {
  cleanupOutdatedCaches,
  createHandlerBoundToURL,
  precacheAndRoute,
  type PrecacheEntry,
} from "workbox-precaching"
import { NavigationRoute, registerRoute } from "workbox-routing"
import { CacheFirst, NetworkFirst, StaleWhileRevalidate, type Strategy } from "workbox-strategies"
import {
  noticeFromPush,
  noticeUrl,
  pickLocale,
  renewBody,
  renewSubscriptionPath,
} from "./features/pwa/push"
import {
  cacheLimits,
  cacheNames,
  cacheRuleFor,
  navigationDenylist,
  networkTimeoutSeconds,
  type CacheRule,
} from "./features/pwa/routes"

declare const self: ServiceWorkerGlobalScope & {
  __WB_MANIFEST: (PrecacheEntry | string)[]
}

// The app shell. The list is written in at build time.
cleanupOutdatedCaches()
precacheAndRoute(self.__WB_MANIFEST)

// Every navigation gets the shell; the app reads its state from the URL.
registerRoute(
  new NavigationRoute(createHandlerBoundToURL("/index.html"), { denylist: navigationDenylist }),
)

/** Plugins for a cache: only whole, successful answers are kept, within its limits. */
function plugins(rule: Exclude<CacheRule, null>) {
  return [
    new CacheableResponsePlugin({ statuses: [200] }),
    new ExpirationPlugin({ ...cacheLimits[rule], purgeOnQuotaError: true }),
  ]
}

const strategies: Record<Exclude<CacheRule, null>, Strategy> = {
  "api-tiles": new StaleWhileRevalidate({
    cacheName: cacheNames["api-tiles"],
    plugins: plugins("api-tiles"),
  }),
  "rain-tiles": new CacheFirst({
    cacheName: cacheNames["rain-tiles"],
    plugins: plugins("rain-tiles"),
  }),
  "basemap-styles": new StaleWhileRevalidate({
    cacheName: cacheNames["basemap-styles"],
    plugins: plugins("basemap-styles"),
  }),
  "basemap-tiles": new CacheFirst({
    cacheName: cacheNames["basemap-tiles"],
    plugins: plugins("basemap-tiles"),
  }),
  "basemap-assets": new CacheFirst({
    cacheName: cacheNames["basemap-assets"],
    plugins: plugins("basemap-assets"),
  }),
  "api-data": new NetworkFirst({
    cacheName: cacheNames["api-data"],
    networkTimeoutSeconds,
    plugins: plugins("api-data"),
  }),
}

registerRoute(
  ({ request, url }) =>
    request.method === "GET" &&
    // Byte ranges are left to the network: a cached whole file must not answer one.
    !request.headers.has("range") &&
    cacheRuleFor(url, self.location.origin) !== null,
  (options) => {
    const rule = cacheRuleFor(options.url, self.location.origin)
    return rule ? strategies[rule].handle(options) : fetch(options.request)
  },
)

self.addEventListener("message", (event) => {
  if ((event.data as { type?: string } | null)?.type === "SKIP_WAITING") void self.skipWaiting()
})

// The first worker takes the open page at once, so it works offline from the first visit.
// Later versions wait for SKIP_WAITING (above) before they activate.
clientsClaim()

/** The device's language for a message that does not name one. */
function deviceLocale() {
  return pickLocale(self.navigator.languages ?? [self.navigator.language])
}

function pushText(event: PushEvent): string | null {
  try {
    return event.data?.text() ?? null
  } catch {
    return null
  }
}

self.addEventListener("push", (event) => {
  const notice = noticeFromPush(pushText(event), deviceLocale())
  event.waitUntil(self.registration.showNotification(notice.title, notice.options))
})

self.addEventListener("notificationclick", (event) => {
  event.notification.close()
  const target = noticeUrl(event.notification.data, self.location.origin)
  event.waitUntil(
    (async () => {
      const windows = await self.clients.matchAll({ type: "window", includeUncontrolled: true })
      // An open window of the app is reused: focused, then sent to the alert.
      const open = windows.find((client) => new URL(client.url).origin === self.location.origin)
      if (open) {
        await open.focus().catch(() => undefined)
        if (open.url === target) return
        const moved = await open.navigate(target).catch(() => null)
        if (moved) return
        // A window this worker does not control cannot be navigated: open a new one instead.
      }
      await self.clients.openWindow(target)
    })(),
  )
})

/** Fired when the push service replaces a subscription (Firefox, and Chrome on key rotation). */
interface PushSubscriptionChangeEvent extends ExtendableEvent {
  readonly newSubscription: PushSubscription | null
  readonly oldSubscription: PushSubscription | null
}

self.addEventListener("pushsubscriptionchange", (rawEvent) => {
  const event = rawEvent as PushSubscriptionChangeEvent
  event.waitUntil(
    (async () => {
      const old = event.oldSubscription
      let next = event.newSubscription ?? (await self.registration.pushManager.getSubscription())
      if (!next && old?.options.applicationServerKey) {
        next = await self.registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: old.options.applicationServerKey,
        })
      }
      if (!next) return
      await fetch(renewSubscriptionPath, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(renewBody(old?.endpoint, next.toJSON())),
      })
    })(),
  )
})
