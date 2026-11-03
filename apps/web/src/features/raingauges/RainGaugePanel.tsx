import { DropletsIcon, XIcon } from "lucide-react"
import { DottedGrid } from "@/components/DottedGrid"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { formatAge, formatClock, formatDay, formatDayTime, formatNumber } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { bandBadge, bandLabel, seriesPath } from "./present"
import { useRainGaugeDetail } from "./queries"
import type { RainGaugeDetail } from "./types"

const width = 300
const height = 96

/** The rolling 24-hour total over time. It is a running sum, so a fall is old rain leaving the window. */
function RollingChart({ series }: { series: RainGaugeDetail["series"] }) {
  const t = useT()
  const { points, peak, low } = seriesPath(series)
  if (points.length < 2) return null
  const path = points
    .map(
      (point, index) =>
        `${index === 0 ? "M" : "L"}${(point.x * width).toFixed(1)} ${((1 - point.y) * height).toFixed(1)}`,
    )
    .join(" ")
  const first = series[0]!
  const last = series.at(-1)!
  return (
    <figure className="flex flex-col gap-2">
      <DottedGrid className="px-3 pt-4 pb-3">
        <svg
          viewBox={`0 0 ${width} ${height}`}
          className="h-24 w-full overflow-visible text-foreground"
          role="img"
          aria-label={t("rain.panel.chartLabel", {
            low: formatNumber(low, 1),
            peak: formatNumber(peak, 1),
          })}
          preserveAspectRatio="none"
        >
          <path
            d={path}
            fill="none"
            stroke="var(--rain-3)"
            strokeWidth={2}
            strokeLinejoin="round"
            strokeLinecap="round"
            vectorEffect="non-scaling-stroke"
          />
        </svg>
      </DottedGrid>
      <figcaption className="numeric flex justify-between gap-2 text-xs text-muted-foreground">
        <span>
          {formatDay(first.at)} {formatClock(first.at)}
        </span>
        <span className="text-center">{t("rain.panel.upTo", { peak: formatNumber(peak, 1) })}</span>
        <span>
          {formatDay(last.at)} {formatClock(last.at)}
        </span>
      </figcaption>
    </figure>
  )
}

/** One live rain gauge: its rolling 24-hour total, band, age and who runs it. */
export function RainGaugePanel({
  id,
  onClose,
  className,
}: {
  id: string
  onClose: () => void
  className?: string
}) {
  const t = useT()
  const detail = useRainGaugeDetail(id)
  const gauge = detail.data?.gauge.properties
  const meta = detail.data?.meta
  return (
    <Panel
      icon={<DropletsIcon />}
      title={gauge?.name ?? (detail.error ? t("rain.panel.title") : t("common.loading"))}
      subtitle={
        gauge ? t("rain.panel.subtitle", { institution: gauge.institution }) : t("rain.panel.title")
      }
      className={cn("w-full", className)}
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label={t("rain.panel.close")}>
          <XIcon />
        </Button>
      }
      footer={
        <p>
          <span lang="en">
            {meta?.attribution ?? "HMIS, via Sri Lanka Water Levels (waterlevels.lk)"}.
          </span>{" "}
          {gauge && `${t("rain.panel.runByNamed", { institution: gauge.institution })} `}
          {t("terms.notOfficial")} {t("terms.emergency")}
        </p>
      }
    >
      {detail.isPending && <Skeleton className="h-24 w-full rounded-xl" />}
      {Boolean(detail.error) && !gauge && (
        <p role="alert" className="text-sm text-foreground">
          {t("rain.panel.error")}
        </p>
      )}
      {gauge && (
        <>
          {gauge.rain24hMm !== null ? (
            <StatValue value={formatNumber(gauge.rain24hMm, 1)} unit={t("rain.panel.unit")} />
          ) : (
            <p className="rounded-xl border border-border bg-inset px-3 py-2 text-sm text-foreground">
              {t("rain.panel.silent")}
              {gauge.lastRain24hMm !== null &&
                gauge.observedAt &&
                ` ${t("rain.panel.lastFigure", {
                  rain: formatNumber(gauge.lastRain24hMm, 1),
                  time: formatDayTime(gauge.observedAt),
                })}`}
            </p>
          )}
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={bandBadge[gauge.band]}>{bandLabel(gauge.band)}</Badge>
            <Badge variant="chip" className="text-foreground">
              {t("rain.panel.measured")}
            </Badge>
          </div>
          <LabeledRows>
            <LabeledRow
              label={t("rain.panel.read")}
              value={
                gauge.observedAt
                  ? t("rain.panel.readValue", {
                      time: formatDayTime(gauge.observedAt),
                      age: formatAge(gauge.observedAt),
                    })
                  : t("rain.panel.noReading")
              }
              tone={gauge.stale ? "warning" : "default"}
            />
            <LabeledRow label={t("rain.panel.runBy")} value={gauge.institution} numeric={false} />
          </LabeledRows>
          {detail.data && detail.data.series.length > 1 && (
            <section className="flex flex-col gap-1.5">
              <h3 className="text-xs font-medium text-muted-foreground">
                {t("rain.panel.rolling")}
              </h3>
              <RollingChart series={detail.data.series} />
              <p className="text-xs text-muted-foreground">{t("rain.panel.rollingNote")}</p>
            </section>
          )}
        </>
      )}
    </Panel>
  )
}
