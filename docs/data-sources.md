# Data-source catalogue

Every source the project uses, or plans to use, is listed here with its
access method, cadence, licence and attribution. A source must be listed
before a connector for it is merged.

Status values: **planned** (not yet implemented), **active**,
**evaluating** (terms or access unconfirmed), **rejected**.

## Hydrology and floods

| Source | Data | Access | Cadence | Licence / terms | Status |
|---|---|---|---|---|---|
| Irrigation Department, Hydrology and Disaster Management Division (`irrigation-gauges`) | River water levels and gauge rainfall at 40 gauging stations, with alert, minor-flood and major-flood thresholds and positions | Public ArcGIS FeatureServer used by the department's gauge dashboard (`services3.arcgis.com/J7ZFXmR8rSmQ3FGf/arcgis/rest/services/gauges_2_view/FeatureServer/0/query`); polled every 15 min (ADR 0004) | About hourly per station | Government data, no licence stated; attribute the department | active |
| `nuuuwan/lk_irrigation` | Scraped copy of the above, with station, river and basin metadata and history since 2025 | Raw JSON on GitHub (`data/alert_data.json`, `data/static/*.json`) | 30 min | Code MIT; data as upstream | evaluating (history before our first fetch) |
| `nuuuwan/lk_dmc_vis` | DMC three-hourly water-level bulletins parsed to JSON; station thresholds and coordinates | Raw JSON on GitHub (`data/static/gauging_stations.json`) | 15 min poll | Code MIT; data as upstream | planned (thresholds) |
| Google Flood Forecasting API | Gauge forecasts, flood status, flash floods, inundation polygons | REST, `floodforecasting.googleapis.com`; API key and acceptance of Google's terms | Daily model runs | Google terms; do not store gauge search results for more than one day | evaluating |
| Reservoir and tank levels (Irrigation Department, Mahaweli Authority) | Storage and spill status of major reservoirs | No machine-readable feed found yet | Daily bulletins | Unknown | evaluating |

## Weather

| Source | Data | Access | Cadence | Licence / terms | Status |
|---|---|---|---|---|---|
| NOAA GFS (NOAA Open Data on AWS) | 10 m wind (U/V), precipitation, temperature forecasts | GRIB2 by byte range using the `.idx` inventory | 4 runs a day, 3-hourly steps to 120 h | US government work, public domain | planned |
| Department of Meteorology | Daily rainfall and temperature per station; forecasts; warnings | Website and bulletins; no public API found | Daily, plus ad hoc warnings | Government data; attribute the department | evaluating |
| `nuuuwan/weather_lk` | Parsed Met Department daily station data | Raw JSON (`data/latest_flat.json`). Do not clone: the repository is about 50 GB | 6 h | Code MIT; data as upstream | planned |
| `nuuuwan/lk_weather_3h` | Met Department three-hourly observations at 24 stations | Raw JSON (`data/alert_data.json`) | 15 min | Code MIT; data as upstream | planned |
| NASA GIBS | IMERG precipitation, infrared cloud imagery | WMTS raster tiles, loaded by the browser | 30 min to 3 h; IMERG early run lags 4 to 7 h | NASA open data; attribution requested | planned |
| OpenWeather / Open-Meteo | Current conditions and point forecasts | REST with API key (OpenWeather) or keyless (Open-Meteo) | Hourly | Provider terms; check commercial-use limits | evaluating |

## Hazards and warnings

| Source | Data | Access | Cadence | Licence / terms | Status |
|---|---|---|---|---|---|
| NBRO landslide early warnings (via DMC) | Warning level per divisional secretariat division | `nuuuwan/lk_dmc_landslides` (`data/latest_flat.json`); upstream is published as PDF notices | 3 h | Government data; code MIT | planned |
| Disaster Management Centre | Situation reports, flood and river warnings, weather advisories | `nuuuwan/lk_dmc` data branches (PDF, text and JSON) | 2 h | Government data; code MIT | planned |
| GDACS | Tropical cyclone tracks and alerts for all basins | RSS / GeoJSON API | Event-driven | Free use with attribution | planned |
| Joint Typhoon Warning Center, IMD RSMC New Delhi | North Indian Ocean cyclone advisories | Text and graphic bulletins | Event-driven | Public advisories | evaluating |
| USGS earthquakes (and `nuuuwan/lk_tsunamis`) | Earthquakes near Sri Lanka, tsunami relevance | GeoJSON feed | 1 to 15 min | Public domain | planned |

## Geography

| Source | Data | Access | Licence / terms | Status |
|---|---|---|---|---|
| `nuuuwan/lk_admin_regions` | Provinces, districts, DS and GN divisions; GeoJSON and TopoJSON at several precisions | Raw files on GitHub | Code MIT | planned |
| `prabhavalabs/lanka-data-layer` | Admin hierarchy, postal codes, census, electoral data; foundry builds SQLite and PMTiles | npm packages `@lanka-data-layer/*`; foundry run offline | MIT; OSM-derived parts ODbL | planned |
| `nuuuwan/lk_rivers` (HydroRIVERS) | River network lines | Raw GeoJSON | HydroSHEDS licence; attribution required | planned |
| OpenFreeMap | Vector basemap | Style URLs, keyless | OpenStreetMap data, ODbL; attribution required | planned |

## Public reports and social signals

| Source | Data | Access | Licence / terms | Status |
|---|---|---|---|---|
| `nuuuwan/lk_news` | Sri Lankan news articles in Sinhala, Tamil and English since 2021 | Raw files on GitHub | Article copyright remains with publishers | evaluating |
| Social platforms (X, Facebook, Reddit) | Eyewitness reports | Official APIs only | Platform terms; personal data must not be stored | evaluating |

## Rules for collectors

- Fetch on a schedule and store; never call a third-party source on
  behalf of an individual web request.
- Send an identifying `User-Agent` with a contact address.
- Respect rate limits and `Cache-Control`; back off on errors.
- Keep the raw payload of each fetch for a limited period so connectors
  can be replayed in tests and parsing errors can be investigated.
- Record source name, source time and fetch time on every record.
- raw.githubusercontent.com is a file host, not an API: it is cached for
  about five minutes and rate limited. Use it for metadata and backfill,
  and poll upstream sources directly for warnings.
