# Architecture decision records

Each significant decision is recorded as a numbered file,
`NNNN-short-title.md`, with the sections Context, Decision and
Consequences. A decision is changed by writing a new record that
supersedes the old one, not by editing an accepted record.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-go-modular-monolith.md) | Go modular monolith with PostgreSQL and PostGIS | Accepted |
| [0002](0002-web-stack.md) | React, Vite, TanStack Query with a persisted cache, and MapLibre | Accepted |
| [0003](0003-repository-tooling.md) | Single repository with a pnpm workspace and a Makefile | Accepted |
| [0004](0004-river-levels.md) | River levels from the Irrigation Department gauge service | Accepted |
| [0005](0005-design-system.md) | shadcn/ui with a token-based theme, light and dark | Accepted |
| [0006](0006-admin-regions-and-search.md) | Administrative regions, vector tiles and place search | Accepted |
| [0007](0007-weather-grid-and-flood-forecast.md) | Weather grid, wind field and flood forecast | Accepted; flood searches superseded in part by 0015 |
| [0008](0008-weather-by-region.md) | Weather by region reads the regions table | Accepted |
| [0009](0009-web-map-and-location-panel.md) | One map of every layer, with a location panel | Accepted |
| [0010](0010-rivers-overflow-risk-and-flood-basins.md) | Rivers, overflow risk and flood basins | Accepted; flood drawing superseded in part by 0015 |
| [0011](0011-official-warnings-and-hazards.md) | Official warnings, landslide levels, earthquakes and cyclones | Accepted |
| [0012](0012-rainfall-stations-air-and-sea.md) | Rainfall, weather stations, air quality and sea conditions | Accepted |
| [0013](0013-reservoirs-and-extra-gauges.md) | Reservoirs and extra river gauges through waterlevels.lk | Accepted; its `data` branch rule superseded in part by 0015 |
| [0014](0014-situation-overview.md) | Situation overview | Accepted |
| [0015](0015-flood-hub-language-and-all-live-gauges.md) | Flood Hub's language, Extended coverage, live rain gauges and Colombo's canals | Accepted |
