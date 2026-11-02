/**
 * Shapes the weather shells render. They are presentational: the
 * integration step maps API responses onto these and passes them in.
 */

export type WeatherCondition =
  | "clear"
  | "clear-night"
  | "partly-cloudy"
  | "cloudy"
  | "drizzle"
  | "rain"
  | "heavy-rain"
  | "thunderstorm"
  | "fog"
  | "windy"

export interface Place {
  name: string
  /** Administrative context, e.g. "Ratnapura District · Sabaragamuwa Province". */
  area: string
  latitude: number
  longitude: number
}

export interface CurrentConditions {
  condition: WeatherCondition
  /** Words for the condition, e.g. "Heavy showers". */
  summary: string
  temperatureC: number
  feelsLikeC: number
  humidityPercent: number
  /** Rainfall in the last 24 hours. */
  rainfall24hMm: number
  pressureHpa: number
  /** When the conditions were observed or modelled (ISO 8601). */
  observedAt: string
}

export interface ForecastHour {
  /** Start of the hour (ISO 8601). */
  time: string
  condition: WeatherCondition
  temperatureC: number
  /** Expected rainfall in the hour. */
  rainfallMm: number
  /** Chance of measurable rain, 0 to 100. */
  rainChancePercent: number
}

export interface WindReading {
  /** Mean speed. */
  speedKmh: number
  gustKmh: number
  /** Direction the wind blows from, in degrees clockwise from north. */
  fromDegrees: number
}

export type AlertSeverity = "advisory" | "watch" | "warning" | "emergency"

export interface WeatherAlert {
  id: string
  title: string
  severity: AlertSeverity
  /** Who issued it, e.g. "Department of Meteorology". */
  issuer: string
  /** Where it applies, e.g. "Ratnapura, Kegalle and Kalutara districts". */
  area: string
  issuedAt: string
  /** When it lapses, if stated. */
  validUntil?: string
  summary?: string
}

export type AdminLevel = "province" | "district" | "ds" | "gn"

export interface AdminArea {
  id: string
  name: string
  level: AdminLevel
  /** Name of the containing area, e.g. the district of a DS division. */
  parentName?: string
}

export type SearchResultKind = "place" | "station" | "area"

export interface SearchResult {
  id: string
  kind: SearchResultKind
  title: string
  /** Second line, e.g. "Kalu Ganga · river gauge". */
  subtitle?: string
  /** Short value on the right, e.g. a level or a temperature. */
  meta?: string
}
