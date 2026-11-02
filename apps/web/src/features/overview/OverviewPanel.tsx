import { ChevronRightIcon, ExternalLinkIcon, LandmarkIcon, XIcon } from "lucide-react"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { formatAge } from "@/lib/format"
import { cn } from "@/lib/utils"
import { groupLabels, groupOrder, sectionTag, shownCounts, shownItems, toneBadge } from "./present"
import type { Overview, OverviewItem, OverviewSection, OverviewTarget } from "./types"

function Age({ at }: { at: string | null }) {
  return <span className="numeric">{at ? formatAge(at) : "time not known"}</span>
}

function Line({ item, onOpen }: { item: OverviewItem; onOpen: (target: OverviewTarget) => void }) {
  const body = (
    <>
      <span className="min-w-0 flex-1">
        <span className="block text-sm text-foreground">{item.title}</span>
        {item.detail && (
          <span className="block text-xs break-words text-muted-foreground">{item.detail}</span>
        )}
        <span className="mt-0.5 block text-[0.6875rem] text-muted-foreground">
          <Age at={item.at} /> · {item.source}
          {item.stale && " · out of date"}
        </span>
      </span>
      <span className="flex shrink-0 flex-col items-end gap-1">
        {item.badge && <Badge variant={toneBadge[item.tone]}>{item.badge}</Badge>}
        {item.estimate && (
          <Badge variant="chip" className="h-5 px-1.5 text-[0.6875rem] text-foreground">
            Estimate
          </Badge>
        )}
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
          Official PDF
          <ExternalLinkIcon className="size-3" aria-hidden="true" />
          <span className="sr-only">(opens in a new tab)</span>
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
  const tag = sectionTag(section)
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
            {section.title}
          </button>
        ) : (
          <h4 id={headingId} className="text-sm font-medium text-foreground">
            {section.title}
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
            {section.available ? "Out of date" : "Not read"}
          </Badge>
        )}
      </div>
      <p className="text-xs text-foreground/90">{section.summary}</p>
      {counts.length > 0 && (
        <ul className="numeric flex flex-wrap gap-x-2.5 gap-y-0.5 text-[0.6875rem] text-muted-foreground">
          {counts.map((count) => (
            <li key={count.key}>
              {count.label} <span className="text-foreground">{count.value}</span>
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
          {more} more
          <ChevronRightIcon />
        </Button>
      )}
      <p className="text-[0.6875rem] text-muted-foreground">
        {section.available ? (
          <>
            Updated <Age at={section.at} /> · {section.source}
          </>
        ) : (
          section.source
        )}
        {section.note && <> · {section.note}</>}
      </p>
    </section>
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
  return (
    <Panel
      icon={<LandmarkIcon />}
      title="Sri Lanka now"
      subtitle={
        overview ? (
          <>
            Composed <Age at={overview.generatedAt} />
          </>
        ) : (
          "What the official and open feeds say"
        )
      }
      className={cn("w-full", className)}
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label="Close Sri Lanka now">
          <XIcon />
        </Button>
      }
    >
      {pending && !overview && (
        <div className="flex flex-col gap-2" role="status" aria-label="Loading the overview">
          <Skeleton className="h-14 w-full rounded-xl" />
          <Skeleton className="h-24 w-full rounded-xl" />
        </div>
      )}
      {failed && !overview && (
        <p role="alert" className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm">
          The overview could not be loaded, so nothing can be said from here. Check with the
          Disaster Management Centre,{" "}
          <a className="underline" href="https://www.dmc.gov.lk/" target="_blank" rel="noreferrer">
            dmc.gov.lk
          </a>
          , or call 117 in an emergency.
        </p>
      )}
      {overview && (
        <>
          <p
            role="status"
            className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm leading-snug font-medium text-foreground"
          >
            {overview.status}
          </p>
          {groupOrder.map((group) => {
            const sections = overview.sections.filter((section) => section.group === group)
            if (sections.length === 0) return null
            return (
              <div key={group} className="flex flex-col gap-3">
                <h3 className="text-xs font-medium text-muted-foreground">{groupLabels[group]}</h3>
                {sections.map((section) => (
                  <Section key={section.id} section={section} onOpen={onOpen} />
                ))}
              </div>
            )
          })}
          <p className="text-xs text-muted-foreground">
            Official warnings are authoritative. Everything marked Estimate is ours. In an emergency
            call 117.
          </p>
        </>
      )}
    </Panel>
  )
}
