import { BellIcon, ExternalLinkIcon, XIcon } from "lucide-react"
import type { ReactNode } from "react"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { formatAge, formatDayTime } from "@/lib/format"
import type { Layer } from "@/lib/layers"
import { bulletinBadge, bulletinTypeLabels, bulletinTypeOrder, formatReportDate } from "./present"
import { useWarnings } from "./queries"
import type { Bulletin, BulletinType, LandslideMeta } from "./types"

interface WarningsPanelProps {
  layer: Layer | undefined
  type?: string
  district?: string
  /** The district's name, when filtering by one. */
  districtName?: string
  onFilter: (next: { type?: string; district?: string }) => void
  onClose: () => void
}

function OfficialLink({ href, children }: { href: string; children: ReactNode }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="inline-flex items-center gap-1 rounded-sm text-xs font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
    >
      {children}
      <ExternalLinkIcon className="size-3" aria-hidden="true" />
      <span className="sr-only">(opens in a new tab)</span>
    </a>
  )
}

function BulletinItem({ bulletin }: { bulletin: Bulletin }) {
  const badge = bulletinBadge(bulletin)
  return (
    <li className="flex flex-col gap-1.5 px-3 py-3">
      <div className="flex items-start justify-between gap-3">
        <p className="min-w-0 text-sm leading-snug font-medium text-foreground">{bulletin.title}</p>
        <Badge variant={badge.variant} className="mt-px shrink-0">
          {badge.label}
        </Badge>
      </div>
      <p className="text-xs text-muted-foreground">
        {bulletin.typeLabel}
        {bulletin.language === "ta" && " · Tamil"}
        {bulletin.messageId && <span className="numeric"> · {bulletin.messageId}</span>}
      </p>
      <p className="numeric text-[0.6875rem] text-muted-foreground">
        <time dateTime={bulletin.issuedAt}>{formatDayTime(bulletin.issuedAt)}</time>
        {" · "}
        {formatAge(bulletin.issuedAt)}
        {bulletin.validUntil && (
          <>
            {" · valid until "}
            <time dateTime={bulletin.validUntil}>{formatDayTime(bulletin.validUntil)}</time>
          </>
        )}
      </p>
      {bulletin.text && (
        <details className="group">
          <summary className="cursor-pointer rounded-sm text-xs text-muted-foreground outline-none select-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring">
            Read an excerpt
          </summary>
          <p className="mt-1.5 text-[0.6875rem] text-muted-foreground">
            English sentences copied unchanged from the PDF. Tables and maps are left out, so read
            the PDF for the whole bulletin.
          </p>
          <div className="mt-1.5 flex flex-col gap-1.5 rounded-lg bg-inset px-2.5 py-2 text-sm text-foreground/90">
            {bulletin.text.split("\n\n").map((paragraph, index) => (
              <p key={index}>{paragraph}</p>
            ))}
          </div>
        </details>
      )}
      {bulletin.districts.length > 0 && (
        <p className="text-xs text-muted-foreground">
          Mentioned in bulletin: {bulletin.districts.map((district) => district.name).join(", ")}
        </p>
      )}
      <p className="flex flex-wrap gap-x-3 gap-y-1">
        <OfficialLink href={bulletin.pdfUrl}>Official PDF</OfficialLink>
        <OfficialLink href={bulletin.sourcePageUrl}>DMC page</OfficialLink>
      </p>
    </li>
  )
}

function LandslideStatusCard({ meta }: { meta: LandslideMeta }) {
  const hasLevels = meta.counts["1"] + meta.counts["2"] + meta.counts["3"] > 0
  return (
    <section
      aria-label="Landslide early warning"
      className="flex flex-col gap-1.5 rounded-xl border border-border bg-inset px-3 py-2.5"
    >
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-xs font-medium text-muted-foreground">Landslide early warning</h3>
        {meta.stale && (
          <Badge variant="chip" className="h-5 text-foreground">
            Out of date
          </Badge>
        )}
      </div>
      <p className="text-sm text-foreground">{meta.note}</p>
      {meta.reportDate && hasLevels && (
        <p className="numeric text-xs text-muted-foreground">
          Report of {formatReportDate(meta.reportDate)}: Level 3 (Red) {meta.counts["3"]} · Level 2
          (Amber) {meta.counts["2"]} · Level 1 (Yellow) {meta.counts["1"]}
        </p>
      )}
      {meta.pdfUrl && (
        <p>
          <OfficialLink href={meta.pdfUrl}>Official report</OfficialLink>
        </p>
      )}
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
}: WarningsPanelProps) {
  const { data, isPending, error } = useWarnings(layer, { type, district })
  const bulletins = data?.bulletins ?? []
  const inForce = new Set((data?.inForce ?? []).map((bulletin) => bulletin.id))
  // Warnings in force first, then everything else newest first.
  const current = bulletins.filter((bulletin) => inForce.has(bulletin.id))
  const rest = bulletins.filter((bulletin) => !inForce.has(bulletin.id))

  return (
    <Panel
      icon={<BellIcon />}
      title="Official warnings"
      subtitle="NBRO, Department of Meteorology and DMC"
      className="w-full"
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label="Close official warnings"
        >
          <XIcon />
        </Button>
      }
    >
      {data && <LandslideStatusCard meta={data.landslide} />}

      <ToggleGroup
        type="single"
        size="sm"
        value={type ?? "all"}
        onValueChange={(value) => {
          if (value) onFilter({ type: value === "all" ? undefined : value, district })
        }}
        aria-label="Show bulletins of type"
        className="w-full"
      >
        <ToggleGroupItem value="all" className="flex-1">
          All
        </ToggleGroupItem>
        {bulletinTypeOrder.map((value: BulletinType) => (
          <ToggleGroupItem key={value} value={value} className="flex-1 px-1.5 text-xs">
            {bulletinTypeLabels[value]}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>

      {district && (
        <div className="flex items-center justify-between gap-2 rounded-xl border border-border bg-inset px-3 py-2 text-sm">
          <span>Mentioned in bulletin: {districtName ?? district}</span>
          <Button variant="ghost" size="sm" onClick={() => onFilter({ type })}>
            Show all
          </Button>
        </div>
      )}

      {isPending && (
        <div className="flex flex-col gap-2" role="status" aria-label="Loading official warnings">
          <Skeleton className="h-20 w-full rounded-xl" />
          <Skeleton className="h-20 w-full rounded-xl" />
        </div>
      )}
      {error != null && !data && (
        <p role="alert" className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm">
          Official warnings could not be loaded. Check with the Disaster Management Centre,{" "}
          <a className="underline" href="https://www.dmc.gov.lk/" target="_blank" rel="noreferrer">
            dmc.gov.lk
          </a>
          .
        </p>
      )}
      {data && bulletins.length === 0 && (
        <p className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm text-muted-foreground">
          {data.meta.fetchedAt
            ? "No bulletins match in the last 45 days."
            : "Bulletins have not been collected yet."}
        </p>
      )}
      {current.length > 0 && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">In force</h3>
          <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border">
            {current.map((bulletin) => (
              <BulletinItem key={bulletin.id} bulletin={bulletin} />
            ))}
          </ul>
        </section>
      )}
      {rest.length > 0 && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">Latest bulletins</h3>
          <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border">
            {rest.map((bulletin) => (
              <BulletinItem key={bulletin.id} bulletin={bulletin} />
            ))}
          </ul>
        </section>
      )}

      {data && (
        <div className="flex flex-col gap-1 text-xs text-muted-foreground">
          <p>
            {data.meta.note} Updated{" "}
            <span className="numeric">{formatAge(data.meta.fetchedAt)}</span>
            {data.meta.upstreamAt && (
              <>
                {", index rebuilt "}
                <span className="numeric">{formatAge(data.meta.upstreamAt)}</span>
              </>
            )}
            .
          </p>
          {data.meta.stale && (
            <Badge variant="chip" className="h-5 text-foreground">
              Out of date
            </Badge>
          )}
          <p>
            Department of Meteorology warnings are shown as the DMC republishes them; its own site
            has not been reachable from our collector.
          </p>
        </div>
      )}
    </Panel>
  )
}
