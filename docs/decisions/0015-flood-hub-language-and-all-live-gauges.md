# ADR 0015: Flood Hub's language, Extended coverage, live rain gauges and Colombo's canals

- Status: Accepted
- Date: 2026-11-03

Supersedes in part ADR 0007 (Google's searches, unverified gauges, the limit on polygons), ADR 0010
(how flood areas and basins are drawn; unverified gauges tinting only with a setting) and ADR 0013
(not reading the `data` branch of `shakee93/sl-water-levels`).

## Context

The owner asked that flood boundaries look like Google Flood Hub's, and for all the live gauges.
Comparing a screenshot of Flood Hub with our map, and reading live data for the first time, showed:

- Flood Hub's boundaries are flash-flood polygons, blocky pixel-grid shapes. On a calm day the
  country is green dots and nothing else. Our basins tinted the whole island.
- Until now the polygon decoder had only met invented data.
- About 275 live rain gauges and 33 Colombo canal gauges were not on the map.

## Decision

### Google's searches

The brief said the searches made one request with no paging. They already followed
`nextPageToken`, but stopped silently after five pages, with no test. Google's own data explains why
that matters: `significantEvents:search` answers one event per page at any page size (one live on
2 November 2026, with a token for a second, empty page), and a page can be empty with a token.

- Follow the token up to 25 pages at a page size of 1,000, and log if that limit is reached.
- The status search no longer stops at an empty page.
- `flashFloods:search` takes `countryCodes` (ADR 0007 said no region filter exists), so it asks for
  `LK`. Live, 131 events worldwide fit one page; `LK` had none and `IN` seven. `significantEvents:search`
  has no such filter and stays worldwide, kept by country here.
- Polygons are read flash floods and events first, then verified gauges, then lower-confidence
  gauges, up to 120 a run. A large flood would otherwise have used the old limit of 40 on gauges'
  polygons and dropped flash-flood areas without a word.
- Each run still replaces what is stored, and nothing older than 24 hours is served or kept.

### What real polygons look like

Eight live polygons from India, fetched with the real key and drawn through the production decoder:

- The response is a KML string in a JSON envelope: an XML declaration, a namespace, a `Placemark`
  named `InundationPolygon`, `altitudeMode` on each polygon, and `MultiGeometry` with up to five
  polygons and holes. The decoder read all of it without change.
- Shapes run along a grid of about 0.05 to 0.1 degrees and meet a curved coast, from 7 to 172 vertices,
  all valid in PostGIS. Every vertex survives storage (the eight stored had the counts Google sent), the
  map source's `tolerance` is low and outlines use mitre joins, so they keep their corners.
- **Highly likely does not sit on likely.** On the event measured, none of the highly likely area
  lies inside the likely one: likely has holes where highly likely is. They look nested, as in the
  owner's screenshot, but do not stack, so each keeps its own colour.
- `eventPolygonId` is documented as the union of the two. Live, on both events checked, it is exactly
  the highly likely polygon. It adds nothing either way and is not read.

Real polygons are not committed. `riverseed polygons` draws them from a local manifest with the
test-data banner, which now also comes on for a seeded area with no seeded gauge.

### Flood Hub's language

| Google | Flood Hub's word | Drawn |
|---|---|---|
| `EXTREME` | Extreme | Dark red dot, `!!!` |
| `SEVERE` | Danger | Red-orange dot, `!!` |
| `ABOVE_NORMAL` | Warning | Orange dot, `!` |
| `UNKNOWN` | No data | Grey dot, `?` |
| `NO_FLOODING` | Normal | Green dot |
| `HIGH` inundation | Very high | Dark blue |
| `MEDIUM` inundation | High | Mid blue |
| `LOW` inundation | Moderate | Pale blue |

- Google's thresholds are named warning, danger and extreme danger, which supports the first three.
  "Warning" here is a forecast, and each panel says it is not an official warning.
- `InundationLevel` means probability or depth according to `inundationMapType`, so that is now read.
  A depth map says so instead of using probability words. The panel gives Google's own word beside
  Flood Hub's.
- Gauges are round with a light ring, as Flood Hub's. Colour is never alone: raised and no-data gauges
  carry a mark, and the trend arrow stays. Our own river gauges stay small, blue when calm and outlined
  in black, so they are not taken for forecasts.
- A flash flood's severity (`likely`, `highly_likely`) was being looked up as a gauge severity, so
  its panel showed no badge. A panel now reads likelihood by kind.

### A calm country

Basins keep a thin outline and a 3.5% tint when calm, none and a dashed outline with no forecast, and
only tint for a warning or above. "No forecast" keeps its row in the key and says nothing is known
there. The map says "No flash-flood areas for Sri Lanka right now" and how old that reading is.

### Extended coverage

Lower-confidence gauges (300 on 2 November) are collected unless
`LANKAWEATHER_GOOGLE_FLOOD_UNVERIFIED=false`. They add about 6 requests a run, about 13 in all, all
with thresholds. They are off on the map behind a toggle showing the count, drawn hollow, and labelled.

Google says to use them with caution, so they count for nothing elsewhere:

- River risk already read verified gauges only.
- A place's summary lists only verified gauges and raises no alert from the others. There are 30 for
  each verified one, so they would also have crowded the verified ones out.
- **A flood area of a lower-confidence gauge counted as verified evidence for its basin**. It now
  counts as unverified, and `AreasAt` leaves it out.
- A basin says `verifiedState`, drawn while the toggle is off.

### Live rain gauges

**Direct HMIS was tried and not used.** `hmis.gov.lk/api/v1/getAllRainfallDevices/24` answered one
unauthenticated request with 327 devices. But the site sends a bearer token, the page calling it sits
behind the login in the site's own routes, the portal has customer requests, bank details and payment
screens, its API Center needs a login and it publishes no terms. An endpoint that fails to check a
token is not a public one, so three requests were made and no more. The owner should raise this
with the department, and ask for access.

So the source is the copy the owner approved, `shakee93/sl-water-levels` `data/hmis/rain-live.json`,
which is 60 KB and read every 30 minutes with its ETag. That repository has no licence, so none of its
code is copied. Credit "HMIS, via Sri Lanka Water Levels (waterlevels.lk)" and the institution.

- **Only 24-hour totals.** Each point is a rolling sum, so no 1-hour or 3-hour figure is derived.
- **Stale is not dry.** A gauge with no reading for 3 hours has `rain24hMm: null`; `lastRain24hMm` keeps
  the figure, and its panel gives it with its date. It is hollow on the map, with no number.
- **Bands:** light under 10 mm, moderate 10 to 50, heavy 50 to 100, very heavy 100 or more, the bands
  waterlevels.lk uses. Colours are one to a band, so the key is exact.
- **Its own layer, on by default.** The `rainfall` layer is a raster with a timeline, off by default.
  The owner wants gauges at once, and a layer that needed the satellite image switched on would not
  do that. A saved set of layers is offered this one once.
- **Zoom.** Drops, a shape no other layer uses. At country zoom only 10 mm or more shows; light rain
  from zoom 8; dry and out-of-date gauges from zoom 9, small and quiet. A wet day can still show over 130.
- **Duplicates.** 25 of the 62 river stations have a device within 500 m, and 8 of 163 Met stations.
  With both layers on, an older point within 500 m of a current gauge steps aside.
- History is kept 7 days, and a gauge the feed has dropped goes after 30.

The overview's rain section names the wettest live gauges once at least 20 report, and otherwise keeps
the river gauges' rain. Totals are "last 24 hours", never "today".

### Colombo's canals

`waterlevels.lk/api/canals`, allowed by its `robots.txt` and documented on its `/data` page, every 30
minutes. It is replaced whole on each read, with the sea level and predicted tide. Credit its `credit`
field and "Data © CUrW / SLLDC (pub.curwsl.org)", say it is context and not a flood warning, and keep
its state words.

**Positions.** The API has none. Neither `/data`, nor the first page of `pub.curwsl.org`, nor anything
else we found gives any, so none are guessed: a list, reached from River levels and from any place within
20 km of Colombo's centre. That distance says where the list is useful, not where any gauge is.

### River gauges

`/api/levels` lists 62 stations, and `river_stations` holds 62: 40 from the Irrigation Department and 22
from this feed. The only difference in names is one alias, Ratnapura for `rathnapura`, handled since
ADR 0013. There is no gap, so no 23-against-22 difference, and nothing changed.

## Consequences

- The map on a calm day looks like Flood Hub's. Basins are now hard to see when calm, on purpose.
- `Warning` now has two meanings on the map: Google's forecast and official warnings. The official
  banner and the word "forecast" on these panels keep them apart.
- A heavy day shows many drops at country zoom. The rule is one number, 10 mm, if it needs raising.
- Rain gauges and canals both depend on one person's unofficial project. If it stops, both age and
  turn stale, and the overview falls back to river gauges' rain.
- Whether the copy of HMIS may be republished is the department's to say, and is unconfirmed.
- The running worker and API must be restarted to pick all of this up.
