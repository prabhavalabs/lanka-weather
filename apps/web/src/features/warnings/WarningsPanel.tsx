import { BellIcon, ExternalLinkIcon, XIcon } from "lucide-react"
import { useEffect, useMemo, useRef, type ReactNode } from "react"
import { Panel } from "@/components/Panel"
import { ScrollRow } from "@/components/ScrollRow"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { formatAge, formatDayTime } from "@/lib/format"
import { isLocale, useLocale, useT } from "@/lib/i18n"
import type { Layer } from "@/lib/layers"
import {
  bulletinBadge,
  bulletinEntries,
  bulletinTypeLabel,
  bulletinTypeOrder,
  formatReportDate,
  landslideLevels,
  landslideLimit,
  landslideStatusNote,
  levelLabel,
  scriptLang,
  type BulletinEntry,
} from "./present"
import { useWarnings } from "./queries"
import { Sentence } from "./Sentence"
import type { BulletinType, LandslideMeta } from "./types"

interface WarningsPanelProps {
  layer: Layer | undefined
  type?: string
  district?: string
  /** The district's name, when filtering by one. */
  districtName?: string
  onFilter: (next: { type?: string; district?: string }) => void
  onClose: () => void
  /** The official / alerts switch over the panel. */
  tabs?: ReactNode
}

function OfficialLink({ href, children }: { href: string; children: ReactNode }) {
  const t = useT()
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="inline-flex items-center gap-1 rounded-sm text-xs font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
    >
      {children}
      <ExternalLinkIcon className="size-3" aria-hidden="true" />
      <span className="sr-only">{t("warnings.newTab")}</span>
    </a>
  )
}

function DayTime({ iso }: { iso: string }) {
  return <time dateTime={iso}>{formatDayTime(iso)}</time>
}

const dmcSite = <OfficialLink href="https://www.dmc.gov.lk/">dmc.gov.lk</OfficialLink>

function BulletinItem({ entry }: { entry: BulletinEntry }) {
  const t = useT()
  const bulletin = entry.merged
  const badge = bulletinBadge(bulletin)
  const language = isLocale(entry.shown.language) ? entry.shown.language : null
  return (
    <li className="flex flex-col gap-1.5 px-3 py-3">
      <div className="flex items-start justify-between gap-3">
        <p
          lang={scriptLang(entry.title)}
          className="min-w-0 text-sm leading-snug font-medium break-words text-foreground"
        >
          {entry.title}
        </p>
        <Badge variant={badge.variant} className="mt-px max-w-[50%] shrink-0 whitespace-normal">
          {badge.label}
        </Badge>
      </div>
      <p className="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs text-muted-foreground">
        <span>{bulletinTypeLabel(bulletin.type)}</span>
        {language && language !== "en" && (
          <Badge variant="chip" className="h-5 text-foreground">
            {t(`warnings.language.${language}`)}
          </Badge>
        )}
        {bulletin.messageId && (
          <span className="numeric">
            {t("common.separator").trim()} {bulletin.messageId}
          </span>
        )}
      </p>
      {entry.note && (
        <p className="text-xs text-foreground">{t(`warnings.edition.${entry.note}`)}</p>
      )}
      <p className="numeric text-[0.6875rem] text-muted-foreground">
        <DayTime iso={entry.shown.issuedAt} />
        {t("common.separator")}
        {formatAge(entry.shown.issuedAt)}
        {bulletin.validUntil && (
          <>
            {t("common.separator")}
            <Sentence
              text={t("warnings.bulletin.validUntil")}
              values={{ time: <DayTime iso={bulletin.validUntil} /> }}
            />
          </>
        )}
      </p>
      {bulletin.withdrawnBy && (
        <p className="text-xs text-foreground">
          <Sentence
            text={t("warnings.bulletin.withdrawnBy")}
            values={{ time: <DayTime iso={bulletin.withdrawnBy.issuedAt} /> }}
          />{" "}
          <OfficialLink href={bulletin.withdrawnBy.pdfUrl}>
            {t("warnings.bulletin.withdrawalNotice")}
          </OfficialLink>
        </p>
      )}
      {bulletin.mayBeWithdrawnBy && (
        <p className="rounded-lg border border-border bg-inset px-2.5 py-2 text-xs text-foreground">
          <Sentence
            text={t("warnings.bulletin.mayBeWithdrawn")}
            values={{ time: <DayTime iso={bulletin.mayBeWithdrawnBy.issuedAt} /> }}
          />{" "}
          <Sentence
            text={t("warnings.bulletin.readOrCheck")}
            values={{
              notice: (
                <OfficialLink href={bulletin.mayBeWithdrawnBy.pdfUrl}>
                  {t("warnings.bulletin.readNotice")}
                </OfficialLink>
              ),
              site: dmcSite,
            }}
          />
        </p>
      )}
      {bulletin.severity === "withdrawal" && bulletin.withdrawnBasins && (
        <p className="text-xs text-foreground">
          {bulletin.withdrawnBasins.length > 0 ? (
            <Sentence
              text={t("warnings.bulletin.withdraws")}
              values={{ basins: <span lang="en">{bulletin.withdrawnBasins.join(", ")}</span> }}
            />
          ) : (
            t("warnings.bulletin.basinsUnread")
          )}
        </p>
      )}
      {entry.text && (
        <details className="group">
          <summary className="cursor-pointer rounded-sm text-xs text-muted-foreground outline-none select-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring">
            {t("warnings.bulletin.readExcerpt")}
          </summary>
          <p className="mt-1.5 text-[0.6875rem] text-muted-foreground">
            {t("warnings.bulletin.excerptNote")}
            {entry.textFromEnglish && <> {t("warnings.edition.englishText")}</>}
          </p>
          <div
            lang="en"
            className="mt-1.5 flex flex-col gap-1.5 rounded-lg bg-inset px-2.5 py-2 text-sm break-words text-foreground/90"
          >
            {entry.text.split("\n\n").map((paragraph, index) => (
              <p key={index}>{paragraph}</p>
            ))}
          </div>
        </details>
      )}
      {bulletin.districts.length > 0 && (
        <p className="text-xs text-muted-foreground">
          {t("warnings.area.mentionedIn", {
            names: bulletin.districts.map((district) => district.name).join(", "),
          })}
        </p>
      )}
      <p className="flex flex-wrap gap-x-3 gap-y-1">
        <OfficialLink href={entry.shown.pdfUrl}>{t("warnings.officialPdf")}</OfficialLink>
        {entry.others.map((other) =>
          isLocale(other.language) ? (
            <OfficialLink key={other.id} href={other.pdfUrl}>
              {t(`warnings.edition.link.${other.language}`)}
            </OfficialLink>
          ) : null,
        )}
        <OfficialLink href={entry.shown.sourcePageUrl}>{t("warnings.dmcPage")}</OfficialLink>
      </p>
    </li>
  )
}

function LandslideStatusCard({ meta }: { meta: LandslideMeta }) {
  const t = useT()
  const hasLevels = meta.counts["1"] + meta.counts["2"] + meta.counts["3"] > 0
  const note = landslideStatusNote(meta, meta.fetchedAt)
  return (
    <section
      aria-label={t("warnings.landslide.title")}
      className="flex flex-col gap-1.5 rounded-xl border border-border bg-inset px-3 py-2.5"
    >
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-xs font-medium text-muted-foreground">
          {t("warnings.landslide.title")}
        </h3>
        {meta.stale && (
          <Badge variant="chip" className="h-5 text-foreground">
            {t("common.outOfDate")}
          </Badge>
        )}
      </div>
      <p className="text-sm text-foreground" lang={note.lang}>
        {note.text}
      </p>
      {meta.reportDate && hasLevels && (
        <p className="numeric text-xs text-muted-foreground">
          {t("warnings.landslide.reportCounts", {
            date: formatReportDate(meta.reportDate),
            counts: landslideLevels
              .map((level) =>
                t("warnings.landslide.levelCount", {
                  level: levelLabel(level),
                  count: meta.counts[level],
                }),
              )
              .join(t("common.separator")),
          })}
        </p>
      )}
      <p className="text-xs text-muted-foreground">{landslideLimit()}</p>
      <p className="flex flex-wrap gap-x-3 gap-y-1">
        {meta.pdfUrl && (
          <OfficialLink href={meta.pdfUrl}>{t("warnings.officialReport")}</OfficialLink>
        )}
        {dmcSite}
      </p>
    </section>
  )
}

/**
 * Official bulletins, newest first, in the issuers' words, each linking to its
 * official PDF. Reads its own filtered list; the header's count comes from the
 * unfiltered one.
 */
export function WarningsPanel({
  layer,
  type,
  district,
  districtName,
  onFilter,
  onClose,
  tabs,
}: WarningsPanelProps) {
  const t = useT()
  const locale = useLocale()
  const { data, isPending, error } = useWarnings(layer, { type, district })
  // The unfiltered list, already loaded for the header, holds the other
  // editions of a bulletin that a filter leaves out (Tamil editions name no
  // district, for one).
  const everything = useWarnings(layer)
  const bulletins = data?.bulletins ?? []
  const pool = everything.data?.bulletins
  const entries = useMemo(
    () => bulletinEntries(data?.bulletins ?? [], pool, locale),
    [data?.bulletins, pool, locale],
  )
  const inForce = new Set((data?.inForce ?? []).map((bulletin) => bulletin.id))
  // Warnings in force first, then everything else newest first. An entry is in
  // force when any of its editions is.
  const isInForce = (entry: BulletinEntry) =>
    entry.editions.some((bulletin) => inForce.has(bulletin.id))
  const current = entries.filter(isInForce)
  const rest = entries.filter((entry) => !isInForce(entry))

  // Bring the chosen filter into view when it sits past the edge of its row,
  // e.g. when a link opens the panel on one. Only the row moves, not the panel.
  const filterRow = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const row = filterRow.current
    const item = row?.querySelector<HTMLElement>('[data-state="on"]')
    if (!row || !item) return
    const box = item.getBoundingClientRect()
    const origin = row.getBoundingClientRect().left - row.scrollLeft
    const left = box.left - origin - 8
    const right = box.right - origin + 8
    if (left < row.scrollLeft) row.scrollLeft = left
    else if (right > row.scrollLeft + row.clientWidth) row.scrollLeft = right - row.clientWidth
  }, [type])

  return (
    <Panel
      icon={<BellIcon />}
      title={t("warnings.panel.title")}
      subtitle={t("warnings.panel.subtitle")}
      className="w-full"
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("warnings.panel.close")}
        >
          <XIcon />
        </Button>
      }
    >
      {tabs}
      {data && <LandslideStatusCard meta={data.landslide} />}

      {/* One row that scrolls sideways: the names stay whole on a narrow panel. */}
      <ScrollRow
        className="shrink-0 overflow-hidden rounded-xl border border-border bg-muted"
        viewportRef={filterRow}
      >
        <ToggleGroup
          type="single"
          size="sm"
          value={type ?? "all"}
          onValueChange={(value) => {
            if (value) onFilter({ type: value === "all" ? undefined : value, district })
          }}
          aria-label={t("warnings.panel.filterLabel")}
          className="w-max min-w-full rounded-none border-0 bg-transparent p-1"
        >
          <ToggleGroupItem value="all" className="flex-auto px-2.5">
            {t("warnings.panel.all")}
          </ToggleGroupItem>
          {bulletinTypeOrder.map((value: BulletinType) => (
            <ToggleGroupItem key={value} value={value} className="flex-auto px-2.5 text-xs">
              {bulletinTypeLabel(value)}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </ScrollRow>

      {district && (
        <div className="flex items-center justify-between gap-2 rounded-xl border border-border bg-inset px-3 py-2 text-sm">
          <span className="min-w-0 break-words">
            {t("warnings.area.mentionedIn", { names: districtName ?? district })}
          </span>
          <Button variant="ghost" size="sm" className="shrink-0" onClick={() => onFilter({ type })}>
            {t("warnings.panel.showAll")}
          </Button>
        </div>
      )}

      {isPending && (
        <div className="flex flex-col gap-2" role="status" aria-label={t("warnings.panel.loading")}>
          <Skeleton className="h-20 w-full rounded-xl" />
          <Skeleton className="h-20 w-full rounded-xl" />
        </div>
      )}
      {error != null && !data && (
        <p role="alert" className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm">
          <Sentence
            text={t("warnings.panel.error")}
            values={{
              site: (
                <a
                  className="underline"
                  href="https://www.dmc.gov.lk/"
                  target="_blank"
                  rel="noreferrer"
                >
                  dmc.gov.lk
                </a>
              ),
            }}
          />
        </p>
      )}
      {data && bulletins.length === 0 && (
        <p className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm text-muted-foreground">
          {data.meta.fetchedAt ? t("warnings.panel.noMatch") : t("warnings.panel.notCollected")}
        </p>
      )}
      {current.length > 0 && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">
            {t("warnings.panel.inForce")}
          </h3>
          <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border">
            {current.map((entry) => (
              <BulletinItem key={entry.key} entry={entry} />
            ))}
          </ul>
        </section>
      )}
      {rest.length > 0 && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">
            {t("warnings.panel.latest")}
          </h3>
          <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border">
            {rest.map((entry) => (
              <BulletinItem key={entry.key} entry={entry} />
            ))}
          </ul>
        </section>
      )}

      {data && (
        <div className="flex flex-col gap-1 text-xs text-muted-foreground">
          <p>
            {t("warnings.panel.sourceNote")}{" "}
            <span className="numeric">
              {data.meta.upstreamAt
                ? t("warnings.panel.updatedRebuilt", {
                    age: formatAge(data.meta.fetchedAt),
                    rebuilt: formatAge(data.meta.upstreamAt),
                  })
                : t("warnings.panel.updated", { age: formatAge(data.meta.fetchedAt) })}
            </span>
          </p>
          {data.meta.stale && (
            <Badge variant="chip" className="h-5 text-foreground">
              {t("common.outOfDate")}
            </Badge>
          )}
          <p>{t("warnings.panel.meteoNote")}</p>
        </div>
      )}
    </Panel>
  )
}
