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
import { sriLankaTimeZone } from "@/lib/format"
import type { RegionDetail } from "../regions/types"
import type { Selection } from "./selection"
import type {
  AlertLevel,
  DerivedAlert,
  OfficialWarning,
  PointForecast,
  RegionLevel,
  RegionRef,
  Summary,
} from "./types"

export const levelWords: Record<RegionLevel, string> = {
  province: "Province",
  district: "District",
  ds: "DS division",
  gn: "GN division",
}

/** A region's name as people say it: "Kandy District", "Central Province". */
export function regionLabel(region: Pick<RegionRef, "level" | "name">): string {
  if (region.level === "district") return `${region.name} District`
  if (region.level === "province") return `${region.name} Province`
  return region.name
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
  return `${Math.abs(lat).toFixed(4)}° ${lat >= 0 ? "N" : "S"}, ${Math.abs(lon).toFixed(4)}° ${lon >= 0 ? "E" : "W"}`
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
    if (!region) return { ...base, name: "Loading…", area: "" }
    const parents = [...region.hierarchy]
      .reverse()
      .map((parent) => regionLabel(parent))
      .join(" › ")
    return {
      ...base,
      name: regionLabel(region),
      localName: region.names.si ?? undefined,
      area: [levelWords[region.level], parents].filter(Boolean).join(" · "),
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
      name: gn.name,
      localName: gn.nameSi ?? undefined,
      area: hierarchyText(regions, ["gn"]),
    }
  }
  return {
    ...base,
    name: summary ? "Off the coast" : "Loading…",
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

/**
 * A derived alert as the alert list shows it: always an estimate, issued
 * by lanka-weather, with the data it came from.
 */
export function estimateAlert(
  alert: DerivedAlert,
  index: number,
  area: string,
  generatedAt: string,
): WeatherAlert {
  return {
    id: `${alert.type}-${index}`,
    title: alert.title,
    severity: levelSeverity[alert.level] ?? "watch",
    estimate: true,
    issuer: "Lanka Weather estimate",
    area,
    issuedAt: alert.validFrom ?? generatedAt,
    validUntil: alert.validUntil ?? undefined,
    summary: alert.detail,
    basis: alert.basis,
  }
}

const officialSeverities: AlertSeverity[] = ["advisory", "watch", "warning", "emergency"]

/** An official warning (Department of Meteorology, DMC), once the API carries them. */
export function officialAlert(
  warning: OfficialWarning,
  index: number,
  area: string,
  generatedAt: string,
): WeatherAlert {
  const severity = officialSeverities.find((value) => value === warning.severity) ?? "warning"
  return {
    id: warning.id ?? `official-${index}`,
    title: warning.title,
    severity,
    issuer: warning.issuer ?? "Department of Meteorology",
    area: warning.area ?? area,
    issuedAt: warning.issuedAt ?? generatedAt,
    validUntil: warning.validUntil ?? undefined,
    summary: warning.summary,
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
