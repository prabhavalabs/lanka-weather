import { CloudRainIcon, WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { formatAge, formatDayTime } from "@/lib/format"
import { cn } from "@/lib/utils"
import {
  floodSeverityBadge,
  floodSeverityLabels,
  floodTrendArrows,
  floodTrendLabels,
  formatWindow,
} from "../floods/present"
import { LevelChart } from "../rivers/LevelChart"
import { formatLevel, statusColors, statusLabels } from "../rivers/present"
import { atRisk, coverageText, formatRain, riskBadge, riskLabels } from "./present"
import { useRiverDetail } from "./queries"
import type { ReachRisk, RiverGauge } from "./types"

const historyHours = 72
const chartedGauges = 4

/** Rough severity of a gauge, to chart the worst ones first. */
const statusRank = { major_flood: 4, minor_flood: 3, alert: 2, normal: 1, unknown: 0 } as const

function thresholds(gauge: RiverGauge) {
  return [
    { label: "Alert", value: gauge.alertLevel, color: statusColors.alert },
    { label: "Minor flood", value: gauge.minorFloodLevel, color: statusColors.minor_flood },
    { label: "Major flood", value: gauge.majorFloodLevel, color: statusColors.major_flood },
  ].filter(
    (threshold): threshold is { label: string; value: number; color: string } =>
      threshold.value !== null,
  )
}

function Stretch({ reach, highlighted }: { reach: ReachRisk; highlighted: boolean }) {
  return (
    <li
      className={cn("flex flex-col gap-1 px-3 py-2", highlighted && "bg-accent")}
      aria-current={highlighted ? "true" : undefined}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="min-w-0 truncate text-sm text-foreground">{reach.name}</span>
        <Badge variant={riskBadge[reach.level]}>{riskLabels[reach.level]}</Badge>
      </div>
      <p className="text-xs text-muted-foreground">
        {reach.reasons[0]?.text ?? "Nothing is known about this stretch."}
      </p>
    </li>
  )
}

interface RiverPanelProps {
  id: string
  /** The stretch clicked on the map, shown first. */
  reachFid?: number
  onClose: () => void
  onSelectRiver: (id: string) => void
  onSelectStation: (id: string) => void
  onSelectRegion: (id: string) => void
  className?: string
}

/**
 * One river: its estimated risk with reasons, each stretch, gauges with their
 * levels, Google's forecast, rain over the catchment, and the areas along
 * stretches at risk. Everything here is marked as an estimate, and the panel
 * sends people to official warnings.
 */
export default function RiverPanel({
  id,
  reachFid,
  onClose,
  onSelectRiver,
  onSelectStation,
  onSelectRegion,
  className,
}: RiverPanelProps) {
  const detail = useRiverDetail(id)
  const river = detail.data

  const close = (
    <Button variant="outline" size="icon" onClick={onClose} aria-label="Close river details">
      <XIcon />
    </Button>
  )

  if (!river) {
    return (
      <Panel icon={<WavesIcon />} title="River" className={cn("w-full", className)} action={close}>
        {detail.error ? (
          <p className="text-sm text-muted-foreground" role="alert">
            This river could not be loaded.
          </p>
        ) : (
          <Skeleton className="h-40 w-full rounded-2xl" role="status" aria-label="Loading river" />
        )}
      </Panel>
    )
  }

  const reaches = [...river.reaches].sort((a, b) =>
    a.fid === reachFid ? -1 : b.fid === reachFid ? 1 : 0,
  )
  const gauges = [...river.gauges].sort(
    (a, b) => (statusRank[b.status] ?? 0) - (statusRank[a.status] ?? 0),
  )
  const others: GaugeListItem[] = gauges.slice(chartedGauges).map((gauge) => ({
    id: gauge.id,
    title: gauge.name,
    subtitle: gauge.stale ? "Out of date" : `Read ${formatAge(gauge.observedAt)}`,
    value: formatLevel(gauge.waterLevel, gauge.unit),
    badge: { variant: gauge.status, label: statusLabels[gauge.status] },
    onSelect: () => onSelectStation(gauge.id),
  }))
  const related: GaugeListItem[] = [
    ...(river.parent
      ? [
          {
            id: river.parent.id,
            title: river.parent.name,
            subtitle: "Flows into",
            badge: {
              variant: riskBadge[river.parent.level],
              label: riskLabels[river.parent.level],
            },
            onSelect: () => onSelectRiver(river.parent!.id),
          },
        ]
      : []),
    ...river.tributaries.map((tributary) => ({
      id: tributary.id,
      title: tributary.name,
      subtitle: "Tributary",
      badge: { variant: riskBadge[tributary.level], label: riskLabels[tributary.level] },
      onSelect: () => onSelectRiver(tributary.id),
    })),
  ]

  return (
    <Panel
      icon={<WavesIcon />}
      title={river.name}
      subtitle={river.nameSi ? <span lang="si">{river.nameSi}</span> : coverageText(river)}
      className={cn("w-full", className)}
      action={close}
      footer={
        <p>
          Sources: {river.meta.attribution}. Gauges read{" "}
          <span className="numeric">{formatAge(river.meta.gaugesObservedAt)}</span>
          {river.meta.rainFetchedAt && (
            <>
              , rain forecast fetched{" "}
              <span className="numeric">{formatAge(river.meta.rainFetchedAt)}</span>
            </>
          )}
          .
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={riskBadge[river.level]}>{riskLabels[river.level]}</Badge>
        <Badge variant="chip" className="text-foreground">
          Estimate
        </Badge>
        {river.meta.seeded && (
          <Badge variant="destructive" className="uppercase">
            Test data
          </Badge>
        )}
      </div>

      <ul className="flex list-disc flex-col gap-1 pl-4 text-sm text-foreground">
        {river.reasons.slice(0, 4).map((reason) => (
          <li key={reason.text}>{reason.text}</li>
        ))}
      </ul>

      <p className="rounded-xl border border-border bg-inset px-3 py-2 text-xs text-muted-foreground">
        {river.note}
      </p>

      <section className="flex flex-col gap-1.5">
        <h3 className="text-xs font-medium text-muted-foreground">
          Stretches · {coverageText(river)}
        </h3>
        <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border">
          {reaches.map((reach) => (
            <Stretch key={reach.fid} reach={reach} highlighted={reach.fid === reachFid} />
          ))}
        </ul>
      </section>

      {river.affected.length > 0 && (
        <section className="flex flex-col gap-2">
          <h3 className="text-xs font-medium text-muted-foreground">
            Areas along stretches at risk
          </h3>
          {river.affected
            .filter((stretch) => atRisk(stretch.level))
            .map((stretch) => (
              <div
                key={stretch.reachFid}
                className="flex flex-col gap-1.5 rounded-xl border border-border px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="min-w-0 truncate text-sm">{stretch.reachName}</span>
                  <Badge variant={riskBadge[stretch.level]}>{riskLabels[stretch.level]}</Badge>
                </div>
                {stretch.districts.length > 0 && (
                  <p className="text-xs text-muted-foreground">
                    Districts:{" "}
                    {stretch.districts.map((region, index) => (
                      <span key={region.id}>
                        {index > 0 && ", "}
                        <button
                          type="button"
                          className="text-foreground underline underline-offset-2"
                          onClick={() => onSelectRegion(region.id)}
                        >
                          {region.name}
                        </button>
                      </span>
                    ))}
                  </p>
                )}
                {stretch.dsDivisions.length > 0 && (
                  <p className="text-xs text-muted-foreground">
                    DS divisions: {stretch.dsDivisions.map((region) => region.name).join(", ")}
                  </p>
                )}
              </div>
            ))}
        </section>
      )}

      {gauges.slice(0, chartedGauges).map((gauge) => {
        const lines = thresholds(gauge)
        return (
          <section key={gauge.id} className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <button
                type="button"
                onClick={() => onSelectStation(gauge.id)}
                className="min-w-0 truncate text-left text-sm font-medium underline-offset-2 hover:underline"
              >
                {gauge.name}
              </button>
              <Badge variant={gauge.status}>{statusLabels[gauge.status]}</Badge>
            </div>
            <p className="numeric text-xs text-muted-foreground">
              {formatLevel(gauge.waterLevel, gauge.unit)} · read {formatAge(gauge.observedAt)}
              {gauge.stale && " · out of date"}
            </p>
            <LevelChart
              readings={gauge.readings}
              thresholds={lines}
              unit={gauge.unit}
              hours={historyHours}
            />
          </section>
        )
      })}
      <GaugeList title="Other gauges" items={others} />

      {river.forecasts.length > 0 && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">
            Google flood forecast on this river
          </h3>
          <LabeledRows>
            {river.forecasts.map((forecast) => (
              <LabeledRow
                key={forecast.gaugeId}
                label={
                  <span className="flex items-center gap-2">
                    <Badge variant={floodSeverityBadge[forecast.severity]}>
                      {floodSeverityLabels[forecast.severity]}
                    </Badge>
                    <span className="numeric">
                      {floodTrendArrows[forecast.trend]} {floodTrendLabels[forecast.trend]}
                    </span>
                  </span>
                }
                value={formatWindow(forecast.forecastStart, forecast.forecastEnd)}
                tone={forecast.stale ? "warning" : "default"}
              />
            ))}
          </LabeledRows>
          <p className="text-xs text-muted-foreground">
            Issued {formatDayTime(river.forecasts[0]!.issuedAt)}. Source: Google Flood Hub.
          </p>
        </section>
      )}

      {river.rain && (
        <section className="flex flex-col gap-1.5">
          <h3 className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
            <CloudRainIcon aria-hidden="true" className="size-3.5" />
            Forecast rain over the catchment
          </h3>
          <LabeledRows>
            <LabeledRow label="Next 24 hours" value={formatRain(river.rain.next24hMm)} />
            <LabeledRow label="Next 72 hours" value={formatRain(river.rain.next72hMm)} />
            <LabeledRow
              label="Catchment"
              value={`${Math.round(river.rain.catchmentKm2).toLocaleString("en-GB")} km²`}
            />
          </LabeledRows>
          <p className="text-xs text-muted-foreground">
            An average over the whole catchment, from weather models.
            {river.rain.stale && " This forecast is out of date, so it is not used."} Weather data
            by Open-Meteo.com.
          </p>
        </section>
      )}

      <GaugeList title="Connected rivers" items={related} />
    </Panel>
  )
}
