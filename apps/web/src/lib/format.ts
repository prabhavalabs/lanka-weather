/**
 * Dates, times and numbers in the reader's language, always in Sri Lanka
 * time and always in Western digits (what Sri Lankan Sinhala and Tamil media
 * print). Month names come from the catalogues rather than `Intl`, whose
 * Sinhala data names the Gregorian months after the traditional lunar ones
 * (ඉල් for November); clocks are written 14:00 in every language.
 *
 * Each function reads the current language, so a component that shows its
 * result must subscribe with `useT()` or `useLocale()` to re-render when the
 * language changes. Pass `locale` to format in another language.
 */
import { getLocale, intlTag, translateIn, type Locale, type MessageKey } from "@/lib/i18n"

/** Times are shown in Sri Lanka time wherever the reader is. */
export const sriLankaTimeZone = "Asia/Colombo"

const formats = new Map<string, Intl.DateTimeFormat>()

/** One cached `Intl.DateTimeFormat` per language and option set, in Sri Lanka time. */
function dateFormat(locale: Locale, name: string, options: Intl.DateTimeFormatOptions) {
  const key = `${locale}:${name}`
  let format = formats.get(key)
  if (!format) {
    format = new Intl.DateTimeFormat(intlTag(locale), {
      ...options,
      numberingSystem: "latn",
      timeZone: options.timeZone ?? sriLankaTimeZone,
    })
    formats.set(key, format)
  }
  return format
}

/** Year, month (1-12), day, hour and minute of a moment in Sri Lanka. */
function colomboParts(date: Date) {
  const parts = dateFormat("en", "parts", {
    year: "numeric",
    month: "numeric",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(date)
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    Number(parts.find((item) => item.type === type)?.value ?? 0)
  return {
    year: part("year"),
    month: part("month"),
    day: part("day"),
    hour: part("hour"),
    minute: part("minute"),
  }
}

type Month = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12

/** "Nov" or "November" in the language. */
export function monthName(month: number, style: "short" | "long" = "short", locale = getLocale()) {
  const key = `format.${style === "short" ? "monthShort" : "monthLong"}.m${month as Month}` as const
  return translateIn(locale, key)
}

/** "14:00" */
export function formatClock(iso: string): string {
  const { hour, minute } = colomboParts(new Date(iso))
  return `${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}`
}

/** "Mon" */
export function formatWeekday(iso: string, locale: Locale = getLocale()): string {
  return dateFormat(locale, "weekday", { weekday: "short" }).format(new Date(iso))
}

/** "Monday" */
export function formatWeekdayLong(iso: string, locale: Locale = getLocale()): string {
  return dateFormat(locale, "weekday-long", { weekday: "long" }).format(new Date(iso))
}

/** "Mon 3 Nov" */
export function formatDay(iso: string, locale: Locale = getLocale()): string {
  const date = new Date(iso)
  const { day, month } = colomboParts(date)
  return translateIn(locale, "format.day", {
    weekday: formatWeekday(iso, locale),
    day,
    month: monthName(month, "short", locale),
  })
}

/** "Mon 3 Nov · 14:00" */
export function formatDayTime(iso: string, locale: Locale = getLocale()): string {
  return translateIn(locale, "format.dayTime", {
    day: formatDay(iso, locale),
    time: formatClock(iso),
  })
}

/** "3 Nov, 14:00": a date and time without the weekday. */
export function formatDateTime(iso: string, locale: Locale = getLocale()): string {
  const { day, month } = colomboParts(new Date(iso))
  return translateIn(locale, "format.dateTime", {
    day,
    month: monthName(month, "short", locale),
    time: formatClock(iso),
  })
}

/**
 * A calendar date, "3 Nov 2026". Takes an ISO moment (read in Sri Lanka time)
 * or a plain "YYYY-MM-DD" date (read as written).
 */
export function formatDate(
  value: string,
  style: "short" | "long" = "short",
  locale: Locale = getLocale(),
): string {
  const plain = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  const { year, month, day } = plain
    ? { year: Number(plain[1]), month: Number(plain[2]), day: Number(plain[3]) }
    : colomboParts(new Date(value))
  return translateIn(locale, "format.date", { day, month: monthName(month, style, locale), year })
}

/** "3 Nov": a calendar day without year or weekday. */
export function formatDayMonth(value: string, locale: Locale = getLocale()): string {
  const plain = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  const { month, day } = plain
    ? { month: Number(plain[2]), day: Number(plain[3]) }
    : colomboParts(new Date(value))
  return translateIn(locale, "format.dayMonth", { day, month: monthName(month, "short", locale) })
}

const compass8 = ["N", "NE", "E", "SE", "S", "SW", "W", "NW"] as const
const compass16 = [
  "N",
  "NNE",
  "NE",
  "ENE",
  "E",
  "ESE",
  "SE",
  "SSE",
  "S",
  "SSW",
  "SW",
  "WSW",
  "W",
  "WNW",
  "NW",
  "NNW",
] as const

export type CompassCode = (typeof compass16)[number]

/** The compass point of a bearing in degrees, as a code: 225 → "SW". */
export function compassCode(degrees: number, points: 8 | 16 = 8): CompassCode {
  const normalised = ((degrees % 360) + 360) % 360
  return points === 16
    ? compass16[Math.round(normalised / 22.5) % 16]!
    : compass8[Math.round(normalised / 45) % 8]!
}

/**
 * The compass point of a bearing in the reader's language: 225 is "SW" in
 * English, "නිරිත" in Sinhala and "தென்மேற்கு" in Tamil.
 */
export function compassPoint(degrees: number, points: 8 | 16 = 8, locale = getLocale()): string {
  return translateIn(locale, `format.compass.${compassCode(degrees, points)}`)
}

/** A compass code in words: "SW" is "south-west". */
export function compassWords(code: CompassCode, locale = getLocale()): string {
  return translateIn(locale, `format.compassLong.${code}`)
}

/** A number with a fixed number of decimals and a true minus sign. */
export function formatNumber(value: number, decimals = 0): string {
  const text = Math.abs(value).toFixed(decimals)
  return value < 0 && Number(text) !== 0 ? `−${text}` : text
}

/**
 * A number with thousands separators, "12,345", at most `decimals` decimals.
 * Grouped in threes in every language: Sri Lankan Tamil writing does not use
 * the Indian lakh grouping that `Intl` gives Tamil.
 */
export function formatInteger(value: number, decimals = 0): string {
  const key = `group:${decimals}`
  let format = numberFormats.get(key)
  if (!format) {
    format = new Intl.NumberFormat("en-GB", {
      maximumFractionDigits: decimals,
      numberingSystem: "latn",
    })
    numberFormats.set(key, format)
  }
  const text = format.format(Math.abs(value))
  return value < 0 && text !== "0" ? `−${text}` : text
}

const numberFormats = new Map<string, Intl.NumberFormat>()

/** How long ago a time was, in words: "just now", "12 min ago", "3 h ago", "2 days ago". */
export function formatAge(
  iso: string | null,
  now: Date = new Date(),
  locale: Locale = getLocale(),
): string {
  if (!iso) return translateIn(locale, "format.never")
  const minutes = Math.round((now.getTime() - new Date(iso).getTime()) / 60_000)
  if (minutes < 1) return translateIn(locale, "format.justNow")
  if (minutes < 60) return translateIn(locale, "format.minutesAgo", { count: minutes })
  const hours = Math.round(minutes / 60)
  if (hours < 48) return translateIn(locale, "format.hoursAgo", { count: hours })
  return translateIn(locale, "format.daysAgo", { count: Math.round(hours / 24) })
}

/** How far ahead a time is: "in 12 min", "in 3 h", "in 2 days"; "now" when past. */
export function formatIn(iso: string, now: Date = new Date(), locale: Locale = getLocale()) {
  const minutes = Math.round((new Date(iso).getTime() - now.getTime()) / 60_000)
  if (minutes < 1) return translateIn(locale, "format.now")
  if (minutes < 60) return translateIn(locale, "format.inMinutes", { count: minutes })
  const hours = Math.round(minutes / 60)
  if (hours < 48) return translateIn(locale, "format.inHours", { count: hours })
  return translateIn(locale, "format.inDays", { count: Math.round(hours / 24) })
}

/** A message key in the `format` namespace, for helpers that build their own phrases. */
export type FormatKey = Extract<MessageKey, `format.${string}`>
