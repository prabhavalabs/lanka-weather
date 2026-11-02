# Rivers, overflow risk and flood basins

The read API for the river network, its estimated risk of overflowing, flood-forecast basins
and past floods. See ADR 0010 for how these are made.

Everything under "risk" is **an estimate by lanka-weather**, and says so with `estimate: true`.
Official warnings come from the Irrigation Department, the Department of Meteorology and the
Disaster Management Centre.

JSON uses the envelope `{"data": …}` or `{"error": {"code", "message"}}`. Tiles are Mapbox Vector
Tiles. Coordinates are `[longitude, latitude]`; boxes are `[west, south, east, north]`. A response
with `meta.seeded: true` holds local test data, and is never real.

## Risk levels

| Level | Means |
|---|---|
| `severe` | At or above major flood level, or forecast above extreme danger, or about to reach either |
| `high` | At or above minor flood level, or forecast above danger, or heavy rain over the catchment |
| `watch` | At or nearing alert level, or forecast above warning, or fairly heavy rain over the catchment |
| `normal` | A current gauge or forecast reads below these |
| `unknown` | Nothing current says either way. **This is not safe.** |

## `GET /v1/rivers/risk`

Every river and stretch, worst first, recomputed at most once a minute.
`Cache-Control: public, max-age=60`. Answers 503 `no_data` until the network has been collected.

```json
{
  "data": {
    "estimate": true,
    "note": "Estimate from gauge readings, Google flood forecasts and forecast rain. Thresholds are the Irrigation Department's. Follow official warnings from the Irrigation Department, the Department of Meteorology and the Disaster Management Centre.",
    "rivers": [
      {
        "id": "kelani-ganga",
        "name": "Kelani Ganga",
        "nameSi": "කැලණි ගඟ",
        "aliases": [],
        "named": true,
        "parentId": null,
        "level": "watch",
        "reasons": [
          {"level": "watch", "kind": "gauge", "text": "Norwood is at 1.25 m, below its alert level of 1.50 m. It is within 0.25 m of that level."},
          {"level": "unknown", "kind": "rain", "text": "Forecast rain over the catchment is 16 mm in the next 24 hours and 46 mm in 72 hours on average."}
        ],
        "uplandKm2": 2325,
        "lengthKm": 132,
        "bbox": [79.86875, 6.82, 80.78, 7.1],
        "gaugedReaches": 5,
        "observedAt": "2026-11-02T19:42:33Z",
        "reaches": [
          {
            "fid": 41405521,
            "riverId": "kelani-ganga",
            "name": "Norwood to Kithulgala",
            "level": "watch",
            "reasons": [],
            "gauged": true,
            "lengthKm": 35,
            "gauges": [
              {"kind": "station", "id": "norwood", "name": "Norwood", "role": "upstream", "waterLevel": 1.25, "unit": "m", "status": "normal", "trend": "steady", "observedAt": "2026-11-02T19:30:00Z", "stale": false},
              {"kind": "station", "id": "kithulgala", "name": "Kithulgala", "role": "downstream", "waterLevel": 1.84, "unit": "m", "status": "normal", "trend": "steady", "observedAt": "2026-11-02T19:30:00Z", "stale": false}
            ],
            "rain": {"next24hMm": 16.5, "next72hMm": 45.5, "catchmentKm2": 409, "stale": false}
          }
        ]
      }
    ],
    "meta": {
      "generatedAt": "2026-11-02T20:30:00Z",
      "networkVersion": "4f1c…",
      "networkBuiltAt": "2026-11-02T20:00:00Z",
      "gaugesObservedAt": "2026-11-02T19:42:33Z",
      "gaugesCurrent": 37,
      "gaugesTotal": 40,
      "forecastIssuedAt": "2026-11-02T18:28:44Z",
      "forecastFetchedAt": "2026-11-02T19:50:00Z",
      "rainFetchedAt": "2026-11-02T19:50:48Z",
      "rainStale": false,
      "attribution": "Irrigation Department of Sri Lanka; HydroRIVERS (Lehner & Grill 2013, HydroSHEDS), extracted by nuuuwan/lk_rivers; …",
      "seeded": false
    }
  }
}
```

- `level` and `reasons` on a river are those of its worst stretch.
- A reason's `kind` is `gauge`, `forecast`, `rain`, `reservoir` or `coverage`. A `reservoir` reason
  names a reservoir directly upstream, and a level of `unknown` on one means it did not raise the
  stretch (ADR 0010).
- `meta.reservoirLinks` has `byPosition`, `byRiver` and `unlinked`: those not placed on the network,
  each with `id`, `name`, `reason` (`no_place`, `too_far`, `conflict`, `no_stretch` or `ambiguous`)
  and `detail`. They count for nothing. `meta.reservoirsFetchedAt` is null when no readings could be
  used, so spills upstream are not known.
- A gauge's `role` is `upstream` (its water reaches this stretch), `downstream` (the gauge at the
  stretch's lower end) or `next` (the first gauge below an ungauged stretch: named, not used as
  evidence).
- `reason.kind` is `gauge`, `forecast`, `rain` or `coverage`.
- A stretch's `fid` is the feature id of its lines in the river tiles.

## `GET /v1/rivers/{id}`

One river in full: the fields above, and

| Field | Holds |
|---|---|
| `parent` | The river it flows into, with its level |
| `tributaries` | Named tributaries, with their levels |
| `gauges` | Each station on it, with thresholds and 72 hours of `readings` |
| `forecasts` | Google's forecast at each verified gauge on it, with `severity`, `trend`, `issuedAt` and the forecast window |
| `rain` | Forecast rain over the catchment of its lowest stretch |
| `affected` | For each stretch at risk, the `districts` and `dsDivisions` it passes through |

Errors: 404 `river_not_found`.

## `GET /v1/floods/basins`

Every official river basin, and each sub-basin with something to say.
`Cache-Control: public, max-age=60`. Answers 503 `no_data` until the basins have been collected.

```json
{
  "data": {
    "basins": [
      {
        "fid": 660,
        "id": "mahaweli-ganga",
        "name": "Mahaweli Ganga",
        "named": true,
        "state": "no_flooding",
        "areaKm2": 10268.4,
        "riverId": "mahaweli-ganga",
        "bbox": [80.46, 6.77, 81.41, 8.5],
        "labelPoint": [80.98, 7.6],
        "gauges": [
          {"kind": "station", "id": "peradeniya", "name": "Peradeniya", "state": "no_flooding", "status": "normal", "trend": "steady", "waterLevel": 2.41, "unit": "m", "observedAt": "2026-11-02T19:30:00Z", "stale": false, "verified": true}
        ],
        "areas": [],
        "onlyUnverified": false,
        "observedAt": "2026-11-02T19:30:00Z"
      }
    ],
    "subbasins": [{"fid": 4121156400, "state": "no_flooding", "onlyUnverified": false}],
    "meta": {
      "generatedAt": "2026-11-02T20:30:00Z",
      "attribution": "Irrigation Department of Sri Lanka; HydroBASINS (Lehner & Grill 2013, HydroSHEDS); Flood forecasts: Google Flood Hub (Flood Forecasting API)",
      "boundarySource": "irrigation-river-basins",
      "subbasinSource": "hydrosheds-hydrobasins",
      "boundaryTime": "2022-09-06T10:09:27Z",
      "gaugesObservedAt": "2026-11-02T19:42:33Z",
      "forecastIssuedAt": "2026-11-02T18:28:44Z",
      "forecastFetchedAt": "2026-11-02T19:50:00Z",
      "includesUnverified": false,
      "seeded": false
    }
  }
}
```

`state`, worst first: `extreme`, `severe`, `above_normal`, `no_flooding`, `no_forecast`.
`no_forecast` means nothing current covers the basin, which is not the same as calm.

A gauge's `state` is empty when it says nothing: it is out of date, or has no thresholds.
`verified` is false for a Google gauge it has not quality-verified, read only with
`LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED=true`. `onlyUnverified` says a basin's state rests on those
alone. A sub-basin not listed has no forecast.

## `GET /v1/floods/basins/{id}`

One basin, by its `id` or `fid`, with `meta`. Errors: 404 `basin_not_found`.

## `GET /v1/floods/past`

Mapped past floods, newest first. **These are past events, not a forecast.**
`Cache-Control: public, max-age=3600`.

```json
{
  "data": {
    "note": "Past flood extents mapped by the Irrigation Department. These show where water reached in past events, not a current forecast.",
    "events": [
      {
        "fid": 2816447019,
        "id": "kelani-ganga-flood-2016-major",
        "name": "Kelani Ganga flood, 2016 (major)",
        "sourceName": "Kelani Ganag Flood 2016 Major",
        "river": "Kelani Ganga",
        "year": 2016,
        "date": null,
        "magnitude": "major",
        "areaKm2": 140.2,
        "districts": ["Colombo", "Gampaha", "Kegalle", "Ratnapura"],
        "bbox": [79.87, 6.84, 80.45, 7.1],
        "service": "Flood_Map"
      }
    ],
    "meta": {
      "source": "irrigation-past-floods",
      "attribution": "Irrigation Department of Sri Lanka",
      "sourceTime": "2025-12-01T03:59:39Z",
      "fetchedAt": "2026-11-02T20:30:00Z"
    }
  }
}
```

`date` is set only when the source names a day. `sourceName` is the event as the source writes it.

## Tiles

All have extent 4096 and `0 <= z <= 14`, answer 204 when empty and 400 `invalid_tile` for
coordinates outside the pyramid.

### `GET /v1/tiles/rivers/{z}/{x}/{y}.pbf?v={networkVersion}`

| Layer | Feature id | Properties | Zooms |
|---|---|---|---|
| `reaches` | Stretch `fid` | `reach`, `river`, `name`, `named`, `width` (1 to 5, by upstream area) | Named rivers from 500 km²: all. Smaller ones from 7 and 8; unnamed from 8 and 9 |
| `streams` | `HYRIV_ID` | `river`, `width` | From 7, by upstream area |
| `labels` | none | `river`, `name`, `main`, `upland` | Named rivers |

Pass `networkVersion` from `/v1/rivers/risk` as `v`. A stretch's lines can be several features with
one id: one per width class. Cached for a week, since a rebuild changes the URL.

### `GET /v1/tiles/river-basins/{z}/{x}/{y}.pbf`

| Layer | Feature id | Properties | Zooms |
|---|---|---|---|
| `basins` | Basin `fid` | `id`, `name`, `named`, `area` | All; basins under 25, 10 and 3 km² join at zooms 7, 8 and 9 |
| `subbasins` | HydroBASINS id | `id`, `basin` (the official basin's `fid`) | From 9 |

### `GET /v1/tiles/past-floods/{z}/{x}/{y}.pbf`

| Layer | Feature id | Properties |
|---|---|---|
| `events` | Event `fid` | `id`, `name`, `year` |

## `GET /v1/summary`

Adds `nearbyRivers`: up to three rivers within about 17 km, nearest first, with the estimate for the
nearest stretch of each.

```json
{"id": "kelani-ganga", "name": "Kelani Ganga", "named": true, "level": "normal", "reachName": "Hanwella to Nagalagam Street", "reason": "Nagalagam Street is at 1.6 ft, below its alert level of 4.0 ft.", "distanceKm": 0.8, "estimate": true}
```

`unavailable` lists `nearbyRivers` if the estimate could not be read.

## Other changes

- `/v1/floods/forecast`: each flood area's properties add `id`, and `meta` adds `seeded`.
- `/v1/layers`: adds `rivers` and `past-floods`; `flood-forecast` adds `tilePath`.

## Credit

Show with this data: "Irrigation Department of Sri Lanka", "HydroRIVERS and HydroBASINS (Lehner & Grill
2013, HydroSHEDS)", "nuuuwan/lk_rivers", "Flood forecasts: Google Flood Hub" and "Weather data by
Open-Meteo.com (CC BY 4.0)", as applicable.
