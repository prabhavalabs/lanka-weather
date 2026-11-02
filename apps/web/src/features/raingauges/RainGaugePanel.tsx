import { DropletsIcon, XIcon } from "lucide-react"
import { DottedGrid } from "@/components/DottedGrid"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { formatAge, formatClock, formatDay, formatDayTime, formatNumber } from "@/lib/format"
import { cn } from "@/lib/utils"
import { bandBadge, bandLabels, seriesPath } from "./present"
import { useRainGaugeDetail } from "./queries"
import type { RainGaugeDetail } from "./types"

const width = 300
const height = 96

/** The rolling 24-hour total over time. It is a running sum, so a fall is old rain leaving the window. */
function RollingChart({ series }: { series: RainGaugeDetail["series"] }) {
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
          aria-label={`Rolling 24-hour rain from ${formatNumber(low, 1)} to ${formatNumber(peak, 1)} mm`}
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
        <span>up to {formatNumber(peak, 1)} mm</span>
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
  const detail = useRainGaugeDetail(id)
  const gauge = detail.data?.gauge.properties
  const meta = detail.data?.meta
  return (
    <Panel
      icon={<DropletsIcon />}
      title={gauge?.name ?? (detail.error ? "Rain gauge" : "Loading…")}
      subtitle={gauge ? `Rain gauge · ${gauge.institution}` : "Rain gauge"}
      className={cn("w-full", className)}
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label="Close rain gauge">
          <XIcon />
        </Button>
      }
      footer={
        <p>
          {meta?.attribution ?? "HMIS, via Sri Lanka Water Levels (waterlevels.lk)"}
          {gauge && `. Run by ${gauge.institution}`}. Not an official warning. In an emergency call
          117.
        </p>
      }
    >
      {detail.isPending && <Skeleton className="h-24 w-full rounded-xl" />}
      {detail.error && !gauge && (
        <p role="alert" className="text-sm text-foreground">
          This rain gauge could not be read just now.
        </p>
      )}
      {gauge && (
        <>
          {gauge.rain24hMm !== null ? (
            <StatValue value={formatNumber(gauge.rain24hMm, 1)} unit="mm in 24 h" />
          ) : (
            <p className="rounded-xl border border-border bg-inset px-3 py-2 text-sm text-foreground">
              This gauge has not reported for over 3 hours, so we cannot say how much rain has
              fallen. That does not mean it has been dry.
              {gauge.lastRain24hMm !== null &&
                gauge.observedAt &&
                ` Its last figure was ${formatNumber(gauge.lastRain24hMm, 1)} mm, ${formatDayTime(gauge.observedAt)}.`}
            </p>
          )}
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={bandBadge[gauge.band]}>{bandLabels[gauge.band]}</Badge>
            <Badge variant="chip" className="text-foreground">
              Measured
            </Badge>
          </div>
          <LabeledRows>
            <LabeledRow
              label="Read"
              value={
                gauge.observedAt
                  ? `${formatDayTime(gauge.observedAt)} · ${formatAge(gauge.observedAt)}`
                  : "No reading"
              }
              tone={gauge.stale ? "warning" : "default"}
            />
            <LabeledRow label="Run by" value={gauge.institution} numeric={false} />
          </LabeledRows>
          {detail.data && detail.data.series.length > 1 && (
            <section className="flex flex-col gap-1.5">
              <h3 className="text-xs font-medium text-muted-foreground">
                Rolling 24-hour total, up to 48 hours
              </h3>
              <RollingChart series={detail.data.series} />
              <p className="text-xs text-muted-foreground">
                Each point is the sum of the 24 hours before it, so it says nothing about any one
                hour.
              </p>
            </section>
          )}
        </>
      )}
    </Panel>
  )
}
