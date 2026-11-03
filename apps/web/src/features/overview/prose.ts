/**
 * The overview's words in the reader's language. The API writes them in
 * English from fixed rules (internal/overview), so each sentence it can write
 * is known: here it is matched and written again from the catalogue, with its
 * numbers and names carried over. The English page shows the API's words as
 * sent. Anything not matched (an issuer's title, a USGS place, a river's
 * reason) stays in English and is marked `lang="en"`. Pure, so it is tested
 * without a browser.
 */
import { compassWords, formatDate, type CompassCode } from "@/lib/format"
import { translateIn, type Locale, type MessageKey } from "@/lib/i18n"

/** Words to show: in the language, or the API's English marked as such. */
export interface Prose {
  text: string
  lang?: "en"
}

/** Fixed words the API sends, and the message for each. */
const exact: Record<string, MessageKey> = {
  // Section titles.
  "Landslide warnings": "overview.title.landslides",
  "Warnings in force": "overview.title.warnings",
  "River gauges": "overview.title.gauges",
  "Rivers at risk": "overview.title.rivers",
  Reservoirs: "overview.title.reservoirs",
  "Reservoirs spilling or releasing": "overview.title.reservoirsMoving",
  Cyclones: "overview.title.cyclones",
  "Cyclones in the North Indian Ocean": "overview.title.cyclonesOcean",
  "Recent significant earthquakes": "overview.title.earthquakes",
  "Weather by district": "overview.title.weather",
  "Rain at gauges, last 24 hours": "overview.title.rain",
  "Wettest gauges, last 24 hours": "overview.title.rainLive",
  "Air quality": "overview.title.air",
  "Highest waves": "overview.title.sea",
  // Tallies and badges.
  "In force": "overview.count.inForce",
  Reporting: "overview.count.reporting",
  Gauges: "overview.count.gauges",
  Current: "overview.count.current",
  "Level 1 (Yellow)": "terms.landslideLevel.l1",
  "Level 2 (Amber)": "terms.landslideLevel.l2",
  "Level 3 (Red)": "terms.landslideLevel.l3",
  Alert: "terms.riverStatus.alert",
  "Minor flood": "terms.riverStatus.minor_flood",
  "Major flood": "terms.riverStatus.major_flood",
  Severe: "rivers.risk.level.severe",
  High: "rivers.risk.level.high",
  Watch: "rivers.risk.level.watch",
  Normal: "rivers.risk.level.normal",
  "Not known": "rivers.risk.level.unknown",
  Spilling: "terms.reservoirStatus.spilling",
  Releasing: "terms.reservoirStatus.releasing",
  Warning: "terms.bulletinSeverity.warning",
  Advisory: "terms.bulletinSeverity.advisory",
  Heavy: "rain.gauges.band.heavy",
  "Very heavy": "rain.gauges.band.veryHeavy",
  // Sources.
  "NBRO via the DMC": "overview.source.landslides",
  "Disaster Management Centre": "terms.agency.dmc",
  "Irrigation Department": "terms.agency.irrigation",
  "Department of Meteorology": "terms.agency.meteo",
  "Estimate by lanka-weather": "overview.source.risk",
  "Irrigation Department and Mahaweli Authority via waterlevels.lk": "overview.source.reservoirs",
  "HMIS, via Sri Lanka Water Levels (waterlevels.lk)": "overview.source.rainGauges",
  "CAMS via Open-Meteo": "overview.source.air",
  "Open-Meteo marine model": "overview.source.sea",
  // Bulletin types, as a warning's line gives them.
  "Landslide report": "overview.item.bulletinType.landslide",
  "Water level and flood": "overview.item.bulletinType.water_level",
  Weather: "overview.item.bulletinType.weather",
  "Situation report": "overview.item.bulletinType.situation",
  // Summaries.
  "Could not be read just now, so nothing can be said from here.": "overview.summary.unavailable",
  "Landslide warnings have not been collected yet.": "overview.summary.landslidesNotYet",
  "We have not been able to check recently, so we cannot say which warnings are in force.":
    "overview.summary.warningsStale",
  "None in force as far as we know.": "overview.summary.warningsNone",
  "No current readings, so we cannot say which are spilling.": "overview.summary.reservoirsStale",
  "No active cyclone in the North Indian Ocean": "overview.summary.cyclonesNone",
  "Cyclones have not been collected yet.": "overview.summary.cyclonesNotYet",
  "We have not been able to check GDACS recently, so this may be out of date.":
    "overview.summary.cyclonesStale",
  "We have not been able to check GDACS recently, so we cannot say whether a cyclone is active. Check with the Department of Meteorology.":
    "overview.summary.cyclonesUnknown",
  "We have not been able to check USGS recently, so this may be out of date.":
    "overview.summary.quakesStale",
  "Hottest, coolest, wettest and windiest of the districts.": "overview.summary.weather",
  "These model values are out of date.": "overview.summary.weatherStale",
  "No gauge has enough current readings to give a total.": "overview.summary.rainStale",
  "The worst value on the model's grid.": "overview.summary.air",
  "The highest value on the model's grid near the coast.": "overview.summary.sea",
  // Notes.
  "Check with the Disaster Management Centre.": "overview.note.checkDmc",
  "Check with the Disaster Management Centre and the Department of Meteorology.":
    "overview.note.checkDmcMeteo",
  "Check with the Department of Meteorology.": "overview.note.checkMeteo",
  "Bulletins reach us through nuuuwan/lk_dmc, usually within hours of being issued. The PDF on dmc.gov.lk is the official version.":
    "overview.note.bulletins",
  "NBRO can lower or end a level before a report's period runs out. We learn of that only from its next report, hours later, so a level shown here may already have been withdrawn. Check with the DMC.":
    "overview.note.landslideLimit",
  "Measured at gauges, against the Irrigation Department's own levels. Most stretches of river have no gauge.":
    "overview.note.gauges",
  "Estimate from gauge readings, Google flood forecasts, forecast rain and reported reservoir spills. Thresholds are the Irrigation Department's. Follow official warnings from the Irrigation Department, the Department of Meteorology and the Disaster Management Centre.":
    "overview.note.rivers",
  "Not an official warning. In an emergency call 117.": "overview.note.reservoirs",
  "GDACS's alert rates the impact on the countries it lists. It is not a warning for Sri Lanka.":
    "overview.note.cyclones",
  "Lanka Weather generates no tsunami warnings. Follow the DMC.": "overview.note.earthquakes",
  "Model values averaged over each district, not measurements.": "overview.note.weather",
  "Measured at river gauges, which cluster in the wet zone.": "overview.note.rain",
  "Measured at NBRO, Department of Meteorology, Irrigation Department and ACCIMT gauges. Not an official warning.":
    "overview.note.rainLive",
  "A model estimate about 40 km across. It does not measure the air at any place.":
    "overview.note.air",
  "Model values, not observations. Follow the Department of Meteorology's sea-area warnings.":
    "overview.note.sea",
  // Parts of the sentence at the top.
  "No river gauge above its alert level.": "overview.status.noGauge",
  "Nothing to report from the feeds below. Follow official warnings.": "overview.status.nothing",
  "1 cyclone active near of Sri Lanka.": "overview.status.cycloneNear",
  // Parts of a line.
  "no figure given": "overview.item.noFigure",
  "USGS tsunami flag, not a warning for Sri Lanka": "overview.item.tsunami",
}

/** The eight directions the API writes, as compass codes. */
const directions: Record<string, CompassCode> = {
  north: "N",
  "north-east": "NE",
  east: "E",
  "south-east": "SE",
  south: "S",
  "south-west": "SW",
  west: "W",
  "north-west": "NW",
}

const colours: Record<string, MessageKey> = {
  Red: "terms.colour.red",
  Amber: "terms.colour.amber",
  Yellow: "terms.colour.yellow",
  Orange: "terms.colour.orange",
  Green: "terms.colour.green",
}

const actions: Record<string, MessageKey> = {
  Watch: "terms.landslideAction.l1",
  Alert: "terms.landslideAction.l2",
  Evacuate: "terms.landslideAction.l3",
}

const aqiBands: Record<string, MessageKey> = {
  good: "environment.aqi.good",
  moderate: "environment.aqi.moderate",
  "unhealthy for sensitive groups": "environment.aqi.unhealthySensitive",
  unhealthy: "environment.aqi.unhealthy",
  "very unhealthy": "environment.aqi.veryUnhealthy",
  hazardous: "environment.aqi.hazardous",
}

const seaStates: Record<string, MessageKey> = {
  calm: "environment.seaState.calm",
  smooth: "environment.seaState.smooth",
  slight: "environment.seaState.slight",
  moderate: "environment.seaState.moderate",
  rough: "environment.seaState.rough",
  "very rough": "environment.seaState.veryRough",
  high: "environment.seaState.high",
  "very high": "environment.seaState.veryHigh",
}

/** The feeds as the status names them. */
const feeds: Record<string, MessageKey> = {
  "landslide levels": "overview.feed.landslides",
  "official warnings": "overview.feed.warnings",
  "river gauges": "overview.feed.gauges",
  "river estimates": "overview.feed.rivers",
  reservoirs: "overview.feed.reservoirs",
  cyclones: "overview.feed.cyclones",
  earthquakes: "overview.feed.earthquakes",
  weather: "overview.feed.weather",
  "gauge rainfall": "overview.feed.rain",
  "air quality": "overview.feed.air",
  "sea conditions": "overview.feed.sea",
}

const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]

/** "31 Oct 2026" as the API writes a date, as "YYYY-MM-DD"; null if it is not one. */
function plainDate(text: string): string | null {
  const match = /^(\d{1,2}) ([A-Z][a-z]{2}) (\d{4})$/.exec(text)
  const month = match ? months.indexOf(match[2]!) + 1 : 0
  if (!match || month === 0) return null
  return `${match[3]}-${String(month).padStart(2, "0")}-${match[1]!.padStart(2, "0")}`
}

/** A date, or a date and time ("2 Nov 2026 18:30", Sri Lanka time), in the language. */
function dateIn(locale: Locale, text: string): string | null {
  const timed = /^(.+) (\d{2}:\d{2})$/.exec(text)
  const date = plainDate(timed ? timed[1]! : text)
  if (!date) return null
  const day = formatDate(date, "short", locale)
  return timed ? translateIn(locale, "format.dayTime", { day, time: timed[2]! }) : day
}

const count = (text: string) => Number(text.replace(/,/g, ""))

/** A rule: a sentence the API writes, and how to write it again; null when a word is unknown. */
type Rule = [RegExp, (match: string[], locale: Locale) => string | null]

const word = (locale: Locale, table: Record<string, MessageKey>, text: string | undefined) => {
  const key = text === undefined ? undefined : table[text]
  return key ? translateIn(locale, key) : null
}

const direction = (locale: Locale, text: string | undefined) => {
  const code = text === undefined ? undefined : directions[text]
  return code ? compassWords(code, locale) : null
}

/** Fills a message whose words all came out; null if one did not. */
function filled(
  locale: Locale,
  key: MessageKey,
  values: Record<string, string | number | null>,
): string | null {
  if (Object.values(values).some((value) => value === null)) return null
  return translateIn(locale, key, values as Record<string, string | number>)
}

const rules: Rule[] = [
  // The sentence at the top.
  [
    /^Landslide warnings shown for (\d+) DS divisions?, up to (.+)\.$/,
    ([, n, level], l) =>
      filled(l, "overview.status.landslides", { count: count(n!), level: word(l, exact, level) }),
  ],
  [
    /^(\d+) official (?:warning or advisory|warnings and advisories) in force\.$/,
    ([, n], l) => translateIn(l, "overview.status.warnings", { count: count(n!) }),
  ],
  [
    /^1 cyclone active ([a-z-]+) of Sri Lanka\.$/,
    ([, where], l) => filled(l, "overview.status.cyclone", { direction: direction(l, where) }),
  ],
  [
    /^(\d+) cyclones active in the North Indian Ocean\.$/,
    ([, n], l) => translateIn(l, "overview.status.cyclones", { count: count(n!) }),
  ],
  [
    /^(\d+) river gauges? at or above alert level\.$/,
    ([, n], l) => translateIn(l, "overview.status.gauges", { count: count(n!) }),
  ],
  [
    /^Estimate: (\d+) river stretch(?:es)? at high or severe\.$/,
    ([, n], l) =>
      translateIn(l, "overview.status.rivers", {
        count: count(n!),
        high: translateIn(l, "rivers.risk.level.high"),
        severe: translateIn(l, "rivers.risk.level.severe"),
      }),
  ],
  [
    /^M(\d+(?:\.\d+)?) earthquake reported in the last day\.$/,
    ([, m], l) => translateIn(l, "overview.status.quake", { magnitude: m! }),
  ],
  [
    /^(\d+) reservoirs? spilling\.$/,
    ([, n], l) => translateIn(l, "overview.status.reservoirs", { count: count(n!) }),
  ],
  [
    /^(Out of date|Could not read): (.+)\.$/,
    ([, lead, list], l) => {
      const names = list!.split(", ").map((name) => word(l, feeds, name))
      if (names.some((name) => name === null)) return null
      const key = lead === "Out of date" ? "overview.status.stale" : "overview.status.unread"
      return translateIn(l, key, { feeds: names.join(", ") })
    },
  ],
  // Summaries.
  [
    /^Last report (.+); none currently active as far as we know\.$/,
    ([, date], l) => filled(l, "overview.summary.landslidesNone", { date: dateIn(l, date!) }),
  ],
  [
    /^Report of (.+), valid until (.+)\.$/,
    ([, date, until], l) =>
      filled(l, "overview.summary.landslidesActive", {
        date: dateIn(l, date!),
        until: dateIn(l, until!),
      }),
  ],
  [
    /^(\d+) (?:warning or advisory|warnings and advisories) in force, in the issuers' words\.$/,
    ([, n], l) => translateIn(l, "overview.summary.warnings", { count: count(n!) }),
  ],
  [
    /^Only (\d+) of (\d+) gauges have a current reading, so we cannot say how the rivers stand\.$/,
    ([, current, total], l) =>
      translateIn(l, "overview.summary.gaugesStale", { current: current!, total: total! }),
  ],
  [
    /^None of the (\d+) gauges reporting is at its alert level\. (\d+) not reporting\.$/,
    ([, current, missing], l) =>
      translateIn(l, "overview.summary.gaugesNone", { current: current!, missing: missing! }),
  ],
  [
    /^(\d+) gauges? at or above the alert level, of (\d+) reporting\.$/,
    ([, n, current], l) =>
      translateIn(l, "overview.summary.gaugesRaised", { count: count(n!), current: current! }),
  ],
  [
    /^Estimate: (\d+) stretch(?:es)? of (\d+) at watch or above\. Nothing measures the (\d+) not known\.$/,
    ([, n, total, unknown], l) =>
      translateIn(l, "overview.summary.rivers", {
        count: count(n!),
        total: total!,
        unknown: unknown!,
        watch: translateIn(l, "rivers.risk.level.watch"),
      }),
  ],
  [
    /^None of the (\d+) with a current reading is spilling or releasing\.$/,
    ([, current], l) => translateIn(l, "overview.summary.reservoirsNone", { current: current! }),
  ],
  [
    /^(\d+) spilling and (\d+) releasing, of (\d+) with a current reading\.$/,
    ([, spilling, releasing, current], l) =>
      translateIn(l, "overview.summary.reservoirs", {
        spilling: spilling!,
        releasing: releasing!,
        current: current!,
      }),
  ],
  [
    /^(\d+) systems? active\.$/,
    ([, n], l) => translateIn(l, "overview.summary.cyclones", { count: count(n!) }),
  ],
  [
    /^(\d+) lines? in the last (\d+) days, of (\d+) listed in (\d+)\.$/,
    ([, n, days, listed, window], l) =>
      translateIn(l, "overview.summary.quakes", {
        count: count(n!),
        days: days!,
        listed: listed!,
        window: window!,
      }),
  ],
  [
    /^None that stand out in the last (\d+) days, of (\d+) listed in (\d+)\.$/,
    ([, days, listed, window], l) =>
      translateIn(l, "overview.summary.quakesNone", {
        days: days!,
        listed: listed!,
        window: window!,
      }),
  ],
  [
    /^No rain recorded at the (\d+) gauges with a total\.$/,
    ([, n], l) => translateIn(l, "overview.summary.rainNone", { count: count(n!) }),
  ],
  [
    /^The wettest of (\d+) gauges with a total\.$/,
    ([, n], l) => translateIn(l, "overview.summary.rain", { count: count(n!) }),
  ],
  [
    /^The wettest of (\d+) gauges reporting\.$/,
    ([, n], l) => translateIn(l, "overview.summary.rainLive", { count: count(n!) }),
  ],
  [
    /^No rain recorded at the (\d+) gauges reporting\.$/,
    ([, n], l) => translateIn(l, "overview.summary.rainLiveNone", { count: count(n!) }),
  ],
  // Badges.
  [
    /^(Warning|Advisory): (Red|Amber|Yellow|Orange|Green)$/,
    ([, severity, colour], l) =>
      filled(l, "overview.badge.withColour", {
        severity: word(l, exact, severity),
        colour: word(l, colours, colour),
      }),
  ],
  [
    /^GDACS (Red|Orange|Green)$/,
    ([, colour], l) => filled(l, "overview.badge.gdacs", { colour: word(l, colours, colour) }),
  ],
  // Line titles.
  [
    /^(Hottest|Coolest|Wettest today|Strongest wind): (.+)$/,
    ([, which, name], l) => {
      const key = {
        Hottest: "overview.item.hottest",
        Coolest: "overview.item.coolest",
        "Wettest today": "overview.item.wettest",
        "Strongest wind": "overview.item.windiest",
      }[which!] as MessageKey
      return translateIn(l, key, { name: name! })
    },
  ],
  [/^Worst: (.+)$/, ([, place], l) => translateIn(l, "overview.item.worst", { place: place! })],
  [
    /^(\d+(?:\.\d+)?) m, ([a-z ]+)$/,
    ([, height, state], l) =>
      filled(l, "overview.item.waves", { height: height!, state: word(l, seaStates, state) }),
  ],
  [
    /^M(\d+(?:\.\d+)?) earthquake$/,
    ([, m], l) => translateIn(l, "overview.item.quake", { magnitude: m! }),
  ],
  // Parts of a line's detail.
  [
    /^(Landslide report|Water level and flood|Weather|Situation report)\. It may already have been withdrawn; check with the DMC\.$/,
    ([, type], l) => filled(l, "overview.item.mayBeWithdrawn", { type: word(l, exact, type) }),
  ],
  [
    /^(.+) District\. NBRO: (Watch|Alert|Evacuate)\.$/,
    ([, district, action], l) =>
      filled(l, "overview.item.landslide", {
        district: district!,
        action: word(l, actions, action),
      }),
  ],
  [
    /^(.+), at (-?\d+(?:\.\d+)? \S+)$/,
    ([, basin, reading], l) =>
      translateIn(l, "overview.item.gauge", { basin: basin!, reading: reading! }),
  ],
  [/^([\d,.]+) cusecs$/, ([, flow], l) => translateIn(l, "overview.item.cusecs", { flow: flow! })],
  [
    /^([\d,.]+) cusecs downstream, which can include power-station flow$/,
    ([, flow], l) => translateIn(l, "overview.item.cusecsDownstream", { flow: flow! }),
  ],
  [
    /^(\d+)% full$/,
    ([, percent], l) => translateIn(l, "overview.item.full", { percent: percent! }),
  ],
  [
    /^([\d,]+) km ([a-z-]+) of Sri Lanka$/,
    ([, distance, where], l) =>
      filled(l, "overview.item.cyclone", { distance: distance!, direction: direction(l, where) }),
  ],
  [
    /^winds to (\d+) km\/h$/,
    ([, speed], l) => translateIn(l, "overview.item.winds", { speed: speed! }),
  ],
  [
    /^([\d,]+) km from Sri Lanka$/,
    ([, distance], l) => translateIn(l, "overview.item.quakeDistance", { distance: distance! }),
  ],
  [
    /^(\d+) mm forecast today$/,
    ([, amount], l) => translateIn(l, "overview.item.rainToday", { amount: amount! }),
  ],
  [
    /^(\d+) km\/h, gusts to (\d+) km\/h$/,
    ([, speed, gusts], l) =>
      translateIn(l, "overview.item.windGusts", { speed: speed!, gusts: gusts! }),
  ],
  [
    /^US AQI (\d+), ([a-z ]+)$/,
    ([, aqi, band], l) =>
      filled(l, "overview.item.aqi", { aqi: aqi!, band: word(l, aqiBands, band) }),
  ],
  [
    /^About (\d+) km ([a-z-]+) of Sri Lanka's coast$/,
    ([, distance, where], l) =>
      filled(l, "overview.item.coast", { distance: distance!, direction: direction(l, where) }),
  ],
]

/** A figure with its unit says the same in every language. */
const figure = /^[−-]?[\d.,]+ ?(?:°C|mm|m|ft|km\/h|kt|%|cusecs)?$/

/** One of the API's sentences or labels in the language; the English as sent if not known. */
export function proseIn(locale: Locale, text: string): Prose {
  if (locale === "en" || text === "" || figure.test(text)) return { text }
  const key = exact[text]
  if (key) return { text: translateIn(locale, key) }
  for (const [pattern, write] of rules) {
    const match = pattern.exec(text)
    if (!match) continue
    const written = write(match, locale)
    if (written !== null) return { text: written }
  }
  return { text, lang: "en" }
}

/** The status at the top, sentence by sentence. */
export function statusIn(locale: Locale, status: string): Prose[] {
  if (locale === "en" || exact[status]) return [proseIn(locale, status)]
  return status
    .split(/(?<=\.) (?=[A-Z0-9])/)
    .filter(Boolean)
    .map((sentence) => proseIn(locale, sentence))
}

/** A line's detail, part by part ("1,551 cusecs · 97% full · Kurunegala"). */
export function detailIn(locale: Locale, detail: string): Prose[] {
  if (locale === "en") return [{ text: detail }]
  return detail.split(" · ").map((part) => proseIn(locale, part))
}
