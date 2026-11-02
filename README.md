# open-weather

A weather and disaster early-warning portal for Sri Lanka. It gathers
forecasts, observations and hazard data from public sources into one
interactive map, so that people can see what is happening now and what
is likely to happen next in their area.

> Status: initial scaffolding. The architecture is being defined in
> `docs/architecture/`.

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

## Planned repository layout

| Path | Contents |
|---|---|
| `apps/web/` | Front end: React, Vite, TanStack Query with a persisted browser cache, map rendering |
| `apps/api/` | Thin HTTP API that serves normalised layers to the front end |
| `apps/ingest/` | Scheduled collectors that fetch, validate and store source data |
| `packages/` | Shared schemas, types and utilities |
| `docs/` | Architecture, data-source catalogue and decision records |

The layout is confirmed in the architecture documentation and may
change before the first release.

## Not versioned

The following are deliberately kept out of Git (see `.gitignore`):

- downloaded or derived data, caches and research material (`data/`,
  `cache/`, `research/`, raster and tile files);
- secrets and local configuration (`.env*`, keys, credentials);
- configuration and instruction files for AI agents and editor
  assistants.

Use `.env.example` files to document required configuration.

## Getting started

Prerequisites: Node.js 24 or later, pnpm, and Docker (for the local
database). Setup and run instructions will be added with the first
application code.

## Contributing

Branching, commit and pull-request conventions are described in
[CONTRIBUTING.md](CONTRIBUTING.md).

## Data sources and attribution

Each data source is listed, with its licence and attribution
requirements, in `docs/data-sources.md` once added. Data is shown for
public information; official warnings from the Department of
Meteorology and the Disaster Management Centre remain authoritative.
