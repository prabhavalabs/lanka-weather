# Reservoirs and extra river gauges

Everything here is collected on a schedule from Sri Lanka Water Levels
(`waterlevels.lk`) and OpenStreetMap (ADR 0013). None of it is an official
warning. In an emergency call 117.

## GET /v1/reservoirs

Every reservoir with its latest reading, as bare GeoJSON
(`application/geo+json`), like `/v1/rivers/levels`. Reservoirs that could not be
placed on the map are included with `"geometry": null`.

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "minneriya",
      "geometry": { "type": "Point", "coordinates": [80.862, 8.024] },
      "properties": {
        "id": "minneriya",
        "name": "Minneriya",
        "district": "Polonnaruwa",
        "river": null,
        "type": "major",
        "status": "releasing",
        "storagePercent": 67,
        "storageFresh": true,
        "flow": { "kind": "downstream", "cusecs": 690, "agency": "Mahaweli Authority", "day": "2026-11-02" },
        "readingDay": "2026-11-02",
        "agency": "Irrigation Department",
        "source": "waterlevels-lk-reservoirs",
        "sourceUrl": "https://waterlevels.lk/reservoirs/minneriya",
        "fetchedAt": "2026-11-02T19:58:02Z",
        "stale": false,
        "mapped": true,
        "osmName": "Minneriya Reservoir",
        "osmKind": "water",
        "altNames": null
      }
    }
  ],
  "meta": {
    "source": "waterlevels-lk-reservoirs",
    "mapSource": "osm-reservoirs",
    "attribution": "Irrigation Department and Mahaweli Authority via Sri Lanka Water Levels (waterlevels.lk); positions © OpenStreetMap contributors (ODbL)",
    "note": "Not an official warning. In an emergency call 117.",
    "fetchedAt": "2026-11-02T19:58:03Z",
    "sourceTime": "2026-11-02T19:58:02Z",
    "mapTime": "2026-11-02T19:46:36Z",
    "generatedAt": "2026-11-02T20:14:34Z",
    "counts": { "total": 142, "mapped": 115, "spilling": 10, "releasing": 2, "holding": 76, "nodata": 54 }
  }
}
```

- `status` is the source's word: `spilling` (over the spillway), `releasing`
  (through gates or outlets), `holding` or `nodata`.
- `storagePercent` is the last figure the source has. `storageFresh` false means
  it is not current. Null is unknown, never zero. `nodata` can come with a fresh
  figure: storage was reported, spill status was not.
- `flow.stale` says the figure's own day is too old to use, which can differ from
  the day the status was read.
- `flow` is null when unknown. `kind` is `spill`, `outlets` or `downstream`. A
  Mahaweli Authority `downstream` flow can include power-station flow, so do not
  describe it as a flood release.
- `readingDay` is a day in Sri Lanka. `stale` is true 48 hours after the end of
  that day; a stale reservoir does not count as spilling or releasing.
- `osmKind` is `water` (a polygon is available) or `dam` (a marker at the dam).

## GET /v1/reservoirs/areas

The polygons of mapped reservoirs as GeoJSON, simplified to about 30 m (about
350 kB, 100 kB compressed), each with `id`, `name`, `status` and `stale`.

## GET /v1/reservoirs/{id}/readings?days=60

`days` is 1 to 400 and defaults to 60. Days the source only repeated an old figure
are left out, so a gap means nothing was read.

```json
{
  "data": {
    "reservoir": { "id": "minneriya", "name": "Minneriya", "status": "releasing" },
    "note": "Not an official warning. In an emergency call 117.",
    "readings": [
      { "day": "2026-11-02", "status": "releasing", "storagePercent": 67, "storageFresh": true,
        "flow": { "kind": "downstream", "cusecs": 690, "agency": "Mahaweli Authority", "day": null },
        "agency": "Irrigation Department" }
    ]
  }
}
```

Errors: 400 `invalid_days`, 404 `reservoir_not_found`.

## GET /v1/summary

Gains `reservoirs`: up to five mapped reservoirs within 50 km that are spilling or
releasing on a current reading, nearest first, each with `id`, `name`,
`distanceKm`, `status`, `storagePercent`, `flow`, `readingDay`, `lon` and `lat`.
`unavailable` includes `reservoirs` if they could not be read. `rivers[]` gains
`source` and `attribution`. These reservoirs raise no derived alert.

## GET /v1/rivers/levels

The layer now includes gauges the Irrigation Department feed lacks. Each station
gains:

- `source`: `irrigation-gauges` or `waterlevels-lk-gauges`;
- `attribution`: who to credit, naming RiverNet where the site does;
- `sourceUrl`: the gauge's page at the source, or null;
- `nextHour`: the experimental forecast, or null.

`unit` can now be `unknown`, with `status` `unknown`: show the level without a
flood verdict and without a unit. `meta.sources` lists each collector with its last
successful fetch; `meta.source` and `meta.fetchedAt` remain the department's feed.

```json
"nextHour": {
  "level": 1.8, "low": 1.76, "high": 1.85, "unit": "m",
  "direction": "rising", "crossing": null, "method": "model",
  "issuedAt": "2026-11-02T18:32:46Z", "experimental": true,
  "attribution": "Sri Lanka Water Levels (waterlevels.lk)"
}
```

`level`, `low` and `high` are in the station's unit; `low` and `high` are null for a
trend estimate. `experimental` is always true, whatever the source's own flag says.
It is null when older than two hours or when the station's unit is unknown, and it
never changes `status`.
