/**
 * The About page: where the data comes from, how it is handled, credits and
 * the disclaimer. English is the source: it fixes the keys.
 */
export const about = {
  link: "Sources, credits and disclaimer",
  title: "About Lanka Weather",
  subtitle: "Where the data comes from, how it is used, and what it cannot tell you",
  close: "Close",
  contents: "On this page",
  reviewed: "Last reviewed {date}",

  notOfficial: {
    heading: "Not an official warning service",
    body: "Lanka Weather is an independent, open-source project by Prabhava Labs. It is not run by, affiliated with or endorsed by the Department of Meteorology, the Disaster Management Centre (DMC), NBRO, the Irrigation Department or any other government agency.",
    official:
      "Official warnings from the Department of Meteorology and the DMC always take precedence over anything shown here.",
    estimates:
      "Anything marked “Estimate” is worked out by this software from public data. It may be wrong, late or missing.",
    stale:
      "Sources can stop updating or go offline without notice. When a source is late, the app says so; it never shows silence as safety.",
    rely: "Do not rely on Lanka Weather as your only source for decisions about safety.",
    emergency: "In an emergency, call 117 (Disaster Management Centre).",
  },

  how: {
    heading: "How it works",
    collect:
      "Our server collects each source on a schedule that matches how often it updates, from every 10 minutes to once a month, and keeps a copy. Apart from the base map, your browser asks only our server; it never sends requests to the sources themselves.",
    record:
      "Every reading keeps the name of its source, the time the source gives it and the time we fetched it. The map shows how old the data is and marks layers that are out of date.",
    official:
      "Official bulletins are shown as their issuers wrote them. Where the text of a PDF can be read cleanly, whole English sentences are quoted unchanged; otherwise only the title is shown, with a link to the official PDF on dmc.gov.lk. The PDFs are linked, never changed.",
    noAi: "No artificial intelligence or language model reads, summarises or rewrites the reports. Everything is read by fixed rules that can be checked in the source code.",
    estimates:
      "River risk, flood risk and the region and point alerts are estimates: they combine gauge levels, official thresholds, reservoir releases, forecasts and rainfall by rules described in the project’s documentation. They are labelled “Estimate” wherever they appear.",
    languages:
      "The Sinhala and Tamil translations are written for this app and are being reviewed. Where a sentence from a source is not known, it stays in its original language.",
  },

  sources: {
    heading: "Data sources",
    intro:
      "Lanka Weather is built on the work of the agencies and projects below. We thank each of them. Their data stays theirs: each keeps its own licence and terms, and we follow them. Where a source has no published licence, we use only what it makes public, credit it, and link back to it.",
    via: "Reaches us through",
    terms: "Terms",
    full: "The full list, with how and how often each source is read, is in the project’s data-source catalogue.",
    catalogue: "Data-source catalogue",
    groups: {
      official: "Sri Lankan government agencies",
      global: "International and open data",
      maps: "Maps and geography",
      community: "Community projects we build on",
    },
    hydroshedsCitation: "Citation",
  },

  /** What each source gives the map, and on what terms; by the ids in features/about/sources.ts. */
  source: {
    dmc: {
      use: "Landslide reports, river water level and flood warnings, weather advisories and situation reports. The official PDFs are linked, never stored.",
      terms: "Public government information; no licence published. Credited and linked.",
    },
    nbro: {
      use: "Landslide early-warning levels for each divisional secretariat division, as published in the DMC’s reports.",
      terms: "Public government information; no licence published. Credited and linked.",
    },
    met: {
      use: "Observations at weather stations, rain at about 60 stations, and the forecasts the DMC republishes.",
      terms: "Public government information; no licence published. Credited and linked.",
    },
    irrigation: {
      use: "River levels and flood thresholds at gauging stations, river basins, maps of past floods, and reservoir figures.",
      terms: "Public government information; no licence published. Credited and linked.",
    },
    hmis: {
      use: "Rolling 24-hour rain at about 275 gauges run by NBRO, the Department of Meteorology, the Irrigation Department and ACCIMT.",
      terms: "Public government information; no licence published. Credited and linked.",
    },
    mahaweli: {
      use: "Storage, spill and release figures for reservoirs in the Mahaweli system.",
      terms: "Public government information; no licence published. Credited and linked.",
    },
    rivernet: {
      use: "River levels at gauges that the Irrigation Department’s feed does not include.",
      terms: "Public information; no licence published. Credited and linked.",
    },
    curw: {
      use: "Water levels at Colombo canal gauges, sea level at the Kelani mouth and the predicted tide. Shown as context, not as flood warnings.",
      terms: "Public information; no licence published. Credited and linked.",
    },
    survey: {
      use: "Boundaries of provinces, districts, DS and GN divisions, and population from the 2024 census.",
      terms: "CC BY-IGO (boundaries, via OCHA HDX). Census figures are public government data.",
    },
    google: {
      use: "Flood forecasts at river gauges, flash floods and the areas they may cover.",
      terms: "Google’s terms of service. Forecast results are kept no longer than 24 hours.",
    },
    noaa: {
      use: "Wind forecasts for the animated wind layer.",
      terms: "Public domain (U.S. government work).",
    },
    nasa: {
      use: "Satellite estimates of rain, every half hour.",
      terms: "NASA open data; free to use with attribution.",
    },
    openmeteo: {
      use: "The weather grid and forecasts, air quality and sea conditions.",
      terms: "CC BY 4.0. Free tier for non-commercial use.",
    },
    cams: {
      use: "The air-quality model behind the air-quality layer. Generated using Copernicus Atmosphere Monitoring Service information.",
      terms: "Copernicus licence; free to use with attribution.",
    },
    gdacs: {
      use: "Tropical cyclones in the North Indian Ocean: position, track and forecast cone.",
      terms: "Free to use with attribution.",
    },
    usgs: {
      use: "Earthquakes of the last 30 days across the Indian Ocean.",
      terms: "Public domain (U.S. government work).",
    },
    osm: {
      use: "The base map, place names for search, and the outlines of reservoirs.",
      terms: "Open Database License (ODbL). © OpenStreetMap contributors.",
    },
    openfreemap: {
      use: "The base map’s tiles and styles, served directly to your browser.",
      terms: "Free to use; OpenStreetMap data under the ODbL.",
    },
    hydrosheds: {
      use: "The river network and its sub-basins, behind the rivers layer and the river risk estimates.",
      terms: "Free to use with attribution and the citation below.",
    },
    naturalearth: {
      use: "Sri Lanka’s coastline, used to fit the weather grid to the land.",
      terms: "Public domain.",
    },
    nuuuwan: {
      use: "Open copies and parsers of DMC reports, NBRO landslide levels, Department of Meteorology reports, administrative boundaries, census figures and river names. Much of the official data reaches us through these repositories.",
      terms: "Code under the MIT licence; the data stays with its original owners.",
    },
    waterlevels: {
      use: "Reservoir figures, extra river gauges, Colombo canal levels and HMIS rain gauges, gathered from the agencies above. Built and run by @shakeeb_x.",
      terms: "Free to read; no licence published. Credited and linked.",
    },
  },

  privacy: {
    heading: "Privacy",
    noAccount:
      "There are no accounts. Lanka Weather does not ask for your name, phone number or e-mail address.",
    device:
      "Your settings, home place and the layers you choose are kept in this browser only. Clearing the site’s data removes them.",
    location:
      "If you use “my location”, your position is used in the browser to move the map. When you open the weather for a place, its coordinates are sent to our server to look it up; our server’s logs record which page of the API was asked for, not your IP address or the place.",
    alerts:
      "If you turn on alerts, we keep what your browser’s push service needs to reach this device, the browser family, your language, the topics, districts and level you choose, and, for “near me”, a point rounded to about 1 km. Alerts sent are kept for 30 days. Turning alerts off deletes the subscription.",
    thirdParty:
      "The base map tiles come straight from OpenFreeMap, so it receives ordinary web requests from your browser. Nothing else on the page loads from another site. There is no advertising and no analytics.",
  },

  legal: {
    heading: "Disclaimer and terms",
    asIs: "Lanka Weather and its data are provided “as is”, without warranty of any kind, express or implied, including accuracy, completeness, timeliness or fitness for a particular purpose.",
    liability:
      "To the extent the law allows, Prabhava Labs and the project’s contributors are not liable for any loss or damage arising from the use of, or reliance on, this app or its data.",
    thirdParty:
      "Data from third parties remains theirs and is subject to their terms. Naming an agency or organisation here credits it as a source; it does not imply that it endorses or is responsible for this app.",
    boundaries:
      "Boundaries and names on the map are shown for reference only and do not express any position on legal or political status.",
    changes:
      "Sources, features and these terms may change. This page will say when they were last reviewed.",
  },

  contact: {
    heading: "Corrections and requests",
    body: "If something looks wrong, or you represent a source and want us to change how your data is used or credited, please open an issue on GitHub. We aim to deal with requests from data owners first.",
    issues: "Open an issue",
  },

  licence: {
    heading: "Open source",
    body: "The source code is public under the Apache License 2.0. The licence covers the code only, not the data from the sources above.",
    code: "Source code on GitHub",
  },
}
