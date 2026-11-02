# ADR 0010: Rivers, overflow risk and flood basins

- Status: Accepted
- Date: 2026-11-02

## Context

The owner's first priority is to mark Sri Lanka's prominent rivers on the
map, to show whether each stretch is at risk of overflowing, and to show
flood-forecast boundaries by area, as Google Flood Hub does.

Three facts shape this:

- Google publishes polygons only for gauges above normal, so its own flood
  areas are usually empty. The map must still show the country in sections
  when nothing is flooding.
- There are about 50 gauges for 142 rivers. Most river stretches have no gauge
  at all, and must never look safe for want of one.
- Late in the work, the Irrigation Department's public ArcGIS organisation
  turned out to hold official basins, rivers, station lists and past flood maps.

## Decision

### Which source for what

We inspected every candidate, and kept each source only for what no better one
gives. Nothing is built twice.

| Need | Source | Why |
|---|---|---|
| Country-level basin boundaries | Department's `river_basins` | Official; see the audit below |
| Sub-basins (zoom 9 and in) | HydroBASINS level 12 | Whole country with `NEXT_DOWN`. The department's `Flood_Map` has named sub-basins, but only for 15 basins and with no flow topology |
| River lines and flow topology | HydroRIVERS, through `nuuuwan/lk_rivers` | The only source with `NEXT_DOWN`, `UPLAND_SKM` and `HYBAS_L12` |
| River names | Official basins, then the station list, then stream polygons, then `lk_rivers` | See "Names" |
| Google gauge to river | `HYBAS_L12` on HydroRIVERS, then proximity | `hybas_<id>` names the sub-basin directly |

Rejected:

- **Department's `rivers`.** 3,379 polylines whose names are the numbers 1 to
  3,379, with no topology. HydroRIVERS does everything it could.
- **Department's `Srilanka_River` as geometry.** 1,833 polygons of stream areas
  from a 1985 survey (`GFCODE` STRMA). Used only for its 1,030 names.
- **A dissolve of HydroBASINS for country-level basins.** Unneeded once the
  official layer proved usable.

### Audit of `river_basins`

The layer has 727 polygons in EPSG:3857, last edited in 2022. It is complete
but not clean:

- **Complete.** It totals 65,643 km², against about 65,600 km² for the island.
  Overlaps between polygons total 7.4 km².
- **Named.** 115 polygons carry 102 distinct names. After cleaning there are
  exactly 103 named basins, which matches the 103 river basins usually counted
  for Sri Lanka.
- **Not clean.** 612 polygons are called `Placemark`: 511 are slivers under 1 km²,
  and 101 are real stretches of coast draining directly to the sea (6,064 km²,
  the largest 942 km² around Jaffna). 12 named polygons are scraps of a basin
  named elsewhere. One name is misspelt (`Telwatta Ganaga`). Two different rivers
  50 km apart are both `Pali Aru`.

`CleanBasins` is a pure function that merges named scraps into their basin,
drops unnamed slivers, keeps large unnamed polygons as "Coastal area", and keeps
same-named rivers apart. The result is 204 basins totalling 65,597 km².

### Names

Each river system takes the name of the official basin holding most of its
catchment. A basin names only its largest system, because the streams beside a
great river's mouth share its polygon but are not that river. The other names
stay searchable as aliases: `lk_rivers` calls Andella Oya "Navakiri Aru".

- **Exception.** The department's own flood maps and stations say Malwathu Oya
  where `river_basins` says Aruvi Aru. People say Malwathu Oya, so it is shown,
  with Aruvi Aru as an alias.
- **Tributaries.** Two tiers. First those named outright: the `Tributory` of each
  station on `hydrostations`, and `lk_rivers`' non-outlet ids, which name a
  tributary by its source segment, not by `MAIN_RIV`. Then those read from
  `Srilanka_River`: the longest connected chain voted for a name in a system, at
  least 8 km long and reaching 100 km². Within a tier the largest river claims
  its course first. A name read from polygons stops at its last polygon, so
  "Kambarawa Ganga" does not run on over the Amban Ganga below it.
- **Errors in `lk_rivers` handled.** Blank names and `(Unnamed)` mean nothing. Two
  ids name the Kelani, one of which is a headwater of the Gurugoda Oya: it is
  ignored, because a chain carrying the main river's name counts only if it joins
  the main river. Headwater names under 25 km² are dropped, as too small to draw.
- **Stations' basin names.** `CanonicalBasin` holds an explicit alias map. "Mee
  Oya" means Mi Oya: there is a separate, much smaller Mee Oya in the north-east,
  so the two cannot be matched by spelling.
- **Sinhala** names are given for 15 rivers whose spelling is certain, and for no
  others.

### Network

`river-network` is a derived collector. It fetches nothing, and rebuilds only when
a fingerprint of its inputs changes: rivers, basins, regions, and every row of
`river_stations`, whichever collector added it.

- **Snapping stations.** The nearest suitable segment within 3 km with at least
  15 km² upstream. Within 500 m of the nearest (the 15 arc-second grid's error) the
  larger river wins, because a rill often runs closer to a gauge than its river.
  Where the department's list says a station stands on its basin's main river, the
  main stem is preferred: Nagalagam Street lies 37 m from a Colombo canal and
  769 m from HydroRIVERS' line of the Kelani. An unknown basin, or one with no match
  nearby, falls back to position alone and is logged. Stations nothing fits are
  logged and left out.
- **Snapping Google gauges.** The largest segment of the gauge's sub-basin within
  5 km, then the nearest suitable. Only verified gauges govern reaches.
- **Reaches.** A stem is split wherever its governing gauges change, walking
  `NEXT_DOWN`. A reach has the nearest gauges upstream (at a confluence, the largest
  branch with any gauge decides, so a small tributary in flood does not colour a
  main stem whose own gauge reads calm) and the next gauge at its lower end. A
  stretch above its first gauge is ungauged, and names the gauge below it, but that
  gauge is not evidence for it. A reach's id is its lowest segment's `HYRIV_ID`, so
  it survives rebuilds.

Result on real data, checked against geography: 142 rivers (134 named, 51 of them
tributaries), 212 reaches (52 gauged), all 50 gauges matched. The Kelani reaches the
sea at 79.87° E, 6.98° N, near Colombo; the Mahaweli at 81.25° E, 8.46° N, in Koddiyar
Bay near Trincomalee; Nagalagam Street is on the Kelani.

### The risk estimate

`Assess` is a pure, table-tested function. It weighs a list of rules, one per kind
of evidence, each returning findings. **To add an input**, such as an upstream
reservoir spilling: add a field to `Inputs` and a rule to `rules` that returns a
finding with `Compounds` set and capped as the evidence deserves. Nothing else
changes; `TestANewRuleNeedsOnlyAFinding` shows it.

| Evidence | Level |
|---|---|
| At or above the gauge's major flood level | severe |
| At or above its minor flood level | high |
| At or above its alert level | watch |
| Below alert, near it and not falling | watch |
| Rising, and would reach the next threshold within 6 hours | that threshold's level |
| Google forecast above normal / severe / extreme | watch / high / severe |
| Google forecast severe and still rising | severe |
| Catchment rain of 50 mm in 24 h or 100 mm in 72 h | watch |
| Catchment rain of 100 mm in 24 h or 200 mm in 72 h | high |
| Rain or similar on a river already at watch or above | one step up |

Why these values:

- **Thresholds** are the Irrigation Department's, published per station. Nothing
  here invents a flood level. Levels and thresholds are compared in the station's
  own unit (ADR 0004).
- **6 hours** is long enough to act on and short enough to trust. The rate comes
  from a window of up to 3 hours (ADR 0004), and extrapolating further than twice
  that says more than the reading does.
- **"Near"** means within a quarter of the department's own gap from alert to minor
  flood at that station. A gauge's datum is arbitrary, so a share of the level means
  nothing, and a fixed height means different things on different rivers. With no
  minor level, 10% of the alert level is used.
- **Rain bands** follow the Department of Meteorology's advisory language ("fairly
  heavy" above 50 mm, "heavy" above 100 mm), the same bands the location panel uses.
  Those describe a place; here they are read against a catchment's average, which is
  harder to reach, so reaching one is at least as serious. The 72-hour bands are
  double, for saturated ground.
- **Rain alone stops at high**, because a forecast is not a measurement of the river.
- **Google's trend counts only at the top.** On an ordinary dry day 9 of the 10
  verified gauges show "rise", so a rising calm forecast says nothing.

What never happens:

- **Nothing reads safe without evidence.** Only a current gauge or a current forecast
  can make a stretch `normal`. A dry rain forecast cannot. Without one, the level is
  `unknown` and the reasons say so, even when rain raises it.
- **Stale readings count as no evidence.** A gauge older than the layer's limit
  (3 hours), or a forecast older than 30 hours, is named and not used.
- **Missing thresholds are tolerated.** Stations may publish some or none. A level is
  read against what exists, and without an alert level a low reading proves nothing.
- **Unverified Google gauges do not affect river risk.**

Every response says `estimate: true` and carries a note citing the departments and
telling people to follow official warnings. The UI marks it "Estimate" everywhere.

Rain reaches a stretch by accumulation, not a stored join: each segment's forecast
rain is sampled at its midpoint and weighted by its local catchment (`CATCH_SKM`), then
accumulated down `NEXT_DOWN`, so each segment knows the mean over everything draining
through it. This takes no polygons and no per-reach lists.

### Compute at request, with no risk table

The estimate is computed from fresh tables on request and kept for a minute. Gauges
are polled every 15 minutes, so a minute costs nothing, and it is fast because the
network stays in memory per version.

A stored risk table would also keep Google-derived data beyond its 24 hours. As it is,
Google's forecast is read only from `flood_gauges`, whose day limit is unchanged. The
network keeps Google's gauge ids in `river_gauge_matches` and `river_reach_gauges`: these
are renewed hourly, `Prune` deletes matches older than a day, and a rebuild then removes
them from reaches.

### Serving

- **Tiles.** Reach parts per width class (by upstream area), so a line widens downstream
  and recolours as one reach. The reach's `HYRIV_ID` is the feature id, so colours are
  expressions on stretch ids and new readings recolour the map without reloading tiles.
- **Tile versions.** River tile URLs carry `networkVersion`, so a rebuild cannot mix old
  lines with new ids, and tiles can be cached for a week.
- **Labels** run along a heavily simplified and smoothed copy of each river. A label
  cannot follow a line stepping along a 15 arc-second grid.
- **By zoom.** Named rivers of 500 km² and more always; smaller named rivers from zoom 7
  and 8; unnamed ones from 8 and 9; other streams from zoom 7 up by upstream area.

### Colour alone never carries risk

Risk uses the status palette, and also: wider lines the higher the risk, marks along the
line (dots for watch, triangles for high, double chevrons for severe), a pale casing on
anything measured, and a dashed line for a stretch nothing measures. Every panel pairs
colour with a word.

### Flood basins

Official basins are always shown while the flood layer is on, tinted and outlined by the
worst current gauge, forecast or Google area in them:

- **No flooding:** a pale tint and a solid outline.
- **No forecast here:** no tint and a dashed outline, so it can never read as calm.
- **Raised:** a strong tint, and a heavier outline in the status colour.

From zoom 9, sub-basins take over the tint. A sub-basin takes the state of its own gauge
and of those above it, down to the next gauge, since a forecast describes the water
leaving a gauge. Upstream of every gauge there is no forecast.

Unverified gauges tint only when `LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED=true` is set, and a
basin resting only on them says so in its panel and in the legend.

Google's polygons change from dashed outlines to solid fills with strong outlines,
inundation graded by how likely the water is, the likeliest drawn on top. They open a
panel with valid times and attribution.

**A click on a basin now takes the click that used to open a region.** With the flood layer
on, which is the default, the click order is: gauge, flood area, river, past flood, basin,
then region. The basin panel offers "Weather at this place". With the flood layer off,
regions take clicks as before. Region colours also default to off for new visitors, so
rivers and basins read first; they stay one control away.

### Past floods

A separate overlay, off by default, shows 25 mapped events from `Flood_Map` (17 events,
1989 to 2019), `Flood__Nov2025` (7) and `Puttalam_Flood__Detwah_30Nov2025` (1, spelled
"Ditwah" on the map). It is hatched in violet, a colour and a pattern no live forecast uses,
drawn under everything current, and labelled as past on the list, the chip and the panel.
Its dated event name comes from the source; a value with no year is skipped and logged.

- **Layers are found by value.** `Flood_Map` repeats its layer groups, and ids do not line up
  with events, so only "Flood Inundation Area" layers are read, by their `Flooding_area`
  value. Spelling is corrected for reading (`Kelani Ganag`, `Mundani Aru`) and the source's
  own text is kept and shown.
- **Geometry.** The extents are traced from rasters. Kala Oya's has 67,952 rings, 12,700 of
  them holes, and took over 5 minutes in `ST_MakeValid`. Rings under 0.1 ha are dropped
  (97.6% of that extent's area stays) and the `structure` method is used. Extents are stored
  whole, simplified for low zooms and cut into pieces for close zooms. Each service is read
  generalised to about 5 m, and the November service one feature at a time, because asking
  for all seven at once times out.
- **What these are.** The services do not say whether an extent was observed or modelled.
  "Kelani Ganga Flood 1989 Critical" may be either, so the panel says "mapped" and
  quotes the source.

### Terms of the department's services

The services publish no licence. Checked on 2026-11-02: `licenseInfo` and
`accessInformation` are empty on every item we use, `copyrightText` is empty on every
service, and the organisation states none. Items are shared publicly. We found no
restrictive terms, and none granting reuse either. We treat them as government data, credit
"Irrigation Department of Sri Lanka" wherever they are shown, and fetch them rarely. This
should be confirmed with the department before any commercial use.

### Rendering tested against real flood conditions

There is no flood today, so `cmd/riverseed` writes one into a local database, using real past
extents for Google's polygon kinds. It requires `LANKAWEATHER_ALLOW_SEED=true`, refuses the
database named `lankaweather`, and marks everything with the source `local-seed`. The API then
reports `seeded: true`, and the map carries a "TEST DATA" banner. Nothing in the service calls it.

### Collection

| Collector | Interval | Retry | Notes |
|---|---|---|---|
| `hydrosheds-hydrorivers` | weekly | 6 h | Reads the GitHub commits first; 3.5 MB when changed |
| `irrigation-river-basins` | weekly | 6 h | Reads `lastEditDate` first |
| `irrigation-past-floods` | monthly | 24 h | Long-running; about 5 minutes when changed |
| `hydrosheds-hydrobasins` | monthly | 24 h | Long-running; 80 MB to a temporary file, with a limit |
| `river-network` | hourly | default | Derived; a few small queries when nothing changed |

A collector implementing `source.LongRunner` runs in its own goroutine, one run at a time, so
a slow download never delays the quarter-hourly gauges. HydroBASINS' host sends no entity tag
and moves `Last-Modified` with no change to the file, so the archive's size names its version.

## Consequences

- People see rivers, risk and basins with no flood in progress, and can tell measured calm
  from nothing known.
- The estimate is only as good as about 50 gauges. Most stretches read "Not known", which
  is the honest answer. Rain can raise them, but nothing else can tell them apart.
- HydroRIVERS' line can run a kilometre or two from the true channel near the coast, and
  gauges are snapped to it. Reaches are approximate near river mouths.
- A river's level is its worst stretch's. A long river can read "Watch" because of one
  upland gauge; its panel shows which.
- Google's ids are kept for a day in the network tables, on top of `flood_gauges`.
- A click now opens a basin rather than a region while the flood layer is on.

## Open questions for the owner

- **Thresholds disagree.** `hydrostations` publishes different levels from the gauge feed
  for many stations: Hanwella is 6.5/8/10 m there and 6.5/7.5/9.5 m in the feed, and Holombuwa
  is 3/3/5 m against 3/4/6 m. We use the feed's, which arrive with each reading. The
  department should say which are current.
- **Public edit capabilities.** `hydrostations` lists `Create, Delete, Update, Editing`, and
  `gauges_2_view`, the source of our readings, lists `Create, Update, Editing`. We did not test
  them. If anonymous edits really work, anyone could alter the readings and thresholds this
  estimate uses. This should be raised with the department.
- **Two stations have no readings.** Calidonia and Thambuththegama are on the list but not in
  the feed.
