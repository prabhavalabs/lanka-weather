# ADR 0003: Single repository with a pnpm workspace and a Makefile

- Status: Accepted
- Date: 2026-11-02

## Context

The repository holds a Go service and one or more TypeScript packages.
TerraGuard kept a separate lockfile per application, which made
dependency updates and CI repetitive.

## Decision

- Go code lives at the repository root (`cmd/`, `internal/`,
  `migrations/`) as one module.
- TypeScript code lives in a pnpm workspace with a single lockfile:
  `apps/web` and, as needed, `packages/*`.
- A root `Makefile` provides `bootstrap`, `dev`, `lint`, `test`, `build`
  and `ci`, which run both toolchains. Turborepo is not used at this
  size.
- Continuous integration runs `make ci` on every pull request.

## Consequences

- One command checks the whole repository before a pull request.
- Adding a build orchestrator later remains possible if the number of
  packages grows.
