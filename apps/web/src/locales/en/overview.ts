/**
 * Messages of the overview feature. English is the source: it fixes the keys.
 *
 * The overview's words come from the API in English (internal/overview). The
 * English page shows them as sent; in Sinhala and Tamil each known sentence is
 * matched and written again from these messages (features/overview/prose.ts),
 * so the English here mirrors the server's sentences. A sentence that is not
 * known stays in English.
 */
export const overview = {
  button: {
    label: "Sri Lanka now, the overview",
    text: "Now",
  },
  panel: {
    title: "Sri Lanka now",
    composed: "Composed {age}",
    subtitle: "What the official and open feeds say",
    close: "Close Sri Lanka now",
    loading: "Loading the overview",
    failed:
      "The overview could not be loaded, so nothing can be said from here. Check with the Disaster Management Centre, {link}, or call 117 in an emergency.",
    footer:
      "Official warnings are authoritative. Everything marked {estimate} is ours. In an emergency call 117.",
    timeNotKnown: "time not known",
    lineOutOfDate: "out of date",
    notRead: "Not read",
    officialPdf: "Official PDF",
    newTab: "(opens in a new tab)",
    more: "{count} more",
    updated: "Updated {age}",
  },
  group: {
    official: "Official",
    water: "Rivers and reservoirs",
    hazards: "Cyclones and earthquakes",
    now: "Right now",
  },
  title: {
    landslides: "Landslide warnings",
    warnings: "Warnings in force",
    gauges: "River gauges",
    rivers: "Rivers at risk",
    reservoirs: "Reservoirs",
    reservoirsMoving: "Reservoirs spilling or releasing",
    cyclones: "Cyclones",
    cyclonesOcean: "Cyclones in the North Indian Ocean",
    earthquakes: "Recent significant earthquakes",
    weather: "Weather by district",
    rain: "Rain at gauges, last 24 hours",
    rainLive: "Wettest gauges, last 24 hours",
    air: "Air quality",
    sea: "Highest waves",
  },
  count: {
    inForce: "In force",
    reporting: "Reporting",
    gauges: "Gauges",
    current: "Current",
    reservoirs: "Reservoirs",
  },
  /** The sentence at the top, one part at a time. */
  status: {
    landslides: {
      one: "Landslide warnings shown for {count} DS division, up to {level}.",
      other: "Landslide warnings shown for {count} DS divisions, up to {level}.",
    },
    warnings: {
      one: "{count} official warning or advisory in force.",
      other: "{count} official warnings and advisories in force.",
    },
    cyclone: "1 cyclone active {direction} of Sri Lanka.",
    cycloneNear: "1 cyclone active near Sri Lanka.",
    cyclones: "{count} cyclones active in the North Indian Ocean.",
    gauges: {
      one: "{count} river gauge at or above alert level.",
      other: "{count} river gauges at or above alert level.",
    },
    rivers: {
      one: "Estimate: {count} river stretch at {high} or {severe}.",
      other: "Estimate: {count} river stretches at {high} or {severe}.",
    },
    quake: "M{magnitude} earthquake reported in the last day.",
    reservoirs: {
      one: "{count} reservoir spilling.",
      other: "{count} reservoirs spilling.",
    },
    noGauge: "No river gauge above its alert level.",
    stale: "Out of date: {feeds}.",
    unread: "Could not read: {feeds}.",
    nothing: "Nothing to report from the feeds below. Follow official warnings.",
  },
  /** The feeds as the status names them. */
  feed: {
    landslides: "landslide levels",
    warnings: "official warnings",
    gauges: "river gauges",
    rivers: "river estimates",
    reservoirs: "reservoirs",
    cyclones: "cyclones",
    earthquakes: "earthquakes",
    weather: "weather",
    rain: "gauge rainfall",
    air: "air quality",
    sea: "sea conditions",
  },
  summary: {
    unavailable: "Could not be read just now, so nothing can be said from here.",
    landslidesNone: "Last report {date}; none currently active as far as we know.",
    landslidesActive: "Report of {date}, valid until {until}.",
    landslidesNotYet: "Landslide warnings have not been collected yet.",
    warnings: {
      one: "{count} warning or advisory in force, in the issuers' words.",
      other: "{count} warnings and advisories in force, in the issuers' words.",
    },
    warningsStale:
      "We have not been able to check recently, so we cannot say which warnings are in force.",
    warningsNone: "None in force as far as we know.",
    gaugesStale:
      "Only {current} of {total} gauges have a current reading, so we cannot say how the rivers stand.",
    gaugesNone:
      "None of the {current} gauges reporting is at its alert level. {missing} not reporting.",
    gaugesRaised: {
      one: "{count} gauge at or above the alert level, of {current} reporting.",
      other: "{count} gauges at or above the alert level, of {current} reporting.",
    },
    rivers: {
      one: "Estimate: {count} stretch of {total} at {watch} or above. Nothing measures the {unknown} not known.",
      other:
        "Estimate: {count} stretches of {total} at {watch} or above. Nothing measures the {unknown} not known.",
    },
    reservoirsStale: "No current readings, so we cannot say which are spilling.",
    reservoirsNone: "None of the {current} with a current reading is spilling or releasing.",
    reservoirs:
      "{spilling} spilling and {releasing} releasing, of {current} with a current reading.",
    cyclones: { one: "{count} system active.", other: "{count} systems active." },
    cyclonesNone: "No active cyclone in the North Indian Ocean",
    cyclonesNotYet: "Cyclones have not been collected yet.",
    cyclonesStale: "We have not been able to check GDACS recently, so this may be out of date.",
    cyclonesUnknown:
      "We have not been able to check GDACS recently, so we cannot say whether a cyclone is active. Check with the Department of Meteorology.",
    quakes: {
      one: "{count} line in the last {days} days, of {listed} listed in {window}.",
      other: "{count} lines in the last {days} days, of {listed} listed in {window}.",
    },
    quakesStale: "We have not been able to check USGS recently, so this may be out of date.",
    quakesNone: "None that stand out in the last {days} days, of {listed} listed in {window}.",
    weather: "Hottest, coolest, wettest and windiest of the districts.",
    weatherStale: "These model values are out of date.",
    rainStale: "No gauge has enough current readings to give a total.",
    rainNone: "No rain recorded at the {count} gauges with a total.",
    rain: "The wettest of {count} gauges with a total.",
    rainLive: "The wettest of {count} gauges reporting.",
    rainLiveNone: "No rain recorded at the {count} gauges reporting.",
    air: "The worst value on the model's grid.",
    sea: "The highest value on the model's grid near the coast.",
  },
  note: {
    checkDmc: "Check with the Disaster Management Centre.",
    checkDmcMeteo: "Check with the Disaster Management Centre and the Department of Meteorology.",
    checkMeteo: "Check with the Department of Meteorology.",
    bulletins:
      "Bulletins reach us through nuuuwan/lk_dmc, usually within hours of being issued. The PDF on dmc.gov.lk is the official version.",
    landslideLimit:
      "NBRO can lower or end a level before a report's period runs out. We learn of that only from its next report, hours later, so a level shown here may already have been withdrawn. Check with the DMC.",
    gauges:
      "Measured at gauges, against the Irrigation Department's own levels. Most stretches of river have no gauge.",
    rivers:
      "Estimate from gauge readings, Google flood forecasts, forecast rain and reported reservoir spills. Thresholds are the Irrigation Department's. Follow official warnings from the Irrigation Department, the Department of Meteorology and the Disaster Management Centre.",
    reservoirs: "Not an official warning. In an emergency call 117.",
    cyclones:
      "GDACS's alert rates the impact on the countries it lists. It is not a warning for Sri Lanka.",
    earthquakes: "Lanka Weather generates no tsunami warnings. Follow the DMC.",
    weather: "Model values averaged over each district, not measurements.",
    rain: "Measured at river gauges, which cluster in the wet zone.",
    rainLive:
      "Measured at NBRO, Department of Meteorology, Irrigation Department and ACCIMT gauges. Not an official warning.",
    air: "A model estimate about 40 km across. It does not measure the air at any place.",
    sea: "Model values, not observations. Follow the Department of Meteorology's sea-area warnings.",
  },
  badge: {
    /** "Warning: Amber"; the severity and colour are `terms`. */
    withColour: "{severity}: {colour}",
    gdacs: "GDACS {colour}",
  },
  item: {
    hottest: "Hottest: {name}",
    coolest: "Coolest: {name}",
    wettest: "Wettest today: {name}",
    windiest: "Strongest wind: {name}",
    worst: "Worst: {place}",
    waves: "{height} m, {state}",
    quake: "M{magnitude} earthquake",
    bulletinType: {
      landslide: "Landslide report",
      water_level: "Water level and flood",
      weather: "Weather",
      situation: "Situation report",
    },
    mayBeWithdrawn: "{type}. It may already have been withdrawn; check with the DMC.",
    landslide: "{district} District. NBRO: {action}.",
    gauge: "{basin}, at {reading}",
    cusecs: "{flow} cusecs",
    cusecsDownstream: "{flow} cusecs downstream, which can include power-station flow",
    noFigure: "no figure given",
    full: "{percent}% full",
    cyclone: "{distance} km {direction} of Sri Lanka",
    winds: "winds to {speed} km/h",
    quakeDistance: "{distance} km from Sri Lanka",
    tsunami: "USGS tsunami flag, not a warning for Sri Lanka",
    rainToday: "{amount} mm forecast today",
    windGusts: "{speed} km/h, gusts to {gusts} km/h",
    aqi: "US AQI {aqi}, {band}",
    coast: "About {distance} km {direction} of Sri Lanka's coast",
  },
  source: {
    landslides: "NBRO via the DMC",
    risk: "Estimate by lanka-weather",
    reservoirs: "Irrigation Department and Mahaweli Authority via waterlevels.lk",
    rainGauges: "HMIS, via Sri Lanka Water Levels (waterlevels.lk)",
    air: "CAMS via Open-Meteo",
    sea: "Open-Meteo marine model",
  },
}
