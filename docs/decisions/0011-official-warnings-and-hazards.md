# ADR 0011: Official warnings, landslide levels, earthquakes and cyclones

- Status: Accepted
- Date: 2026-11-02

## Context

The hazard layers were planned, and the location panel's official slot said
warnings were coming. Four sources were checked on 2026-11-02:

- `nuuuwan/lk_dmc_landslides` publishes NBRO's level per DS division
  (`data/all.json`, 592 reports since 2019) with the same codes as our
  regions. Of 158 codes it has ever used, 156 are in our regions; LK-2304 and
  LK-2316 appear only in reports from 2025-12-01 to 2026-05-17.
- `nuuuwan/lk_dmc` publishes an index of the last 100 documents of four kinds
  of DMC report, and for each document `doc.json`, `doc.txt`, `blocks.json` and
  the PDF at `data/<dataset>/2020s/<year>/<doc_id>/`.
- USGS's FDSN service and the GDACS API both answer as documented.
- `www.meteo.gov.lk` and `meteo.gov.lk` both timed out, and again when retried.

Questions the sources leave open: how long a landslide report stays current,
whether bulletin text can be shown, what may be highlighted on the map, and how
to avoid implying safety when a feed goes quiet.

## Decision

### How long a landslide report stays current

- **Reports state their own period**, in English in the text: "Validity
  Period: From 16:00 hrs on 31.10.2026 To 16:00 hrs on 01.11.2026". It has been
  24 hours in every report read. It is read per report; if a report states
  none, 24 hours from issue is assumed and the panel says "assumed". Reports
  with no matching bulletin are assumed issued at 16:00, NBRO's usual time.
- **Reports are reissued while a risk lasts** and stop when it ends: the last
  report on 2026-11-02 was dated 31 October, the 38th update of a series begun
  on 3 October. So no recent report means none is in force, *if* the feed is
  alive.
- The landslide levels reach `lk_dmc_landslides` about 4.5 hours after issue,
  and the DMC index sooner. A 12-hour grace follows the end of a period, so
  a report does not vanish in the gap before its successor arrives.
- The status is decided by a pure function (`warnings.Assess`):

| Status | When | Map | Words |
|---|---|---|---|
| `active` | within the stated period | levels | Report of <date>, valid until <time> |
| `lapsed` | up to 12 h after it | levels | its validity ended; a newer report may not have reached us |
| `pending` | a newer bulletin exists without parsed levels | as before | levels have not reached us; read the official report |
| `none` | later, and the feed is fresh | nothing | Last report <date>; none currently active as far as we know |
| `unknown` | later, and the feed is stale | nothing | we cannot say whether a warning is active; check with the DMC |

- A feed is **fresh** only if both our last read of the levels and the
  upstream's own rebuild time (`summary.json`, whose `time_updated` is Sri
  Lanka time) are within the layer's 12-hour limit. A fetch can succeed against
  an upstream that has quietly stopped.
- `lk_dmc_landslides` commits only when something changes, so its commit date
  says nothing about whether it is alive; the DMC index's rebuild time does.
- A division left out of a report in force is "not named in the report. That
  does not mean it is safe." The words "safe", "clear" and "all clear" are
  never used of a place.

### Bulletin text

- **Text is usable only in part.** The Sinhala extracts with broken glyphs,
  the Tamil is in a legacy font that extracts as Latin gibberish ("kz;rhpT"),
  tables run together ("m15.00"), and some bulletins break every line into its
  own paragraph. Many bulletins also hold clean English paragraphs.
- **Only whole English sentences are kept, unchanged, in order.** A paragraph
  must be plain ASCII, start with a capital, end a sentence, have at least eight
  words with a fifth of them common English words, and contain no gibberish or
  run-together cells. Anything else is left out, so a kept excerpt can be
  incomplete, and the interface says so and links the PDF.
- Landslide reports keep no body: their lines lose the level columns they belong
  to. Water level reports and situation reports are tables, and fail the rules.
  They are shown as title and link.
- A bulletin's own colour ("COLOR: Red", "Colour – Amber") and period are read
  from its text. Legends ("Color: -Red Warning") do not match.
- The index's `lang` says "en" for Tamil editions; the title decides.

### What counts as a warning

- `warning` if the title says warning or the bulletin says Red; `advisory` if
  it says advisory, Amber or Yellow; `withdrawal`; otherwise `routine`.
- A warning or advisory is **in force** within its stated period, or for 24
  hours from issue if it states none. The header's count is these, one per
  warning (Tamil editions and numbered updates are the same warning), plus
  one if a landslide report is showing.
- A withdrawal does not end earlier warnings: it lists basins that cannot be
  read reliably. Earlier warnings run their period, with the withdrawal shown
  beside them. This errs towards showing too much for up to a day.
- The banner is raised by a level 2 or 3 division on the map and by bulletins
  of severity `warning`. It is not raised by level 1, advisories or routine
  reports. Dismissing it remembers what it announced, so anything new shows.

### Map highlights for official warnings

- Districts are outlined only for a landslide level in force (solid), or where
  the clean text of a warning or advisory **in force** names them by exact word
  (dashed, "Mentioned in bulletin").
- A word match says only that a bulletin names the place, and it can be a place
  on a coastal route rather than the area warned. So it is never worded more
  strongly. Routine bulletins highlight nothing on the map; they are listed in
  the location panel for the district they name.
- A bulletin without clean text highlights nothing, and the layer says an
  unmarked district is not a clear one.

### Landslide drawing

- The region tiles carry DS divisions only from zoom 8 (ADR 0006), and the map
  opens at about zoom 7, so filling divisions by feature state over the tiles
  would show nothing at country scale. The layer's API returns the named
  divisions' `geom_simple` instead (about 63 kB gzipped for the 71 divisions of
  a bad day). This also allows `fill-pattern`, which feature state cannot drive.
- A level is never told by colour alone: dots, lines or cross-hatch, an outline
  by level, and an "L2" label. The district highlight does use the tiles.

### Earthquakes

- Two queries every 15 minutes: M4.5+ for 15° S–25° N, 60–105° E, and M2.5+ for
  3–13° N, 75–87° E (USGS's catalogue is thin there, and returned nothing for
  2026). `includedeleted=true` reports deletions; anything else USGS stops
  listing in the 30-day window is removed too. An empty answer changes nothing.
- The tsunami flag is shown as USGS's indicator with the words that it is not a
  warning for Sri Lanka, and people are sent to the DMC. We generate no tsunami
  warnings.

### Cyclones

- Current events in 0–35° N, 40–100° E, hourly, with a geometry request per
  event. GDACS leaves `iscurrent` set for days (on 2026-11-02 for a storm whose
  last advisory was 2026-10-30), so an event with no advisory for 48 hours does
  not count.
- Stored and drawn: observed and forecast track, positions, the overall wind
  buffers and the uncertainty cone. The per-step wind polygons are dropped, as
  the overall buffers enclose them.
- The calm state, "No active cyclone in the North Indian Ocean", shows only
  with a fresh fetch; otherwise the layer says it cannot tell.
- GDACS's alert rates impact on the countries it lists; the panel says that it
  is not a warning for Sri Lanka.

### The map moves further

The pan limit widens from 70–92° E to 55–110° E, 20° S–30° N and the minimum zoom
goes from 5 to 3, because most earthquakes that matter for tsunamis are off
Sumatra, outside the old limit.

### Sources not used

- The Department of Meteorology's site timed out, so it stays `evaluating`. Its
  forecasts and warnings are shown as the DMC republishes them.
- JTWC and IMD RSMC New Delhi stay `evaluating`; GDACS's track draws on JTWC.
- `docs/data-sources.md` asks that warnings be polled from upstream rather than
  raw.githubusercontent.com. This takes the indexes for now because the DMC's
  site has no feed; the delay is stated in the interface.

## Consequences

- Warnings reach us late: an hour or more for bulletins, hours for landslide
  levels. The interface says so and links dmc.gov.lk, and `pending` covers the
  gap. A direct DMC collector would shorten it.
- Everything official depends on one person's parsing pipeline. If it stops, the
  layers say "out of date" and "cannot say" rather than going calm, but they
  stop being useful.
- Excerpts can omit content that was in tables or images. They are labelled as
  excerpts, and the PDF stays one click away.
- A division's history is kept indefinitely; 592 reports hold about 16,000 rows.
- A mention match can highlight a coastal district for a warning about deep sea
  areas. The label says what it is.
