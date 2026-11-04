import { WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { LegendSwatch } from "@/components/LegendSwatch"
import { Panel } from "@/components/Panel"
import { ProgressBar } from "@/components/ProgressBar"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { LevelChart } from "./LevelChart"
import {
  formatAge,
  formatLevel,
  formatRate,
  headroom,
  levelNumber,
  nextHourLine,
  stationThresholds,
  statusLabels,
  trendArrows,
  trendLabels,
} from "./present"
import { useStationHistory } from "./queries"
import { Template } from "./Template"
import type { StationProperties } from "./types"

const historyHours = 72

interface StationPanelProps {
  station: StationProperties
  onClose: () => void
  className?: string
}

/** Details of one gauging station: current level, thresholds and recent history. */
export function StationPanel({ station, onClose, className }: StationPanelProps) {
  const t = useT()
  const history = useStationHistory(station.id, historyHours)
  const room = headroom(station)
  const thresholds = stationThresholds(station)
  const level = station.waterLevel === null ? "–" : levelNumber(station.waterLevel, station.unit)
  const unit = station.unit === "unknown" ? undefined : station.unit
  const rising = station.trend === "rising"
  const nextHour = nextHourLine(station.nextHour)

  return (
    <Panel
      icon={<WavesIcon />}
      title={station.name}
      subtitle={t("rivers.station.subtitle", { basin: station.basin })}
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("rivers.station.close")}
        >
          <XIcon />
        </Button>
      }
      footer={
        <p>
          {t("rivers.station.source", { attribution: station.attribution })}{" "}
          {station.sourceUrl && (
            <>
              <a
                href={station.sourceUrl}
                target="_blank"
                rel="noreferrer"
                className="underline underline-offset-2 hover:text-foreground"
              >
                {t("rivers.station.sourceLink")}
              </a>
              .{" "}
            </>
          )}
          {t("rivers.station.official")}
        </p>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <Badge variant={station.status}>{statusLabels[station.status]}</Badge>
          {station.stale && (
            <Badge variant="chip" className="text-foreground">
              {t("common.outOfDate")}
            </Badge>
          )}
        </div>
        <StatValue
          label={t("rivers.station.waterLevel")}
          value={level}
          unit={unit}
          delta={
            station.ratePerHour !== null
              ? `${trendArrows[station.trend]} ${formatRate(station.ratePerHour, station.unit)}`
              : undefined
          }
          deltaLabel={
            station.ratePerHour !== null ? trendLabels[station.trend].toLowerCase() : undefined
          }
          caption={
            room
              ? t(`rivers.station.below.${room.next}`, {
                  amount: formatLevel(room.remaining, station.unit),
                })
              : station.waterLevel === null
                ? t("rivers.station.noReading")
                : station.unit === "unknown"
                  ? t("rivers.station.noUnit")
                  : thresholds.length === 0
                    ? t("rivers.station.noThresholds")
                    : t("rivers.station.aboveMajor")
          }
          aside={room ? `${Math.round(room.fraction * 100)}%` : undefined}
        />
        {room && (
          <ProgressBar
            value={room.fraction}
            label={t(`rivers.station.towards.${room.next}`)}
            color={rising ? undefined : "var(--chart-2)"}
          />
        )}
      </div>

      {nextHour && (
        <p className="rounded-xl border border-border bg-inset px-3 py-2 text-xs text-muted-foreground">
          <Template
            text={t("rivers.station.nextHour")}
            values={{
              value: (
                <span className="numeric whitespace-nowrap text-foreground">{nextHour.value}</span>
              ),
              issued: nextHour.issued,
            }}
          />
        </p>
      )}

      <LabeledRows>
        <LabeledRow
          label={t("rivers.station.observed")}
          value={
            station.stale
              ? t("rivers.station.observedStale", { age: formatAge(station.observedAt) })
              : formatAge(station.observedAt)
          }
          tone={station.stale ? "warning" : "default"}
        />
        {station.ratePerHour === null && (
          <LabeledRow
            label={t("rivers.station.trend")}
            value={trendLabels[station.trend]}
            numeric={false}
          />
        )}
        {station.rainfallMm !== null && (
          <LabeledRow
            label={t("rivers.station.rainfall")}
            value={`${station.rainfallMm.toFixed(1)} mm`}
          />
        )}
        {thresholds.map((threshold) => (
          <LabeledRow
            key={threshold.label}
            leading={<LegendSwatch color={threshold.color} />}
            label={threshold.label}
            value={formatLevel(threshold.value, station.unit)}
          />
        ))}
      </LabeledRows>

      <div>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground">
          {t("rivers.station.history", { hours: historyHours })}
        </h3>
        {history.isPending && (
          <Skeleton
            className="h-40 w-full rounded-2xl"
            role="status"
            aria-label={t("rivers.station.loadingHistory")}
          />
        )}
        {history.error && (
          <p
            className="rounded-2xl border border-border bg-inset py-6 text-center text-xs text-muted-foreground"
            role="alert"
          >
            {t("rivers.station.historyUnavailable")}
          </p>
        )}
        {history.data && (
          <LevelChart
            readings={history.data.readings}
            thresholds={thresholds}
            unit={unit ?? t("rivers.station.noUnitShort")}
            hours={historyHours}
          />
        )}
      </div>
    </Panel>
  )
}
