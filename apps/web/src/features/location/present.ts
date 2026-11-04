/**
 * Maps the API's location answers (`/v1/summary`, `/v1/weather/point`,
 * `/v1/regions/{id}`) onto the shapes the weather shells render. Pure, so
 * the mapping is tested without a browser.
 */
import { conditionFromWmo } from "@/components/weather/conditions"
import type {
  AlertSeverity,
  CurrentConditions,
  ForecastDay,
  ForecastHour,
  Place,
  WeatherAlert,
  WindReading,
} from "@/components/weather/types"
import { formatNumber, sriLankaTimeZone } from "@/lib/format"
import {
  getLocale,
  localName,
  translate,
  translateIn,
  type LocalNames,
  type Locale,
  type MessageKey,
} from "@/lib/i18n"
import { regionBareName } from "../regions/present"
import type { RegionDetail } from "../regions/types"
import type { Selection } from "./selection"
import type {
  AlertLevel,
  DerivedAlert,
  PointForecast,
  RegionLevel,
  RegionRef,
  Summary,
} from "./types"

/** The level's name in the current language: "District", "DS division". */
export function levelWord(level: RegionLevel): string {
  return translate(`terms.adminLevel.${level}`)
}

/**
 * The level names in the current language, read when used. Prefer
 * `levelWord(level)`; this keeps `levelWords[level]` working.
 */
export const levelWords: Readonly<Record<RegionLevel, string>> = {
  get province() {
    return levelWord("province")
  },
  get district() {
    return levelWord("district")
  },
  get ds() {
    return levelWord("ds")
  },
  get gn() {
    return levelWord("gn")
  },
}

/**
 * A region's name as people say it, in the current language when the data
 * has it: "Kandy District", "මහනුවර දිස්ත්‍රික්කය", "Central Province". A
 * local name that already ends in the level word is kept as it is.
 */
export function regionLabel(
  region: Pick<RegionRef, "level" | "name"> & Partial<Pick<LocalNames, "nameSi" | "nameTa">>,
): string {
  if (region.level !== "district" && region.level !== "province") return localName(region)
  return translate(`location.region.${region.level}`, { name: regionBareName(region) })
}

/** The regions containing a place, smallest first: "Katukele › Gangawata Korale › Kandy District › Central Province". */
export function hierarchyText(regions: Summary["regions"], skip: RegionLevel[] = []): string {
  if (!regions) return ""
  return [regions.gn, regions.ds, regions.district, regions.province]
    .filter((region): region is RegionRef => region !== null && !skip.includes(region.level))
    .map(regionLabel)
    .join(" › ")
}

/** "7.2931° N, 80.6350° E" */
export function formatCoordinates(lat: number, lon: number): string {
  return translate("location.coordinates", {
    lat: Math.abs(lat).toFixed(4),
    latHemisphere: translate(lat >= 0 ? "location.hemisphere.north" : "location.hemisphere.south"),
    lon: Math.abs(lon).toFixed(4),
    lonHemisphere: translate(lon >= 0 ? "location.hemisphere.east" : "location.hemisphere.west"),
  })
}

/**
 * The name in another script, for under the heading: the Sinhala name when
 * reading English, the English name when reading Sinhala or Tamil (when the
 * heading is a local name).
 */
function otherName(
  names: LocalNames,
  locale: Locale = getLocale(),
  level?: RegionLevel,
): Pick<Place, "localName" | "localNameLang"> {
  if (locale === "en") {
    if (!names.nameSi) return {}
    // A province's or district's Sinhala name, mended and with its whole level word.
    const localName =
      level === "district" || level === "province"
        ? translateIn("si", `location.region.${level}`, {
            name: regionBareName({ level, ...names }, "si"),
          })
        : names.nameSi
    return { localName, localNameLang: "si" }
  }
  return localName(names, locale) !== names.name
    ? { localName: names.name, localNameLang: "en" }
    : {}
}

/**
 * The heading of the panel. A searched place keeps its name; a point
 * clicked on the map is named after the GN division containing it; a
 * region is named after itself, with its level and parents below.
 */
export function placeFor(
  selection: Selection,
  location: { lat: number; lon: number },
  summary?: Summary | null,
  region?: RegionDetail | null,
): Place {
  const base = { latitude: location.lat, longitude: location.lon }
  if (selection.kind === "region") {
    if (!region) return { ...base, name: translate("common.loading"), area: "" }
    const names: LocalNames = {
      name: region.name,
      nameSi: region.names.si,
      nameTa: region.names.ta,
    }
    const parents = [...region.hierarchy]
      .reverse()
      .map((parent) => regionLabel(parent))
      .join(" › ")
    return {
      ...base,
      name: regionLabel({ level: region.level, ...names }),
      ...otherName(names, getLocale(), region.level),
      area: [levelWord(region.level), parents].filter(Boolean).join(" · "),
    }
  }
  const regions = summary?.regions ?? null
  if (selection.kind === "point" && selection.name) {
    return { ...base, name: selection.name, area: hierarchyText(regions) }
  }
  const gn = regions?.gn
  if (gn) {
    return {
      ...base,
      name: localName(gn),
      ...otherName(gn),
      area: hierarchyText(regions, ["gn"]),
    }
  }
  return {
    ...base,
    name: summary ? translate("location.offCoast") : translate("common.loading"),
    area: formatCoordinates(location.lat, location.lon),
  }
}

const num = (value: number | null | undefined) =>
  value === null || value === undefined || !Number.isFinite(value) ? null : value

/** Total of the next 24 hourly values, or null without any. */
export function sumNext(values: (number | null)[] | undefined, hours = 24): number | null {
  if (!values) return null
  let total = 0
  let seen = 0
  for (const value of values.slice(0, hours)) {
    if (value === null) continue
    total += value
    seen += 1
  }
  return seen === 0 ? null : Math.round(total * 10) / 10
}

/** Current conditions, or null when the point has none (outside the grid, nothing collected). */
export function currentFrom(point: PointForecast | null | undefined): CurrentConditions | null {
  const current = point?.current
  if (!current || current.temperatureC === null) return null
  const { condition, summary } = conditionFromWmo(
    current.weatherCode,
    current.isDay,
    current.windSpeedKmh,
  )
  return {
    condition,
    summary,
    temperatureC: current.temperatureC,
    feelsLikeC: num(current.apparentTemperatureC),
    humidityPercent: num(current.relativeHumidityPct),
    rainfallMm: num(current.precipitationMm),
    rainNext24hMm: sumNext(point?.hourly?.precipitationMm),
    rainChancePercent: num(current.precipitationProbabilityPct),
    cloudCoverPercent: num(current.cloudCoverPct),
    pressureHpa: num(current.pressureHpa),
    uvIndex: num(current.uvIndex),
    dewPointC: num(current.dewPointC),
    observedAt: current.validAt,
  }
}

export function windFrom(point: PointForecast | null | undefined): WindReading | null {
  const current = point?.current
  if (!current || current.windSpeedKmh === null || current.windDirectionDeg === null) return null
  return {
    speedKmh: current.windSpeedKmh,
    gustKmh: current.windGustsKmh ?? current.windSpeedKmh,
    fromDegrees: current.windDirectionDeg,
  }
}

/** The hourly forecast; hours without a temperature are left out. */
export function hoursFrom(point: PointForecast | null | undefined): ForecastHour[] {
  const hourly = point?.hourly
  if (!hourly) return []
  const hours: ForecastHour[] = []
  hourly.times.forEach((time, index) => {
    const temperatureC = hourly.temperatureC[index]
    if (temperatureC === null || temperatureC === undefined) return
    const { condition } = conditionFromWmo(
      hourly.weatherCode[index],
      hourly.isDay[index],
      hourly.windSpeedKmh[index],
    )
    hours.push({
      time,
      condition,
      temperatureC,
      rainfallMm: hourly.precipitationMm[index] ?? 0,
      rainChancePercent: Math.round(hourly.precipitationProbabilityPct[index] ?? 0),
      windSpeedKmh: num(hourly.windSpeedKmh[index]),
      gustKmh: num(hourly.windGustsKmh[index]),
      windFromDegrees: num(hourly.windDirectionDeg[index]),
    })
  })
  return hours
}

export function daysFrom(point: PointForecast | null | undefined): ForecastDay[] {
  const daily = point?.daily
  if (!daily) return []
  return daily.dates.map((date, index) => {
    const { condition, summary } = conditionFromWmo(daily.weatherCode[index], 1)
    return {
      date,
      condition,
      summary,
      minC: num(daily.temperatureMinC[index]),
      maxC: num(daily.temperatureMaxC[index]),
      rainMm: num(daily.precipitationSumMm[index]),
      rainChancePercent: num(daily.precipitationProbabilityMaxPct[index]),
      windMaxKmh: num(daily.windSpeedMaxKmh[index]),
      gustMaxKmh: num(daily.windGustsMaxKmh[index]),
      uvMax: num(daily.uvIndexMax[index]),
    }
  })
}

const levelSeverity: Record<AlertLevel, AlertSeverity> = {
  yellow: "watch",
  orange: "warning",
  red: "emergency",
}

const levelKey = { yellow: "Yellow", orange: "Orange", red: "Red" } as const

/** Days from one Sri Lanka date to another (YYYY-MM-DD). */
function daysBetween(from: string, to: string): number {
  return Math.round((Date.parse(to) - Date.parse(from)) / 86_400_000)
}

/**
 * When a forecast alert's window falls, in words, as the server words it:
 * a window starting today is the next 24 hours, else tomorrow or the day after.
 */
function alertWhen(validFrom: string | null | undefined, generatedAt: string): string | null {
  if (!validFrom || Number.isNaN(Date.parse(validFrom)) || Number.isNaN(Date.parse(generatedAt))) {
    return null
  }
  const days = daysBetween(localDate(new Date(generatedAt)), localDate(new Date(validFrom)))
  if (days <= 0) return translate("location.alert.when.next24h")
  if (days === 1) return translate("location.alert.when.tomorrow")
  if (days === 2) return translate("location.alert.when.dayAfter")
  return null
}

/**
 * A derived alert's title and detail in the current language, rebuilt from
 * its type, level and numbers. Null parts keep the server's English words.
 */
function alertText(
  alert: DerivedAlert,
  generatedAt: string,
): { title: string | null; detail: string | null } {
  const level = levelKey[alert.level]
  if (!level) return { title: null, detail: null }
  const value = typeof alert.value === "number" ? formatNumber(alert.value) : null
  const threshold = typeof alert.threshold === "number" ? formatNumber(alert.threshold) : null
  const when = alertWhen(alert.validFrom, generatedAt)
  const forecast = (key: MessageKey) =>
    value !== null && threshold !== null && when !== null
      ? translate(key, { value, threshold, when })
      : null
  switch (alert.type) {
    case "heavy_rain":
      return {
        title: translate(`location.alert.title.rain${level}`),
        detail: forecast("location.alert.rain"),
      }
    case "strong_wind": {
      const kind = alert.detail.startsWith("Gusts")
        ? "location.alert.gusts"
        : alert.detail.startsWith("Winds")
          ? "location.alert.winds"
          : null
      return {
        title: translate(`location.alert.title.wind${level}`),
        detail: kind ? forecast(kind) : null,
      }
    }
    case "heat":
      return {
        title: translate(`location.alert.title.heat${level}`),
        detail:
          value !== null && threshold !== null
            ? translate("location.alert.heat", { value, threshold })
            : null,
      }
    case "river_level": {
      const name = /^The (.+) gauge, /.exec(alert.detail)?.[1]
      return {
        title: translate(`location.alert.title.river${level}`),
        detail:
          name && value !== null
            ? translate(`location.alert.river${level}`, { name, distance: value })
            : null,
      }
    }
    case "flood_forecast":
      return value !== null
        ? {
            title: translate("location.alert.title.floodGauge"),
            detail: translate(`location.alert.floodGauge${level}`, { distance: value }),
          }
        : {
            title: translate("location.alert.title.floodArea"),
            detail: translate("location.alert.floodArea"),
          }
  }
  return { title: null, detail: null }
}

const forecastBasis = "Open-Meteo forecast, interpolated from the weather grid"

/** The server's words for what an alert was derived from, in the current language. */
function basisText(basis: string): string | null {
  if (basis === forecastBasis) return translate("location.alert.basis.forecast")
  if (basis === `${forecastBasis}; heat index from temperature and humidity`) {
    return translate("location.alert.basis.heat")
  }
  if (basis === "Irrigation Department river gauge reading and the station's own thresholds") {
    return translate("location.alert.basis.rivers")
  }
  if (basis === "Google Flood Forecasting") return translate("location.alert.basis.floods")
  const source = /^(.+) gauge reading and the station's published thresholds$/.exec(basis)?.[1]
  return source ? translate("location.alert.basis.riversOther", { source }) : null
}

/**
 * A derived alert as the alert list shows it: always an estimate, issued
 * by lanka-weather, with the data it came from. Its words are rebuilt in the
 * current language from the alert's type and numbers; what cannot be rebuilt
 * keeps the server's English, marked as English.
 */
export function estimateAlert(
  alert: DerivedAlert,
  index: number,
  area: string,
  generatedAt: string,
): WeatherAlert {
  const text = alertText(alert, generatedAt)
  const basis = alert.basis ? basisText(alert.basis) : null
  return {
    id: `${alert.type}-${index}`,
    title: text.title ?? alert.title,
    ...(text.title === null ? { titleLang: "en" } : {}),
    severity: levelSeverity[alert.level] ?? "watch",
    estimate: true,
    issuer: translate("location.alert.issuer"),
    area,
    issuedAt: alert.validFrom ?? generatedAt,
    validUntil: alert.validUntil ?? undefined,
    summary: text.detail ?? alert.detail,
    ...(text.detail === null && alert.detail ? { summaryLang: "en" } : {}),
    basis: basis ?? alert.basis,
    ...(basis === null && alert.basis ? { basisLang: "en" } : {}),
  }
}

const dateFormat = new Intl.DateTimeFormat("en-CA", {
  timeZone: sriLankaTimeZone,
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
})

/** Today's date in Sri Lanka, YYYY-MM-DD. */
export function localDate(now: Date = new Date()): string {
  return dateFormat.format(now)
}
