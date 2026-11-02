# lanka-weather

A weather and disaster early-warning portal for Sri Lanka. It gathers
forecasts, observations and hazard data from public sources into one
interactive map, so that people can see what is happening now and what
is likely to happen next in their area.

> Status: early development. The service serves the layer catalogue and
> the map shows Sri Lanka; data collectors are being added layer by
> layer. See `docs/architecture/overview.md`.

## Purpose

Sri Lanka has frequent, often sudden, severe weather: monsoon floods,
landslides, cyclones and strong winds. The relevant information already
exists but is spread across many sources and formats. This project
brings those sources together, normalises them, and presents them as
layers on a single map of Sri Lanka, with warnings that are easy to
understand.

Planned data layers include:

- wind speed and direction (animated), rainfall and temperature;
- cyclone tracks and marine warnings;
- flood forecasts (Google Flood Forecasting) and river gauge water levels;
- reservoir and tank levels, with overflow risk;
- Department of Meteorology forecasts and warnings;
- Disaster Management Centre (DMC) alerts and NBRO landslide warnings;
- current conditions from weather APIs;
- public reports and social signals, where licensing allows.

## Repository layout

| Path | Contents |
|---|---|
| `cmd/lankaweather/` | Go entry point: `serve` (HTTP API); `worker` and `migrate` follow |
| `internal/` | Go packages: layer catalogue, HTTP API, connectors, ingest, store, regions and place search |
| `apps/web/` | Front end: React, Vite, TanStack Query with a persisted IndexedDB cache, MapLibre, shadcn/ui |
| `brand/` | Logo, wordmark and icon sources, with usage and colours |
| `packages/` | Shared TypeScript packages, when needed |
| `docs/` | Architecture overview, data-source catalogue and decision records |
| `.github/` | Pull-request template and CI workflow |

## Not versioned

The following are deliberately kept out of Git (see `.gitignore`):

- downloaded or derived data, caches and research material (`data/`,
  `cache/`, `research/`, raster and tile files);
- secrets and local configuration (`.env*`, keys, credentials);
- configuration and instruction files for AI agents and editor
  assistants.

Use `.env.example` files to document required configuration.

## Getting started

Prerequisites: Go 1.25 or later, Node.js 24 or later, pnpm (enabled
through Corepack: `corepack enable`) and Docker.

```bash
cp .env.example .env   # local settings; .env is never committed
make bootstrap         # install Node and Go dependencies
make db-up             # PostgreSQL with PostGIS on 127.0.0.1:55440
make collect           # migrate, then fetch every source once (about 2 minutes)
make dev-api           # terminal 1: API on http://localhost:8080
make dev-worker        # terminal 2: scheduler that keeps data current
make dev-web           # terminal 3: web app on http://localhost:5173
```

The Vite dev server proxies `/v1` and `/healthz` to the API. Set
`LANKAWEATHER_API_URL` to point it elsewhere, and `LANKAWEATHER_ADDR` to
change the API's listen address.

In development, `http://localhost:5173/?kit` shows the design kit: every
component in the current theme with placeholder data. After changing the
artwork in `brand/`, regenerate the icons with
`pnpm --filter @lanka-weather/web icons`.

### Commands

| Command | Purpose |
|---|---|
| `lankaweather migrate` | Apply database migrations |
| `lankaweather serve` | HTTP API |
| `lankaweather worker` | Scheduler: runs each collector at its interval; one leader across replicas |
| `lankaweather collect` | Run every collector once and exit |

### API

| Endpoint | Returns |
|---|---|
| `GET /healthz`, `GET /readyz` | Liveness; readiness including the database |
| `GET /v1/layers` | Layer catalogue with refresh cadence and staleness limits |
| `GET /v1/rivers/levels` | River gauging stations as GeoJSON: level, flood status, trend, data age |
| `GET /v1/rivers/stations/{id}/readings?hours=72` | One station's readings (1 to 720 hours) |
| `GET /v1/regions?level=province\|district\|ds` | Region boundaries as GeoJSON, simplified per level; `level=gn&parent=LK-2130` for the GN divisions of one district or DS division |
| `GET /v1/tiles/regions/{z}/{x}/{y}.pbf` | Mapbox Vector Tiles of every level (`province` z0+, `district` z6+, `ds` z8+, `gn` z10+, up to z14) |
| `GET /v1/regions/{id}?geometry=true` | One region: names in English, Sinhala and Tamil, hierarchy, centroid, bounding box, area, 2024 population |
| `GET /v1/regions/lookup?lat=6.9271&lon=79.8612` | The province, district, DS and GN division containing a point |
| `GET /v1/places/search?q=kandy&limit=10` | Cities, towns, villages and regions by name, typo-tolerant, with coordinates, hierarchy and bounding box |

Region ids are Department of Census and Statistics codes: `LK-2`
(province), `LK-21` (district), `LK-2130` (DS division), `LK-2130120`
(GN division). Boundary responses carry an `ETag` and a day's
`Cache-Control`. The full contract, with example responses, is in
[ADR 0006](docs/decisions/0006-admin-regions-and-search.md#api-contract).

Before opening a pull request, run everything CI runs:

```bash
make ci          # format, lint, type checks, generated-code check, tests, builds;
                 # integration tests too when LANKAWEATHER_TEST_DATABASE_URL is set
```

## Contributing

Branching, commit and pull-request conventions are described in
[CONTRIBUTING.md](CONTRIBUTING.md).

## Data sources and attribution

Each data source is listed, with its licence and attribution
requirements, in [docs/data-sources.md](docs/data-sources.md). Data is shown for
public information; official warnings from the Department of
Meteorology and the Disaster Management Centre remain authoritative.
