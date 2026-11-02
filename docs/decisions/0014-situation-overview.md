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

## Consequences

- Each load makes one request, kept for a minute by the API and the browser. It reads about a
  dozen services, cached where they already cache, in about a second.
- A number here is as current as its source, which is why every line shows its age and source.
- Anything new added to the map needs a section here to be seen in this view.
