# ADR 0001: Go modular monolith with PostgreSQL and PostGIS

- Status: Accepted
- Date: 2026-11-02

## Context

The back end has two jobs: collect data from many public sources on a
schedule, and serve a thin read API to the web application. Both a
Node/TypeScript and a Go implementation were considered.

TerraGuard, an earlier project by the same team, already solved several
of the hard parts in Go: decoding GRIB2 wind fields from NOAA GFS by
byte range, a connector contract whose decoders can be tested by
replaying recorded payloads, a scheduler with Postgres leader election,
and a Google Flood Forecasting connector.

## Decision

- The back end is written in Go, as one binary with the sub-commands
  `serve`, `worker` and `migrate`.
- Routing uses `chi`; database access uses `pgx` with `sqlc`-generated
  queries; migrations use `goose`.
- PostgreSQL with PostGIS is the only datastore. It also holds the job
  queue, so no Redis or message broker is needed.
- The HTTP contract is described in `api/openapi.yaml`; the web client is
  generated from it.

## Consequences

- One language for collectors and API keeps the data model in one place;
  the shared contract with the web app is the OpenAPI document rather
  than shared TypeScript types.
- Code patterns from TerraGuard can be adopted. Copying TerraGuard code
  requires confirmation that its owner permits reuse.
- Contributors need both Go and Node toolchains.
