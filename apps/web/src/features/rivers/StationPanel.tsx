import { WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { LegendSwatch } from "@/components/LegendSwatch"
import { Panel } from "@/components/Panel"
import { ProgressBar } from "@/components/ProgressBar"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import { LevelChart } from "./LevelChart"
import {
  formatAge,
  formatLevel,
  formatRate,
  headroom,
  statusColors,
  statusLabels,
  trendArrows,
  trendLabels,
} from "./present"
import { useStationHistory } from "./queries"
import type { StationProperties } from "./types"

const historyHours = 72

interface StationPanelProps {
  station: StationProperties
  onClose: () => void
  className?: string
}

/** Details of one gauging station: current level, thresholds and recent history. */
export function StationPanel({ station, onClose, className }: StationPanelProps) {
  const history = useStationHistory(station.id, historyHours)
  const room = headroom(station)
  const thresholds = [
    { label: "Alert", value: station.alertLevel, color: statusColors.alert },
    { label: "Minor flood", value: station.minorFloodLevel, color: statusColors.minor_flood },
    { label: "Major flood", value: station.majorFloodLevel, color: statusColors.major_flood },
  ].filter(
    (threshold): threshold is { label: string; value: number; color: string } =>
      threshold.value !== null,
  )
  const level =
    station.waterLevel === null ? "–" : station.waterLevel.toFixed(station.unit === "ft" ? 1 : 2)
  const rising = station.trend === "rising"

  return (
    <Panel
      icon={<WavesIcon />}
      title={station.name}
      subtitle={`${station.basin} · river gauge`}
      className={cn("w-full", className)}
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label="Close station details">
          <XIcon />
        </Button>
      }
      footer={
        <p>
          Source: Irrigation Department of Sri Lanka. Official flood warnings come from the
          Irrigation Department and the DMC.
        </p>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <Badge variant={station.status}>{statusLabels[station.status]}</Badge>
          {station.stale && (
            <Badge variant="chip" className="text-foreground">
              Out of date
            </Badge>
          )}
        </div>
        <StatValue
          label="Water level"
          value={level}
          unit={station.unit}
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
              ? `${formatLevel(room.remaining, station.unit)} below ${room.next} level`
              : thresholds.length === 0
                ? "This station publishes no flood thresholds."
                : "Above major flood level"
          }
          aside={room ? `${Math.round(room.fraction * 100)}%` : undefined}
        />
        {room && (
          <ProgressBar
            value={room.fraction}
            label={`Level between thresholds, towards ${room.next}`}
            color={rising ? undefined : "var(--chart-2)"}
          />
        )}
      </div>

      <LabeledRows>
        <LabeledRow
          label="Observed"
          value={`${formatAge(station.observedAt)}${station.stale ? " · out of date" : ""}`}
          tone={station.stale ? "warning" : "default"}
        />
        {station.ratePerHour === null && (
          <LabeledRow label="Trend" value={trendLabels[station.trend]} numeric={false} />
        )}
        {station.rainfallMm !== null && (
          <LabeledRow label="Rainfall at gauge" value={`${station.rainfallMm.toFixed(1)} mm`} />
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
          Last {historyHours} hours
        </h3>
        {history.isPending && (
          <Skeleton
            className="h-40 w-full rounded-2xl"
            role="status"
            aria-label="Loading history"
          />
        )}
        {history.error && (
          <p
            className="rounded-2xl border border-border bg-inset py-6 text-center text-xs text-muted-foreground"
            role="alert"
          >
            History is unavailable.
          </p>
        )}
        {history.data && (
          <LevelChart
            readings={history.data.readings}
            thresholds={thresholds}
            unit={station.unit}
            hours={historyHours}
          />
        )}
      </div>
    </Panel>
  )
}
