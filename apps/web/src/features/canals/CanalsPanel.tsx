import { ExternalLinkIcon, WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { formatAge, formatDayTime, formatNumber } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import {
  changeText,
  groupCanals,
  seaTrendText,
  shownState,
  stateBadge,
  stateLabel,
} from "./present"
import { Template } from "../rivers/Template"
import { useCanals } from "./queries"
import type { CanalGauge } from "./types"

/** A gauge's last 24 hours as a small line: a shape, with its figures beside it. */
function Sparkline({ points }: { points: CanalGauge["last24h"] }) {
  if (points.length < 2) return <span className="inline-block h-5 w-14" aria-hidden="true" />
  const levels = points.map((point) => point.level_m)
  const low = Math.min(...levels)
  const span = Math.max(Math.max(...levels) - low, 0.05)
  const path = points
    .map(
      (point, index) =>
        `${index === 0 ? "M" : "L"}${((index / (points.length - 1)) * 56).toFixed(1)} ${(18 - ((point.level_m - low) / span) * 16).toFixed(1)}`,
    )
    .join(" ")
  return (
    <svg viewBox="0 0 56 20" className="h-5 w-14 shrink-0 text-muted-foreground" aria-hidden="true">
      <path d={path} fill="none" stroke="currentColor" strokeWidth={1.25} strokeLinejoin="round" />
    </svg>
  )
}

function GaugeRow({ gauge }: { gauge: CanalGauge }) {
  const t = useT()
  const state = shownState(gauge)
  const change = gauge.stale ? null : changeText(gauge.change3hM)
  return (
    <li className="flex flex-col gap-1 py-2">
      <span className="flex items-center justify-between gap-2">
        <span className="min-w-0 truncate text-sm text-foreground" title={gauge.name}>
          {gauge.name}
        </span>
        <Badge variant={stateBadge(state)} className="shrink-0">
          {stateLabel(state)}
        </Badge>
      </span>
      <span className="flex items-center justify-between gap-2">
        <span className="numeric min-w-0 truncate text-xs text-muted-foreground">
          {[
            gauge.levelM !== null && !gauge.stale ? `${formatNumber(gauge.levelM, 2)} m` : null,
            change,
            gauge.observedAt
              ? gauge.stale
                ? t("rivers.canals.lastRead", { age: formatAge(gauge.observedAt) })
                : formatAge(gauge.observedAt)
              : t("rivers.canals.noReading"),
          ]
            .filter(Boolean)
            .join(t("common.separator"))}
        </span>
        {!gauge.stale && <Sparkline points={gauge.last24h} />}
      </span>
    </li>
  )
}

/**
 * Colombo's canal levels. The readings are context and not a flood warning, in the
 * source's own words. The source publishes no positions, so this is a list, not pins.
 */
export function CanalsPanel({ onClose, className }: { onClose: () => void; className?: string }) {
  const t = useT()
  const canals = useCanals(true)
  const data = canals.data
  const groups = data ? groupCanals(data.gauges) : []
  const seaTrend = data?.sea?.trend ? seaTrendText(data.sea.trend) : null
  const tide = data?.tide.events[0]
  return (
    <Panel
      icon={<WavesIcon />}
      title={t("rivers.canals.title")}
      subtitle={
        data
          ? t("rivers.canals.subtitle", {
              reporting: data.meta.reporting,
              total: data.meta.total,
            })
          : t("rivers.canals.subtitleEmpty")
      }
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("rivers.canals.close")}
        >
          <XIcon />
        </Button>
      }
      footer={
        <p>
          {data?.meta.attribution ?? "Data © CUrW / SLLDC (pub.curwsl.org)"}.{" "}
          {data?.meta.credit && `${data.meta.credit}. `}
          {t("rivers.canals.through", {
            via: data?.meta.via ?? "Sri Lanka Water Levels (waterlevels.lk)",
          })}
          {data?.meta.upstreamUrl && (
            <>
              {" "}
              <a
                href={data.meta.upstreamUrl}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 underline underline-offset-2"
              >
                pub.curwsl.org
                <ExternalLinkIcon className="size-3" aria-hidden="true" />
              </a>
            </>
          )}
          .
        </p>
      }
    >
      <p className="rounded-xl border border-border bg-inset px-3 py-2 text-sm text-foreground">
        {t("rivers.canals.context")} {t("terms.emergency")}
      </p>
      {canals.isPending && <Skeleton className="h-48 w-full rounded-xl" />}
      {Boolean(canals.error) && !data && (
        <p role="alert" className="text-sm text-foreground">
          {t("rivers.canals.error")}
        </p>
      )}
      {data && data.meta.stale && data.meta.total > 0 && (
        <p role="status" className="text-sm text-foreground">
          {t("rivers.canals.allStale")}
        </p>
      )}
      {data && (
        <LabeledRows>
          <LabeledRow
            label={t("rivers.canals.read")}
            value={formatAge(data.meta.fetchedAt)}
            tone={data.meta.stale ? "warning" : "default"}
          />
          <LabeledRow label={t("rivers.canals.raised")} value={String(data.meta.raised)} />
          {data.sea?.levelM !== null && data.sea?.levelM !== undefined && (
            <LabeledRow
              label={t("rivers.canals.sea")}
              value={
                seaTrend ? (
                  <Template
                    text={t("rivers.canals.seaValue")}
                    values={{
                      level: formatNumber(data.sea.levelM, 2),
                      trend: seaTrend.english ? (
                        <span lang="en">{seaTrend.text}</span>
                      ) : (
                        seaTrend.text
                      ),
                    }}
                  />
                ) : (
                  `${formatNumber(data.sea.levelM, 2)} m`
                )
              }
            />
          )}
          {tide && (
            <LabeledRow
              label={t(
                tide.kind === "high"
                  ? "rivers.canals.tide.high"
                  : tide.kind === "low"
                    ? "rivers.canals.tide.low"
                    : "rivers.canals.tide.generic",
              )}
              value={formatDayTime(tide.at)}
            />
          )}
        </LabeledRows>
      )}
      {groups.map((group) => (
        <section key={group.name} className="flex flex-col">
          <h3 className="text-xs font-medium text-muted-foreground">
            {group.name || t("rivers.canals.otherGauges")}
          </h3>
          <ul className="flex flex-col divide-y divide-border">
            {group.gauges.map((gauge) => (
              <GaugeRow key={gauge.id} gauge={gauge} />
            ))}
          </ul>
        </section>
      ))}
      {data && (
        <p className="text-xs text-muted-foreground">
          {t("rivers.canals.footnote")} {data.tide.note && <span lang="en">{data.tide.note}</span>}
        </p>
      )}
    </Panel>
  )
}
