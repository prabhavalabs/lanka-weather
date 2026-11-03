import { RadioTowerIcon, TriangleAlertIcon, XIcon } from "lucide-react"
import type * as React from "react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { formatAge, formatDay, formatDayTime, formatNumber } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import {
  feedNotice,
  formatRain,
  formatTemperature,
  markerTemperature,
  rainLabel,
  readingLabel,
} from "./present"
import { useMetStation } from "./queries"
import { TemperatureChart } from "./TemperatureChart"

interface MetStationPanelProps {
  id: string
  onClose: () => void
  className?: string
}

const historyDays = 7

/** A sentence with `{name}` placeholders filled by React nodes, such as links. */
function withNodes(text: string, nodes: Record<string, React.ReactNode>): React.ReactNode[] {
  return text
    .split(/(\{\w+\})/)
    .map((part, index) => {
      const name = /^\{(\w+)\}$/.exec(part)?.[1]
      return name && name in nodes ? <span key={index}>{nodes[name]}</span> : part
    })
    .filter((part) => part !== "")
}

const linkClass = "underline underline-offset-2 hover:text-foreground"

/** One Department of Meteorology station: its latest reading and history. All observed. */
export function MetStationPanel({ id, onClose, className }: MetStationPanelProps) {
  const t = useT()
  const detail = useMetStation(id)
  const station = detail.data?.station.properties
  const meta = detail.data?.meta
  const notice = meta ? feedNotice(meta) : null
  const shown = station ? markerTemperature(station) : null
  const isMax = station?.temperatureC === null && station?.tempMaxC !== null

  return (
    <Panel
      icon={<RadioTowerIcon />}
      title={station?.name ?? (detail.error ? t("stations.panel.title") : t("common.loading"))}
      subtitle={
        station?.wmoId
          ? t("stations.panel.subtitleWmo", { wmo: station.wmoId })
          : station && !station.reportsTemperature
            ? t("stations.panel.subtitleRain")
            : t("stations.panel.subtitle")
      }
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("stations.panel.close")}
        >
          <XIcon />
        </Button>
      }
      footer={
        meta && (
          <p>
            {withNodes(t("stations.panel.source"), {
              meteo: (
                <a href={meta.sourceUrl} target="_blank" rel="noreferrer" className={linkClass}>
                  {t("stations.panel.meteo")}
                </a>
              ),
              copy3h: (
                <a
                  href="https://github.com/nuuuwan/lk_weather_3h"
                  target="_blank"
                  rel="noreferrer"
                  className={linkClass}
                >
                  lk_weather_3h
                </a>
              ),
              copyDaily: (
                <a
                  href="https://github.com/nuuuwan/weather_lk"
                  target="_blank"
                  rel="noreferrer"
                  className={linkClass}
                >
                  weather_lk
                </a>
              ),
            })}
          </p>
        )
      }
    >
      {detail.isPending && (
        <div className="flex flex-col gap-3" role="status" aria-label={t("stations.panel.loading")}>
          <Skeleton className="h-10 w-36" />
          <Skeleton className="h-40 w-full rounded-2xl" />
        </div>
      )}
      {detail.error && !detail.data && (
        <p
          className="rounded-2xl border border-border bg-inset px-4 py-6 text-center text-sm"
          role="alert"
        >
          {t("stations.panel.error")}
        </p>
      )}
      {station && (
        <>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="chip" className="text-foreground">
              {t("stations.panel.observed")}
            </Badge>
            {station.stale && (
              <Badge variant="chip" className="text-foreground">
                {t("stations.panel.outOfDate")}
              </Badge>
            )}
          </div>
          {notice && (
            <p
              role="status"
              className="flex items-start gap-2 rounded-xl border border-border bg-inset px-3 py-2.5 text-sm text-foreground"
            >
              <TriangleAlertIcon
                className="mt-0.5 size-4 shrink-0 text-brand-text"
                aria-hidden="true"
              />
              {notice}
            </p>
          )}
          {shown !== null && (
            <StatValue
              label={isMax ? t("stations.panel.dailyMaximum") : t("stations.panel.temperature")}
              value={formatNumber(shown, 1)}
              unit="°C"
              caption={
                station.observedAt
                  ? `${formatDayTime(station.observedAt)} · ${formatAge(station.observedAt)}`
                  : undefined
              }
            />
          )}
          <LabeledRows>
            <LabeledRow
              label={t("stations.panel.observedRow")}
              value={
                station.stale
                  ? t("stations.panel.observedStale", { age: formatAge(station.observedAt) })
                  : formatAge(station.observedAt)
              }
              tone={station.stale ? "warning" : "default"}
            />
            <LabeledRow
              label={t("stations.panel.readingRow")}
              value={readingLabel(station)}
              numeric={false}
            />
            {station.rainMm !== null && (
              <LabeledRow label={rainLabel(station)} value={formatRain(station.rainMm)} />
            )}
            {station.rain24hMm !== null && station.rainHours !== 24 && (
              <LabeledRow
                label={t("stations.panel.rain24h")}
                value={formatRain(station.rain24hMm)}
              />
            )}
            {station.humidityPct !== null && (
              <LabeledRow
                label={t("stations.panel.humidity")}
                value={`${Math.round(station.humidityPct)}%`}
              />
            )}
            {station.tempMaxC !== null && (
              <LabeledRow
                label={t("stations.panel.maximum")}
                value={formatTemperature(station.tempMaxC)}
              />
            )}
            {station.tempMinC !== null && (
              <LabeledRow
                label={t("stations.panel.minimum")}
                value={formatTemperature(station.tempMinC)}
              />
            )}
            {station.elevationM !== null && (
              <LabeledRow
                label={t("stations.panel.elevation")}
                value={`${Math.round(station.elevationM)} m`}
              />
            )}
            {station.positionApproximate && (
              <LabeledRow
                label={t("stations.panel.position")}
                value={t("stations.panel.approximate")}
                numeric={false}
              />
            )}
          </LabeledRows>
          {detail.data && detail.data.threeHourly.length > 0 && (
            <div>
              <h3 className="mb-2 text-xs font-medium text-muted-foreground">
                {t("stations.panel.history", { count: historyDays })}
              </h3>
              <TemperatureChart readings={detail.data.threeHourly} days={historyDays} />
            </div>
          )}
          {detail.data && detail.data.daily.length > 0 && (
            <div>
              <h3 className="mb-1 text-xs font-medium text-muted-foreground">
                {t("stations.panel.dailyReports")}
              </h3>
              <LabeledRows>
                {detail.data.daily
                  .slice(-5)
                  .reverse()
                  .map((day) => (
                    <LabeledRow
                      key={day.date}
                      label={formatDay(day.observedAt)}
                      value={`${formatRain(day.rainMm)}${
                        day.tempMaxC !== null && day.tempMinC !== null
                          ? ` · ${formatNumber(day.tempMinC)}–${formatNumber(day.tempMaxC)}°`
                          : ""
                      }`}
                    />
                  ))}
              </LabeledRows>
            </div>
          )}
        </>
      )}
    </Panel>
  )
}
