# Contributing

Thank you for helping. Lanka Weather is a volunteer, open-source project,
and bug reports, data corrections, translations and code are all welcome.

By taking part you agree to follow the [code of conduct](CODE_OF_CONDUCT.md).
Report security problems privately, as described in [SECURITY.md](SECURITY.md),
not as issues.

## 1. Before you start

- For a small fix (a typo, a translation, an obvious bug) open a pull
  request directly.
- For anything larger, such as a new layer, a new data source or a change
  to how alerts work, open an issue first so the approach can be agreed
  before you write the code.
- Issues labelled `good first issue` are a good place to begin.

## 2. Set up

Prerequisites: Go (the version in `go.mod`), Node.js 24 or later, pnpm
(through Corepack: `corepack enable`) and Docker.

```bash
# 1. Fork the repository on GitHub, then:
git clone https://github.com/<you>/lanka-weather.git
cd lanka-weather
git remote add upstream https://github.com/prabhavalabs/lanka-weather.git

# 2. Install and start the local database
cp .env.example .env
make bootstrap
make db-up
```

No API keys are needed to develop or to run `make ci`. Two features stay
off until you add keys to `.env`:

- the Google flood forecast (`LANKAWEATHER_GOOGLE_API_KEY`, a Google Cloud
  key with the Flood Forecasting API enabled);
- push notifications (`LANKAWEATHER_VAPID_*`; generate a pair with
  `go run ./cmd/lankaweather vapid-keys`).

Set `LANKAWEATHER_CONTACT` to your own URL or email address. It is sent to
every data source in the User-Agent, so their operators know who to contact
about your traffic.

The README's "Getting started" section shows how to collect data and run
the API, worker and web app.

## 3. Branches

| Branch | Purpose | Created from |
|---|---|---|
| `main` | Always releasable. Protected: changes arrive only through pull requests. | n/a |
| `feature/<short-description>` | New capability, documentation or refactoring | `upstream/main` |
| `bugfix/<short-description>` | Fix for a defect | `upstream/main` |

```bash
git fetch upstream
git switch -c feature/river-gauge-layer upstream/main
```

- Lowercase kebab case, for example `feature/river-gauge-layer` or
  `bugfix/wind-layer-antimeridian`.
- If an issue exists, put its number first: `feature/42-flood-forecast-ingest`.
- Keep branches short-lived. Before opening or updating a pull request,
  rebase on `upstream/main` (`git fetch upstream && git rebase upstream/main`).

## 4. Commit messages

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
| `map` | Map rendering, overlays and animation (`apps/web/src/features`) |
| `api` | HTTP API (`internal/httpapi`) |
| `ingest` | Source collectors and schedules (`internal/connectors`, `internal/ingest`) |
| `rivers` | River, reservoir and flood-basin logic (`internal/rivers`) |
| `layers` | Data-layer definitions and normalisation (`internal/layers`) |
| `db` | Database schema, migrations and queries (`migrations`, `queries`) |
| `infra` | Deployment and local infrastructure (`deploy`, `compose.yaml`, `Dockerfile`) |
| `docs` / `architecture` | Documentation and decision records |
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

## 5. Before opening a pull request

```bash
make ci                  # everything CI runs, in order:
                         #   gofmt, go vet, Prettier, ESLint, tsc
                         #   sqlc generated-code check
                         #   go test, Vitest
                         #   Go and Vite production builds
git status --porcelain   # nothing unintended staged
```

After changing `queries/` or `migrations/`, run `make generate` and commit
the result.

Integration tests run against a real database. To run them locally:

```bash
make db-test   # creates the lankaweather_test database
LANKAWEATHER_TEST_DATABASE_URL=postgres://lankaweather:lankaweather@127.0.0.1:55440/lankaweather_test?sslmode=disable make test-integration
```

## 6. Pull requests

- Push the branch to your fork and open a pull request against
  `prabhavalabs/lanka-weather:main`.
- Each pull request covers one logical change.
- The title follows the commit-message format, for example
  `feat(ingest): add resource governor`. It becomes the squash-commit
  subject, and the description becomes its body.
- The description uses the repository template
  (`.github/pull_request_template.md`): summary, changes, motivation,
  testing performed, and risks and rollback. For map changes, add a
  screenshot.
- If this is your first contribution, a maintainer approves the CI run
  before it starts.
- CI must pass and review conversations must be resolved. A maintainer
  reviews the pull request and merges it with **squash and merge**.

## 7. What is never committed

- Data of any kind: `data/`, caches, rasters (GRIB, NetCDF, GeoTIFF), tile
  archives, database dumps.
- Secrets and local configuration: `.env*`, keys, credentials. Keep
  personal editor and tool configuration out of the repository too; your
  global Git excludes file is the place for it.
- Research workspace material (`research/`).

## 8. Data sources

Every new data source must be added to `docs/data-sources.md` with its
owner, access method, update frequency, licence and attribution text.
Respect each source's terms of use and rate limits; collectors must
identify themselves and cache responses rather than fetch per user
request.

Test fixtures in `testdata/` should be as small as possible, and edited
or made up where the source's terms do not allow redistribution.

Official warnings from the Department of Meteorology and the Disaster
Management Centre are authoritative. Anything the project derives, such
as a risk score, is labelled as an estimate.

## 9. Translations

The interface is in English, Sinhala and Tamil. A change to user-facing
text needs all three, in `apps/web/src/locales/{en,si,ta}`, and must
follow the terms in [docs/i18n-glossary.md](docs/i18n-glossary.md). If you
cannot translate a string, say so in the pull request and a maintainer
will find help.

## 10. Licence

Lanka Weather is licensed under the [Apache License 2.0](LICENSE).
Contributions are accepted under the same licence (section 5). By opening a
pull request you confirm that you wrote the change, or otherwise have the
right to submit it under that licence.
