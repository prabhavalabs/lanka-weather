# ADR 0012: Rainfall, weather stations, air quality and sea conditions

- Status: Accepted
- Date: 2026-11-02

## Context

The rainfall and weather-stations layers were planned but empty, and the
map had nothing on air quality or the sea. Their sources raise questions the
earlier records leave open:

- `docs/data-sources.md` had the browser load NASA's tiles, which the
  project rule against calling third parties for a visitor forbids.
- Met Department observations reach us only through scraped copies on
  GitHub, because `www.meteo.gov.lk` has no API and timed out when tried. On
  2 November 2026 the copies were 4 days (3-hourly) and 10 days (daily)
  behind, although the second repository is pushed hourly.
- The 3-hourly README heads its time column "UT", which would be wrong by
  5.5 hours if believed.
- Open-Meteo's free tier is already 72 % spent by the weather grid (ADR 0007).

## Decision

### Satellite rain (NASA GIBS IMERG)

- **Collection:** every 30 minutes the collector asks GIBS which half hours
  it has, with the WMTS domains request (about 370 bytes, against 5.8 MB for
  the capabilities document), and fetches those it lacks, newest first.
- **Tiles:** `IMERG_Precipitation_Rate_30min` on
  `GoogleMapsCompatible_Level6`, so zoom 6 is the deepest native zoom. It is
  the only zoom kept: the map's lowest zoom (5) already draws zoom 6 tiles,
  and deeper zooms overzoom them. Nine tiles cover 73.1–90° E, 0–16.6° N:
  Sri Lanka in the middle tile and the rain approaching it. About 3 KB each,
  so 12 hours take about 0.6 MB.
- **Retention:** 24 half hours, counted back from the newest image and not
  from now, so an outage leaves the layer as it was and marked stale.
- **Failures:** a step is stored whole, in one transaction. A step GIBS lists
  but answers 404 for (it was seen at 14:00 UTC during testing) waits for the
  next run and is not a failure. An all-transparent tile is stored as empty,
  also not a failure, and served as a one-pixel transparent PNG.
- **Storage is PostgreSQL `bytea`, not disk under `cache/`.** The worker and
  the API are separate processes that may not share a disk, every other
  collected field is already in the database (`wind_fields` is the precedent),
  and pruning and "stored whole or not at all" come with a transaction. The
  volume is under a megabyte. PNGs do not compress, so TOAST is told not to
  try (`STORAGE EXTERNAL`).
- **Serving:** `/v1/rain/tiles/{time}/{z}/{x}/{y}.png` with
  `immutable` cache headers, because a stored half hour is never refetched.
  GIBS's "best" layer can later replace an early-run image with a better
  one; with 12 hours kept we do not chase that.
- **Colours:** the legend is NASA's own, taken from the layer's colour map
  (`GPM_Precipitation_Rate.xml`) and checked against a recorded excerpt by a
  test. NASA draws the heaviest rain nearly black, which a dark basemap would
  swallow, so the dark theme lifts the image with `raster-brightness-min` and
  applies the same sum to the legend.
- **Staleness:** the image lags about 6 hours by nature, so the layer's stale
  limit is 10 hours. The interface always shows the image's age and says
  "Estimate".

### Met stations

- **Times are local.** All but 22 of 39,253 readings fall at minute 30, while
  WMO synoptic hours are on the hour UTC, which only a UTC+05:30 clock turns
  into minute 30. The coolest slot is 05:30 and the warmest 11:30–14:30. The 22
  are one batch stamped 17:31 to 17:52, as a scrape reached them one by one;
  they belong to the 17:30 report.
- **A daily report covers the 24 hours to 08:30 local on its date.** Its rain
  equals the sum of the eight 3-hourly readings over that window (Galle
  14.6 mm and Anuradhapura 12.2 mm on 23 October 2026, exactly). `time_ut` is
  local midnight of the date, so readings are stored at 08:30 local.
- **Fetching the daily feed:** the file is in time order, so an HTTP range for
  its last 256 KB gets about 30 reports from a 10 MB file, including any days
  a worker missed. `latest_flat.json` would be smaller but holds one report.
  The 3-hourly file is read conditionally with its ETag.
- **Positions:** the geocoded `flat_places.json` holds names at (0, 0), and
  many at the geocoder's answer for "Sri Lanka" (7.873, 80.772). So the 24
  synoptic stations use a table in code: 21 from NOAA's ISD station history,
  matched on the WMO number, and 3 with local numbers (Mattala, Polonnaruwa,
  Moneragala) from the list, each within 2.1 km of OpenStreetMap's place. All
  24 lie within 7 km of their geocoded towns. Rain-only stations use the list
  with those positions removed, and the panel calls them approximate. Stations
  with no usable position are not stored; the collector logs their names.
- **Fallback:** a current 3-hourly reading wins. When it is out of date a newer
  daily report takes over. When both are old, the newest reading stays and is
  marked stale, on the marker (hollow), in the list and in a plain sentence in
  the panel. Stale limits are 6 hours for 3-hourly and 36 hours for daily
  readings, since a daily report comes at irregular hours.

### Observed rainfall

- **Met stations:** the daily report's total, or the sum of the 3-hourly rain
  over the day to the newest reading when at least 6 of 8 are there.
- **River gauges:** `river_readings.rainfall_mm` is rain since the reading
  before it. The 24-hour total is their sum, given at least 18 readings. The
  department does not document the field, but its own bulletin settles it, as
  checked on 2026-11-02:
  - **The Irrigation Department's bulletin agrees.** Its evening water-level
    report prints "9 Hr RF in mm at 6:00", on the DMC's site. Over four evenings
    (30 October to 2 November), of the 144 rows with both figures, our sum of
    readings from 09:00 to 18:00 local matches 141 within 0.1 mm (Manampitiya
    76.1, Thaldena 49.8, Deraniyagala 26.0). The others: Nawalapitiya is 0.3 mm
    under, and Padiyathalawa differs both ways (30 against our 0 one day, 0
    against our 46.7 another), as that gauge's readings arrive in late batches.
  - **It is not a running total.** Hourly values fall and rise within a day
    (Thaldena 49.8, 16.3, 1.3, 1.7, 4.0) and have no reset at any hour.
  - **`waterlevels.lk` agrees where it can.** Its `rain.fallen_24h_mm` matched
    our 24-hour totals at 31 of 32 stations with rain figures, and is a sum from
    the same feed, so it is a check on the arithmetic rather than on the meaning.
    Wellawaya is the difference: it reads 21 mm against our 15 mm, counting
    twice a 6 mm reading entered nine minutes apart with the same level.
  - **Repeats were being handled wrongly, and now are not.** The first rule
    counted once any readings within 10 minutes. The bulletin shows the two cases
    differ. The same level and rain entered twice is one reading: Ellagawa's
    9.5 mm appears twice on 30 October, and the bulletin's 11.5 mm for that
    window is 9.5 + 2. Readings seconds apart with different levels are a backlog,
    and each holds its own rain: Pitabeddara's 4.4 and 0 mm on 1 November, where
    the bulletin's 38.9 mm is 34.5 + 4.4, and the old rule would have lost the 4.4.
- Totals older than a week are left out of the layer. An old total keeps a
  hollow point but no number, so it cannot read as today's rain.

### Air quality and sea conditions

- **Grids:** air quality on 0.4° (CAMS's own resolution, 74 nodes, land-masked
  as the weather grid is); sea state on 0.5° from 79–82.5° E, 5–10.5° N (75 sea
  nodes), which is the waters near the coast rather than the open ocean.
- **Budget:** both every 3 hours, batched at 50 coordinates and paced as the
  weather grid is, about 1,200 weighted calls a day. With the weather grid that
  is about 8,400 of 10,000, assuming the APIs share one limit per address. A
  test fails above 8,500.
- **Storage** reuses `weather_runs` and `weather_fields` with two more kinds,
  so pruning and "stored whole" are shared. The grids stay out of
  `weather_grid_nodes`, which regions join to.
- **Sea field:** the field has to stop at the coast but MapLibre cannot mask a
  raster, and the basemap has no land polygon to put over it. So the land
  (the Natural Earth coastline of ADR 0007) is cut out of the image in a canvas.
  That coast is about 1 km coarse, so the field fades as the map zooms in; at
  those zooms a 55 km grid has nothing to say about a harbour anyway.
- **Coastal places:** a place has a sea row if it is on the water or within
  about 16 km of it, reading the nearest sea node up to 75 km off. Colombo's
  nearest node is 40 km away.
- **Labels:** both are model data. The interface says "Estimate" for each
  value and tells users to follow official sea-area warnings.

## Consequences

- Today the Met layer is visibly out of date, because its sources are. Its
  readings are still the only measured temperatures we have. A source that
  answers directly should replace these copies when one is found.
- River gauges are the only current measured rainfall, and they cluster in the
  wet zone.
- 41 of the 204 names in the last 120 days of daily reports (10 % of rows) have no usable
  position and are not stored, including Victoria and Laxapana.
- A second API replica or worker reads the same rows; there is no shared disk
  to arrange.
- Adding air quality and sea state leaves the free tier about 16 % of margin.
  Running a second worker on one address would exceed it (ADR 0007 warns of
  the same).
