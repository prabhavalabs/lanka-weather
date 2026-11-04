import { ChevronRightIcon, ExternalLinkIcon, LandmarkIcon, XIcon } from "lucide-react"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { formatAge } from "@/lib/format"
import { useLocale, useT, type Locale } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { groupLabel, groupOrder, sectionTag, shownCounts, shownItems, toneBadge } from "./present"
import { detailIn, proseIn, statusIn, type Prose } from "./prose"
import type { Overview, OverviewItem, OverviewSection, OverviewTarget } from "./types"

function Age({ at }: { at: string | null }) {
  const t = useT()
  const locale = useLocale()
  return (
    <span className="numeric">
      {at ? formatAge(at, undefined, locale) : t("overview.panel.timeNotKnown")}
    </span>
  )
}

/** The API's words in the language; what stays in English says so. */
function Words({ prose }: { prose: Prose }) {
  return prose.lang ? <span lang={prose.lang}>{prose.text}</span> : <>{prose.text}</>
}

/** Several parts, joined as the API joined them. */
function Parts({ parts, between }: { parts: Prose[]; between: string }) {
  return parts.map((part, index) => (
    <span key={index}>
      {index > 0 && between}
      <Words prose={part} />
    </span>
  ))
}

/** Text from the API in the language, as one string (for attributes and badges). */
function plain(locale: Locale, text: string) {
  return proseIn(locale, text).text
}

function Line({ item, onOpen }: { item: OverviewItem; onOpen: (target: OverviewTarget) => void }) {
  const t = useT()
  const locale = useLocale()
  const separator = t("common.separator")
  const body = (
    <>
      <span className="min-w-0 flex-1">
        <span className="block text-sm break-words text-foreground">
          <Words prose={proseIn(locale, item.title)} />
        </span>
        {item.detail && (
          <span className="block text-xs break-words text-muted-foreground">
            <Parts parts={detailIn(locale, item.detail)} between=" · " />
          </span>
        )}
        {(item.badge || item.estimate) && (
          // Under the title rather than beside it: Sinhala and Tamil badges are long, and
          // beside the title they would squeeze it into a narrow column.
          <span className="mt-1 flex flex-wrap gap-1">
            {item.badge && (
              <Badge variant={toneBadge[item.tone]} className="h-auto max-w-full whitespace-normal">
                <Words prose={proseIn(locale, item.badge)} />
              </Badge>
            )}
            {item.estimate && (
              <Badge variant="chip" className="h-5 px-1.5 text-[0.6875rem] text-foreground">
                {t("terms.estimate")}
              </Badge>
            )}
          </span>
        )}
        <span className="mt-0.5 block text-[0.6875rem] break-words text-muted-foreground">
          <Age at={item.at} />
          {separator}
          <Words prose={proseIn(locale, item.source)} />
          {item.stale && `${separator}${t("overview.panel.lineOutOfDate")}`}
        </span>
      </span>
      {item.target && (
        <ChevronRightIcon
          className="mt-0.5 size-4 shrink-0 text-muted-foreground"
          aria-hidden="true"
        />
      )}
    </>
  )
  return (
    <li>
      {item.target ? (
        <button
          type="button"
          onClick={() => onOpen(item.target!)}
          className="flex w-full items-start gap-2.5 px-3 py-2 text-left outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
        >
          {body}
        </button>
      ) : (
        <div className="flex items-start gap-2.5 px-3 py-2">{body}</div>
      )}
      {item.href && (
        <a
          href={item.href}
          target="_blank"
          rel="noreferrer"
          className="mb-2 ml-3 inline-flex items-center gap-1 rounded-sm text-xs font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
        >
          {t("overview.panel.officialPdf")}
          <ExternalLinkIcon className="size-3" aria-hidden="true" />
          <span className="sr-only">{t("overview.panel.newTab")}</span>
        </a>
      )}
    </li>
  )
}

function Section({
  section,
  onOpen,
}: {
  section: OverviewSection
  onOpen: (target: OverviewTarget) => void
}) {
  const t = useT()
  const locale = useLocale()
  const separator = t("common.separator")
  const tag = sectionTag(section, locale)
  const { items, more } = shownItems(section)
  const counts = shownCounts(section)
  const headingId = `overview-${section.id}`
  return (
    <section aria-labelledby={headingId} className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-1.5">
        {section.target ? (
          <button
            type="button"
            id={headingId}
            onClick={() => onOpen(section.target!)}
            className="rounded-sm text-left text-sm font-medium text-foreground underline-offset-2 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
          >
            <Words prose={proseIn(locale, section.title)} />
          </button>
        ) : (
          <h4 id={headingId} className="text-sm font-medium text-foreground">
            <Words prose={proseIn(locale, section.title)} />
          </h4>
        )}
        {tag && (
          <Badge
            variant={section.official ? "brand" : "chip"}
            className="h-5 px-1.5 text-[0.6875rem]"
          >
            {tag}
          </Badge>
        )}
        {(section.stale || !section.available) && (
          <Badge variant="chip" className="h-5 px-1.5 text-[0.6875rem] text-foreground">
            {section.available ? t("common.outOfDate") : t("overview.panel.notRead")}
          </Badge>
        )}
      </div>
      <p className="text-xs text-foreground/90">
        <Words prose={proseIn(locale, section.summary)} />
      </p>
      {counts.length > 0 && (
        <ul className="numeric flex flex-wrap gap-x-2.5 gap-y-0.5 text-[0.6875rem] text-muted-foreground">
          {counts.map((count) => (
            <li key={count.key}>
              {plain(locale, count.label)} <span className="text-foreground">{count.value}</span>
            </li>
          ))}
        </ul>
      )}
      {items.length > 0 && (
        <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border">
          {items.map((item) => (
            <Line key={item.id} item={item} onOpen={onOpen} />
          ))}
        </ul>
      )}
      {more > 0 && section.target && (
        <Button
          variant="ghost"
          size="sm"
          className="self-start"
          onClick={() => onOpen(section.target!)}
        >
          {t("overview.panel.more", { count: more })}
          <ChevronRightIcon />
        </Button>
      )}
      <p className="text-[0.6875rem] text-muted-foreground">
        {section.available && (
          <>
            <UpdatedAge at={section.at} />
            {separator}
          </>
        )}
        <Words prose={proseIn(locale, section.source)} />
        {section.note && (
          <>
            {separator}
            <Words prose={proseIn(locale, section.note)} />
          </>
        )}
      </p>
    </section>
  )
}

/** "Updated 5 min ago", with the age where the language puts it. */
function UpdatedAge({ at }: { at: string | null }) {
  const t = useT()
  const [before, after] = t("overview.panel.updated").split("{age}")
  return (
    <>
      {before}
      <Age at={at} />
      {after}
    </>
  )
}

/** "Composed 5 min ago". */
function ComposedAge({ at }: { at: string }) {
  const t = useT()
  const [before, after] = t("overview.panel.composed").split("{age}")
  return (
    <>
      {before}
      <Age at={at} />
      {after}
    </>
  )
}

interface OverviewPanelProps {
  overview: Overview | undefined
  pending: boolean
  failed: boolean
  onOpen: (target: OverviewTarget) => void
  onClose: () => void
  className?: string
}

/**
 * Sri Lanka now: a first look composed by fixed rules, official warnings first.
 * Each line shows its age and source, and opens its layer and panel.
 */
export function OverviewPanel({
  overview,
  pending,
  failed,
  onOpen,
  onClose,
  className,
}: OverviewPanelProps) {
  const t = useT()
  const locale = useLocale()
  const [failedBefore, failedAfter] = t("overview.panel.failed").split("{link}")
  const [footerBefore, footerAfter] = t("overview.panel.footer").split("{estimate}")
  return (
    <Panel
      icon={<LandmarkIcon />}
      title={t("overview.panel.title")}
      subtitle={overview ? <ComposedAge at={overview.generatedAt} /> : t("overview.panel.subtitle")}
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("overview.panel.close")}
        >
          <XIcon />
        </Button>
      }
    >
      {pending && !overview && (
        <div className="flex flex-col gap-2" role="status" aria-label={t("overview.panel.loading")}>
          <Skeleton className="h-14 w-full rounded-xl" />
          <Skeleton className="h-24 w-full rounded-xl" />
        </div>
      )}
      {failed && !overview && (
        <p role="alert" className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm">
          {failedBefore}
          <a className="underline" href="https://www.dmc.gov.lk/" target="_blank" rel="noreferrer">
            dmc.gov.lk
          </a>
          {failedAfter}
        </p>
      )}
      {overview && (
        <>
          <p
            role="status"
            className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm leading-snug font-medium text-foreground"
          >
            <Parts parts={statusIn(locale, overview.status)} between=" " />
          </p>
          {groupOrder.map((group) => {
            const sections = overview.sections.filter((section) => section.group === group)
            if (sections.length === 0) return null
            return (
              <div key={group} className="flex flex-col gap-3">
                <h3 className="text-xs font-medium text-muted-foreground">
                  {groupLabel(group, locale)}
                </h3>
                {sections.map((section) => (
                  <Section key={section.id} section={section} onOpen={onOpen} />
                ))}
              </div>
            )
          })}
          <p className="text-xs text-muted-foreground">
            {footerBefore}
            {t("terms.estimate")}
            {footerAfter}
          </p>
        </>
      )}
    </Panel>
  )
}
