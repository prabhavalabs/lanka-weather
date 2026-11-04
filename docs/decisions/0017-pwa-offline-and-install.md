# ADR 0017: Installable, offline-capable app, and an install guide

- Status: Accepted
- Date: 2026-11-03

## Context

ADR 0002 says the app must stay useful offline with the last data it fetched. The
persisted query cache does that for data, but only once the page itself has loaded,
and with no service worker the page does not load without a connection. Floods and
storms are also when mobile networks fail.

Push alerts are the main reason to install. On iPhone and iPad (iOS and iPadOS 16.4
and later), web push works only for a site added to the Home Screen, never in a
browser tab. Most people here have never installed a web app. They open links from
WhatsApp or Facebook, whose built-in browsers cannot install anything.

## Decision

### Service worker

`vite-plugin-pwa` with the `injectManifest` strategy builds `apps/web/src/sw.ts` into
`/sw.js` with the Workbox modules (`workbox-precaching`, `-routing`, `-strategies`,
`-expiration`, `-cacheable-response`). We write the worker ourselves, not let the
plugin generate one, because it also handles push.

- **Precache the app shell**: HTML, scripts (including MapLibre's `.mjs` worker),
  styles, the Geist, Noto Sans Sinhala and Noto Sans Tamil fonts, and icons. That is
  about 3 MB. Font subsets the interface never shows (Cyrillic, Vietnamese) and the
  manifest screenshots are left out. Every navigation outside `/v1/`, `/api/` and
  `/healthz` is answered with the precached `index.html`, and the app reads its state
  from the URL.
- **Runtime caches** (`features/pwa/routes.ts`), each with an entry limit and an age
  limit, and `purgeOnQuotaError`:

  | Requests                                                  | Strategy                   | Limit                 |
  | --------------------------------------------------------- | -------------------------- | --------------------- |
  | Our vector tiles `/v1/tiles/…pbf`                         | stale-while-revalidate     | 1500 entries, 30 days |
  | IMERG rain tiles `/v1/rain/tiles/<time>/…png` (immutable) | cache first                | 600, 3 days           |
  | OpenFreeMap styles and TileJSON                           | stale-while-revalidate     | 20, 30 days           |
  | OpenFreeMap tiles (URL names the planet build)            | cache first                | 3000, 30 days         |
  | OpenFreeMap glyphs and sprites                            | cache first                | 400, 90 days          |
  | Other `/v1/` JSON and GeoJSON                             | network first, 4 s timeout | 200, 2 days           |
  | `/v1/places/…` (search), `/v1/push/…`, `/healthz`         | network only               | none                  |

  Tiles average about 5–60 KB, so the tile caches stay within a few tens of
  megabytes in the worst case.

- **API data is cached in the worker too, on purpose.** The persisted query cache
  (IndexedDB) is the main offline copy, and when the browser reports that it is
  offline, TanStack Query sends no request at all. The worker's copy covers two cases
  the query cache cannot. One is the first start after an app update: the cache's
  `buster` is the build id, so an update empties it. The other is a connection that
  claims to be online but is not, where the 4-second timeout answers from the cache.
  The large wind fields (about 39 KB each, kept out of IndexedDB) fall under the same
  200-entry limit.
- **Only whole, successful answers are kept** (status 200). A 503 "not collected
  yet", a 404 or an opaque response is never cached. Range requests are left to the
  network.
- Cache names carry a version (`lw-api-data-v1` and so on), so a change of policy
  can start clean. `cleanupOutdatedCaches` removes old precaches.

### Updates

Updates use the prompt style. A new worker installs in the background and waits.
The app shows a sonner toast, "A new version is ready", with Reload and Later. Only
Reload posts `SKIP_WAITING`, and the page then reloads once. The app is never
reloaded under someone who is reading a warning. It looks for updates every hour
and when the app comes back into view, at most every five minutes. It registers
after the window's `load` event. That keeps the worker from competing with the
first paint, and it means a version already waiting from an earlier visit is still
offered, because sonner drops toasts raised before its toaster mounts. The dev
server runs without a worker unless `VITE_PWA_DEV=1`.

The first worker claims the open page (`clientsClaim`), so the app works offline
from the first visit. Later workers wait as above.

### Push

The worker shows push messages with this payload, sent as JSON in the push data:

```json
{
  "title": "…",
  "body": "…",
  "url": "/?warning=123",
  "tag": "warning-123",
  "severity": "info | watch | warning | danger",
  "sentAt": "2026-11-03T04:30:00Z",
  "requireInteraction": false,
  "lang": "en | si | ta"
}
```

Parsing is pure and tested (`features/pwa/push.ts`):

- `url` must be a same-origin path. Anything else becomes `/`.
- `tag` replaces an earlier notice with the same tag, and `renotify` makes the
  replacement alert again.
- `danger` always requires interaction. `warning` and `danger` vibrate.
- The icon is `/icon-192.png`. The badge is `/badge-96.png`, the mark in white on
  transparent.
- A message with no readable title or body still shows a generic notice in the
  device's language: browsers penalise push messages that show nothing.

A tapped notice focuses an open window of the app and navigates it to `url`, or
opens a new one. On `pushsubscriptionchange` the worker re-subscribes if it has
to, with the old subscription's key. It then POSTs
`{ "oldEndpoint": string | null, "subscription": PushSubscriptionJSON }` to
`/api/push/subscriptions/renew`, which the push alerts API provides. The worker
exposes nothing else; subscription code uses `navigator.serviceWorker.ready`.

### Manifest

There is one manifest, `public/site.webmanifest`, written by hand and checked by
`features/pwa/manifest.test.ts`. The plugin is set to `manifest: false`.

- `id` and `scope` are `/`, and `start_url` is `/?source=pwa`. The app removes the
  `source` marker at start-up so it is not carried into shared links.
- `display` is `standalone`, with `display_override` `["standalone", "minimal-ui"]`.
- It has `categories`, `lang`, `dir`, and the 192, 512 and maskable 512 icons.
- `theme_color` and `background_color` are the dark brand colours, which match the
  icon. `user_preferences` gives both schemes where browsers support it, and
  `index.html` keeps a `theme-color` per scheme.
- Shortcuts open views the URL already understands: official warnings
  (`?warnings=all`), river water-level warnings (`?warnings=water_level`), and
  reservoirs spilling or releasing (`?reservoirs=active`).
- The screenshots, narrow 1080×1920 and wide 1920×1080 WebP of about 100–160 KB,
  were captured from the running app.
- For Apple devices, `index.html` sets `apple-mobile-web-app-capable`,
  `mobile-web-app-capable`, and the `black-translucent` status bar (the layout
  already keeps clear of the safe-area insets), and links 180, 167 and 152 px touch
  icons. There are no startup images: they need one image per device size and
  orientation, and iOS falls back to the background colour.

### Install guide

The guide is a dialog on wide screens and a bottom drawer on phones
(`features/install`). It opens from:

- "Install app" in the sidebar footer and in the phone's layer drawer;
- the link `/?install`, for sending over WhatsApp or SMS;
- `openInstallGuide({ reason })`, or the `lanka-weather:open-install-guide` window
  event, for the push alerts flow;
- a gentle suggestion. It comes from the second visit (a visit is a load after 30
  minutes away), 25 seconds in, at most three times and 14 days apart. "Not now"
  silences it for 90 days. It never appears once the app is installed, nor in
  browsers that cannot install.

The user agent picks the steps to show first. A chooser shows every other device
and browser: iPhone or iPad with Safari, Chrome, Edge and Firefox; Android with
Chrome, Samsung Internet, Edge and Firefox; a computer with Chrome, Edge, Safari
and Firefox.

- iPadOS sends a Mac user agent; it is told from a Mac by its touch points.
- In-app browsers (Facebook, Messenger, Instagram, TikTok, LINE, WhatsApp,
  Snapchat, WeChat, X, LinkedIn, Viber, the Google app and bare Android WebViews)
  get an explanation, "Copy link", and on Android "Open in Chrome" (an `intent:`
  link).
- Where the browser offered `beforeinstallprompt`, one large Install button calls
  `prompt()`.
- Inside the installed app the guide says so, and links to alert settings if the
  push code registered them (`registerAlertSettingsOpener`).
- `appinstalled` closes the guide and thanks the person.

Each step pairs a short sentence with a drawing of the real button. The button
names stay in the browser's English, in all three languages, because that is what
the phone shows. Sinhala and Tamil are complete (`locales/*/install.ts` and
`pwa.ts`), and the longest Tamil text fits at 360 px. Motion follows the global
`prefers-reduced-motion` rule.

## Consequences

- The app opens and shows the last map with no connection, including after an
  update, within bounded storage.
- Production hosting must serve `/sw.js` and `/site.webmanifest` with
  `Cache-Control: no-cache`, so updates are found, and `/assets/*` as immutable.
  The Go server does not serve the built app yet. Whoever adds that must set these
  headers.
- The worker and the guide follow browser behaviour that changes: menu wording,
  iOS share placement, and Chrome's install icon. The steps live in one file per
  language and need checking against new browser releases. They have not been
  tried on every real device.
- The renew endpoint path, `/api/push/subscriptions/renew`, is the push alerts
  contract. The rest of the API is under `/v1/`. If push moves under `/v1/`, change
  `renewSubscriptionPath` in `features/pwa/push.ts`.
- A catalogue namespace must not have a key named `other`: the language layer
  reads an object with a string `other` as a plural message.
