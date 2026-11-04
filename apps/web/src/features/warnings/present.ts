/**
 * Words and colours for official warnings. Levels, titles and colours are the
 * issuers' own; this only chooses how they look and in which language the
 * interface around them reads. Pure, so it is tested without a browser.
 *
 * Bulletin titles and text are the issuer's words and are never translated.
 * The DMC publishes some bulletins in separate Sinhala and Tamil editions;
 * `bulletinEntries` pairs the editions of one bulletin so the reader's own
 * edition is shown first.
 */
import type { AlertSeverity, WeatherAlert } from "@/components/weather/types"
import { formatClock, formatDate, formatDateTime, formatDayTime } from "@/lib/format"
import { getLocale, translate, translateIn, type Locale } from "@/lib/i18n"
import type {
  BannerItem,
  Bulletin,
  BulletinSeverity,
  BulletinType,
  DistrictHighlight,
  LandslideHere,
  LandslideMeta,
  LandslideStatus,
  OfficialAt,
} from "./types"

export const landslideLevels = [3, 2, 1] as const
export type LandslideLevel = (typeof landslideLevels)[number]

/** "Level 2 (Amber)": NBRO's level as its reports print it, in the reader's language. */
export function levelLabel(level: LandslideLevel, locale: Locale = getLocale()): string {
  return translateIn(locale, `terms.landslideLevel.l${level}`)
}

/** NBRO's word for what to do at a level: Watch, Alert or Evacuate. */
export function levelAction(level: LandslideLevel, locale: Locale = getLocale()): string {
  return translateIn(locale, `terms.landslideAction.l${level}`)
}

/** "Level 2 (Amber) · Alert" */
export function levelWords(level: LandslideLevel, locale: Locale = getLocale()): string {
  return translateIn(locale, "warnings.landslide.levelAction", {
    level: levelLabel(level, locale),
    action: levelAction(level, locale),
  })
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

export function severityLabel(severity: BulletinSeverity): string {
  return translate(`terms.bulletinSeverity.${severity}`)
}

export const bulletinTypeOrder: BulletinType[] = [
  "landslide",
  "water_level",
  "weather",
  "situation",
]

/** The type's name in the reader's language, in place of the API's English `typeLabel`. */
export function bulletinTypeLabel(type: BulletinType): string {
  return translate(`terms.bulletinType.${type}`)
}

type BadgeTone = "alert" | "minor_flood" | "major_flood" | "normal" | "unknown" | "chip"

type ColourKey = "red" | "amber" | "yellow" | "orange" | "green"

const colourBadge: Record<ColourKey, BadgeTone> = {
  red: "major_flood",
  amber: "minor_flood",
  orange: "minor_flood",
  yellow: "alert",
  green: "normal",
}

function colourKey(colour: string | null | undefined): ColourKey | null {
  const key = colour?.trim().toLowerCase()
  return key && key in colourBadge ? (key as ColourKey) : null
}

/** The badge for a colour a bulletin states; the word is always on it. */
export function colourVariant(colour: string | null | undefined): BadgeTone {
  const key = colourKey(colour)
  return key ? colourBadge[key] : "unknown"
}

/** A stated colour in the reader's language: "Red" is "රතු" in Sinhala. Unknown words stay. */
export function colourWord(colour: string, locale: Locale = getLocale()): string {
  const key = colourKey(colour)
  return key ? translateIn(locale, `terms.colour.${key}`) : colour
}

/** A bulletin's badge: its stated colour, else how its title reads. */
export function bulletinBadge(
  bulletin: Pick<Bulletin, "severity" | "colour"> & Partial<Pick<Bulletin, "withdrawnBy">>,
): {
  variant: BadgeTone
  label: string
} {
  if (bulletin.withdrawnBy) return { variant: "chip", label: translate("warnings.withdrawn") }
  if (bulletin.severity === "routine") return { variant: "chip", label: severityLabel("routine") }
  if (bulletin.severity === "withdrawal") {
    return { variant: "normal", label: severityLabel("withdrawal") }
  }
  const word = severityLabel(bulletin.severity)
  if (bulletin.colour) {
    return {
      variant: colourVariant(bulletin.colour),
      label: translate("warnings.badgeColour", {
        severity: word,
        colour: colourWord(bulletin.colour),
      }),
    }
  }
  return { variant: bulletin.severity === "warning" ? "major_flood" : "alert", label: word }
}

/** A report's local date: "2026-10-31" is "31 Oct 2026". */
export function formatReportDate(date: string): string {
  return formatDate(date)
}

/** "1 Nov, 16:00" in Sri Lanka time. */
export function formatShort(iso: string): string {
  return formatDateTime(iso)
}

/** "1 Nov 2026 16:00", as the landslide notes give a time. */
function reportTime(iso: string, locale: Locale): string {
  return `${formatDate(iso, "short", locale)} ${formatClock(iso)}`
}

/** Reports that can show levels on the map. */
export function levelsShown(status: LandslideMeta["status"]): boolean {
  return status === "active" || status === "lapsed" || status === "pending"
}

/** Text and, when it is the server's English prose, its language. */
export interface Words {
  text: string
  lang?: "en"
}

interface ReportState {
  status: LandslideStatus
  stale: boolean
  reportDate: string | null
  validUntil: string | null
  /** The server's English sentence, used when the facts cannot say more. */
  note: string
}

/**
 * The sentence about the landslide report, from its status and dates. It says
 * what `warnings.Assess` says, in the reader's language; when the facts are not
 * all there, the server's English sentence is kept and marked.
 */
export function landslideStatusNote(
  report: ReportState,
  checkedAt?: string | null,
  locale: Locale = getLocale(),
): Words {
  const t = (key: Parameters<typeof translateIn>[1], values?: Record<string, string | number>) =>
    translateIn(locale, key, values)
  const server: Words = { text: report.note, lang: "en" }
  const date = report.reportDate ? formatDate(report.reportDate, "short", locale) : null
  const checked = checkedAt ? reportTime(checkedAt, locale) : null
  const staleTail = () =>
    report.stale
      ? ` ${checked ? t("warnings.status.staleSince", { checked }) : t("warnings.status.staleRecently")}`
      : ""
  switch (report.status) {
    case "waiting":
      return { text: t("warnings.status.waiting") }
    case "pending":
      // Only the server knows when the newer report was issued.
      return locale === "en" && report.note ? server : { text: t("warnings.status.pending") }
    case "active":
      if (!date || !report.validUntil) return server
      return {
        text:
          t("warnings.status.active", { date, until: reportTime(report.validUntil, locale) }) +
          staleTail(),
      }
    case "lapsed":
      if (!date || !report.validUntil) return server
      return {
        text:
          t("warnings.status.lapsed", { date, until: reportTime(report.validUntil, locale) }) +
          staleTail(),
      }
    case "unknown":
      if (!date) return server
      return {
        text: checked
          ? t("warnings.status.unknown", { date, checked })
          : t("warnings.status.unknownRecently", { date }),
      }
    case "none":
      if (!date) return server
      return { text: t("warnings.status.none", { date }) }
  }
  return server
}

/** What the feed cannot tell: that a level was ended early. */
export function landslideLimit(): string {
  return translate("warnings.status.limit")
}

/** One line for the layer list. */
export function landslideSummary(meta: LandslideMeta, shown: number): string {
  if (meta.status === "waiting") return translate("warnings.summary.waiting")
  if (shown > 0) {
    const parts = landslideLevels
      .filter((level) => meta.counts[level] > 0)
      .map((level) =>
        translate("warnings.summary.levelCount", { level, count: meta.counts[level] }),
      )
    return parts.join(translate("common.separator"))
  }
  if (meta.status === "none" && meta.reportDate) {
    return translate("warnings.summary.none", { date: formatReportDate(meta.reportDate) })
  }
  if (meta.status === "unknown") return translate("warnings.summary.unknown")
  return landslideStatusNote(meta, meta.fetchedAt).text
}

/** Banner items not dismissed yet. */
export function visibleBanner<T extends Pick<BannerItem, "key">>(
  items: T[],
  dismissed: string[],
): T[] {
  return items.filter((item) => !dismissed.includes(item.key))
}

/** The most serious item leads the banner: red before amber, landslide first on a tie. */
export function bannerLead<T extends Pick<BannerItem, "colour">>(items: T[]): T | undefined {
  const rank = (item: T) =>
    item.colour === "Red" ? 3 : item.colour === "Amber" ? 2 : item.colour === "Yellow" ? 1 : 0
  return [...items].sort((a, b) => rank(b) - rank(a))[0]
}

/** Dismissals only need to remember what is still being announced. */
export function pruneDismissed(dismissed: string[], items: Pick<BannerItem, "key">[]): string[] {
  const live = new Set(items.map((item) => item.key))
  return dismissed.filter((key) => live.has(key))
}

/** "Kandy District" */
export function districtName(name: string): string {
  return translate("warnings.area.district", { district: name })
}

/** Why a district is highlighted, in words for the hover chip. */
export function highlightLines(district: DistrictHighlight): string[] {
  const lines: string[] = []
  if (district.landslideLevel > 0) {
    lines.push(
      translate("warnings.highlight.landslide", {
        level: levelLabel(district.landslideLevel as LandslideLevel),
      }),
    )
  }
  if (district.mentioned.length > 0) {
    lines.push(translate("warnings.highlight.mentioned", { title: district.mentioned[0]!.title }))
    if (district.mentioned.length > 1) {
      lines.push(translate("warnings.highlight.more", { count: district.mentioned.length - 1 }))
    }
  }
  return lines
}

// Editions -------------------------------------------------------------------

/** Editions of one bulletin issued this far apart still pair. */
const editionWindow = 2 * 60 * 60 * 1000

/** Languages in the order editions anchor an entry. */
const editionOrder = ["en", "ta", "si"]

const languageWords =
  /\s*(?:[-–(]\s*)?\b(?:tamil|sinhala|sinhalese)(?:\s+(?:language|version|medium))?\b\s*\)?/gi

/**
 * A title reduced to what its editions share, as `warnings.sameWarning` does:
 * "Weather Report (Tamil Language)" and "Weather Report" are one.
 */
export function titleStem(title: string): string {
  return title
    .replace(languageWords, " ")
    .replace(/\bno\.?\s*\d+\b/gi, " ")
    .replace(/[-–()]/g, " ")
    .toLowerCase()
    .replace(/\s+/g, " ")
    .trim()
}

/**
 * The title as shown beside a language badge: "(Tamil Language)", "- Tamil" and
 * the like are dropped, since the badge says it. An English edition is shown as
 * published.
 */
export function editionTitle(bulletin: Pick<Bulletin, "title" | "language">): string {
  if (bulletin.language === "en") return bulletin.title
  const stripped = bulletin.title
    .replace(languageWords, " ")
    .replace(/\s+/g, " ")
    .replace(/[\s\-–:,]+$/, "")
    .trim()
  return stripped || bulletin.title
}

/** The language of a piece of the issuer's text, by its script. Titles are mostly English. */
export function scriptLang(text: string): Locale {
  if (/[඀-෿]/.test(text)) return "si"
  if (/[஀-௿]/.test(text)) return "ta"
  return "en"
}

function isEditionLanguage(language: string): language is Locale {
  return language === "en" || language === "si" || language === "ta"
}

export type EditionNote = "onlyEn" | "onlySi" | "onlyTa" | "missingSi" | "missingTa"

const onlyNote: Record<Locale, EditionNote> = { en: "onlyEn", si: "onlySi", ta: "onlyTa" }

/** One bulletin as the list shows it: the reader's edition first, the others as links. */
export interface BulletinEntry {
  /** A stable key: the first edition's id. */
  key: string
  /** The edition shown: the reader's language if published, else English, else the one there is. */
  shown: Bulletin
  /** The other editions, linked. */
  others: Bulletin[]
  /** Every edition, newest first. */
  editions: Bulletin[]
  /** Why the reader's edition is not the one shown, if it is not. */
  note: EditionNote | null
  /** The excerpt: the shown edition's, else the English edition's. Always English sentences. */
  text: string | null
  /** The excerpt comes from the English edition, not the one shown. */
  textFromEnglish: boolean
  /**
   * The shown edition with what its editions state between them: the colour,
   * period and districts are read from the English text, so a Tamil edition
   * borrows them.
   */
  merged: Bulletin
  /** The shown edition's title, ready to show beside its language badge. */
  title: string
}

function editionKey(bulletin: Bulletin): string {
  return `${bulletin.type}|${titleStem(bulletin.title)}`
}

function timeOf(bulletin: Bulletin): number {
  return new Date(bulletin.issuedAt).getTime()
}

/**
 * Groups bulletins into editions of one bulletin: same type, same title stem
 * and issued within two hours, at most one per language, nearest first. A
 * bulletin with no partner is an entry of its own, so nothing is hidden.
 */
export function groupEditions(bulletins: Bulletin[]): Bulletin[][] {
  const seen = new Set<string>()
  const unique = bulletins.filter((bulletin) => {
    if (seen.has(bulletin.id)) return false
    seen.add(bulletin.id)
    return true
  })
  const languages = [
    ...editionOrder,
    ...new Set(
      unique.map((bulletin) => bulletin.language).filter((l) => !editionOrder.includes(l)),
    ),
  ]
  const groups: Bulletin[][] = []
  for (const language of languages) {
    const items = unique.filter((bulletin) => bulletin.language === language)
    const pairs: { gap: number; item: Bulletin; group: Bulletin[] }[] = []
    for (const item of items) {
      for (const group of groups) {
        const anchor = group[0]!
        if (group.some((member) => member.language === language)) continue
        if (editionKey(anchor) !== editionKey(item)) continue
        const gap = Math.abs(timeOf(anchor) - timeOf(item))
        if (gap <= editionWindow) pairs.push({ gap, item, group })
      }
    }
    pairs.sort((a, b) => a.gap - b.gap)
    const placed = new Set<string>()
    const filled = new Set<Bulletin[]>()
    for (const { item, group } of pairs) {
      if (placed.has(item.id) || filled.has(group)) continue
      group.push(item)
      placed.add(item.id)
      filled.add(group)
    }
    for (const item of items) if (!placed.has(item.id)) groups.push([item])
  }
  for (const group of groups) group.sort((a, b) => timeOf(b) - timeOf(a))
  return groups
}

/** How one group of editions reads in a language. */
export function presentEditions(editions: Bulletin[], locale: Locale = getLocale()): BulletinEntry {
  const own = editions.find((bulletin) => bulletin.language === locale)
  const english = editions.find((bulletin) => bulletin.language === "en")
  const shown = own ?? english ?? editions[0]!
  const others = editions.filter((bulletin) => bulletin !== shown)

  // An English reader with no English edition but several others has the
  // language badges to go by.
  let note: EditionNote | null = null
  if (!own) {
    if (others.length === 0) {
      if (isEditionLanguage(shown.language)) note = onlyNote[shown.language]
    } else if (locale === "si") note = "missingSi"
    else if (locale === "ta") note = "missingTa"
  }

  const text = shown.text ?? english?.text ?? null
  const textFromEnglish = shown.text === null && english !== undefined && english.text !== null

  const pick = <K extends keyof Bulletin>(key: K): Bulletin[K] => {
    if (shown[key] !== null && shown[key] !== undefined) return shown[key]
    const other = [english, ...others].find(
      (bulletin) => bulletin && bulletin[key] !== null && bulletin[key] !== undefined,
    )
    return other ? other[key] : shown[key]
  }
  const districts = new Map<string, Bulletin["districts"][number]>()
  for (const bulletin of editions) {
    for (const district of bulletin.districts) districts.set(district.id, district)
  }
  const withdrawnBy = pick("withdrawnBy")
  const merged: Bulletin = {
    ...shown,
    colour: pick("colour"),
    messageId: pick("messageId"),
    validFrom: pick("validFrom"),
    validUntil: pick("validUntil"),
    withdrawnBy,
    mayBeWithdrawnBy: withdrawnBy ? null : pick("mayBeWithdrawnBy"),
    withdrawnBasins: pick("withdrawnBasins"),
    current: editions.some((bulletin) => bulletin.current),
    districts: [...districts.values()],
  }

  return {
    key: editions[editions.length - 1]!.id,
    shown,
    others,
    editions,
    note,
    text,
    textFromEnglish,
    merged,
    title: editionTitle(shown),
  }
}

/**
 * The bulletins to list, each with its other editions. `pool` is every
 * bulletin known (e.g. the unfiltered list), where editions left out of a
 * filtered list are found. Entries keep the order of `bulletins`.
 */
export function bulletinEntries(
  bulletins: Bulletin[],
  pool: Bulletin[] = [],
  locale: Locale = getLocale(),
): BulletinEntry[] {
  const order = new Map(bulletins.map((bulletin, index) => [bulletin.id, index]))
  return groupEditions([...bulletins, ...pool])
    .filter((group) => group.some((bulletin) => order.has(bulletin.id)))
    .map((group) => ({
      group,
      at: Math.min(...group.map((bulletin) => order.get(bulletin.id) ?? Infinity)),
    }))
    .sort((a, b) => a.at - b.at)
    .map(({ group }) => presentEditions(group, locale))
}

/** A banner item as the banner shows it, in the reader's language. */
export interface BannerView extends BannerItem {
  /** The language of the title, for its `lang`. */
  titleLang: Locale
  /** The detail is the server's English prose. */
  detailLang?: "en"
  /** The language of the edition announced, when it is a bulletin not in English. */
  edition?: Locale
}

/**
 * The banner in the reader's language: a bulletin's own edition when the DMC
 * published one, its type in the reader's words, and the landslide counts
 * from the banner's key ("landslide:<date>:<level 3>:<level 2>").
 */
export function bannerView(
  items: BannerItem[],
  pool: Bulletin[],
  locale: Locale = getLocale(),
): BannerView[] {
  const groups = groupEditions(pool)
  return items.map((item) => {
    if (item.kind === "landslide") {
      const [, , red, amber] = item.key.split(":")
      const counts = { 3: Number(red), 2: Number(amber) }
      if (!Number.isInteger(counts[3]) || !Number.isInteger(counts[2])) {
        return {
          ...item,
          title: translateIn(locale, "warnings.landslide.title"),
          titleLang: locale,
          detailLang: "en",
        }
      }
      const part = (level: 2 | 3) =>
        translateIn(locale, "warnings.banner.landslideDetail", {
          level: levelLabel(level, locale),
          count: counts[level],
        })
      const detail =
        counts[3] > 0 && counts[2] > 0
          ? translateIn(locale, "warnings.banner.landslideDetailMore", {
              detail: part(3),
              level: levelLabel(2, locale),
              count: counts[2],
            })
          : part(counts[3] > 0 ? 3 : 2)
      return {
        ...item,
        title: translateIn(locale, "warnings.landslide.title"),
        titleLang: locale,
        detail,
      }
    }
    const id = item.key.startsWith("bulletin:") ? item.key.slice("bulletin:".length) : item.key
    const group = groups.find((editions) => editions.some((bulletin) => bulletin.id === id))
    if (!group) return { ...item, titleLang: scriptLang(item.title), detailLang: "en" }
    const entry = presentEditions(group, locale)
    const shown = entry.shown
    return {
      ...item,
      title: entry.title,
      titleLang: scriptLang(entry.title),
      detail: translateIn(locale, `terms.bulletinType.${shown.type}`),
      colour: item.colour ?? entry.merged.colour,
      pdfUrl: shown.pdfUrl || item.pdfUrl,
      edition:
        isEditionLanguage(shown.language) && shown.language !== "en" ? shown.language : undefined,
    }
  })
}

// The location panel ------------------------------------------------------------

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

function issuer(type: BulletinType): string {
  return type === "landslide" ? translate("warnings.issuer.nbroViaDmc") : "DMC"
}

/** A bulletin as the alert list shows it: the issuer's title and words, with its link. */
export function bulletinAlert(bulletin: Bulletin, area: string): WeatherAlert {
  const badge = bulletinBadge(bulletin)
  return {
    id: bulletin.id,
    // As published: the alert list shows no language badge, so "(Tamil Language)" stays.
    title: bulletin.title,
    severity: bulletin.colour === "Red" ? "emergency" : severityFor[bulletin.severity],
    badgeLabel: badge.label,
    issuer: translate("warnings.issuer.withType", {
      issuer: issuer(bulletin.type),
      type: bulletinTypeLabel(bulletin.type),
    }),
    area,
    issuedAt: bulletin.issuedAt,
    validUntil: bulletin.validUntil ?? undefined,
    summary: bulletin.text ?? undefined,
    excerpt: bulletin.text !== null,
    link: { href: bulletin.pdfUrl, label: translate("warnings.officialPdf") },
  }
}

/** The landslide level of the place's DS division as the alert list shows it, if it has one. */
export function landslideAlert(here: LandslideHere | null | undefined): WeatherAlert | null {
  if (!here?.level) return null
  const level = here.level.level
  return {
    id: `landslide-${here.dsId}`,
    title: translate("warnings.landslide.title"),
    severity: levelSeverity[level],
    badgeLabel: levelWords(level),
    issuer: translate("warnings.issuer.nbroViaDmc"),
    area:
      here.ds && here.district
        ? translate("warnings.area.dsDistrict", { ds: here.ds, district: here.district })
        : here.district
          ? districtName(here.district)
          : (here.ds ?? ""),
    issuedAt: here.issuedAt ?? here.reportDate ?? "",
    validUntil: here.validUntil ?? undefined,
    link: here.pdfUrl
      ? { href: here.pdfUrl, label: translate("warnings.officialReport") }
      : undefined,
  }
}

/**
 * What to say about landslides here when the division has no level. It says
 * only what is known: not being named is not a statement of safety.
 */
export function landslideNote(here: LandslideHere | null | undefined): string | undefined {
  if (!here) return undefined
  if (here.level) return landslideStatusNote(here).text
  if ((here.status === "active" || here.status === "lapsed") && here.reportDate) {
    const date = formatReportDate(here.reportDate)
    return here.ds
      ? translate("warnings.status.notNamed", { ds: here.ds, date })
      : translate("warnings.status.notNamedHere", { date })
  }
  return landslideStatusNote(here).text
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
  const place = districtName
    ? translate("warnings.area.mentionedIn", { names: districtName })
    : translate("warnings.area.mentioned")
  for (const bulletin of official.warnings) {
    if (bulletin.current) alerts.push(bulletinAlert(bulletin, place))
    else bulletins.push(bulletinAlert(bulletin, place))
  }
  const country = translate("warnings.area.sriLanka")
  for (const bulletin of official.general) {
    if (bulletin.current) alerts.push(bulletinAlert(bulletin, country))
    else bulletins.push(bulletinAlert(bulletin, country))
  }
  return {
    alerts,
    bulletins,
    note: landslideNote(official.landslide) ?? translate("warnings.panel.sourceNote"),
  }
}

/** "Issued Mon 3 Nov · 16:00" */
export function issuedText(iso: string): string {
  return translate("common.issuedAt", { time: formatDayTime(iso) })
}
