# Rainfall, weather stations, air quality and sea conditions

All of these read the database; no request reaches a source. Map data is
returned bare; errors use the envelope `{"error":{"code","message"}}`. Times
are RFC 3339 UTC. `503 no_data` means nothing has been collected yet, and the
web client asks again each minute. See ADR 0012.

## `GET /v1/rain/steps`

The satellite images there are. Cached 2 minutes.

| Field                                      | Meaning                                                                                                              |
| ------------------------------------------ | -------------------------------------------------------------------------------------------------------------------- |
| `tilePath`                                 | `/v1/rain/tiles/{time}/{z}/{x}/{y}.png`                                                                              |
| `minZoom`, `maxZoom`, `tileSize`, `bounds` | The tile set: zoom 6, 256 px, west, south, east, north                                                               |
| `steps[]`                                  | Oldest first: `time` (for the URL), `validAt` and `endsAt` (the half hour shown), `fetchedAt`, `tiles`, `emptyTiles` |
| `newestValidAt`, `fetchedAt`               | The newest image                                                                                                     |
| `stale`                                    | The newest image is older than 10 hours                                                                              |
| `estimate`                                 | Always true: a satellite estimate, not a measurement                                                                 |
| `legend`                                   | `unit` (mm/h), `stops[]` of `rateMmH` and `color`, and the colour map they come from. Under 0.1 mm/h is transparent  |

## `GET /v1/rain/tiles/{time}/{z}/{x}/{y}.png`

One tile, from `steps[].time` such as `20261102T1330Z`. Cached a week,
immutable. A tile with no rain is a transparent PNG. `404 tile_not_found` for
another time or place.

## `GET /v1/rain/observed`

GeoJSON points with rain measured over 24 hours; `application/geo+json`.
Cached 5 minutes. Totals older than 7 days are left out.

Properties: `id`, `stationId`, `kind` (`met` or `gauge`), `name`, `rainMm`,
`windowEnd` (when the 24 hours close), `method` (`daily_report`, `sum_3h` or
`sum_gauge`), `source` and `stale`. Meta: `attribution`, `generatedAt`,
`observed`, and `fresh` and `stale` counts. Stale limits: 6 hours for summed
3-hourly readings, 36 for a daily report, 3 for a gauge.

## `GET /v1/stations`

Department of Meteorology stations as GeoJSON. Cached 5 minutes.

Properties: `id`, `name`, `wmoId`, `elevationM`, `positionApproximate`,
`reading` (`3h`, `daily` or `none`), `observedAt`, `temperatureC` (3-hourly
only), `humidityPct`, `rainMm` over `rainHours`, `rain24hMm`, `tempMinC`,
`tempMaxC`, `reportsTemperature`, `stale`.

Meta: `attribution`, `sourceUrl`, `generatedAt`, `observed: true`,
`threeHourly` and `daily` (each `source`, `newestObservedAt`, `fetchedAt`,
`stale`), and `stale`, true when neither feed is current.

A current 3-hourly reading is used. If it is out of date and the daily report
is newer, the daily report is used, so `temperatureC` is null and the
extremes carry the temperature. If both are old, the newer stays, marked stale.

## `GET /v1/stations/{id}?days=7`

In the envelope: `station` (a feature as above), `threeHourly[]`
(`observedAt`, `temperatureC`, `humidityPct`, `rainMm`), `daily[]` (`date`,
`observedAt`, `rainMm`, `tempMinC`, `tempMaxC`) and `meta`. History runs back
`days` (1 to 45) from each feed's newest reading, so a feed that stopped still
has a history; daily reports go 30 days further. `404 station_not_found`,
`400 invalid_days`.

## `GET /v1/air/current`

US AQI and PM2.5 from the CAMS model on a 0.4° grid. Same shape as
`/v1/weather/current`, so one rendering serves both: `source`, `model`,
`attribution`, `validAt`, `fetchedAt`, `grid`, `units`, `values` (`usAqi`,
`pm25`; one per lattice node, null where masked) and `note`. Cached 5 minutes.

## `GET /v1/sea/current`

Wave model values on a 0.5° grid over the sea. The same shape, with `values`
`waveHeightM`, `waveDirectionDeg`, `wavePeriodS`, `swellHeightM`,
`swellDirectionDeg` and `swellPeriodS` (null on land), and `land`: Sri Lanka's
land as closed longitude, latitude rings, simplified to about 1 km, for
clipping a field to the coast. Directions are where waves come from.

## Additions to `GET /v1/summary`

Each is null when it does not apply, and listed in `unavailable` only if it
failed.

- `observed`: the nearest station within 75 km that reports temperature: the
  station properties above, with `distanceKm`, `lon`, `lat` and `attribution`.
- `airQuality`: `usAqi`, `category` (`good`, `moderate`,
  `unhealthy_sensitive`, `unhealthy`, `very_unhealthy` or `hazardous`, the
  EPA's bands), `pm25`, `validAt`, `fetchedAt`, `stale`, `estimate`,
  `attribution`.
- `sea`, on the water or within about 16 km of it: `waveHeightM`,
  `wavePeriodS`, `waveDirectionDeg`, `swellHeightM`, `swellPeriodS`,
  `swellDirectionDeg`, `state` (the WMO word), `distanceKm` to the node read,
  `validAt`, `fetchedAt`, `stale`, `estimate`, `attribution`, `note`.
