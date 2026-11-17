# ADR 0020: Timely official warnings, notices that say where, and a list of notices

- Status: Accepted
- Date: 2026-11-17

Extends ADR 0011 (official warnings), ADR 0018 (push alerts) and ADR 0019 (flood-risk estimate).

## Context

The owner reported that alerts came late, and that notices did not say clearly where they applied.
The production database from 3 to 17 November 2026 showed:

- **Official bulletins reached people 80 to 170 minutes after the DMC issued them.** They came only
  through `nuuuwan/lk_dmc`, whose indexes rebuild about every two hours, and we read those hourly.
  "Flood Warning for Kuda Ganga, Gin and Attanagalu Oya" (DMC 22:00, 11 November) was sent at 00:46.
- **NBRO landslide levels reached people 4 to 8 hours after NBRO issued them** (16:00): their
  structured feed rebuilds once a day at about 20:40, and we read it every two hours.
- **Heavy-rain notices came 30 to 50 minutes after the reading**: the copy refreshes every 30
  minutes and we read it every 30 minutes, out of phase.
- **One device had silently lost two topics.** The app sent its stored list of topics to the server
  on every start, so `heavy_rain` and `flood_risk`, added on the server by migrations 00014 and
  00015, were removed again the next time the app opened. The extreme rain in Matara and the "very
  high" flood-risk estimates on 12 November were never sent to that device.
- **Notices did not say where.** A flood warning whose text names rivers but no districts was sent
  to everyone as "Official warning · Sri Lanka"; gathered notices said "Kandy and 4 more"; a
  cyclone was "663 km away" with no direction; the earthquake template's `{place}` was overwritten
  by the district placeholder and printed "Sri Lanka" instead of USGS's location.
- Tapping a notice opened the map, and nothing listed the notices a device had received.

## Decision

### Timeliness

- A new collector, `dmc-site`, reads the first page of each of the DMC's four report lists on
  dmc.gov.lk every 10 minutes (four small HTML pages). Each row is stored under the identifier
  `nuuuwan/lk_dmc` gives it (time and title slug, an MD5 when long), so both sources fill one row;
  every one of the 40 rows read on 17 November matched production's existing identifier.
- For a new warning, advisory, withdrawal or landslide report, `dmc-site` reads the PDF with
  poppler's `pdftotext` (installed in the API image) and runs the existing text rules on it, so
  districts, colour and validity are known minutes after posting. Text that reads badly is left to
  the index. Without `pdftotext`, text waits for the index as before.
- The index (`dmc-reports`) is read every 30 minutes, the landslide levels every 20 minutes and the
  HMIS rain gauges every 10 minutes (one conditional request when unchanged).
- NBRO's landslide report names its districts in its table of locations. A new reading,
  `LandslideIssued`, raises one notice per district named in the report in force ("Landslide
  warning · Kandy, Nuwara Eliya, Badulla +2"), under the same tag as the level notices that follow
  when the levels arrive. Its state is kept apart from the levels' (kind `landslide_issued`) so
  neither ends the other's subjects.

### Notices say what, where, when and who

- A flood warning with no districts in its text is placed in the districts holding at least 5% of
  each basin its title names ("Gin River Basin" is Galle and Matara), from the Irrigation
  Department's basins (`official_basins`) intersected with the districts. The notice says
  "Districts in the basin: …". One with no basin we know stays national and says the bulletin names
  no districts.
- Titles name places: a short list that fits ("Galle, Matara +3"); bodies name every district.
  Bulletin titles say their kind ("Flood warning", "Weather advisory"). River notices name the
  river and gauge, the reading and the threshold. Landslide notices name up to four divisions.
  Cyclones give a direction from the coast and the district of the nearest coast. Every notice says
  when the issuer issued or measured it, when known.
- All of this is written in English, Sinhala and Tamil from the same placeholders; a test checks
  every kind renders completely in every language.

### Notices can be read again

- The payload carries the notice's `id` and `kind`, and its link adds `notice=<id>`. A gathered
  notice's tag carries its id, so it no longer replaces the previous gathered notice in the tray.
- `POST /api/push/inbox` returns the notices sent to a subscription in the last 30 days (as long
  as events are kept), written in the requested language, with the issuer, whether it is official,
  every district and each event on its own. The auth secret proves ownership, so it is a POST.
- The app keeps a copy of each notice the service worker receives, merges it with the server's list
  and shows both in the bell's "Received" tab until the person clears them. `?notice=` opens a
  notice's details over the map.
- The app now remembers which topics it has shown the person, and turns on any new default topic it
  has not shown; migration 00016 gives `heavy_rain` and `flood_risk` back to the subscriptions that
  lost them.

## Consequences

- dmc.gov.lk is read every 10 minutes (four pages, plus a PDF when a warning appears). If the site
  changes its table, `dmc-site` fails loudly and the index still delivers, later.
- Google's flash-flood forecasts still arrive when Google's API returns them (28 to 183 minutes after
  their stated issue time in November 2026); we read them every 10 minutes and cannot do better.
- A district named by NBRO gets two notices: the early one, then its level. Both share a tag, so the
  second replaces the first in the tray.
