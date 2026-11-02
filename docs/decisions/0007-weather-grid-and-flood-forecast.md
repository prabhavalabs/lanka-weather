# ADR 0007: Weather grid, wind field and flood forecast

- Status: Accepted
- Date: 2026-11-02

## Context

The map and the location panel need weather for all of Sri Lanka: current
conditions to colour the map, a forecast for any place a person picks, an
animated wind layer, and Google's flood forecasts. Four constraints shape
the design:

- **No per-visitor calls to third parties** (architecture overview, goal 4).
  Every source is fetched on a schedule, stored with provenance and served
  from the database.
- **Open-Meteo's free tier** is for non-commercial use. It allows fewer than
  10,000 API calls a day, 5,000 an hour and 600 a minute, and the limits
  apply per IP address. A request may carry many coordinates, but each
  location is weighted as
  `max(1, variables / 10) × max(1, days / 14)` calls.
- **Google Flood Forecasting's terms** do not allow gauge search results to
  be kept for more than a day. The Google Weather API is disabled on the
  project's Google Cloud key (it answers 403).
- **The wind layer** must cover the sea as well as the island. Particles
  should not stop at the coast, and cyclones form over the Bay of Bengal.
  The layer should reuse TerraGuard's particle renderer and field format;
  TerraGuard's owner permits reuse.

## Decision

### Weather grid (Open-Meteo)

- **Grid:** a regular 0.2° lattice from 79.4° E, 5.6° N to 82.2° E,
  10.0° N (15 × 23 nodes), defined in code (`internal/grid`).
- **Land mask:** a node is kept when any of the four cells it is a corner
  of touches land. Land comes from a simplified Natural Earth coastline,
  which is public domain. This keeps 214 nodes: 136 on land and 78 near-shore
  sea nodes. Every point on land then has all four surrounding nodes, so
  interpolation never has to guess at the coast.
- **Cell selection:** land nodes ask Open-Meteo for a land model cell and
  sea nodes for a sea cell. Requests carry 50 coordinates each and are
  paced at 300 weighted calls a minute.
- **Current conditions, hourly:** 10 variables (weight 1 per node):
  - temperature, apparent temperature, relative humidity;
  - precipitation, weather code, cloud cover, mean-sea-level pressure;
  - wind speed, direction and gusts at 10 m.

  Dew point (Magnus formula) and day or night (solar elevation) are
  computed rather than requested. UV index and precipitation probability
  for "now" come from the forecast's nearest hour.
- **Forecast, every three hours:** 12 hourly variables over yesterday and
  the next eight days (weight 1.2 per node):
  - the current-conditions variables;
  - precipitation probability and UV index.

  The daily summary for seven local days (UTC+05:30) is derived from these
  hours:
  - temperature maximum and minimum, apparent-temperature maximum;
  - precipitation total, precipitation-probability maximum;
  - wind and gust maxima, dominant wind direction;
  - UV-index maximum, and the highest weather code.

  This avoids paying for daily variables and keeps local days whole.
- **Wind at 100 m:** not collected. Open-Meteo offers 80 m and 120 m, not
  100 m, and the particle layer uses the 10 m wind from GFS (below).
- **Call budget:** 214 nodes × (24 × 1.0 + 8 × 1.2) = **7,190 weighted calls
  a day**, 72 % of the free tier. The busiest hour costs 471 calls, against
  a limit of 5,000. A unit test fails if the defaults exceed 7,500 a day.

  Spacing alternatives:

  | Spacing | Nodes | Calls a day |
  |---|---|---|
  | 0.25° | 142 | 4,800 |
  | **0.2°** | **214** | **7,190** |
  | 0.15° | 349 | about 11,700 |
  | 0.1° | 704 | about 23,600 |

  0.1° does not fit, even with current conditions alone (about 16,900 a
  day). The models behind Open-Meteo's `best_match` over Sri Lanka (ECMWF
  IFS, DWD ICON, NOAA GFS) run at 9 to 25 km. A 0.2° (22 km) grid therefore
  loses little of their detail, and bilinear interpolation gives smooth
  fields between nodes.
- **Storage:** each fetch is a `weather_runs` row: source, model, grid, the
  node order, source time and fetch time. Each variable is a
  `weather_fields` row: a `real[]` of steps × nodes, with source, first
  valid time and fetch time.

  Runs older than a day are deleted, but the newest run of each kind is
  always kept. If a source stops, its old data is then served marked stale
  rather than vanishing.
- **Serving:** the API keeps the newest current and forecast runs decoded in
  memory and checks for newer runs every 30 seconds.
  - The current grid is encoded and gzipped once per run.
  - Point forecasts are bilinear between the four surrounding nodes. Weather
    code and day or night take the nearest node, and wind direction comes
    from interpolated wind components.
  - Open-Meteo is never called for a request.
- **Retries:** a request that fails with a server error (Open-Meteo answers
  "The service is overloaded" with 503) is retried after 20 and 60 seconds.
  A run is stored whole or not at all.

### Wind field (NOAA GFS, not Open-Meteo)

- The particle layer reads the GFS 10 m wind from NOAA Open Data on AWS:
  - keyless and in the public domain;
  - each run's 0.25° GRIB2 file per forecast hour;
  - only the U and V messages, by HTTP byte range from the `.idx`
    inventory, which is about 1.9 MB of a 500 MB file.
- The field is cropped to the map's panning limit (70–92° E, 2° S–18° N;
  89 × 81 points) and stored as little-endian int16 tenths of a metre per
  second (`wind_fields`). The API serves it in the format of TerraGuard's
  `WindField`, so TerraGuard's renderer (`wind/field.ts`, `renderer.ts`)
  can be reused unchanged.
- Forecast hours 0 to 72 are stored every 3 hours (25 fields). A run is used
  only once its 72-hour file is published. The collector checks every
  30 minutes. Runs more than 30 hours old are deleted, but never the newest
  complete run.
- **Why GFS:** the Open-Meteo grid is land-masked and could not afford the
  sea. A 0.25° wind-only rectangle over the same region would be 7,209
  locations, about 58,000 calls a day at three-hourly steps; even a
  10° square around the island would be 1,681 locations, more than 13,000
  calls a day. GFS gives a
  full regional field over the sea at 0.25°, the resolution of the models
  Open-Meteo blends, at no call cost. It costs about 47 MB of download per
  run, about 190 MB a day.
- The GRIB2 decoder (`internal/grib2`) is TerraGuard's. It reads templates
  3.0, 5.0, 5.2 and 5.3. Tests build messages in code, including a
  hand-packed 5.3 message, because GRIB files are never committed.

### Google Flood Forecasting (Sri Lanka only)

- **Gauge statuses:** `floodStatus:searchLatestFloodStatusByArea` with
  `regionCode: "LK"` returns the 10 quality-verified gauges, all HYBAS
  virtual gauges.
  - The 310 unverified gauges, which Flood Hub shows only in expert mode,
    are read only when `LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED=true`.
  - Thresholds come from `gaugeModels:batchGet`.
- **Flash floods and significant events:** these searches take no region
  filter, so they are read worldwide (one request each) and only events
  whose affected countries include LK are kept.
- **Polygons:** for anything above normal, the notification, inundation,
  flash-flood and event polygons are read from `serializedPolygons/{id}`
  (KML or GeoJSON) into PostGIS, with at most 40 a run.
- **Storage and terms:**
  - Collection is hourly, at about five requests a run.
  - Each run replaces the stored set.
  - Rows fetched more than 24 hours ago are neither served nor kept, as
    Google's terms require.
  - The key is sent only in the `X-Goog-Api-Key` header. It is never logged
    or sent to the browser.
- **Google Weather API:** not used, because it is disabled on the project.
  A Google Weather source could be added behind configuration and kept off
  until the API is enabled.

### Location summary and derived alerts

- `GET /v1/summary` combines:
  - the interpolated weather, with the next 24 hours and 3 days;
  - the three nearest river gauges within 50 km;
  - the three nearest Google gauges within 75 km;
  - the flood areas containing the place.
- Derived alerts are labelled `estimate: true` with their basis. The
  thresholds are listed in `docs/api/weather-and-floods.md`. An `official`
  slot, empty for now, will hold Department of Meteorology and DMC warnings,
  which are authoritative.

### Regions

- `GET /v1/weather/regions` aggregates the grid over administrative
  regions read through a `RegionSource` interface:
  - Regions larger than a grid cell take the area-weighted mean over the
    cells they intersect, and the extremes over cells mostly inside them.
  - Smaller regions, and every GN division, take the value interpolated at
    a point inside them.
- The regions table belongs to migration 00002 on another branch. Until it
  is merged, the source is a stub that answers 503 `not_available`.
  `LANKAWEATHER_WEATHER_REGIONS=postgis` switches to the PostGIS source.

## Consequences

- Weather covers the whole island at 0.2°. The map can colour it, and the
  location panel can show a forecast for any point without a third-party
  call.
- Running a second worker (for example in development) on the same IP
  address spends the same Open-Meteo budget again. Only one worker should
  collect per address. `lankaweather collect` costs about 470 weighted calls.
- Point values are model-grid values: they do not resolve valleys, hill
  stations or the sea breeze, and say so through the `grid` and
  `interpolation` fields of the response.
- The wind layer and the weather grid come from different models (GFS
  alone versus Open-Meteo's blend). Wind speeds shown in the panel and drawn
  by particles can differ slightly.
- Google's flood layer empties if collection stops for a day; this is
  required by its terms, and the layer is marked stale before then.
- Migration 00003 follows 00001 with a gap. A database that has applied
  00003 will need goose's allow-missing option, or a rebuild, once 00002 is
  merged.
