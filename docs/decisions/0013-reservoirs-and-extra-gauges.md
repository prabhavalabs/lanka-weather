# ADR 0013: Reservoirs and extra river gauges through waterlevels.lk

- Status: Accepted
- Date: 2026-11-02

## Context

The `reservoirs` layer had no source. ADR 0004's collector reads 40 river
gauges from the Irrigation Department, which leaves basins with no gauge on
the map. The agencies publish reservoir figures as daily sheets and reports,
with no machine-readable feed we have found.

Sri Lanka Water Levels (`waterlevels.lk`) is an independent, unofficial
community site by @shakeeb_x. It compiles the Irrigation Department's reservoir
sheet and gauges, the Mahaweli Authority's daily report and RiverNet (the
Disaster Management Centre's sensors, by Leecom SCADA). Its `/data` page says the
data "is free, needs no key and allows any website to read it", and its
`robots.txt` allows `/api/levels` and `/api/reservoirs`, among others. Each
response has a `license` field pointing at `about#data-sources`; no formal
licence is stated. Its source repository `shakee93/sl-water-levels` has no
licence, so none of its code is used, nor its raw `data` branch.

## Decision

### Source and credit

- Read only `/api/reservoirs` (every 6 hours) and `/api/levels` (every 30
  minutes), on a schedule, with our User-Agent. The API sends no `ETag` or
  `Last-Modified`, so requests cannot be made conditional.
- Credit "Sri Lanka Water Levels (waterlevels.lk)" and the underlying agency
  wherever the data appears. RiverNet is named only for the 17 gauges whose page
  on the site credits it, read by hand on 2026-11-02, because the API has no
  per-station credit. The other five extra gauges name no agency, and the panel
  says so rather than guess.
- Say on every reservoir panel that this is not an official warning and to call
  117 in an emergency.

### Reservoirs

- A row per reservoir per calendar day in Sri Lanka (`snapshot_day`), replaced by
  later fetches that day and kept indefinitely. `reading_day` is the day the
  figures were read. Keying on the fetch day means a stale repeat can never
  overwrite the reading it repeats.
- History shows only days actually read. A status of `nodata` is shown as no
  data, and a figure with `storageFresh` false as a last known figure with its
  date. Null is unknown, never zero. Live data has `nodata` with a fresh storage
  figure, mostly from the Mahaweli Authority: the figure is shown, the status
  stays `nodata`.
- A reading is stale 48 hours after the end of its reading day in Sri Lanka. A
  stale "spilling" does not count as spilling anywhere.
- `flow.kind` is `spill`, `outlets` or `downstream` in live data. A Mahaweli
  Authority `downstream` flow can include power-station flow, so it is labelled
  as flow and carries a note that it is not necessarily a flood release.

### Reservoir positions

- The feed has no coordinates. A third collector, `osm-reservoirs`, queries
  Overpass weekly (about 15 MB) for named water bodies and dams, and matches them
  by normalised name or a code alias inside the reservoir's district. Polygons
  are stored; a reservoir known only by its dam gets a marker.
- The feed gives no district for most Mahaweli reservoirs, so these come from a
  table in code. A reservoir with no district is never mapped on its name alone.
- The feed's district is sometimes the Irrigation Department's range office, not
  where the tank is. A candidate within about 15 km of the district is accepted
  only if nothing else in the island qualifies under that name.
- A reservoir that does not match stays off the map and its panel says "Location
  not mapped". On 2026-11-02, 115 of 142 matched (81%).
- A run that maps fewer than 40 reservoirs is rejected and the last map is kept.

### Extra gauges

- Only stations the Irrigation Department feed lacks enter `river_stations`, with
  `source` set to the collector, so everything that reads those tables picks them
  up: status colours, staleness, trend and the summary. The department's feed
  stays authoritative and direct. Nothing from this source is written to one of
  its stations, enforced in the plan and in the upsert's `WHERE`.
- Identity is the slug of the name, through an alias table (`Ratnapura` is the
  department's `rathnapura`). A report within 150 m of another source's station
  is dropped as a suspected duplicate and logged. The collector does not run
  until the department's stations are stored, or every report would look new.
- `LatestRiverObservation` is now per source. Otherwise a reading from this feed
  would move the point the department's collector resumes from, and an outage of
  the department's feed would leave a permanent gap.
- Levels and marks are stored in the published unit. The site gives marks in feet,
  rounded to 0.01 ft, so converting back and rounding to the centimetre returns
  the agency's figure: tested for every 5 cm up to 25 m. A quiet gauge publishes no
  unit or marks, so a report with no unit keeps the last known ones.

### Ampara

The department's feed labelled Ampara in metres by default. It has read exactly 0
at all 173 observations stored, has no thresholds, and the site gives its unit as
`unknown`. Unit values now include `unknown`, Ampara uses it, and it is shown as a
bare number with no flood verdict. Whether the gauge works is for the department
to say; rainfall at the same station is reported normally.

### Next-hour forecast

Stored for every station from `/api/levels`, in feet as published, and kept 30
days. It appears on the station panel as one line labelled experimental and
credited, converted to the station's unit, with its issue time, and is hidden when
null or older than 2 hours. It is always labelled experimental, even where the
source's own `experimental` flag is false, and it never changes a status or an
alert. A station with no known unit does not show it.

## Consequences

- If the site stops or changes, its collectors fail visibly in `source_runs`. The
  reservoirs layer and the extra gauges age and turn stale; the department's
  gauges and every other layer are not affected. The fallback for reservoirs is the
  upstream: the department's reservoir sheet and the Mahaweli Authority's report,
  which would need new parsers.
- We depend on one person's unofficial project. Its terms could change; none is
  formal today. Check its `/data` page when reviewing sources.
- RiverNet credit is a table checked by hand and can drift. New gauges from the feed
  show "upstream agency not stated" until the table is updated.
- About 19% of reservoirs are list-only until OpenStreetMap or the alias table
  covers them. The weekly log names each with its reason.
- Readings from the extra gauges arrive at most every 30 minutes, although RiverNet
  reports about every 10.
