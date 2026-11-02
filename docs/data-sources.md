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
| Google Flood Forecasting API (`google-flood-forecasting`) | Latest flood status, trend and forecast window at gauges in Sri Lanka (`regionCode` LK; 10 quality-verified HYBAS gauges), gauge thresholds, and the flash floods, significant events and inundation polygons affecting Sri Lanka | REST, `floodforecasting.googleapis.com`: `floodStatus:searchLatestFloodStatusByArea`, `gaugeModels:batchGet`, `flashFloods:search`, `significantEvents:search`, `serializedPolygons/{id}`. Key `LANKAWEATHER_GOOGLE_API_KEY`, sent only in the `X-Goog-Api-Key` header. Polled hourly, about five requests a run (ADR 0007) | Daily model runs; flash floods a few times a day | Google's terms: gauge search results are kept no more than one day (rows older than 24 h are neither served nor stored). Attribute "Google Flood Hub (Flood Forecasting API)" | active |
| Reservoir and tank levels (Irrigation Department, Mahaweli Authority) | Storage and spill status of major reservoirs | No machine-readable feed found yet | Daily bulletins | Unknown | evaluating |

## Weather

| Source | Data | Access | Cadence | Licence / terms | Status |
|---|---|---|---|---|---|
| NOAA GFS (`noaa-gfs`, NOAA Open Data on AWS) | 10 m wind (U/V) on the 0.25° grid, cropped to 70–92° E, 2° S–18° N, every 3 h to 72 h, for the animated wind layer | GRIB2 by HTTP byte range using the `.idx` inventory (`noaa-gfs-bdp-pds.s3.amazonaws.com`); about 1.9 MB per forecast hour, 47 MB per run. Checked every 30 min (ADR 0007) | 4 runs a day | US government work, public domain; credit "NOAA GFS" | active |
| Department of Meteorology | Daily rainfall and temperature per station; forecasts; warnings | Website and bulletins; no public API found. `www.meteo.gov.lk` and `meteo.gov.lk` timed out on 2026-11-02, also when retried; its forecasts and warnings are shown as the DMC republishes them (ADR 0011) | Daily, plus ad hoc warnings | Government data; attribute the department | evaluating |
| `nuuuwan/weather_lk` | Parsed Met Department daily station data | Raw JSON (`data/latest_flat.json`). Do not clone: the repository is about 50 GB | 6 h | Code MIT; data as upstream | planned |
| `nuuuwan/lk_weather_3h` | Met Department three-hourly observations at 24 stations | Raw JSON (`data/alert_data.json`) | 15 min | Code MIT; data as upstream | planned |
| NASA GIBS | IMERG precipitation, infrared cloud imagery | WMTS raster tiles, loaded by the browser | 30 min to 3 h; IMERG early run lags 4 to 7 h | NASA open data; attribution requested | planned |
| Open-Meteo (`open-meteo`) | Current conditions and an hourly forecast (9 days) at the 214 nodes of the 0.2° weather grid; the daily summary is derived from the hours | Keyless REST, `api.open-meteo.com/v1/forecast`, 50 coordinates per request, `best_match` models. Current conditions hourly (10 variables), forecast every 3 h (12 variables): about 7,190 of the free tier's 10,000 weighted calls a day (ADR 0007) | Model runs every 1 to 6 h | Free tier for non-commercial use only: under 10,000 calls a day, 5,000 an hour, 600 a minute, per IP address. Data under CC BY 4.0; show "Weather data by Open-Meteo.com" with a link next to the data | active |
| OpenWeather | Current conditions and point forecasts | REST with API key | Hourly | Provider terms; check commercial-use limits | evaluating |
| Google Weather API | Current conditions and forecasts | REST, Google Cloud key | Hourly | Google terms | evaluating: the API is disabled on the project's key (403). An optional source behind configuration, off until the API is enabled, may be added |

## Hazards and warnings

| Source | Data | Access | Cadence | Licence / terms | Status |
|---|---|---|---|---|---|
| NBRO landslide early warnings via DMC (`nuuuwan-lk-dmc-landslides`) | NBRO's level per divisional secretariat division, every report since 2019 | `nuuuwan/lk_dmc_landslides` (`data/all.json`, about 550 kB) from raw.githubusercontent.com, every 2 h; upstream is published as PDF notices, which reach this file about 4.5 h after issue (ADR 0011) | About daily while a risk lasts | Government data; code MIT. Attribute NBRO and the DMC | active |
| Disaster Management Centre (`dmc-reports`) | Landslide reports, water level and flood warnings, weather reports and advisories, situation reports: index metadata, official PDF links, and clean English excerpts where the text allows | `nuuuwan/lk_dmc` data branches: `data/<dataset>/docs_last100.tsv`, `summary.json` and each new document's `doc.txt`, from raw.githubusercontent.com, hourly (ADR 0011). PDFs are linked, never stored | Several a day; indexes rebuilt about hourly | Government data; code MIT. Attribute the DMC and link the PDF | active |
| GDACS (`gdacs`) | Current tropical cyclones in the North Indian Ocean: position, track, forecast, wind buffers, cone and alert | GeoJSON API, `www.gdacs.org/gdacsapi/api/events/geteventlist/SEARCH` and one `polygons/getgeometry` per current event, hourly (ADR 0011) | Event-driven | Free use with attribution; credit "Global Disaster Alert and Coordination System (GDACS), gdacs.org" | active |
| Joint Typhoon Warning Center, IMD RSMC New Delhi | North Indian Ocean cyclone advisories. GDACS's track draws on JTWC | Text and graphic bulletins | Event-driven | Public advisories | evaluating |
| USGS earthquakes (`usgs-earthquakes`) | Earthquakes of the last 30 days: M4.5+ across the Indian Ocean, M2.5+ near Sri Lanka, with USGS's tsunami flag | FDSN event service, `earthquake.usgs.gov/fdsnws/event/1/query`, two queries every 15 min (ADR 0011) | Minutes | US government work, public domain; credit USGS | active |
| `nuuuwan/lk_tsunamis` | Tsunami history for Sri Lanka | Raw files on GitHub | Occasional | Code MIT; data as upstream | planned |

## Geography

| Source | Data | Access | Licence / terms | Status |
|---|---|---|---|---|
| `nuuuwan/lk_admin_regions` (`nuuuwan-lk-admin-regions`) | Boundaries of the 9 provinces, 25 districts, 340 DS and 14,019 GN divisions, with DCS codes, English, Sinhala and Tamil names, area and centre | `data/geo/topojson/e4_medium/{provinces,districts,dsds,gnds}.topojson` (about 43 MB) from raw.githubusercontent.com at the commit the GitHub API reports as the latest for that directory; checked weekly, fetched only when the commit changes (ADR 0006) | Geometry from the Survey Department of Sri Lanka via OCHA HDX (COD-AB, CC BY-IGO); code MIT. Attribute "Survey Department of Sri Lanka and Department of Census and Statistics via OCHA HDX and lk_admin_regions" | active |
| `nuuuwan/lk_census_2024` (`nuuuwan-lk-census-2024`) | 2024 Census of Population and Housing: population per province, district, DS and GN division | `data/Person-Sex/data.tsv` (0.6 MB), loaded with the boundaries | Government data from the Department of Census and Statistics; code MIT. Attribute the department | active |
| OpenStreetMap via the Overpass API (`osm-places`) | About 7,000 named places (`place` = city, town, suburb, village, quarter, neighbourhood, hamlet, locality, isolated_dwelling) with Sinhala and Tamil names where tagged, for place search | One Overpass query a week (bounding box 79.5–81.95°E, 5.85–9.9°N, about 1.6 MB), `overpass-api.de` then `overpass.kumi.systems`; the instances ask for under 10,000 queries and 1 GB a day | ODbL. Attribute "© OpenStreetMap contributors"; a derived database made public must be offered under the ODbL | active |
| GeoNames LK dump | About 17,800 populated places, alternate names with language codes | `download.geonames.org/export/dump/LK.zip` | CC BY 4.0 | evaluating (more villages than OpenStreetMap, but no town and village classes and few populations; see ADR 0006) |
| `prabhavalabs/lanka-data-layer` | Admin hierarchy, postal codes, census, electoral data; foundry builds SQLite and PMTiles | npm packages `@lanka-data-layer/*`; foundry run offline | MIT; OSM-derived parts ODbL | planned |
| HydroRIVERS, through `nuuuwan/lk_rivers` (`hydrosheds-hydrorivers`) | 5,699 river segments with flow topology (`NEXT_DOWN`), upstream area and sub-basin, and `lk_rivers`' table of river names | `data/sri_lanka_rivers.geojson` (3.5 MB) and `data/static/riv_id_to_name.json` from raw.githubusercontent.com at the commit the GitHub API reports for each; checked weekly, fetched only when one changes (ADR 0010) | HydroSHEDS licence: free with attribution. Attribute "HydroRIVERS (Lehner & Grill 2013, HydroSHEDS)"; `lk_rivers`' code is MIT | active |
| HydroBASINS level 12 (`hydrosheds-hydrobasins`) | 501 sub-basins of Sri Lanka with `NEXT_DOWN`, for the flood layer's finer boundaries | `data.hydrosheds.org/file/hydrobasins/standard/hybas_as_lev12_v1c.zip` (80 MB shapefile for Asia), streamed to a temporary file with a limit and clipped; checked monthly by its size, since the host's `Last-Modified` moves with no change to the file (ADR 0010) | HydroSHEDS licence: free with attribution. Attribute "HydroBASINS (Lehner & Grill 2013, HydroSHEDS)" | active |
| Irrigation Department: `river_basins` (`irrigation-river-basins`) | The official river basins: 103 named, plus coastal areas, 204 after cleaning | Public ArcGIS FeatureServer of the organisation `J7ZFXmR8rSmQ3FGf`, as for gauges; checked weekly by `lastEditDate` (ADR 0010) | No licence or terms published, restrictive or otherwise. Government data; attribute the department | active |
| Irrigation Department: `Srilanka_River`, `hydrostations` (`irrigation-river-basins`) | Names only: 1,030 named stream polygons (1985 survey), and the tributary each station stands on | As above, loaded with the basins | As above | active |
| Irrigation Department: `rivers` | 3,379 polylines named with the numbers 1 to 3,379, no topology | As above | As above | rejected (HydroRIVERS has names and topology; ADR 0010) |
| OpenFreeMap | Vector basemap | Style URLs, keyless | OpenStreetMap data, ODbL; attribution required | planned |
| Natural Earth (1:10m Admin 0) | Sri Lanka's coastline, simplified to 0.01° and embedded in `internal/grid` to mask the weather grid | Downloaded once; not fetched at run time | Public domain | active |

## Past events

| Source | Data | Access | Licence / terms | Status |
|---|---|---|---|---|
| Irrigation Department: `Flood_Map`, `Flood__Nov2025`, `Puttalam_Flood__Detwah_30Nov2025` (`irrigation-past-floods`) | 25 mapped extents of past floods, 1989 to 2025, with event name and year. Past events, not forecasts | Public ArcGIS FeatureServers of the organisation `J7ZFXmR8rSmQ3FGf`, in EPSG:5234 or 3857, queried in EPSG:4326. Layers found by their `Flooding_area` value, not by id. Checked monthly by `lastEditDate` (ADR 0010) | No licence or terms published, restrictive or otherwise. Government data; attribute the department | active |

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
