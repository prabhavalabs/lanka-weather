/** Times are shown in Sri Lanka time wherever the reader is. */
export const sriLankaTimeZone = "Asia/Colombo"

const hourFormat = new Intl.DateTimeFormat("en-GB", {
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
  timeZone: sriLankaTimeZone,
})

const dayFormat = new Intl.DateTimeFormat("en-GB", {
  weekday: "short",
  day: "numeric",
  month: "short",
  timeZone: sriLankaTimeZone,
})

const weekdayFormat = new Intl.DateTimeFormat("en-GB", {
  weekday: "short",
  timeZone: sriLankaTimeZone,
})

/** "14:00" */
export function formatClock(iso: string): string {
  return hourFormat.format(new Date(iso))
}

/** "Mon 3 Nov" */
export function formatDay(iso: string): string {
  return dayFormat.format(new Date(iso)).replace(",", "")
}

/** "Mon" */
export function formatWeekday(iso: string): string {
  return weekdayFormat.format(new Date(iso))
}

/** "Mon 3 Nov · 14:00" */
export function formatDayTime(iso: string): string {
  return `${formatDay(iso)} · ${formatClock(iso)}`
}

const compass = ["N", "NE", "E", "SE", "S", "SW", "W", "NW"] as const

/** The eight-point compass name of a bearing in degrees: 225 → "SW". */
export function compassPoint(degrees: number): (typeof compass)[number] {
  const normalised = ((degrees % 360) + 360) % 360
  return compass[Math.round(normalised / 45) % 8]!
}

/** A number with a fixed number of decimals and a true minus sign. */
export function formatNumber(value: number, decimals = 0): string {
  const text = Math.abs(value).toFixed(decimals)
  return value < 0 && Number(text) !== 0 ? `−${text}` : text
}

/** How long ago a time was, in words: "just now", "12 min ago", "3 h ago", "2 days ago". */
export function formatAge(iso: string | null, now: Date = new Date()): string {
  if (!iso) return "never"
  const minutes = Math.round((now.getTime() - new Date(iso).getTime()) / 60_000)
  if (minutes < 1) return "just now"
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.round(minutes / 60)
  if (hours < 48) return `${hours} h ago`
  return `${Math.round(hours / 24)} days ago`
}
