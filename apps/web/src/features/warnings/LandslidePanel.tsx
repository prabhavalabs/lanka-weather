import { ExternalLinkIcon, MountainIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { DetailPlaceholder } from "@/components/DetailPlaceholder"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatAge, formatDayTime } from "@/lib/format"
import { localName, useLocale, useT } from "@/lib/i18n"
import {
  formatReportDate,
  landslideLimit,
  landslideStatusNote,
  levelBadge,
  levelLabel,
  levelWords,
} from "./present"
import { useDivision } from "./queries"
import { LevelSwatch } from "./WarningLegends"

interface LandslidePanelProps {
  id: string
  onClose: () => void
}

/** One DS division's landslide warning, in NBRO's words, with recent history. */
export function LandslidePanel({ id, onClose }: LandslidePanelProps) {
  const t = useT()
  const locale = useLocale()
  const { data, isPending } = useDivision(id)
  if (!data) {
    return (
      <DetailPlaceholder
        icon={<MountainIcon />}
        title={t("warnings.division.title")}
        message={isPending ? undefined : t("warnings.division.notFound")}
        onClose={onClose}
      />
    )
  }
  const { report, level, history } = data
  const pdf = report.pdfUrl
  const name = localName(data, locale)
  // The other name beside the title: Sinhala for other readers, English for Sinhala ones.
  const otherName = name === data.name ? data.nameSi : data.name
  const note = landslideStatusNote(report, report.fetchedAt)
  return (
    <Panel
      icon={<MountainIcon />}
      title={name}
      subtitle={
        <>
          {otherName && (
            <>
              <span lang={name === data.name ? "si" : "en"}>{otherName}</span>
              {t("common.separator")}
            </>
          )}
          {t("warnings.division.subtitle", { district: data.district })}
        </>
      }
      className="w-full"
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("warnings.division.close")}
        >
          <XIcon />
        </Button>
      }
      footer={
        <p>
          {t("warnings.division.footer", {
            source: report.attribution,
            age: formatAge(report.fetchedAt),
          })}
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        {level ? (
          <Badge variant={levelBadge[level.level]} className="whitespace-normal">
            {levelWords(level.level)}
          </Badge>
        ) : (
          <Badge variant="chip" className="whitespace-normal text-foreground">
            {t("warnings.division.noLevel")}
          </Badge>
        )}
        {report.stale && (
          <Badge variant="chip" className="text-foreground">
            {t("common.outOfDate")}
          </Badge>
        )}
      </div>
      <p className="text-sm text-foreground" lang={note.lang}>
        {note.text}
      </p>
      <p className="text-xs text-muted-foreground">
        {landslideLimit()}{" "}
        <a
          href="https://www.dmc.gov.lk/"
          target="_blank"
          rel="noreferrer"
          className="font-medium text-brand-text underline underline-offset-2 hover:text-foreground"
        >
          dmc.gov.lk
          <span className="sr-only"> {t("warnings.newTab")}</span>
        </a>
      </p>
      {!level && (report.status === "active" || report.status === "lapsed") && (
        <p className="text-sm text-foreground">{t("warnings.division.notNamed", { name })}</p>
      )}
      <LabeledRows>
        <LabeledRow label={t("terms.adminLevel.ds")} value={name} numeric={false} />
        <LabeledRow label={t("terms.adminLevel.district")} value={data.district} numeric={false} />
        {report.reportDate && (
          <LabeledRow
            label={t("warnings.division.reportDate")}
            value={formatReportDate(report.reportDate)}
          />
        )}
        {report.issuedAt && report.validityStated && (
          <LabeledRow label={t("common.issued")} value={formatDayTime(report.issuedAt)} />
        )}
        {report.validUntil && (
          <LabeledRow
            label={
              report.validityStated
                ? t("common.validUntil")
                : t("warnings.division.validUntilAssumed")
            }
            value={formatDayTime(report.validUntil)}
          />
        )}
        {report.messageId && (
          <LabeledRow label={t("warnings.division.message")} value={report.messageId} />
        )}
      </LabeledRows>
      {pdf && (
        <a
          href={pdf}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1.5 self-start rounded-sm text-sm font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
        >
          {t("warnings.division.officialReport")}
          <ExternalLinkIcon className="size-3.5" aria-hidden="true" />
          <span className="sr-only">{t("warnings.newTab")}</span>
        </a>
      )}
      <section className="flex flex-col gap-1.5">
        <h3 className="text-xs font-medium text-muted-foreground">
          {t("warnings.division.history", { name })}
        </h3>
        {history.length === 0 ? (
          <p className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm text-muted-foreground">
            {t("warnings.division.noHistory")}
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
                <span className="flex min-w-0 items-center gap-2">
                  <LevelSwatch level={entry.level} />
                  {levelLabel(entry.level)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </Panel>
  )
}
