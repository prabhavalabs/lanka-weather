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
  /** The name in the reader's language, else in English. */
  name: string
  /**
   * The name in another script, shown beside the area: the Sinhala name when
   * reading English, the English name when reading Sinhala or Tamil.
   */
  localName?: string
  /** The language of `localName` (BCP 47); Sinhala when not given. */
  localNameLang?: string
  /** Administrative context, smallest first, e.g. "Mahanuwara › Gangawata Korale › Kandy › Central". */
  area: string
  latitude: number
  longitude: number
}

export interface CurrentConditions {
  condition: WeatherCondition
  /** Words for the condition, e.g. "Heavy showers". */
  summary: string
  temperatureC: number
  feelsLikeC: number | null
  humidityPercent: number | null
  /** Rain in the current hour. */
  rainfallMm: number | null
  /** Rain expected over the next 24 hours, when a forecast is available. */
  rainNext24hMm?: number | null
  /** Chance of rain in the current hour, 0 to 100. */
  rainChancePercent?: number | null
  cloudCoverPercent?: number | null
  pressureHpa: number | null
  uvIndex?: number | null
  dewPointC?: number | null
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
  windSpeedKmh?: number | null
  gustKmh?: number | null
  /** Direction the wind blows from, degrees clockwise from north. */
  windFromDegrees?: number | null
}

/** One local day of the forecast. */
export interface ForecastDay {
  /** Local date, YYYY-MM-DD (Sri Lanka time). */
  date: string
  condition: WeatherCondition
  summary: string
  minC: number | null
  maxC: number | null
  rainMm: number | null
  rainChancePercent: number | null
  windMaxKmh: number | null
  gustMaxKmh: number | null
  uvMax: number | null
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
  /**
   * True for an alert lanka-weather derived from forecasts and gauges: it is
   * an estimate, never an official warning, and is labelled so.
   */
  estimate?: boolean
  /** For estimates, the data it was derived from. */
  basis?: string
  /**
   * The language of `title`, `summary` or `basis` when it is not the
   * reader's, e.g. "en" for English text from the server or a bulletin.
   */
  titleLang?: string
  summaryLang?: string
  basisLang?: string
  /** Who issued it, e.g. "Department of Meteorology". */
  issuer: string
  /** Where it applies, e.g. "Ratnapura, Kegalle and Kalutara districts". */
  area: string
  issuedAt: string
  /** When it lapses, if stated. */
  validUntil?: string
  summary?: string
  /** An issuer's own words for the level, shown in place of the severity's. */
  badgeLabel?: string
  /** `summary` is an excerpt of a longer official document. */
  excerpt?: boolean
  /** The official document. */
  link?: { href: string; label: string }
}

export type AdminLevel = "province" | "district" | "ds" | "gn"

export interface AdminArea {
  id: string
  name: string
  level: AdminLevel
  /** Name of the containing area, e.g. the district of a DS division. */
  parentName?: string
}

export type SearchResultKind = "place" | "station" | "reservoir" | "area" | "river"

export interface SearchResult {
  id: string
  kind: SearchResultKind
  title: string
  /** The name in Sinhala or Tamil, shown after the title. */
  localName?: string
  /** The language of `localName` (BCP 47); Sinhala when not given. */
  localNameLang?: string
  /** Second line, e.g. "Kalu Ganga · river gauge". */
  subtitle?: string
  /** Short value on the right, e.g. a level or a temperature. */
  meta?: string
}
