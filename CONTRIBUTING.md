# Contributing

This document defines how changes are proposed, written and merged in this repository. It applies to every contributor.

## 1. Branching model

| Branch | Purpose | Created from | Merges into |
|---|---|---|---|
| `main` | Always releasable. Never committed to directly after the initial commit. | n/a | n/a |
| `feature/<short-description>` | New capability, documentation or refactoring | `main` | `main` via pull request |
| `bugfix/<short-description>` | Fix for a defect | `main` | `main` via pull request |

Naming rules:
- Lowercase kebab case, for example `feature/river-gauge-layer` or `bugfix/wind-layer-antimeridian`.
- If an issue exists, put its number first: `feature/42-flood-forecast-ingest`.
- Keep branches short-lived. Rebase on `main` before opening or updating a pull request.

## 2. Commit messages

Commits follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/).

```
<type>(<optional scope>): <subject>

<body>

<footer>
```

**Types**

| Type | Use for |
|---|---|
| `feat` | A new capability |
| `fix` | A defect fix |
| `docs` | Documentation only |
| `refactor` | Code change that neither fixes a defect nor adds a capability |
| `perf` | Performance improvement |
| `test` | Adding or correcting tests |
| `build` | Dependencies, packaging, build tooling |
| `ci` | Continuous-integration configuration |
| `chore` | Maintenance that does not change behaviour |
| `revert` | Reverting an earlier commit |

**Scopes** in this repository:

| Scope | Area |
|---|---|
| `web` | Front-end application (`apps/web`) |
| `map` | Map rendering, overlays and animation |
| `api` | HTTP API (`apps/api`) |
| `ingest` | Source collectors and schedules (`apps/ingest`) |
| `layers` | Data-layer definitions and normalisation |
| `db` | Database schema and migrations |
| `shared` | Shared packages (`packages/`) |
| `infra` | Deployment and local infrastructure |
| `docs` / `architecture` | Documentation and design records |
| `deps` | Dependency updates |
| `repo` | Repository-wide configuration |

**Subject line**
- Imperative mood ("add", not "added" or "adds").
- Lowercase, no trailing full stop, at most 72 characters.

**Body**
- Explains *what* changed and *why*, not how.
- Wrapped at 72 characters.
- Uses bullet points for lists of changes.

**Footer**
- References issues (`Refs: #42`, `Closes: #42`).
- Declares breaking changes (`BREAKING CHANGE: <description>`).

**Authorship**
- A commit is authored solely by the person who makes it.
- Commit messages contain no co-author trailers and no references to the tools used to produce the change.

Example:

```
feat(ingest): collect river gauge levels from the irrigation department

River levels are the earliest local signal of flooding. Collecting them
every 15 minutes and comparing each reading with the gauge's alert and
major-flood thresholds lets the map flag stations before an official
warning is issued.

- fetch the gauge bulletin and parse readings per station
- store readings with source timestamp and fetch timestamp
- mark readings older than two hours as stale

Refs: #12
```

## 3. Pull requests

- Each pull request covers one logical change and targets `main`.
- The title follows the commit-message format, for example `feat(ingest): add resource governor`. It becomes the squash-commit subject.
- The description uses the repository template (`.github/pull_request_template.md`):
  - summary;
  - changes;
  - motivation;
  - testing performed;
  - risks and rollback.
- Descriptions are written in a formal, factual tone. They describe the change and its evidence, not how it was produced.
- Every pull request is reviewed before merge. It is merged with **squash and merge**, and the branch is then deleted.

## 4. What is never committed

- Data of any kind: `data/`, caches, rasters (GRIB, NetCDF, GeoTIFF), tile archives, database dumps.
- Secrets and local configuration: `.env*`, keys, credentials.
- Configuration and instruction files for AI agents and editor assistants, for example `AGENTS.md`, `CLAUDE.md`, `.claude/`, `.cursor/`, `.github/copilot-instructions.md`. The full list is in `.gitignore`.
- Research workspace material (`research/`).

## 5. Before opening a pull request

```bash
make ci                  # everything CI runs, in order:
                         #   gofmt, go vet, Prettier, ESLint, tsc
                         #   go test, Vitest
                         #   Go and Vite production builds
git status --porcelain   # nothing unintended staged
```

## 6. Data sources

Every new data source must be added to `docs/data-sources.md` with its
owner, access method, update frequency, licence and attribution text.
Respect each source's terms of use and rate limits; collectors must
identify themselves and cache responses rather than fetch per user
request.
