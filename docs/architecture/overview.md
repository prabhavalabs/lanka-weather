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

Internal packages (planned):

| Package | Responsibility |
|---|---|
| `internal/layers` | Catalogue of map layers: identifier, kind, source, refresh cadence, staleness limit, attribution |
| `internal/source` | Contract every connector implements. Decoding is pure: no network or database, so recorded payloads can be replayed in tests |
| `internal/connectors/<name>` | One package per source (irrigation, dmc, meteo, nbro, googleflood, gfs, gdacs, usgs) |
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

The map uses MapLibre GL with OpenFreeMap basemaps, PMTiles for static
geography (administrative regions, rivers), GeoJSON sources for stations
and warnings, raster tiles for imagery, and a canvas overlay for
animated wind.

### PostgreSQL + PostGIS

Stores stations, readings, warnings, regions and the job queue. PostGIS
supplies spatial joins (which gauges are in which district, which
divisions are under a landslide warning) and bounding-box queries.

## Data flow for one layer: river levels

1. The worker polls the Irrigation Department gauge service every
   15 minutes.
2. The connector decodes the payload into readings: station, level,
   unit, observed time.
3. Readings are upserted; the station's alert, minor-flood and
   major-flood thresholds are joined to give a status and a trend.
4. `GET /v1/layers/river-levels` returns GeoJSON with status, trend,
   observed time and fetched time; the response sets `Cache-Control`
   to match the cadence.
5. The web app renders the stations, colours them by status and shows
   the age of each reading.

## Geography and identifiers

Administrative regions come from `nuuuwan/lk_admin_regions` and the
`prabhavalabs/lanka-data-layer` packages. The two use different code
styles (`LK-11` and `LK11`); a crosswalk table maps them so every
warning can be joined to a district and divisional secretariat
division.

## Non-goals for the first release

- User accounts, alert subscriptions and push notifications.
- Our own hydrological or meteorological models.
- Social-media ingestion. News and public reports are evaluated later,
  subject to each platform's terms.

## Related documents

- [Data-source catalogue](../data-sources.md)
- [Architecture decision records](../decisions/)
