/**
 * Who the data on the map comes from, for the About page: the owner, a link,
 * and the copies or services it reaches us through. Names stay as their owners
 * write them; what each is used for and on what terms is in the `about`
 * messages, by id. docs/data-sources.md is the full record, with access
 * methods and cadence; keep the two in step.
 */

export type SourceGroup = "official" | "global" | "maps" | "community"

export type SourceId =
  | "dmc"
  | "nbro"
  | "met"
  | "irrigation"
  | "hmis"
  | "mahaweli"
  | "rivernet"
  | "curw"
  | "survey"
  | "google"
  | "noaa"
  | "nasa"
  | "openmeteo"
  | "cams"
  | "gdacs"
  | "usgs"
  | "osm"
  | "openfreemap"
  | "hydrosheds"
  | "naturalearth"
  | "nuuuwan"
  | "waterlevels"

export interface Link {
  name: string
  url: string
}

export interface Source {
  id: SourceId
  group: SourceGroup
  /** The owner, as they name themselves. */
  owner: string
  url: string
  /** The copies, parsers or services the data reaches us through, if not directly. */
  via?: Link[]
}

const nuuuwan = (repo: string): Link => ({
  name: `nuuuwan/${repo}`,
  url: `https://github.com/nuuuwan/${repo}`,
})
const waterlevels: Link = { name: "Sri Lanka Water Levels", url: "https://waterlevels.lk" }

export const sourceGroups: readonly SourceGroup[] = ["official", "global", "maps", "community"]

export const sources: readonly Source[] = [
  {
    id: "dmc",
    group: "official",
    owner: "Disaster Management Centre (DMC)",
    url: "https://www.dmc.gov.lk",
    via: [{ name: "dmc.gov.lk", url: "https://www.dmc.gov.lk" }, nuuuwan("lk_dmc")],
  },
  {
    id: "nbro",
    group: "official",
    owner: "National Building Research Organisation (NBRO)",
    url: "https://www.nbro.gov.lk",
    via: [{ name: "DMC reports", url: "https://www.dmc.gov.lk" }, nuuuwan("lk_dmc_landslides")],
  },
  {
    id: "met",
    group: "official",
    owner: "Department of Meteorology",
    url: "https://www.meteo.gov.lk",
    via: [nuuuwan("weather_lk"), nuuuwan("lk_weather_3h")],
  },
  {
    id: "irrigation",
    group: "official",
    owner: "Irrigation Department",
    url: "https://www.irrigation.gov.lk",
    via: [
      {
        name: "Irrigation Department ArcGIS services",
        url: "https://services3.arcgis.com/J7ZFXmR8rSmQ3FGf/arcgis/rest/services",
      },
      waterlevels,
    ],
  },
  {
    id: "hmis",
    group: "official",
    owner: "Hydro-Meteorological Information System (HMIS)",
    url: "https://hmis.gov.lk",
    via: [
      waterlevels,
      { name: "sl-water-levels", url: "https://github.com/shakee93/sl-water-levels" },
    ],
  },
  {
    id: "mahaweli",
    group: "official",
    owner: "Mahaweli Authority of Sri Lanka",
    url: "https://mahaweli.gov.lk",
    via: [waterlevels],
  },
  {
    id: "rivernet",
    group: "official",
    owner: "RiverNet (DMC / Leecom SCADA)",
    url: "https://rivernet.lk",
    via: [waterlevels],
  },
  {
    id: "curw",
    group: "official",
    owner: "CUrW and SLLDC",
    url: "https://www.curwsl.org",
    via: [waterlevels],
  },
  {
    id: "survey",
    group: "official",
    owner: "Survey Department and Department of Census and Statistics",
    url: "https://www.statistics.gov.lk",
    via: [
      { name: "OCHA HDX (COD-AB)", url: "https://data.humdata.org/dataset/cod-ab-lka" },
      nuuuwan("lk_admin_regions"),
      nuuuwan("lk_census_2024"),
    ],
  },
  {
    id: "google",
    group: "global",
    owner: "Google Flood Hub (Flood Forecasting API)",
    url: "https://sites.research.google/floods/",
  },
  {
    id: "noaa",
    group: "global",
    owner: "NOAA Global Forecast System (GFS)",
    url: "https://www.ncei.noaa.gov/products/weather-climate-models/global-forecast",
    via: [
      { name: "NOAA Open Data on AWS", url: "https://registry.opendata.aws/noaa-gfs-bdp-pds/" },
    ],
  },
  {
    id: "nasa",
    group: "global",
    owner: "NASA GPM IMERG",
    url: "https://gpm.nasa.gov/data/imerg",
    via: [
      {
        name: "NASA GIBS",
        url: "https://www.earthdata.nasa.gov/engage/open-data-services-software/earthdata-developer-portal/gibs-api",
      },
    ],
  },
  {
    id: "openmeteo",
    group: "global",
    owner: "Open-Meteo.com",
    url: "https://open-meteo.com",
  },
  {
    id: "cams",
    group: "global",
    owner: "Copernicus Atmosphere Monitoring Service (CAMS)",
    url: "https://atmosphere.copernicus.eu",
    via: [{ name: "Open-Meteo.com", url: "https://open-meteo.com" }],
  },
  {
    id: "gdacs",
    group: "global",
    owner: "Global Disaster Alert and Coordination System (GDACS)",
    url: "https://www.gdacs.org",
  },
  {
    id: "usgs",
    group: "global",
    owner: "U.S. Geological Survey (USGS)",
    url: "https://earthquake.usgs.gov",
  },
  {
    id: "osm",
    group: "maps",
    owner: "OpenStreetMap contributors",
    url: "https://www.openstreetmap.org/copyright",
  },
  {
    id: "openfreemap",
    group: "maps",
    owner: "OpenFreeMap",
    url: "https://openfreemap.org",
  },
  {
    id: "hydrosheds",
    group: "maps",
    owner: "HydroSHEDS (HydroRIVERS, HydroBASINS)",
    url: "https://www.hydrosheds.org",
    via: [nuuuwan("lk_rivers")],
  },
  {
    id: "naturalearth",
    group: "maps",
    owner: "Natural Earth",
    url: "https://www.naturalearthdata.com",
  },
  {
    id: "nuuuwan",
    group: "community",
    owner: "Nuwan Senaratna (nuuuwan)",
    url: "https://github.com/nuuuwan",
    via: [
      nuuuwan("lk_dmc"),
      nuuuwan("lk_dmc_landslides"),
      nuuuwan("weather_lk"),
      nuuuwan("lk_weather_3h"),
      nuuuwan("lk_admin_regions"),
      nuuuwan("lk_census_2024"),
      nuuuwan("lk_rivers"),
    ],
  },
  {
    id: "waterlevels",
    group: "community",
    owner: "Sri Lanka Water Levels (waterlevels.lk)",
    url: "https://waterlevels.lk",
  },
]

/** The citation HydroSHEDS asks of anyone using HydroRIVERS and HydroBASINS. */
export const hydroshedsCitation =
  "Lehner, B., Grill G. (2013): Global river hydrography and network routing: baseline data and new approaches to study the world’s large river systems. Hydrological Processes, 27(15): 2171–2186."

/** Where corrections and requests from source owners go. */
export const issuesUrl = "https://github.com/prabhavalabs/lanka-weather/issues"
export const repositoryUrl = "https://github.com/prabhavalabs/lanka-weather"
