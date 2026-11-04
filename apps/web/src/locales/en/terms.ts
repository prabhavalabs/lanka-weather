/**
 * Official terminology shared by several features: the agencies' own words
 * for levels, colours and divisions (docs/i18n-glossary.md gives the sources
 * of the Sinhala and Tamil forms). Change these only against the glossary.
 */
export const terms = {
  /** Irrigation Department river gauge levels. */
  riverStatus: {
    normal: "Normal",
    alert: "Alert",
    minor_flood: "Minor flood",
    major_flood: "Major flood",
    unknown: "No thresholds",
  },
  /** The thresholds themselves, as gauge tables name them. */
  riverLevel: {
    alert: "Alert level",
    minor_flood: "Minor flood level",
    major_flood: "Major flood level",
  },
  trend: {
    rising: "Rising",
    falling: "Falling",
    steady: "Steady",
    unknown: "Trend unknown",
  },
  /** NBRO landslide early-warning levels, as its reports print them. */
  landslideLevel: {
    l1: "Level 1 (Yellow)",
    l2: "Level 2 (Amber)",
    l3: "Level 3 (Red)",
  },
  /** NBRO's words for what to do at each level. */
  landslideAction: {
    l1: "Watch",
    l2: "Alert",
    l3: "Evacuate",
  },
  /** Department of Meteorology colour-coded notices. */
  colour: {
    red: "Red",
    amber: "Amber",
    yellow: "Yellow",
    orange: "Orange",
    green: "Green",
  },
  bulletinSeverity: {
    warning: "Warning",
    advisory: "Advisory",
    withdrawal: "Withdrawal",
    routine: "Report",
  },
  bulletinType: {
    landslide: "Landslide",
    water_level: "Water level",
    weather: "Weather",
    situation: "Situation",
  },
  /** Weather systems as the Department of Meteorology names them. */
  weatherSystem: {
    lowPressure: "Low-pressure area",
    depression: "Depression",
    deepDepression: "Deep depression",
    cyclonicStorm: "Cyclonic storm",
    severeCyclonicStorm: "Severe cyclonic storm",
    cyclone: "Cyclone",
  },
  reservoirStatus: {
    spilling: "Spilling",
    releasing: "Releasing",
    holding: "Holding",
    nodata: "No data",
  },
  /** Administrative divisions, one and many. */
  adminLevel: {
    province: "Province",
    district: "District",
    ds: "DS division",
    gn: "GN division",
    dsLong: "Divisional Secretariat division",
    gnLong: "Grama Niladhari division",
  },
  adminCount: {
    province: { one: "{count} province", other: "{count} provinces" },
    district: { one: "{count} district", other: "{count} districts" },
    ds: { one: "{count} DS division", other: "{count} DS divisions" },
    gn: { one: "{count} GN division", other: "{count} GN divisions" },
  },
  /** Official names of the agencies; acronyms stay as they are. */
  agency: {
    dmc: "Disaster Management Centre",
    nbro: "National Building Research Organisation",
    meteo: "Department of Meteorology",
    irrigation: "Irrigation Department",
    earlyWarning: "Natural Hazards Early Warning Centre",
  },
  /** Derived figures always read as estimates; issuers' warnings as official. */
  estimate: "Estimate",
  estimated: "Estimated",
  official: "Official",
  officialWarning: "Official warning",
  notOfficial: "Not an official warning.",
  emergency: "In an emergency call 117.",
}
