import { ExternalLinkIcon, MountainIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { DetailPlaceholder } from "@/components/DetailPlaceholder"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatAge, formatDayTime } from "@/lib/format"
import { formatReportDate, levelBadge } from "./present"
import { useDivision } from "./queries"
import { LevelSwatch } from "./WarningLegends"

interface LandslidePanelProps {
  id: string
  onClose: () => void
}

/** One DS division's landslide warning, in NBRO's words, with recent history. */
export function LandslidePanel({ id, onClose }: LandslidePanelProps) {
  const { data, isPending } = useDivision(id)
  if (!data) {
    return (
      <DetailPlaceholder
        icon={<MountainIcon />}
        title="Landslide warning"
        message={isPending ? undefined : "This DS division could not be found."}
        onClose={onClose}
      />
    )
  }
  const { report, level, history } = data
  const pdf = report.pdfUrl
  return (
    <Panel
      icon={<MountainIcon />}
      title={data.name}
      subtitle={
        <>
          {data.nameSi && (
            <>
              <span lang="si">{data.nameSi}</span>
              {" · "}
            </>
          )}
          DS division, {data.district} District
        </>
      }
      className="w-full"
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label="Close landslide warning"
        >
          <XIcon />
        </Button>
      }
      footer={
        <p>
          Source: {report.attribution}. Updated{" "}
          <span className="numeric">{formatAge(report.fetchedAt)}</span>.
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        {level ? (
          <Badge variant={levelBadge[level.level]}>
            {level.label} · {level.action}
          </Badge>
        ) : (
          <Badge variant="chip" className="text-foreground">
            No level in force as far as we know
          </Badge>
        )}
        {report.stale && (
          <Badge variant="chip" className="text-foreground">
            Out of date
          </Badge>
        )}
      </div>
      <p className="text-sm text-foreground">{report.note}</p>
      {!level && (report.status === "active" || report.status === "lapsed") && (
        <p className="text-sm text-foreground">
          {data.name} is not named in this report. That does not mean it is safe.
        </p>
      )}
      <LabeledRows>
        <LabeledRow label="DS division" value={data.name} numeric={false} />
        <LabeledRow label="District" value={data.district} numeric={false} />
        {report.reportDate && (
          <LabeledRow label="Report date" value={formatReportDate(report.reportDate)} />
        )}
        {report.issuedAt && report.validityStated && (
          <LabeledRow label="Issued" value={formatDayTime(report.issuedAt)} />
        )}
        {report.validUntil && (
          <LabeledRow
            label={report.validityStated ? "Valid until" : "Valid until (assumed)"}
            value={formatDayTime(report.validUntil)}
          />
        )}
        {report.messageId && <LabeledRow label="Message" value={report.messageId} />}
      </LabeledRows>
      {pdf && (
        <a
          href={pdf}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1.5 self-start rounded-sm text-sm font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
        >
          Official DMC report
          <ExternalLinkIcon className="size-3.5" aria-hidden="true" />
          <span className="sr-only">(opens in a new tab)</span>
        </a>
      )}
      <section className="flex flex-col gap-1.5">
        <h3 className="text-xs font-medium text-muted-foreground">
          Recent reports naming {data.name}
        </h3>
        {history.length === 0 ? (
          <p className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm text-muted-foreground">
            No report we hold names this division.
          </p>
        ) : (
          <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border">
            {history.slice(0, 10).map((entry) => (
              <li
                key={entry.date}
                className="flex items-center justify-between gap-3 px-3 py-2 text-sm"
              >
                <span className="numeric text-muted-foreground">
                  {formatReportDate(entry.date)}
                </span>
                <span className="flex items-center gap-2">
                  <LevelSwatch level={entry.level} />
                  {entry.label}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </Panel>
  )
}
