/**
 * Words and colours for official warnings. Levels, titles and colours are the
 * issuers' own; this only chooses how they look. Pure, so it is tested without
 * a browser.
 */
import type { AlertSeverity, WeatherAlert } from "@/components/weather/types"
import { formatDayTime, sriLankaTimeZone } from "@/lib/format"
import type {
  BannerItem,
  Bulletin,
  BulletinSeverity,
  BulletinType,
  DistrictHighlight,
  LandslideHere,
  LandslideMeta,
  OfficialAt,
} from "./types"

export const landslideLevels = [3, 2, 1] as const
export type LandslideLevel = (typeof landslideLevels)[number]

/** As NBRO's reports print them, and NBRO's words for what to do. */
export const levelLabels: Record<LandslideLevel, string> = {
  1: "Level 1 (Yellow)",
  2: "Level 2 (Amber)",
  3: "Level 3 (Red)",
}

export const levelActions: Record<LandslideLevel, string> = {
  1: "Watch",
  2: "Alert",
  3: "Evacuate",
}

export const levelTokens: Record<LandslideLevel, `--${string}`> = {
  1: "--status-alert",
  2: "--status-minor-flood",
  3: "--status-major-flood",
}

export const levelColors: Record<LandslideLevel, string> = {
  1: "var(--status-alert)",
  2: "var(--status-minor-flood)",
  3: "var(--status-major-flood)",
}

export const levelBadge = {
  1: "alert",
  2: "minor_flood",
  3: "major_flood",
} as const satisfies Record<LandslideLevel, string>

export const severityLabels: Record<BulletinSeverity, string> = {
  warning: "Warning",
  advisory: "Advisory",
  withdrawal: "Withdrawal",
  routine: "Report",
}

export const bulletinTypeOrder: BulletinType[] = [
  "landslide",
  "water_level",
  "weather",
  "situation",
]

export const bulletinTypeLabels: Record<BulletinType, string> = {
  landslide: "Landslide",
  water_level: "Water level",
  weather: "Weather",
  situation: "Situation",
}

type BadgeTone = "alert" | "minor_flood" | "major_flood" | "normal" | "unknown" | "chip"

const colourBadge: Record<string, BadgeTone> = {
  Red: "major_flood",
  Amber: "minor_flood",
  Yellow: "alert",
  Green: "normal",
}

/** The badge for a colour a bulletin states; the word is always on it. */
export function colourVariant(colour: string | null | undefined): BadgeTone {
  return (colour && colourBadge[colour]) || "unknown"
}

/** A bulletin's badge: its stated colour, else how its title reads. */
export function bulletinBadge(
  bulletin: Pick<Bulletin, "severity" | "colour"> & Partial<Pick<Bulletin, "withdrawnBy">>,
): {
  variant: BadgeTone
  label: string
} {
  if (bulletin.withdrawnBy) return { variant: "chip", label: "Withdrawn" }
  if (bulletin.severity === "routine") return { variant: "chip", label: "Report" }
  if (bulletin.severity === "withdrawal") return { variant: "normal", label: "Withdrawal" }
  const word = severityLabels[bulletin.severity]
  if (bulletin.colour) {
    return { variant: colourVariant(bulletin.colour), label: `${word} · ${bulletin.colour}` }
  }
  return { variant: bulletin.severity === "warning" ? "major_flood" : "alert", label: word }
}

const dateFormat = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
  timeZone: "UTC",
})

/** A report's local date: "2026-10-31" is "31 Oct 2026". */
export function formatReportDate(date: string): string {
  return dateFormat.format(new Date(`${date}T00:00:00Z`))
}

const clockFormat = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
  timeZone: sriLankaTimeZone,
})

/** "1 Nov, 16:00" in Sri Lanka time. */
export function formatShort(iso: string): string {
  return clockFormat.format(new Date(iso))
}

/** Reports that can show levels on the map. */
export function levelsShown(status: LandslideMeta["status"]): boolean {
  return status === "active" || status === "lapsed" || status === "pending"
}

/** One line for the layer list. */
export function landslideSummary(meta: LandslideMeta, shown: number): string {
  if (meta.status === "waiting") return "Not collected yet"
  if (shown > 0) {
    const parts = landslideLevels
      .filter((level) => meta.counts[level] > 0)
      .map((level) => `Level ${level}: ${meta.counts[level]}`)
    return parts.join(" · ")
  }
  if (meta.status === "none" && meta.reportDate) {
    return `None active as far as we know · last ${formatReportDate(meta.reportDate)}`
  }
  if (meta.status === "unknown") return "Cannot say · feed out of date"
  return meta.note
}

/** Banner items not dismissed yet. */
export function visibleBanner(items: BannerItem[], dismissed: string[]): BannerItem[] {
  return items.filter((item) => !dismissed.includes(item.key))
}

/** The most serious item leads the banner: red before amber, landslide first on a tie. */
export function bannerLead(items: BannerItem[]): BannerItem | undefined {
  const rank = (item: BannerItem) =>
    item.colour === "Red" ? 3 : item.colour === "Amber" ? 2 : item.colour === "Yellow" ? 1 : 0
  return [...items].sort((a, b) => rank(b) - rank(a))[0]
}

/** Dismissals only need to remember what is still being announced. */
export function pruneDismissed(dismissed: string[], items: BannerItem[]): string[] {
  const live = new Set(items.map((item) => item.key))
  return dismissed.filter((key) => live.has(key))
}

/** Why a district is highlighted, in words for the hover chip. */
export function highlightLines(district: DistrictHighlight): string[] {
  const lines: string[] = []
  if (district.landslideLevel > 0) {
    lines.push(`Landslide warning: ${levelLabels[district.landslideLevel as LandslideLevel]}`)
  }
  if (district.mentioned.length > 0) {
    lines.push(`Mentioned in bulletin: ${district.mentioned[0]!.title}`)
    if (district.mentioned.length > 1) lines.push(`and ${district.mentioned.length - 1} more`)
  }
  return lines
}

const severityFor: Record<BulletinSeverity, AlertSeverity> = {
  warning: "warning",
  advisory: "advisory",
  withdrawal: "advisory",
  routine: "advisory",
}

const levelSeverity: Record<LandslideLevel, AlertSeverity> = {
  1: "watch",
  2: "warning",
  3: "emergency",
}

const issuers: Record<BulletinType, string> = {
  landslide: "NBRO, via DMC",
  water_level: "DMC",
  weather: "DMC",
  situation: "DMC",
}

/** A bulletin as the alert list shows it: the issuer's title and words, with its link. */
export function bulletinAlert(bulletin: Bulletin, area: string): WeatherAlert {
  const badge = bulletinBadge(bulletin)
  return {
    id: bulletin.id,
    title: bulletin.title,
    severity: bulletin.colour === "Red" ? "emergency" : severityFor[bulletin.severity],
    badgeLabel: badge.label,
    issuer: `${issuers[bulletin.type]} · ${bulletin.typeLabel}`,
    area,
    issuedAt: bulletin.issuedAt,
    validUntil: bulletin.validUntil ?? undefined,
    summary: bulletin.text ?? undefined,
    excerpt: bulletin.text !== null,
    link: { href: bulletin.pdfUrl, label: "Official PDF" },
  }
}

/** The landslide level of the place's DS division as the alert list shows it, if it has one. */
export function landslideAlert(here: LandslideHere | null | undefined): WeatherAlert | null {
  if (!here?.level) return null
  const level = here.level.level
  return {
    id: `landslide-${here.dsId}`,
    title: "Landslide early warning",
    severity: levelSeverity[level],
    badgeLabel: `${here.level.label} · ${here.level.action}`,
    issuer: "NBRO, via DMC",
    area: [here.ds, here.district && `${here.district} District`].filter(Boolean).join(", "),
    issuedAt: here.issuedAt ?? here.reportDate ?? "",
    validUntil: here.validUntil ?? undefined,
    link: here.pdfUrl ? { href: here.pdfUrl, label: "Official report" } : undefined,
  }
}

/**
 * What to say about landslides here when the division has no level. It says
 * only what is known: not being named is not a statement of safety.
 */
export function landslideNote(here: LandslideHere | null | undefined): string | undefined {
  if (!here) return undefined
  if (here.level) return here.note
  if ((here.status === "active" || here.status === "lapsed") && here.reportDate) {
    return `${here.ds ?? "This place"} is not named in the landslide report of ${formatReportDate(here.reportDate)}. That does not mean it is safe.`
  }
  return here.note
}

/**
 * The location panel's official slot: warnings that count (the landslide level
 * and warnings or advisories in force) and bulletins that are only for reading.
 */
export function officialSlot(
  official: OfficialAt | null | undefined,
  districtName?: string | null,
) {
  const alerts: WeatherAlert[] = []
  const bulletins: WeatherAlert[] = []
  if (!official) return { alerts, bulletins, note: undefined }
  const landslide = landslideAlert(official.landslide)
  if (landslide) alerts.push(landslide)
  const place = districtName ? `Mentioned in bulletin: ${districtName}` : "Mentioned in bulletin"
  for (const bulletin of official.warnings) {
    if (bulletin.current) alerts.push(bulletinAlert(bulletin, place))
    else bulletins.push(bulletinAlert(bulletin, place))
  }
  for (const bulletin of official.general) {
    if (bulletin.current) alerts.push(bulletinAlert(bulletin, "Sri Lanka"))
    else bulletins.push(bulletinAlert(bulletin, "Sri Lanka"))
  }
  return { alerts, bulletins, note: landslideNote(official.landslide) ?? official.note }
}

/** "Issued Mon 3 Nov · 16:00" */
export function issuedText(iso: string): string {
  return `Issued ${formatDayTime(iso)}`
}
