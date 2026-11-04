# ADR 0009: One map of every layer, with a location panel

- Status: Accepted
- Date: 2026-11-02

## Context

ADRs 0004 to 0008 give the web application river levels, administrative
regions and place search, a weather grid, a wind field, Google flood
forecasts, weather by region and a location summary. The interface had
presentational shells for most of them (ADR 0005) and drew only river
levels. Bringing everything onto one map raises questions the earlier
records leave open:

- in what order layers are drawn, when each adds itself to the map on
  its own schedule and again after every theme change;
- how a weather field and region colours stop at the coast;
- how the animated wind sits under labels and gauges;
- how 14,019 GN divisions are coloured without loading 3.2 MB;
- what a click on the map opens, and how a selection is shared;
- how the selected place stays visible behind floating cards and a
  phone sheet;
- how the first download stays under the 1,600 kB chunk warning.

## Decision

### Layer order

- Invisible anchor layers (`slot:*`, `features/map/slots.ts`) are
  inserted into each basemap style as it loads. Data layers are added
  just under their slot's anchor, so the order holds whichever layer
  is added first. Bottom to top:
  1. **fills**: the weather field and region colours;
  2. a copy of the basemap's water layer, which clips the fills to the
     coast at every zoom;
  3. **lines**: region boundaries, over roads;
  4. **areas**: Google flood areas;
  5. **particles**: the animated wind;
  6. the basemap's labels;
  7. **points**: river gauges, flood gauges and their labels.
- Colours come from theme tokens through `lib/colour.ts`: continuous
  ramps for painted fields, stepped expressions for choropleths.

### Weather field

- The 214-node grid is interpolated bilinearly into an image about
  280 × 443 pixels, its rows spaced evenly in Web Mercator so it lines
  up with the map, and painted through a lookup table of the variable's
  ramp. Sea nodes without a value are left out and the others
  reweighted, so the field runs on smoothly to the coast, where the
  water mask hides the rest.
- It shows temperature, rain this hour, humidity, cloud cover or wind
  speed. While it is on, region colours are hidden; the boundaries stay.

### Wind

- TerraGuard's particle renderer is ported with its tests. Its 2D
  canvas is not laid over the map: a WebGL custom layer
  (`features/wind/layer.ts`) draws it as a screen-sized texture in the
  particles slot, with its own vertex array, uploading only frames that
  changed. Labels and gauges stay readable on top. The map repaints
  every frame while particles move; measured at 60 frames a second on
  a laptop.
- Particle colours are the wind legend's km/h steps, so the legend and
  the particles agree. With `prefers-reduced-motion`, still streamlines
  with arrowheads are drawn instead.
- A scrubber offers the run's 3-hourly steps from now to 72 hours, with
  play and pause; the next step is fetched ahead. Fields are kept out of
  IndexedDB.

### Regions

- Boundaries come from the region vector tiles. The chosen level is
  coloured with feature state keyed by the tiles' numeric ids, so new
  weather recolours the map without reloading tiles.
- GN divisions are coloured district by district: from zoom 10, where
  the tiles carry them, the districts on screen are read from the
  rendered GN features and `/v1/weather/regions?level=gn&parent=` is
  fetched for each (about 11 KB gzipped). A first view of a town loads
  one to five districts rather than the 270 KB national list, panning
  adds only new districts, and the answers stay out of IndexedDB.
- Below a level's first zoom a notice offers to zoom in.

### Selection, panels and the URL

- One click handler decides what a click opens: a river or flood gauge
  (with a few pixels' slack), else a region (its weather at its label
  point, with its aggregate), else the point itself.
- The selection is written in the URL: `?at=lat,lon&name=`,
  `?region=LK-21`, `?station=`, `?gauge=`. Opening a panel adds a
  history entry, so "back" closes it.
- The location panel reads `/v1/summary` for regions, nearby gauges,
  alerts and the official slot, and `/v1/weather/point` for 72 hours
  and 7 days. Derived alerts are labelled "Estimate" with their basis;
  the official slot is shown first and says Met Department and DMC
  warnings are coming until they are collected.
- The selected place is moved into the part of the map the cards leave
  clear (`features/map/reveal.ts`): right of the sidebar and left of
  the detail column on wide screens, between the top bar and the sheet
  on phones. On phones the detail panel is a non-modal bottom sheet
  with a peek height and a full height.

### Loading

- Search, the location panel and the wind renderer load on demand
  (`React.lazy`); the design kit already did. The search trigger and
  the region area picker live apart from the command menu, so the map
  does not load it.
- MapLibre GL has a chunk of its own: about 1.1 MB, cached across
  app releases. The app's chunk is about 580 kB (184 kB gzipped). The
  chunk warning limit is unchanged.
- Answers that say data is not collected yet (503 `no_data`,
  `not_available`, `no_wind_field`) are not retried at once; the layer
  asks again every minute and the interface says the data is still
  loading. Other failures raise one toast per query.

## Consequences

- A new data layer picks a slot and needs no knowledge of the others.
- The wind layer costs a repaint of the map per frame while it moves;
  switching it off, or reduced motion, stops that.
- GN colours appear district by district as the map moves; a view
  across many districts at zoom 10 makes several small requests.
- The URL carries no map position: a shared link centres on what it
  selects.
- Region clicks open the region rather than the exact point; with
  regions off, a click gives the weather at the point itself.
