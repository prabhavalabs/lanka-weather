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
| `internal/` | Go packages: layer catalogue, HTTP API, and later connectors, ingest and store |
| `apps/web/` | Front end: React, Vite, TanStack Query with a persisted IndexedDB cache, MapLibre |
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

Prerequisites: Go 1.25 or later, Node.js 24 or later and pnpm (enabled
through Corepack: `corepack enable`).

```bash
make bootstrap   # install Node and Go dependencies
make dev-api     # terminal 1: API on http://localhost:8080
make dev-web     # terminal 2: web app on http://localhost:5173
```

The Vite dev server proxies `/v1` and `/healthz` to the API. Set
`LANKAWEATHER_API_URL` to point it elsewhere, and `LANKAWEATHER_ADDR` to
change the API's listen address.

Before opening a pull request, run everything CI runs:

```bash
make ci          # format, lint, type-check, tests and builds
```

## Contributing

Branching, commit and pull-request conventions are described in
[CONTRIBUTING.md](CONTRIBUTING.md).

## Data sources and attribution

Each data source is listed, with its licence and attribution
requirements, in [docs/data-sources.md](docs/data-sources.md). Data is shown for
public information; official warnings from the Department of
Meteorology and the Disaster Management Centre remain authoritative.
