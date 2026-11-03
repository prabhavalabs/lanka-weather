/**
 * Which requests the service worker caches, and how (ADR 0017). Pure URL
 * tests, shared by src/sw.ts and the tests.
 *
 * - Our vector tiles (`/v1/tiles/...pbf`): stale-while-revalidate. Region,
 *   basin and river geometry changes rarely; river tiles carry a version.
 * - IMERG rain tiles (`/v1/rain/tiles/<time>/...png`): cache first. Each URL
 *   names its time and the API marks it immutable.
 * - OpenFreeMap styles and TileJSON: stale-while-revalidate (small, may
 *   change). Its tiles, glyphs and sprites: cache first; tile URLs carry
 *   the planet build, so they do not change under the same URL.
 * - Other `/v1/` JSON and GeoJSON: network first with a short timeout. The
 *   persisted query cache (ADR 0002) already keeps the last good data; the
 *   service worker's copy covers a start after an app update (which empties
 *   that cache) and connections that claim to be online but are not.
 * - Search, health and push requests are never cached.
 */

export type CacheRule =
  | "api-tiles"
  | "rain-tiles"
  | "basemap-styles"
  | "basemap-tiles"
  | "basemap-assets"
  | "api-data"
  | null

export const openFreeMapOrigin = "https://tiles.openfreemap.org"

/** Cache names, versioned so a change of policy starts clean. */
export const cacheNames = {
  "api-tiles": "lw-api-tiles-v1",
  "rain-tiles": "lw-rain-tiles-v1",
  "basemap-styles": "lw-basemap-styles-v1",
  "basemap-tiles": "lw-basemap-tiles-v1",
  "basemap-assets": "lw-basemap-assets-v1",
  "api-data": "lw-api-data-v1",
} as const satisfies Record<Exclude<CacheRule, null>, string>

const day = 24 * 60 * 60

/**
 * Storage bounds per cache: entries and age in seconds. Tiles are about
 * 5 to 60 KB, so the tile caches stay within a few tens of megabytes.
 */
export const cacheLimits = {
  "api-tiles": { maxEntries: 1500, maxAgeSeconds: 30 * day },
  "rain-tiles": { maxEntries: 600, maxAgeSeconds: 3 * day },
  "basemap-styles": { maxEntries: 20, maxAgeSeconds: 30 * day },
  "basemap-tiles": { maxEntries: 3000, maxAgeSeconds: 30 * day },
  "basemap-assets": { maxEntries: 400, maxAgeSeconds: 90 * day },
  "api-data": { maxEntries: 200, maxAgeSeconds: 2 * day },
} as const satisfies Record<Exclude<CacheRule, null>, object>

/** Seconds to wait for the network before answering from the cache. */
export const networkTimeoutSeconds = 4

/** API paths that are never cached: answers that only make sense live. */
const uncachedApi = [/^\/v1\/places\//, /^\/v1\/push\//]

/** The caching rule for a GET request, or null to leave it to the network. */
export function cacheRuleFor(url: URL, origin: string): CacheRule {
  if (url.origin === openFreeMapOrigin) {
    if (url.pathname.startsWith("/styles/")) return "basemap-styles"
    if (url.pathname.startsWith("/fonts/") || url.pathname.startsWith("/sprites/")) {
      return "basemap-assets"
    }
    // Tiles end in .pbf; the TileJSON for a planet build is the bare path.
    if (/\.(pbf|mvt)$/.test(url.pathname)) return "basemap-tiles"
    return "basemap-styles"
  }
  if (url.origin !== origin) return null
  const path = url.pathname
  if (!path.startsWith("/v1/")) return null
  if (uncachedApi.some((pattern) => pattern.test(path))) return null
  if (/^\/v1\/rain\/tiles\/.+\.png$/.test(path)) return "rain-tiles"
  if (/^\/v1\/tiles\/.+\.(pbf|mvt)$/.test(path)) return "api-tiles"
  return "api-data"
}

/** Navigations the app shell must not answer: the API and the health check. */
export const navigationDenylist = [/^\/v1\//, /^\/api\//, /^\/healthz/]
