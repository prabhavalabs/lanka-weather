# ADR 0008: Weather by region reads the regions table

- Status: Accepted
- Date: 2026-11-02

## Context

ADR 0006 loads administrative regions into the `regions` table (migration
00002): 9 provinces, 25 districts, 340 DS divisions and 14,019 GN
divisions, with GiST-indexed boundaries and a label point inside each
region.

ADR 0007 serves `GET /v1/weather/regions` by aggregating the weather grid
over regions read through a `RegionSource` interface. The two were built
on separate branches. Until they were merged, the region source was a
stub that answered `503 not_available`, and the PostGIS source had to be
switched on with `LANKAWEATHER_WEATHER_REGIONS=postgis`. It expected a
relation named `admin_regions` with the columns `id`, `level`, `name` and
`geom`.

Three questions remained once both were on one branch:

- how the weather read side should reach the regions table;
- whether the join of regions to grid cells is fast enough to run in the
  API, in particular for the 14,019 GN divisions;
- how a client asks for the GN divisions of one area.

Measured on the full boundaries (about 10 m precision) against the 0.2°
grid, the cell join takes about 14 ms for provinces, 11 ms for districts
and 9 ms for the 30 DS divisions larger than a cell. Reading the 14,019 GN
label points takes about 65 ms.

## Decision

- **A view is the contract.** Migration 00004 creates the view
  `admin_regions` over `regions`. It exposes `id`, `level`, `name`,
  `parent_id`, `province_id`, `district_id`, `ds_id`, `area_km2`,
  `label_point` and `geom`. The weather package reads only this view, so
  the regions table can change without touching the aggregation.
- **PostGIS is the default.** The API reads the view without any setting.
  - `LANKAWEATHER_WEATHER_REGIONS_RELATION` can name another relation with
    the same columns.
  - `LANKAWEATHER_WEATHER_REGIONS=none` switches region weather off. The
    stub then answers `503 not_available`.
  - A level with no regions yet also answers `503 not_available`, and a
    cell level before the grid's cells are stored answers `503 no_data`.
    Neither is cached, so data appears as soon as the collectors have run.
- **Weights are computed in the API and kept in memory.** No table of
  weights is stored:
  - Each level's regions and cell weights are loaded on first use, and in
    the background when the API starts.
  - They are kept for six hours, since boundaries change a few times a
    year.
  - The aggregate of each weather run is computed once per level.
  - A stored table would need refreshing whenever either the regions or
    the grid changed, from two collectors that run independently, for a
    join that costs milliseconds.
- **GN divisions use their point.** Each takes the value interpolated at
  its precomputed `label_point`, not the cells it intersects. A GN division
  is a few square kilometres, against about 490 km² for a cell.
- **`parent` filters by ancestry.** `/v1/weather/regions` accepts `parent`
  as `/v1/regions` does. The filter uses the region's stored province,
  district and DS ids, so it is exact.
  - Unlike `/v1/regions`, GN divisions are also served for the whole
    country. Region weather carries no geometry: the full list is 3.2 MB
    of JSON (270 KB gzipped) and is computed once per run.
- **The summary names the place.** `/v1/summary` includes the province,
  district, DS and GN division containing the point, from the same lookup
  as `/v1/regions/lookup`. The lookup is one GiST-indexed query.

## Consequences

- Weather by region works on a fresh database with no extra setting, once
  `migrate` and `collect` have run.
- A change to the regions table must keep the view's columns. The
  integration test `internal/weather/regions_integration_test.go` writes
  to the table and reads through the view.
- A boundary reload reaches region weather within six hours, or at the
  next API restart.
- Each API replica computes the weights itself, in about 0.3 s in the
  background at start-up. This needs no coordination between replicas.
