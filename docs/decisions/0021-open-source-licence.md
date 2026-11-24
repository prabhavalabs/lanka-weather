# ADR 0021: Apache-2.0 licence and reuse of TerraGuard code

- Status: Accepted
- Date: 2026-11-24

Supersedes the TerraGuard condition in ADR 0001's consequences, and
ADR 0001's statement that the HTTP contract lives in `api/openapi.yaml`.

## Context

The repository is public and is meant to accept pull requests from
anyone. Without a licence, nobody may reuse the code or contribute to
it.

Several parts were copied from TerraGuard, an earlier project by
Perfectus Technology Limited, and then modified: the GRIB2 decoder, the
GFS and Google Flood Forecasting connectors, and the wind renderer.
TerraGuard is not public. It is licensed under the Apache License 2.0,
and its copyright holder permits this reuse. ADR 0001 recorded that
permission as still to be confirmed.

ADR 0001 also said the HTTP contract would be an OpenAPI document that
the web client is generated from. The contract is documented in
`docs/api/` instead, and the web client's types are written by hand.

## Decision

- The repository is licensed under the Apache License 2.0
  (`LICENSE`). It is the licence of the reused TerraGuard code, so the
  repository carries one licence. It is permissive, it grants patent
  rights, and section 5 accepts contributions under the same terms
  without a separate agreement.
- `NOTICE` names the files derived from TerraGuard, with its copyright
  line, and the other bundled third-party material. Files copied from
  TerraGuard in future are added to it.
- The licence covers the source code only. Data collected and served
  by a running instance keeps each source's own terms, listed in
  `docs/data-sources.md`.
- The API contract is documented in `docs/api/`.

## Consequences

- Anyone may use, change and redistribute the code, provided they keep
  `LICENSE` and `NOTICE`.
- Contributions are accepted under Apache-2.0, as `CONTRIBUTING.md`
  states.
- The project name and `brand/` artwork are not licensed as trademarks.
