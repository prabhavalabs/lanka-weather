# ADR 0022: Warnings-first default layers, a focus mask and a home place

- Status: Accepted
- Date: 2026-11-25

Supersedes the default layers in ADR 0014 ("The map with everything
together").

## Context

Early feedback from people trying the map:

- The first view was busy. Ten layers were on at the first visit:
  every warning, then rivers, river levels, reservoirs, rain gauges
  and boundaries. A newcomer had to work out what mattered.
- A warning picked from "Sri Lanka now" did not stand out. A landslide
  division was outlined in the signal orange over an orange or red
  fill, among two dozen like it, and the map only panned to it.
- The house button showed the whole island. People read it as "home"
  and expected to set their own town and come back to it.

## Decision

- **Default layers are the warnings.** The first visit shows official
  warnings, landslide warnings, Google's flood forecast, cyclones and
  earthquakes. Everything else is a switch away in the layer list, and
  "Sri Lanka now" switches on the layer behind any line it opens.
  The defaults live in `apps/web/src/features/layers/defaults.ts`.
- **Browsers are moved on once.** A browser whose saved layers are
  exactly the old defaults gets the new ones; a set someone chose is
  kept. `lanka-weather:layers-defaults` records that this has run.
- **A focus mask for selected areas.** A selected landslide division
  or flood area sits in a hole cut out of a grey veil over the map,
  outlined in the text colour on a halo, and an opened division is
  fitted into view. The warnings panel greys out the districts other
  than the ones it is about: one chosen district, the districts a kind
  of warning names, or every highlighted district. A warning that
  names no district (one for the whole island) leaves the map as it is.
  The veil lies over areas and rivers but under labels and gauges, and
  is not a hit layer.
- **A home place.** Any place or region's weather panel can make it
  the home place. The map's house button goes back there and reopens
  its weather; "show all of Sri Lanka" has a button of its own. The
  home place is kept in the browser only (`lanka-weather:home`).

## Consequences

- People who never changed the old defaults lose rivers, gauges,
  reservoirs and boundaries from their first view and turn them on
  again if they want them. Official warnings are never hidden by the
  change.
- The district veil covers land only: it is drawn from the district
  boundaries, so the sea around the island stays clear.
- The home place is not sent to the server. Using it for push alerts
  would need its own decision and consent wording.
