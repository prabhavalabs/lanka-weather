# ADR 0014: Situation overview

- Status: Accepted
- Date: 2026-11-02

## Context

The map shows sixteen layers, each with its own panel. Someone opening it during a
storm has to know where to look. They need a first answer to "what is going on now?"
that cannot mislead: a single summary is exactly where stale data could pass as calm.

## Decision

`GET /v1/overview` composes the existing services, and a panel shows it first on load.

- **No language model.** `overview.Build` and `Sentence` are pure, fixed and tested.
  A sentence that fails at 2 a.m. in a flood must be one someone can read in the code and
  check. Nothing is summarised: official words are passed through unchanged.
- **Nothing is computed twice.** Each number comes from the service that already owns it,
  so the panel cannot disagree with the layer it opens. Two small additions: the worst air
  and highest wave on their grids.
- **Partial failure is the normal case.** A failing source empties its section, which says it
  could not be read, and the sentence names it. As with `/v1/summary`, the request only fails
  if the service itself does.

### The sentence

Most serious first: official levels and warnings, cyclones, gauges at alert, estimated high
water (labelled), a large earthquake in the last day, reservoirs spilling.

- **Never "safe" or "all clear".** A test bans them. The one calm thing it may say is "No river
  gauge above its alert level", which is a fact about gauges, not rivers: about 50 gauges serve
  142 rivers. The wording is deliberately not "no river above its alert level".
- **A stale feed is left out and named.** That sentence appears only while at least half the
  gauges are current. Stale reservoirs do not count as spilling, a stale cyclone feed does not count,
  and each is listed at the end as "Out of date: …". A feed that failed is "Could not read: …".
- **Calm words in a section need a fresh feed too.** "None in force as far as we know" appears
  only when the warnings feed is current; otherwise it says it cannot say.

### What counts as worth a line

- **Earthquakes.** Of the last 7 days: M6 and above anywhere on the list; M5 and above within
  1,500 km, which takes in the Andaman arc and the near Indian Ocean; anything within 300 km;
  and anything USGS flags for a tsunami, as its indicator and not a warning. Most of a normal month
  is M4.5 to M5 at great distance, which the layer still lists. Three lines at most.
- **Reservoirs.** All spilling, then all releasing, on current readings only, largest flow first.
  A spill under the river estimate's 100 cusecs line (ADR 0010) carries no warning tone.
- **Rivers.** Named rivers at watch or above, worst first, up to five. Unnamed streams count in the
  tally but do not take a line.
- **Now.** Extremes only. These are model values, except gauge rain, and say so.

### Interface

- **First on load.** On a wide screen it fills the detail column. On a phone it is the existing
  sheet at its peek height, so the map stays usable above it. A link to a particular panel wins.
- **Going back.** A line opens its layer and panel with the overview left behind it, so closing
  the panel comes back to the list. Clicking the map or choosing a search result dismisses it, as
  that means the map is being used. It stays reachable from the header.
- **Dismissal lasts until the next load.** This is safety information: yesterday's dismissal should
  not hide today's cyclone.

### The map with everything together

The layers were built apart and first seen together here, on 2 November, at desktop and
phone width in both themes. What changed:

- **Default layers, safety first:** official warnings, landslide warnings, cyclones and
  earthquakes; then rivers, river levels, flood forecast and reservoirs; then boundaries.
  **Wind is now off by default.** Its animation runs over everything, its timeline takes the
  bottom of a phone, and it answers a different question. People who saved their layers keep
  what they chose. Temperature, rainfall, stations, air, sea and past floods stay one switch away.
- **Calm points stay small.** At country scale calm gauges and forecasts shrink, and the
  rising halo waits for zoom 5.5, so the rivers can be read. A raised gauge stays large at any zoom,
  so nothing that needs attention shrinks with them.
- **One map key.** Ten legend cards covered half the map once the sidebar closed. The hazard and
  river keys share one "Map key" button, whose column scrolls and is remembered. A layer that
  colours the whole map keeps its own legend card, as its colours mean nothing without it.
- **The layer list.** A layer's notes open from a chevron, and when it is switched on, rather
  than always, which had pushed the second group off the sidebar. Official warnings lead their
  group, and past floods come last in theirs.
- **Attribution.** It starts folded behind its button at every width, not only on phones, because
  with every source named it ran under the cards. Opened, it stays in the clear part of the map.
- **Toolbar.** The search box gives way, and the level selector moves to the layer's notes when
  the toolbar has less than 42 rem.
- **Opening view.** With the pan limit widened for Sumatra (55 to 110° E, 20° S to 30° N), the map
  still opens on Sri Lanka, now fitted into what the cards leave clear and below the banner.
- **Sri Lanka's own name** comes off the basemap: three lines of script across the middle of the
  island, on top of the data. Neighbours keep theirs.
- **Phone sheet.** It opens at 14.5 rem for the overview: the sentence in view and the map above it.
- **Temperature.** The 21 to 24 °C band was sky blue, and read as water when zoomed in. The cool end
  now runs violet to orchid, then green-teal from 24 °C, in both themes.

Checked but not changed: labels and points stay over fills; past floods are hatched violet under
everything current and stay distinct in both themes, including beside the violet temperature band,
which is a flat field and not a hatch.

## Consequences

- Each load makes one request, kept for a minute by the API and the browser. It reads about a
  dozen services, cached where they already cache, in about a second.
- A number here is as current as its source, which is why every line shows its age and source.
- Anything new added to the map needs a section here to be seen in this view.
