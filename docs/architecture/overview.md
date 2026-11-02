# Architecture overview

lanka-weather collects weather, hydrological and hazard data about Sri
Lanka from public sources, normalises it, stores it with provenance, and
serves it to a map-centred web application.

## Goals

1. **One map of Sri Lanka** on which every relevant layer can be
   switched on together: wind, rain, cyclones, river levels, reservoirs,
   flood forecasts, landslide warnings and official bulletins.
2. **Earlier warning.** Surface rising river levels and approaching
   thresholds as soon as the source publishes them, not when a bulletin
   is read out on the news.
3. **Honest data.** Every value carries its source, the time the source
   observed or issued it, and the time we fetched it. Stale layers are
   marked as stale. Derived risk is labelled as an estimate; official
   Department of Meteorology and DMC warnings remain authoritative.
4. **Cheap to run.** One service and one database. No per-visitor calls
   to third-party APIs.

## System context

```
 Public sources                      lanka-weather                        People
 ─────────────                       ─────────────                        ──────
 Irrigation Dept gauges ─┐
 DMC bulletins / PDFs  ──┤     ┌──────────────────────────┐
 Met Dept observations ──┤     │ worker (Go)              │
 NBRO landslide levels ──┼────▶│  connectors → records    │
 Google Flood Forecast ──┤     │  scheduler, jobs         │
 NOAA GFS (wind, rain) ──┤     └────────────┬─────────────┘
 GDACS / JTWC cyclones ──┤                  ▼
 USGS earthquakes      ──┤     ┌──────────────────────────┐
 NASA GIBS imagery     ──┘     │ PostgreSQL + PostGIS     │
                               │  readings, warnings,     │
 nuuuwan data repos  ─ backfill│  stations, regions, jobs │
                               └────────────┬─────────────┘
                                            ▼
                               ┌──────────────────────────┐        ┌──────────────┐
                               │ api (Go, same binary)    │──JSON─▶│ web (React,  │
                               │  /v1/layers, /v1/...     │        │ Vite, MapLibre)
                               └──────────────────────────┘        └──────────────┘
```

Imagery tiles (NASA GIBS rain and cloud) and basemap tiles
(OpenFreeMap) are loaded by the browser directly, because they are
public, cacheable tile services designed for that purpose. Every other
source is reached only by the worker.

## Components

### `cmd/lankaweather` (Go)

A single binary with sub-commands, following the modular-monolith
pattern used in TerraGuard:

| Command | Role |
|---|---|
| `serve` | HTTP API and, in production, the built web application |
| `worker` | Scheduler and connectors; one leader elected with a Postgres advisory lock |
| `migrate` | Database migrations (goose) |
| `collect` | Run every collector once and exit |

Internal packages (planned):

| Package | Responsibility |
|---|---|
| `internal/layers` | Catalogue of map layers: identifier, kind, source, refresh cadence, staleness limit, attribution |
| `internal/source` | Contract every connector implements. Decoding is pure: no network or database, so recorded payloads can be replayed in tests |
| `internal/connectors/<name>` | One package per source (irrigation, adminregions, osmplaces; planned: dmc, meteo, nbro, googleflood, gfs, gdacs, usgs) |
| `internal/regions` | Region codes and crosswalk; boundaries as GeoJSON and vector tiles, region detail, point lookup |
| `internal/places` | Place and region search |
| `internal/geo` | TopoJSON decoding, simplification along shared edges, well-known binary |
| `internal/ingest` | Scheduler, leader election, retries, fetch log |
| `internal/store` | Database access (pgx, sqlc) |
| `internal/httpapi` | Routes, request validation, response envelopes, caching headers |
| `internal/weather`, `internal/grib2` | GFS wind fields, decoded on demand and cached in memory |

### `apps/web` (TypeScript)

React with Vite. Data fetching uses TanStack Query; the query cache is
persisted to IndexedDB so that the last known state of every layer is
shown immediately on a return visit or on a poor connection, then
refreshed in the background. Each layer's `staleTime` and refetch
interval come from the layer catalogue served by the API, so cadence is
defined once.

The map uses MapLibre GL with OpenFreeMap basemaps, vector tiles from the
API for administrative regions, PMTiles for other static geography
(rivers), GeoJSON sources for stations
and warnings, raster tiles for imagery, and a canvas overlay for
animated wind.

### PostgreSQL + PostGIS

Stores stations, readings, warnings, regions and the job queue. PostGIS
supplies spatial joins (which gauges are in which district, which
divisions are under a landslide warning) and bounding-box queries.

## Data flow for one layer: river levels

1. The worker polls the Irrigation Department gauge service every
   15 minutes (ADR 0004).
2. The connector decodes the payload into readings: station, level,
   unit, observed time.
3. Readings are upserted; the station's alert, minor-flood and
   major-flood thresholds are joined to give a status and a trend.
4. `GET /v1/rivers/levels` returns GeoJSON with status, trend,
   observed time and fetched time; the response sets `Cache-Control`
   to match the cadence.
5. The web app renders the stations, colours them by status and shows
   the age of each reading.

## Geography and identifiers

Administrative regions come from `nuuuwan/lk_admin_regions` (ADR 0006):
9 provinces, 25 districts, 340 divisional secretariat (DS) divisions and
14,019 Grama Niladhari (GN) divisions, with 2024 census populations from
`nuuuwan/lk_census_2024`. They are loaded weekly into the `regions`
table, one row per region with its `level` and boundary
(`geometry(MultiPolygon, 4326)`, GiST-indexed), so warnings, gauges and
weather-grid cells can be joined to regions by code or by
`ST_Intersects`.

A region's id is its Department of Census and Statistics (DCS) code with
one hyphen after the country prefix; each code starts with its parent's:

| Level | DCS id | Compact (lanka-data-layer) | ISO 3166-2 |
|---|---|---|---|
| Province | `LK-1` | `LK1` | `LK-1` |
| District | `LK-11` | `LK11` | `LK-11`, except Northern Province (below) |
| DS division | `LK-1103` | `LK1103` | none |
| GN division | `LK-1103005` | `LK1103005` | none |

The crosswalk is in `internal/regions`: `ParseID` accepts the compact
and fully hyphenated (`LK-1-1-03`) forms and returns the DCS id, and
`ISOCode` maps DCS district codes to ISO 3166-2, which numbers the
Northern Province's districts differently: DCS `LK-42` Mannar is ISO
`LK-43`, `LK-43` Vavuniya is `LK-44`, `LK-44` Mullaitivu is `LK-45` and
`LK-45` Kilinochchi is `LK-42`. The ISO code is stored on each province
and district row (`iso_code`).

Boundaries are served as vector tiles for every level and as GeoJSON for
provinces, districts and DS divisions. Populated places come from
OpenStreetMap; place and region names in English, Sinhala and Tamil are
searchable with typo tolerance (`pg_trgm`).

## Non-goals for the first release

- User accounts, alert subscriptions and push notifications.
- Our own hydrological or meteorological models.
- Social-media ingestion. News and public reports are evaluated later,
  subject to each platform's terms.

## Related documents

- [Data-source catalogue](../data-sources.md)
- [Architecture decision records](../decisions/)
