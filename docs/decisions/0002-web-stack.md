# ADR 0002: React, Vite, TanStack Query with a persisted cache, and MapLibre

- Status: Accepted
- Date: 2026-11-02

## Context

The front end is a single-page map application. People will open it
during storms, often on mobile data, when the latest data matters most
and connections are worst. Next.js and Vercel services are excluded by
project policy.

## Decision

- **Build:** React with Vite and TypeScript in strict mode.
- **Server state:** TanStack Query. The query cache is persisted to
  IndexedDB (`@tanstack/react-query-persist-client` with an `idb-keyval`
  persister), so the last known data for each layer is shown at once on
  a return visit and refreshed in the background. The persister's
  `buster` is tied to the application build so incompatible caches are
  discarded.
- **Cadence:** each layer's `staleTime` and refetch interval come from
  the layer catalogue returned by `GET /v1/layers`.
- **Map:** MapLibre GL with keyless OpenFreeMap basemaps, PMTiles for
  static geography, GeoJSON for stations and warnings, raster tiles for
  imagery, and a canvas overlay for animated wind.
- **Client state:** small UI preferences (enabled layers, basemap) are
  kept in local storage, separate from the query cache.

## Consequences

- The application remains useful offline with the last fetched data;
  the interface must show the age of every layer so old data is not
  mistaken for current data.
- Persisted data counts against browser storage quotas; large payloads
  such as wind fields are excluded from persistence.
