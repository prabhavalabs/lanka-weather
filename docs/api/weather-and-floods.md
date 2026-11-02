# API contract: weather, wind, flood forecasts and location summary

The web application reads these endpoints to draw the weather layers and fill
the location panel. ADR 0007 records the design decisions behind them.

Every response comes from data the worker has already collected. No
request reaches Open-Meteo, NOAA or Google.

## Conventions

Times, units and numbers:

- Times are RFC 3339 in UTC (`2026-11-02T14:45:00Z`).
- Local dates are Sri Lanka Standard Time (UTC+05:30), written `YYYY-MM-DD`.
- Units are in the field names: `temperatureC`, `precipitationMm`,
  `windSpeedKmh`, `relativeHumidityPct`, `windDirectionDeg` (the direction
  the wind blows *from*, clockwise from north).
- `weatherCode` is a WMO weather interpretation code (0 clear, 1–3
  cloud, 45/48 fog, 51–57 drizzle, 61–67 rain, 80–82 showers, 95–99
  thunderstorm).
- `isDay` is 1 or 0.
- Weather numbers are rounded to one decimal. A value the source did not
  give is `null`.

Response shapes:

- **Map data is bare.** `/v1/weather/current` and `/v1/weather/wind` return
  plain JSON, and `/v1/floods/forecast` returns plain GeoJSON. Other
  endpoints wrap their result in `{"data": …}`.
- **Errors are always wrapped:**
  `{"error": {"code": "…", "message": "…"}}`.
- **Gzip:** `/v1/weather/current` and `/v1/weather/wind` are gzipped when the
  request sends `Accept-Encoding: gzip`. They are compressed once per stored
  run, so the cost is paid once.

Errors shared by these endpoints:

| Status | Code | When |
|---|---|---|
| 400 | `invalid_coordinates` | `lat`/`lon` missing, not numbers, or out of range |
| 400 | `invalid_time` | `t` is not RFC 3339 or `now` |
| 400 | `invalid_level` | `level` is not `province`, `district`, `ds` or `gn` |
| 400 | `invalid_parent` | `parent` is not a region code above `level` (regions only) |
| 404 | `outside_grid` | The point is outside the weather grid (79.4–82.2° E, 5.6–10.0° N) |
| 404 | `no_wind_field` | No stored run covers `t` |
| 503 | `no_data` | No weather run has been collected yet |
| 503 | `not_available` | Administrative regions are not loaded yet (regions only) |
| 503 | `unavailable` | The service is not configured in this deployment |
| 500 | `internal` | Anything else; details are logged, not returned |

The layer catalogue (`GET /v1/layers`) gives each layer's `dataPath`,
`refreshSeconds` and `staleAfterSeconds`:

| Layer | Kind | Data path | Refresh | Stale after |
|---|---|---|---|---|
| `temperature` | `grid` | `/v1/weather/current` | 1 h | 3 h |
| `wind` | `wind` | `/v1/weather/wind` | 1 h | 12 h |
| `flood-forecast` | `geojson` | `/v1/floods/forecast` | 1 h | 30 h |

Data age is judged against `validAt` for the weather grid, `runAt` for wind
and `issuedAt` for flood gauges.

Attribution must be shown next to the data:

- **Open-Meteo:** "Weather data by Open-Meteo.com", linked to
  https://open-meteo.com/ (CC BY 4.0).
- **GFS:** "NOAA GFS".
- **Google:** "Flood forecasts: Google Flood Hub (Flood Forecasting API)".

## `GET /v1/weather/current`

Current conditions on the whole grid, for colouring the map.

The grid is a regular lattice: the node with column `i` and row `j` lies at
`lon0 + i × step`, `lat0 + j × step`, and its value is at index
`j × nx + i` of every array. Rows run **south to north**. Nodes outside the
land mask (open sea) are `null`; 214 of the 345 nodes have values.

UV index and precipitation probability are not observed. They come from the
forecast hour nearest `validAt`; `forecastFetchedAt` says when that forecast
was fetched.

Example: 21.6 KB, 4.2 KB gzipped. Arrays are cut here.

```json
{
  "source": "open-meteo",
  "model": "best_match",
  "attribution": "Weather data by Open-Meteo.com (CC BY 4.0)",
  "validAt": "2026-11-02T14:45:00Z",
  "fetchedAt": "2026-11-02T14:48:39.616272Z",
  "grid": { "id": "lk-0.2", "lon0": 79.4, "lat0": 5.6, "step": 0.2, "nx": 15, "ny": 23 },
  "units": { "temperatureC": "°C", "windSpeedKmh": "km/h", "weatherCode": "wmo", "…": "…" },
  "values": {
    "temperatureC": [null, null, null, null, null, null, null, null, null, null, null, null, null, null, null, null, null, null, null, 26.2, 25, 25.1, "…"],
    "apparentTemperatureC": ["…"],
    "relativeHumidityPct": ["…"],
    "dewPointC": ["…"],
    "precipitationMm": ["…"],
    "precipitationProbabilityPct": ["…"],
    "cloudCoverPct": ["…"],
    "pressureHpa": ["…"],
    "windSpeedKmh": ["…"],
    "windDirectionDeg": ["…"],
    "windGustsKmh": ["…"],
    "weatherCode": ["…"],
    "uvIndex": ["…"],
    "isDay": ["…"]
  },
  "forecastFetchedAt": "2026-11-02T14:49:20.173895Z"
}
```

`Cache-Control: public, max-age=300`. A new grid appears hourly.

## `GET /v1/weather/point?lat=&lon=`

Current conditions, a 72-hour hourly forecast from the current hour, and a
7-day daily forecast from today, interpolated at a point.

How values are read from the grid:

- Bilinear between the four surrounding nodes.
- `weatherCode` and `isDay` from the nearest node.
- Wind direction from interpolated wind components, so that 350° and 10°
  give 0°, not 180°.

Daily values are local days. Each is the day's extreme or total at each
node, then interpolated.

The values are those of a 0.2° model grid (about 22 km). They do not
resolve valleys, hill stations or sea breezes.

`current` and `meta.currentFetchedAt` are `null` without a current run;
`hourly`, `daily` and `meta.forecastFetchedAt` are `null` without a
forecast run. `currentStale` is true when `validAt` is over 3 hours old;
`forecastStale` when the forecast was fetched over 12 hours ago.

Example for Kandy (`lat=7.2906&lon=80.6337`): 8.2 KB. Arrays are cut to
two entries here.

```json
{
  "data": {
    "location": { "lat": 7.2906, "lon": 80.6337 },
    "current": {
      "validAt": "2026-11-02T14:45:00Z",
      "temperatureC": 21.9,
      "apparentTemperatureC": 26.5,
      "relativeHumidityPct": 99.1,
      "dewPointC": 21.8,
      "precipitationMm": 0.1,
      "precipitationProbabilityPct": 81.6,
      "cloudCoverPct": 97.1,
      "pressureHpa": 1016,
      "windSpeedKmh": 2.1,
      "windDirectionDeg": 61,
      "windGustsKmh": 7.2,
      "weatherCode": 3,
      "uvIndex": 0,
      "isDay": 0
    },
    "hourly": {
      "times": ["2026-11-02T14:00:00Z", "2026-11-02T15:00:00Z"],
      "temperatureC": [22.1, 21.9],
      "apparentTemperatureC": [26.8, 26.4],
      "relativeHumidityPct": [98.9, 98.7],
      "dewPointC": [21.9, 21.7],
      "precipitationMm": [0.2, 0.4],
      "precipitationProbabilityPct": [87.8, 81.6],
      "cloudCoverPct": [98.7, 97.1],
      "pressureHpa": [1015.1, 1016.2],
      "windSpeedKmh": [2, 2.3],
      "windDirectionDeg": [24, 74],
      "windGustsKmh": [7, 7.1],
      "weatherCode": [51, 3],
      "uvIndex": [0, 0],
      "isDay": [0, 0]
    },
    "daily": {
      "dates": ["2026-11-02", "2026-11-03"],
      "temperatureMaxC": [28.6, 27.7],
      "temperatureMinC": [21.2, 21.1],
      "apparentTemperatureMaxC": [34, 33.1],
      "precipitationSumMm": [8.5, 30.9],
      "precipitationProbabilityMaxPct": [100, 98.8],
      "windSpeedMaxKmh": [8.9, 9.6],
      "windGustsMaxKmh": [24.5, 27.5],
      "windDirectionDominantDeg": [194, 252],
      "uvIndexMax": [8, 8],
      "weatherCode": [80, 95]
    },
    "meta": {
      "source": "open-meteo",
      "model": "best_match",
      "attribution": "Weather data by Open-Meteo.com (CC BY 4.0)",
      "grid": { "id": "lk-0.2", "lon0": 79.4, "lat0": 5.6, "step": 0.2, "nx": 15, "ny": 23 },
      "interpolation": "bilinear between the four surrounding grid nodes; weather code and day/night from the nearest node; wind direction from interpolated wind components",
      "units": { "temperatureC": "°C", "…": "…" },
      "currentFetchedAt": "2026-11-02T14:48:39.616272Z",
      "forecastFetchedAt": "2026-11-02T14:49:20.173895Z",
      "currentStale": false,
      "forecastStale": false
    }
  }
}
```

The hourly arrays hold 72 entries, from the hour containing now. The daily
arrays hold 7 entries, from today. Both may be shorter when the stored
forecast is old.

## `GET /v1/weather/wind?t=`

The 10 m wind for the animated particle layer.

The field is valid at the 3-hour step nearest `t` (RFC 3339, or `now`, the
default). It comes from the newest GFS run that has that step. The format
is TerraGuard's `WindField`, so TerraGuard's `wind/field.ts`
(`decodeWind`, `sampleInto`) and `renderer.ts` read it unchanged.

- `u` (eastward) and `v` (northward) are base64 of little-endian int16s;
  multiply by `scale` to get m/s.
- The value `j × nx + i` lies at `lat0 + j × dlat`, `lon0 + i × dlon`. Rows
  run north to south (`dlat` is negative).
- The grid covers the map's panning limit: 70–92° E, 2° S–18° N, at 0.25°.
- Steps run from `firstValidAt` to `lastValidAt`, `stepHours` apart (0 to
  72 hours after `runAt`). Offer only those times.

Example: 38.9 KB, 16 KB gzipped. Base64 is cut here.

```json
{
  "source": "noaa-gfs",
  "model": "GFS 0.25°",
  "level": "10 m above ground",
  "runAt": "2026-11-02T06:00:00Z",
  "forecastHour": 24,
  "validAt": "2026-11-03T06:00:00Z",
  "firstValidAt": "2026-11-02T06:00:00Z",
  "lastValidAt": "2026-11-05T06:00:00Z",
  "stepHours": 3,
  "grid": { "nx": 89, "ny": 81, "lon0": 70, "lat0": 18, "dlon": 0.25, "dlat": -0.25 },
  "scale": 0.1,
  "u": "6//z//n/+f/3//b/7//k/9//…",
  "v": "6f/r/+v/7P/q/+b/4//s//7/…",
  "attribution": "NOAA GFS",
  "fetchedAt": "2026-11-02T14:16:15.599535Z"
}
```

`Cache-Control: public, max-age=600`. Do not persist wind fields in the
IndexedDB cache (ADR 0002).

## `GET /v1/weather/regions?level=province|district|ds|gn&parent=`

Current weather and today's range for every administrative region of a
level, for a choropleth. Region ids are the codes of `/v1/regions` and the
vector tiles (ADR 0006); a tile feature's numeric id is the code's digits
(`LK-1103005` is feature `1103005`).

`parent` (optional) limits the list to the regions within one region above
`level`, written as in `/v1/regions` (`LK-11`, `lk11`):
`level=gn&parent=LK-1103` returns the GN divisions of one DS division.
Unlike `/v1/regions`, GN divisions are also served for the whole country
(14,019 regions: 3.2 MB of JSON, 270 KB gzipped), because the values are
small and the aggregate is computed once per weather run.

How each region's values are computed:

- **Regions larger than one grid cell** (about 490 km²: provinces,
  districts, the 30 largest DS divisions) aggregate the cells they
  intersect:
  - means are weighted by the share of each cell inside the region;
  - the lowest and highest values are taken over cells at least 10 %
    inside it.
  - `method` is `"cells"` and `cells` counts them.
- **Smaller regions, and every GN division,** take the value interpolated
  at a point inside them (the region's `label_point`, computed when the
  regions are loaded). `method` is `"point"`.

The fields of each region:

| Field | Meaning |
|---|---|
| `temperatureC` | Current temperature (mean) |
| `temperatureMinC`, `temperatureMaxC` | Today's forecast low and high (local day) |
| `precipitationMm` | Current hour's precipitation (mean) |
| `precipitationTodayMm` | Today's forecast total (mean) |
| `windSpeedMaxKmh`, `windGustsMaxKmh` | Highest current wind and gust |

Until the regions collector has loaded a level, this answers
`503 not_available`; until the first weather run is stored, `503 no_data`:

```json
{ "error": { "code": "not_available", "message": "administrative regions are not loaded yet" } }
```

Example (`level=district`, cut to one region):

```json
{
  "data": {
    "level": "district",
    "regions": [
      {
        "id": "LK-11", "name": "Colombo", "method": "cells", "cells": 6,
        "temperatureC": 26.1, "temperatureMinC": 24.3, "temperatureMaxC": 30.2,
        "precipitationMm": 0.1, "precipitationTodayMm": 5.4,
        "windSpeedMaxKmh": 11.2, "windGustsMaxKmh": 22.5
      }
    ],
    "validAt": "2026-11-02T14:45:00Z",
    "fetchedAt": "2026-11-02T14:48:39Z",
    "forecastFetchedAt": "2026-11-02T14:49:20Z",
    "source": "open-meteo",
    "attribution": "Weather data by Open-Meteo.com (CC BY 4.0)"
  }
}
```

With `parent`, `data.parent` repeats the parent's code.

### Regions: how the join works

ADR 0008 records this design. The endpoint reads regions through
`weather.RegionSource`:

```go
type Region struct {
    ID, Name                     string
    ProvinceID, DistrictID, DSID string    // ancestors, or the region itself at its level
    Lon, Lat                     float64   // a point inside the region
    Nodes                        []int     // lattice indexes of grid nodes whose cells intersect it
    Weights                      []float64 // share of each cell inside the region, 0 to 1
}

type RegionSource interface {
    Regions(ctx context.Context, level string) ([]Region, error)
}
```

`weather.PostGISRegions`, the default, reads the `admin_regions` view
(migration 00004) over the `regions` table (migration 00002):

| Column | Type | Notes |
|---|---|---|
| `id` | `text` | DCS code, unique per region; returned as `id` |
| `level` | `text` | `province`, `district`, `ds` or `gn` |
| `name` | `text` | English name |
| `province_id`, `district_id`, `ds_id` | `text` | Ancestors, or the region itself at its own level; `NULL` below it |
| `area_km2` | `double precision` | Decides between cells and a point |
| `label_point` | `geometry(Point, 4326)` | A point inside the region |
| `geom` | `geometry(MultiPolygon, 4326)` | Boundary as published, GiST-indexed |

How it runs:

- Province, district and DS levels join `weather_grid_nodes` (migration
  00003) on `grid = 'lk-0.2' AND ST_Intersects(cell, geom)`, and weigh
  each cell by `ST_Area(ST_Intersection(cell, geom)) / ST_Area(cell)`.
  On the full boundaries this takes about 15 ms per level.
- GN divisions skip the join: their 14,019 points are read in about 65 ms,
  and the four grid nodes around each point are found in memory.
- Each level's regions and weights are kept in memory for six hours, and
  loaded in the background when the API starts. Each weather run's
  aggregate is computed once per level and kept until the next run; a
  `parent` only filters it.
- Before the regions are loaded, or before the grid cells are stored, a
  level is not cached, so it is picked up as soon as the collectors have
  run.

Configuration:

- `LANKAWEATHER_WEATHER_REGIONS_RELATION` names another relation with the
  same columns.
- `LANKAWEATHER_WEATHER_REGIONS=none` switches region weather off; the
  endpoint then answers `503 not_available`.

## `GET /v1/floods/forecast`

Google's flood forecasts for Sri Lanka as GeoJSON.

Each gauge is a `Point` feature with `properties.kind = "gauge"`:

| Property | Meaning |
|---|---|
| `severity` | `no_flooding`, `above_normal`, `severe`, `extreme` or `unknown` |
| `trend` | `rise`, `fall`, `no_change` or `unknown` |
| `forecastStart`, `forecastEnd` | The forecast window |
| `thresholds` | The gauge model's warning, danger and extreme-danger levels, in `unit` (`m3/s` or `m`) |
| `qualityVerified` | Whether Google has verified the gauge |
| `stale` | True when issued more than 30 hours ago |

Flood areas are `MultiPolygon` features. `properties.kind` is one of:

- `notification`;
- `inundation_high`, `inundation_medium` or `inundation_low` (gauges above
  normal);
- `flash_likely` or `flash_highly_likely`;
- `significant_event`.

Each area has `severity`, `validFrom`, `validUntil`, the `gaugeId` it belongs
to, and source `details` (for example affected population).

Nothing fetched more than 24 hours ago is served, as Google's terms
require. `meta.configured` is false when no API key is set.

Example: 6.1 KB. On 2 November 2026 Sri Lanka had 10 verified gauges, all
at `no_flooding`, and no areas; one gauge is shown here.

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "hybas_4121155030",
      "geometry": { "type": "Point", "coordinates": [80.7687499999958, 8.931249999999011] },
      "properties": {
        "kind": "gauge",
        "gaugeId": "hybas_4121155030",
        "gaugeSource": "HYBAS",
        "siteName": "",
        "river": "",
        "qualityVerified": true,
        "severity": "no_flooding",
        "trend": "no_change",
        "issuedAt": "2026-11-02T13:23:42.01453Z",
        "forecastStart": "2026-11-07T00:00:00Z",
        "forecastEnd": "2026-11-08T00:00:00Z",
        "thresholds": { "warning": 59.47, "danger": 112.88, "extremeDanger": 211.15, "unit": "m3/s" },
        "fetchedAt": "2026-11-02T14:50:47Z",
        "stale": false
      }
    }
  ],
  "meta": {
    "source": "google-flood-forecasting",
    "attribution": "Flood forecasts: Google Flood Hub (Flood Forecasting API)",
    "region": "LK",
    "configured": true,
    "fetchedAt": "2026-11-02T14:50:47.299015Z",
    "generatedAt": "2026-11-02T14:51:01.897245Z",
    "gauges": 10,
    "areas": 0
  }
}
```

A flood area feature, for illustration:

```json
{
  "type": "Feature",
  "id": "GiQKIDQ5NWRi…EAQ",
  "geometry": { "type": "MultiPolygon", "coordinates": [[[[81.0, 8.1], [81.4, 8.1], [81.4, 8.4], [81.0, 8.4], [81.0, 8.1]]]] },
  "properties": {
    "kind": "notification", "gaugeId": "hybas_4121156400", "severity": "severe",
    "issuedAt": "2026-11-02T12:22:59Z", "validFrom": "2026-11-03T00:00:00Z", "validUntil": "2026-11-04T00:00:00Z",
    "details": {}, "fetchedAt": "2026-11-02T13:00:00Z"
  }
}
```

`Cache-Control: public, max-age=300`.

## `GET /v1/summary?lat=&lon=`

Everything known about one place, for the search panel after a place is
picked. It contains:

- **`regions`:** the province, district, DS and GN division containing the
  place (`/v1/regions/lookup`), each with `id`, `level`, `name`, `nameSi`
  and `nameTa`. `null` outside Sri Lanka; a level is `null` where no
  region of it contains the place.
- **`weather`:** the point forecast above, cut to 24 hours and 3 days.
  `null` outside the grid or before any run.
- **`rivers`:** up to three Irrigation Department gauges within 50 km,
  nearest first, with distance, status, trend, level and thresholds. Same
  fields as `/v1/rivers/levels`.
- **`floodForecast.gauges`:** up to three Google gauges within 75 km.
- **`floodForecast.areas`:** the Google flood areas that contain the place.
- **`alerts`:** derived alerts, most serious first. See below.
- **`official`:** NBRO, Department of Meteorology and DMC warnings, which are
  authoritative: the landslide level of the place's DS division, bulletins
  naming its district and the latest general bulletins. See
  [warnings and hazards](warnings-and-hazards.md).
- **`unavailable`:** the parts that could not be filled (`weather`,
  `regions`, `rivers`, `floodForecast`, `official`). The rest of the summary is
  still returned.

Example for Colombo (`lat=6.9271&lon=79.8612`): 5.9 KB. The arrays and the
second and third rivers are cut here.

```json
{
  "data": {
    "location": { "lat": 6.9271, "lon": 79.8612 },
    "generatedAt": "2026-11-02T14:51:02Z",
    "regions": {
      "province": { "id": "LK-1", "level": "province", "name": "Western", "nameSi": "බස්නාහිර පළාත", "nameTa": "மேல் மாகாணம்" },
      "district": { "id": "LK-11", "level": "district", "name": "Colombo", "nameSi": "කොළඹ දිස්ත්‍රික්ක", "nameTa": "கொழும்பு மாவட்டம்" },
      "ds": { "id": "LK-1103", "level": "ds", "name": "Colombo", "nameSi": "කොළඹ", "nameTa": "கொழும்பு" },
      "gn": { "id": "LK-1103140", "level": "gn", "name": "Suduwella", "nameSi": "සුදුවැල්ල" }
    },
    "weather": {
      "location": { "lat": 6.9271, "lon": 79.8612 },
      "current": {
        "validAt": "2026-11-02T14:45:00Z", "temperatureC": 25.6, "apparentTemperatureC": 30.4,
        "relativeHumidityPct": 91.1, "dewPointC": 24, "precipitationMm": 0.1, "precipitationProbabilityPct": 83.3,
        "cloudCoverPct": 96.5, "pressureHpa": 1014.3, "windSpeedKmh": 8, "windDirectionDeg": 148,
        "windGustsKmh": 15.1, "weatherCode": 51, "uvIndex": 0, "isDay": 0
      },
      "hourly": { "times": ["2026-11-02T14:00:00Z", "…"], "temperatureC": [25.7, "…"], "…": "…" },
      "daily": { "dates": ["2026-11-02", "2026-11-03", "2026-11-04"], "precipitationSumMm": [5.1, 13.1, 9.4], "temperatureMaxC": [29.8, 28.5, 28.6], "…": "…" },
      "meta": { "source": "open-meteo", "…": "…" }
    },
    "rivers": [
      {
        "id": "nagalagam-street", "name": "Nagalagam Street", "basin": "Kelani Ganga", "distanceKm": 3.9,
        "status": "normal", "trend": "rising", "waterLevel": 2, "unit": "ft",
        "alertLevel": 4, "minorFloodLevel": 5, "majorFloodLevel": 7,
        "observedAt": "2026-11-02T14:34:42.397Z", "stale": false, "lon": 79.8786, "lat": 6.9581
      }
    ],
    "floodForecast": { "gauges": [], "areas": [] },
    "alerts": [],
    "official": {
      "status": "available",
      "landslide": { "status": "none", "level": null, "note": "Last report 31 Oct 2026; none currently active as far as we know." },
      "warnings": [],
      "general": [],
      "note": "Bulletins reach us through nuuuwan/lk_dmc, usually within hours of being issued. The PDF on dmc.gov.lk is the official version.",
      "stale": false,
      "fetchedAt": "2026-11-02T19:53:04Z"
    },
    "unavailable": []
  }
}
```

### Derived alerts

Each alert has the following fields:

| Field | Meaning |
|---|---|
| `type` | `heavy_rain`, `strong_wind`, `heat`, `river_level` or `flood_forecast` |
| `level` | `yellow` (be aware), `orange` (be prepared) or `red` (take action) |
| `title`, `detail` | Short English text |
| `value`, `threshold`, `unit` | The value that triggered the alert and the threshold it reached, where they apply |
| `validFrom`, `validUntil` | The window the alert applies to |
| `estimate` | Always `true` |
| `basis` | The data the alert was derived from |

Derived alerts are **estimates made by lanka-weather**, not warnings, and the
interface must say so. Each type has at most one alert, except river and
flood alerts, which come one per gauge or area.

| Type | Measure | Yellow | Orange | Red |
|---|---|---|---|---|
| `heavy_rain` | Rain over the next 24 hours, or tomorrow's or the next day's local total (mm) | ≥ 50 | ≥ 100 | ≥ 150 |
| `strong_wind` | Sustained 10 m wind over the next 24 hours or the next two days (km/h) | ≥ 40 | ≥ 60 | ≥ 90 |
| `strong_wind` | Gusts over the same windows (km/h); the higher level wins | ≥ 60 | ≥ 80 | ≥ 100 |
| `heat` | Highest heat index over the next 24 hours (°C), NWS formula from temperature and humidity | ≥ 33 (extreme caution) | ≥ 42 (danger) | ≥ 54 (extreme danger) |
| `river_level` | Irrigation Department gauge within 25 km, reading not stale | alert level | minor flood | major flood |
| `flood_forecast` | Google gauge within 30 km | above normal | severe | extreme |
| `flood_forecast` | Place inside a Google flood area | area of an above-normal gauge; flash flood likely; event with < 50,000 people | area of a severe gauge; flash flood highly likely; event with 50,000 to 99,999 people | area of an extreme gauge; event with ≥ 100,000 people |

The rain steps follow the Department of Meteorology's "above 50, 100 and
150 mm" advisories. The heat steps follow the heat-index categories its
advisories use.

An alert, for illustration:

```json
{
  "type": "heavy_rain",
  "level": "orange",
  "title": "Heavy rain likely",
  "detail": "About 117 mm of rain is forecast in the next 24 hours (threshold 100 mm).",
  "value": 117,
  "threshold": 100,
  "unit": "mm",
  "validFrom": "2026-11-02T14:00:00Z",
  "validUntil": "2026-11-03T14:00:00Z",
  "estimate": true,
  "basis": "Open-Meteo forecast, interpolated from the weather grid"
}
```

`Cache-Control: public, max-age=120`.

## Timings

Measured locally against the collected database, on Apple silicon:

| Endpoint | Size | Gzipped | First request | Warm |
|---|---|---|---|---|
| `/v1/weather/current` | 21.6 KB | 4.2 KB | 233 ms (loads the runs) | 1 ms |
| `/v1/weather/point` (Kandy) | 8.2 KB | n/a | 1.5 ms | 0.7 ms |
| `/v1/weather/wind` | 38.9 KB | 16 KB | 11 ms | 0.5 ms |
| `/v1/floods/forecast` | 6.1 KB | n/a | 30 ms | 5 ms |
| `/v1/summary` (Colombo) | 5.9 KB | n/a | 27 ms | 6 ms |
| `/v1/weather/regions?level=province` | 2.2 KB | 0.6 KB | 57 ms (loads the runs) | 0.5 ms |
| `/v1/weather/regions?level=district` | 5.8 KB | 1.1 KB | 0.6 ms | 0.5 ms |
| `/v1/weather/regions?level=ds` | 76 KB | 9.2 KB | 0.9 ms | 1 ms |
| `/v1/weather/regions?level=gn` | 3.2 MB | 270 KB | 18 ms | 12 ms (28 ms gzipped) |
| `/v1/weather/regions?level=gn&parent=LK-1103` | 8.2 KB | 1 KB | 0.8 ms | 0.4 ms |

The region joins run when the API starts (all four levels in 0.3 s), so
the first region request only aggregates the run.
