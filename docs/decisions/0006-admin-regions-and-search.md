# ADR 0006: Administrative regions, vector tiles and place search

- Status: Accepted
- Date: 2026-11-02

## Context

People need to see Sri Lanka's administrative boundaries, find the
region they live in, and search for their city, town or village. Warnings
are issued per district and per divisional secretariat (DS) division,
and region-level weather (for example the mean temperature of a
district) will be computed by joining the weather grid to region
polygons, so the boundaries must also support fast spatial joins.

Sri Lanka has four administrative levels below the country:

| Level | Count | Code |
|---|---:|---|
| Province | 9 | `LK-1` |
| District | 25 | `LK-11` |
| Divisional secretariat (DS) division | 340 | `LK-1103` |
| Grama Niladhari (GN) division | 14,019 | `LK-1103005` |

Codes are those of the Department of Census and Statistics (DCS); a
region's code starts with its parent's code.

## Decision

### Sources

- **Boundaries:** `nuuuwan/lk_admin_regions`, the `e4_medium` TopoJSON
  files (`data/geo/topojson/e4_medium/{provinces,districts,dsds,gnds}.topojson`,
  about 43 MB together, positions quantised to a few metres and simplified
  to about 10 m). Their properties carry the DCS code, the English name,
  Sinhala and Tamil names, area and centre. The geometry comes from the
  Survey Department through OCHA's HDX (CC BY-IGO); the repository's
  code is MIT. Coarser files distort GN divisions in towns (a Colombo GN
  division is about 400 m across); finer ones exceed GitHub's 25 MB limit
  for the GN level.
- **Population:** `nuuuwan/lk_census_2024`, `data/Person-Sex/data.tsv`
  (0.6 MB): the 2024 Census of Population and Housing per province,
  district, DS and GN division, with the same codes.
- **Places:** OpenStreetMap `place=*` nodes (city, town, suburb, village,
  quarter, neighbourhood, hamlet, locality, isolated_dwelling) with a
  name, through the Overpass API: about 7,000 places, 1.6 MB, one query
  of about 45 seconds a week. OpenStreetMap is chosen over GeoNames
  because its place values give the importance ranking directly
  (city > town > village > hamlet), it carries `name:si` and `name:ta`,
  and the basemap already credits OpenStreetMap. GeoNames' LK dump
  (CC BY 4.0, about 17,800 populated places) has more villages but marks
  nearly all of them only as "populated place" with no population, so it
  cannot rank a town above a village; GN division names (14,019, mostly
  village names, with Sinhala names) fill much of that gap. GeoNames
  remains an option if village coverage proves short.
- Nothing is vendored. Each collector fetches its source on a schedule
  and keeps the result in PostgreSQL; no request is proxied to a source.

### Collection

- `nuuuwan-lk-admin-regions` runs weekly (retried after an hour on
  failure). It asks the GitHub API for the latest commit touching the
  boundary directory and the census directory (two small requests), and
  stops when both commits are already loaded. Otherwise it fetches the
  files at those commits (`raw.githubusercontent.com/<repo>/<sha>/<path>`),
  so one load never mixes versions.
- A load is rejected unless it has at least 9 provinces, 25 districts,
  300 DS and 13,000 GN divisions; regions whose parent is missing are
  dropped (more than 1% fails the load). The load replaces the regions in
  one transaction: rows are staged with `COPY`, made valid, merged by
  code, and regions no longer published are deleted.
- `osm-places` runs weekly (retried after three hours, as public Overpass
  instances are often busy), trying `overpass-api.de` then
  `overpass.kumi.systems`. It queries a bounding box rather than the
  country relation, which is far cheaper for the server; the box
  (79.5–81.95°E, 5.85–9.9°N) excludes the Indian coast. A response with
  fewer than 3,000 places or an Overpass `remark` (time-out) is rejected.
- Every region row records `source`, `source_version` (commit SHA),
  `source_time` (commit time) and `fetched_at`; places record
  `source_time` (Overpass `timestamp_osm_base`) and `fetched_at`.
  `source_versions` keeps the version last loaded per source.
- Each place is assigned the GN division containing it, or the nearest
  within about 3 km (places on the simplified coastline), which gives
  its hierarchy. Loading either source refreshes the assignment.

### Identifiers

- The region id is the DCS code with one hyphen: `LK-1`, `LK-11`,
  `LK-1103`, `LK-1103005`. The API also accepts `LK11`,
  `lk-11` and `LK-1-1-03` and normalises them.
- `province_id`, `district_id` and `ds_id` hold the containing region at
  each level, or the region itself at its own level, so
  `GROUP BY district_id` over GN rows aggregates to districts.
- `iso_code` holds the ISO 3166-2 code of provinces and districts. It
  equals the DCS code except in the Northern Province, where ISO numbers
  the districts in a different order (DCS `LK-42` Mannar is ISO `LK-43`,
  `LK-43` Vavuniya is `LK-44`, `LK-44` Mullaitivu is `LK-45`, `LK-45`
  Kilinochchi is `LK-42`).
- `fid` is the code's digits as a number (`1103005`). Levels have
  different digit counts, so it is unique across levels; vector tiles
  use it as the feature id (MapLibre feature state needs numbers).

### Storage and spatial joins

- `regions.geom` is `geometry(MultiPolygon, 4326)` with a GiST index,
  valid (`ST_MakeValid`) and as published. Joins with the weather grid
  should use it:

  ```sql
  -- Mean temperature per district from grid points (points as geometry).
  SELECT r.id, r.name, avg(g.temperature_c)
  FROM regions r
  JOIN weather_grid g ON ST_Intersects(r.geom, g.geom)
  WHERE r.level = 'district'
  GROUP BY r.id, r.name;
  ```

  A GN or small DS division may contain no grid point of a coarse grid;
  use the cell containing `label_point`, or area weights from
  `ST_Intersection(r.geom, cell)`, for those.
- `geom_simple` is the boundary simplified for its level (about 220 m for
  provinces, 110 m districts, 55 m DS and 20 m GN divisions). It is
  simplified in the collector along the TopoJSON's shared arcs, so
  neighbouring regions still meet without gaps or overlaps; PostGIS 3.5
  with GEOS 3.11 has no coverage simplification, and
  `ST_SimplifyPreserveTopology` simplifies each polygon on its own.
- `geom_3857` and `geom_simple_3857` hold the same boundaries in Web
  Mercator for tiles, so tiles never reproject. `label_point` is the
  centre of the largest inscribed circle (a point on the surface for GN
  divisions), always inside the region; `centroid` is the source's
  area-weighted centre, which can fall outside a concave region.

### Serving: tiles and GeoJSON

- **Vector tiles serve every level.** `GET /v1/tiles/regions/{z}/{x}/{y}.pbf`
  returns a Mapbox Vector Tile rendered by PostGIS (`ST_AsMVT`) with one
  layer per level: `province` from zoom 0, `district` from 6, `ds` from 8
  and `gn` from 10, to zoom 14 (clients overzoom beyond it). Below zoom
  9 the simplified boundaries are used. The densest zoom-10 tile, near
  Colombo, is about 270 KB (150 KB gzip); the zoom-12 tile over central
  Colombo is 17 KB (9 KB gzip).
- **GeoJSON serves small sets.** `GET /v1/regions?level=` returns the
  simplified boundaries of all provinces (150 KB, 46 KB gzip), districts
  (260 KB, 79 KB gzip) or DS divisions (1.4 MB, 355 KB gzip), and the GN
  divisions of one district or DS division (`parent` required). All
  14,000 GN divisions as GeoJSON would be about 15 MB even simplified,
  so the whole GN level is available only as tiles.
- Both are cached: rendered bodies are kept in memory (64 MB, least
  recently used) keyed by data version; responses carry
  `Cache-Control: public, max-age=86400, stale-while-revalidate=604800`
  and a weak `ETag` derived from the boundary and census commits and the
  load time, and `If-None-Match` is answered with 304. Tiles with no
  boundary are 204. JSON, GeoJSON and tiles are gzip-compressed when the
  client accepts it.

### Lookup

`GET /v1/regions/lookup` returns the region containing a point at every
level with one indexed `ST_Intersects` query. A level is `null` when the
point is outside every region of it (at sea, or just off the simplified
coast).

### Search

- `search_names` holds every name of every region and place, one row per
  name and language (English, Sinhala, Tamil, OpenStreetMap alternative
  names, and "<name> District", "<name> Province" and "<name> Divisional
  Secretariat"), folded by `search_fold()` (lower case, accents and
  zero-width joiners removed, punctuation as spaces; Sinhala and Tamil
  letters unchanged). It has a `pg_trgm` GIN index and a
  `text_pattern_ops` index for prefixes.
- A name matches when it starts with the query or, from three letters,
  when the query's trigram word similarity to it is at least 0.5 (one or
  two typos in a short word, set per transaction).
- Match quality: 1 for an exact name; 0.8 to 0.9 for a prefix and 0.7 to
  0.8 for the prefix of a later word, higher the more of the name the
  query covers; otherwise 0.8 × word similarity.
- Importance: city 1.0, province 0.9, district 0.85, town 0.75, DS
  division 0.6, suburb 0.55, village 0.45, quarter 0.4, neighbourhood
  0.35, GN division 0.35, hamlet 0.25, locality 0.2, isolated dwelling
  0.1, plus up to 0.1 for population on a log scale.
- Score = 0.6 × match + 0.4 × importance; each place or region appears
  once, with its best-matching name. "Kandy" returns the city, then the
  district, then a hamlet named kandy; "kandi" still returns the city
  first; "මහනුවර" and "கண்டி" find Kandy.
- On the full data a search takes 4–10 ms end to end (p95 under 10 ms).

## API contract

All JSON responses use the envelope `{"data": …}` or
`{"error": {"code", "message"}}`, except GeoJSON collections, which are
returned bare so map libraries can load them. Coordinates are
`[longitude, latitude]`; bounding boxes are `[west, south, east, north]`.
Region `level` is one of `province`, `district`, `ds`, `gn`.

### `GET /v1/regions?level={level}&parent={id}`

GeoJSON `FeatureCollection` of one level with simplified `MultiPolygon`
boundaries, ordered by id. `parent` (optional; required for `gn`, and
then a district or DS division) restricts the result to regions within
that ancestor. Errors: 400 `invalid_level`, `invalid_parent`,
`parent_required`. Content type `application/geo+json`; `ETag`, 304.

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "LK-11",
      "geometry": {"type": "MultiPolygon", "coordinates": [[[[79.84, 6.97], …]]]},
      "properties": {
        "id": "LK-11", "level": "district", "name": "Colombo",
        "nameSi": "කොළඹ දිස්ත්‍රික්ක", "nameTa": "கொழும்பு மாவட்டம்",
        "parentId": "LK-1", "provinceId": "LK-1", "districtId": "LK-11", "dsId": null,
        "isoCode": "LK-11", "gnNumber": null, "areaKm2": 688.17,
        "population": 2375415, "populationYear": 2024, "labelPoint": [79.954179, 6.851105]
      }
    }
  ],
  "meta": {
    "source": "nuuuwan-lk-admin-regions", "populationSource": "nuuuwan-lk-census-2024",
    "attribution": "Survey Department of Sri Lanka and Department of Census and Statistics, via OCHA HDX (CC BY-IGO) and nuuuwan/lk_admin_regions; population: Census of Population and Housing 2024",
    "version": "1-13e6784dce0e-445e4ad9f740-tnvogt",
    "sourceVersion": "13e6784dce0ec813c3fd5db3886a730af82cf887",
    "sourceTime": "2026-06-23T09:08:39Z", "fetchedAt": "2026-11-02T14:05:17.645485Z",
    "simplifiedDegrees": 0.001
  }
}
```

### `GET /v1/tiles/regions/{z}/{x}/{y}.pbf`

Mapbox Vector Tile (`application/vnd.mapbox-vector-tile`), extent 4096,
`0 <= z <= 14`. Layers `province` (z ≥ 0), `district` (z ≥ 6), `ds`
(z ≥ 8) and `gn` (z ≥ 10). Feature id: the numeric code (`1103005`).
Properties: `id`, `level`, `name`, `nameSi`, `nameTa`, `parentId`,
`provinceId`, `districtId`, `dsId` (absent when null). 204 when no
boundary crosses the tile; 400 `invalid_tile`. A MapLibre source:

```js
map.addSource("regions", {
  type: "vector",
  tiles: [location.origin + "/v1/tiles/regions/{z}/{x}/{y}.pbf"],
  minzoom: 0,
  maxzoom: 14,
  promoteId: "id", // or use the numeric feature id
})
map.addLayer({ id: "gn-lines", type: "line", source: "regions", "source-layer": "gn", minzoom: 10 })
```

The catalogue entry `admin-regions` (`GET /v1/layers/admin-regions`)
gives `tilePath`, `dataPath`, `minZoom` and `maxZoom`.

### `GET /v1/regions/{id}?geometry=true|false`

One region. `geometry=true` adds its simplified boundary. Errors: 400
`invalid_region_id`, `invalid_geometry`; 404 `region_not_found`.

```json
{
  "data": {
    "id": "LK-2130", "level": "ds", "name": "Gangawata Korale",
    "names": {"en": "Gangawata Korale", "si": "ගඟවටකෝරලේ", "ta": "கண்டி"},
    "isoCode": null, "gnNumber": null, "parentId": "LK-21",
    "hierarchy": [
      {"id": "LK-2", "level": "province", "name": "Central", "nameSi": "මධ්‍යම පළාත", "nameTa": "மத்திய மாகாணம்"},
      {"id": "LK-21", "level": "district", "name": "Kandy", "nameSi": "මහනුවර දිස්ත්‍රික", "nameTa": "கண்டி மாவட்டம்"}
    ],
    "childCount": 64,
    "centroid": [80.638555, 7.273186], "labelPoint": [80.637317, 7.286554],
    "bbox": [80.584327, 7.217799, 80.730507, 7.329909],
    "areaKm2": 59.61, "population": 153329, "populationYear": 2024,
    "source": {
      "name": "nuuuwan-lk-admin-regions", "version": "13e6784dce0ec813c3fd5db3886a730af82cf887",
      "sourceTime": "2026-06-23T09:08:39Z", "fetchedAt": "2026-11-02T14:05:17.645485Z",
      "attribution": "Survey Department of Sri Lanka and Department of Census and Statistics, via OCHA HDX (CC BY-IGO) and nuuuwan/lk_admin_regions; population: Census of Population and Housing 2024"
    }
  }
}
```

### `GET /v1/regions/lookup?lat={lat}&lon={lon}`

The containing region at each level; `null` where there is none. The
point must be within 5.5–10.5°N, 79–82.5°E. Errors: 400 `invalid_point`,
`out_of_range`.

```json
{
  "data": {
    "lat": 6.9271, "lon": 79.8612,
    "province": {"id": "LK-1", "level": "province", "name": "Western", "nameSi": "බස්නාහිර පළාත", "nameTa": "மேல் மாகாணம்"},
    "district": {"id": "LK-11", "level": "district", "name": "Colombo", "nameSi": "කොළඹ දිස්ත්‍රික්ක", "nameTa": "கொழும்பு மாவட்டம்"},
    "ds": {"id": "LK-1103", "level": "ds", "name": "Colombo", "nameSi": "කොළඹ", "nameTa": "கொழும்பு"},
    "gn": {"id": "LK-1103140", "level": "gn", "name": "Suduwella", "nameSi": "සුදුවැල්ල"}
  }
}
```

### `GET /v1/places/search?q={query}&limit={n}`

Places and regions matching `q` (1–100 characters, English, Sinhala or
Tamil), best first; `limit` 1–50, default 10. `type` is `place` or
`region`; `kind` is a place kind (`city`, `town`, `suburb`, `village`,
`quarter`, `neighbourhood`, `hamlet`, `locality`, `isolated_dwelling`)
or a region level. `bbox` is set for regions only. `hierarchy` lists the
containing regions from the province down (for a place, down to its GN
division). Errors: 400 `invalid_query`, `invalid_limit`.
`Cache-Control: public, max-age=3600`.

```json
{
  "data": {
    "query": "Kandy",
    "results": [
      {
        "id": "osm-node-2908772560", "type": "place", "kind": "city", "name": "Kandy",
        "names": {"en": "Kandy", "si": "මහනුවර", "ta": "கண்டி"}, "matchedName": "Kandy",
        "lat": 7.293121, "lon": 80.635036, "bbox": null, "population": 125000,
        "hierarchy": [
          {"id": "LK-2", "level": "province", "name": "Central"},
          {"id": "LK-21", "level": "district", "name": "Kandy"},
          {"id": "LK-2130", "level": "ds", "name": "Gangawata Korale"},
          {"id": "LK-2130120", "level": "gn", "name": "Mahanuwara"}
        ],
        "score": 1.029
      },
      {
        "id": "LK-21", "type": "region", "kind": "district", "name": "Kandy",
        "names": {"en": "Kandy", "si": "මහනුවර දිස්ත්‍රික", "ta": "கண்டி மாவட்டம்"}, "matchedName": "Kandy",
        "lat": 7.334296, "lon": 80.86275, "bbox": [80.421171, 6.939991, 81.01643, 7.490084],
        "population": 1461895,
        "hierarchy": [{"id": "LK-2", "level": "province", "name": "Central"}],
        "score": 0.975
      }
    ],
    "attribution": "Places © OpenStreetMap contributors (ODbL); regions: Survey Department of Sri Lanka and Department of Census and Statistics, via OCHA HDX (CC BY-IGO) and nuuuwan/lk_admin_regions; population: Census of Population and Housing 2024"
  }
}
```

## Consequences

- One weekly job keeps 14,393 regions current, at the cost of about
  43 MB from GitHub only when upstream changes; an unchanged week costs
  two GitHub API calls.
- The map can show every level at every zoom from one tile endpoint, and
  any future polygon layer (warnings per DS division) can join regions by
  code or by geometry.
- Search depends on OpenStreetMap's coverage of villages; GN division
  names compensate for most gaps. The ODbL requires attribution, which
  the search response carries, and that a derived database made public be
  offered under the ODbL; the collector and query are open source, which
  documents how the places table is derived.
- Upstream GN files lack Tamil names; Tamil search reaches provinces,
  districts, DS divisions and OpenStreetMap places only.
- The GitHub API allows 60 unauthenticated requests an hour per address;
  the collector uses two a week, but a shared address could be rate
  limited, in which case the run fails and is retried after an hour.
