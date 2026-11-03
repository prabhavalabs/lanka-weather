import { CloudRainIcon, WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { formatAge, formatDayTime, formatInteger } from "@/lib/format"
import { localName, useLocale, useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import {
  floodSeverityBadge,
  floodSeverityLabels,
  floodTrendArrows,
  floodTrendLabels,
  formatWindow,
} from "../floods/present"
import { LevelChart } from "../rivers/LevelChart"
import { formatLevel, stationThresholds, statusLabels } from "../rivers/present"
import { Template } from "../rivers/Template"
import {
  atRisk,
  coverageText,
  formatRain,
  reasonBasis,
  reasonText,
  riskBadge,
  riskLabels,
  riverName,
} from "./present"
import { useRiverDetail } from "./queries"
import type { ReachRisk, RiskReason } from "./types"

const historyHours = 72
const chartedGauges = 4

/** Rough severity of a gauge, to chart the worst ones first. */
const statusRank = { major_flood: 4, minor_flood: 3, alert: 2, normal: 1, unknown: 0 } as const

/**
 * A reason in the reader's language, or the server's English sentence under
 * a translated note of what it rests on.
 */
function Reason({ reason }: { reason: RiskReason }) {
  const t = useT()
  const shown = reasonText(reason)
  if (!shown.english) return <>{shown.text}</>
  return (
    <>
      <span className="font-medium">{reasonBasis(reason.kind)}</span>
      {t("common.separator")}
      <span lang="en">{shown.text}</span>
    </>
  )
}

function Stretch({ reach, highlighted }: { reach: ReachRisk; highlighted: boolean }) {
  const t = useT()
  const first = reach.reasons[0]
  return (
    <li
      className={cn("flex flex-col gap-1 px-3 py-2", highlighted && "bg-accent")}
      aria-current={highlighted ? "true" : undefined}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="min-w-0 truncate text-sm text-foreground" title={reach.name}>
          {reach.name}
        </span>
        <Badge variant={riskBadge[reach.level]} className="shrink-0">
          {riskLabels[reach.level]}
        </Badge>
      </div>
      <p className="text-xs break-words text-muted-foreground">
        {first ? <Reason reason={first} /> : t("rivers.risk.panel.stretchUnknown")}
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
  const t = useT()
  const locale = useLocale()
  const detail = useRiverDetail(id)
  const river = detail.data

  const close = (
    <Button
      variant="outline"
      size="icon"
      onClick={onClose}
      aria-label={t("rivers.risk.panel.close")}
    >
      <XIcon />
    </Button>
  )

  if (!river) {
    return (
      <Panel
        icon={<WavesIcon />}
        title={t("rivers.risk.panel.title")}
        className={cn("w-full", className)}
        action={close}
      >
        {detail.error ? (
          <p className="text-sm text-muted-foreground" role="alert">
            {t("rivers.risk.panel.loadError")}
          </p>
        ) : (
          <Skeleton
            className="h-40 w-full rounded-2xl"
            role="status"
            aria-label={t("rivers.risk.panel.loading")}
          />
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
    subtitle: gauge.stale
      ? t("common.outOfDate")
      : t("rivers.risk.panel.readAge", { age: formatAge(gauge.observedAt) }),
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
            subtitle: t("rivers.risk.panel.flowsInto"),
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
      subtitle: t("rivers.risk.panel.tributary"),
      badge: { variant: riskBadge[tributary.level], label: riskLabels[tributary.level] },
      onSelect: () => onSelectRiver(tributary.id),
    })),
  ]

  // The title is the name in the reader's language; the other script, when
  // there is one, goes beneath it so both stay recognisable.
  const title = riverName(river, locale)
  const otherName = locale === "si" ? river.name : river.nameSi
  return (
    <Panel
      icon={<WavesIcon />}
      title={title}
      subtitle={
        otherName && otherName !== title ? (
          <span lang={locale === "si" ? "en" : "si"}>{otherName}</span>
        ) : (
          coverageText(river)
        )
      }
      className={cn("w-full", className)}
      action={close}
      footer={
        <p>
          <Template
            text={t(
              river.meta.rainFetchedAt
                ? "rivers.risk.panel.footerRain"
                : "rivers.risk.panel.footer",
            )}
            values={{
              attribution: river.meta.attribution,
              gauges: <span className="numeric">{formatAge(river.meta.gaugesObservedAt)}</span>,
              rain: <span className="numeric">{formatAge(river.meta.rainFetchedAt)}</span>,
            }}
          />
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={riskBadge[river.level]}>{riskLabels[river.level]}</Badge>
        <Badge variant="chip" className="text-foreground">
          {t("terms.estimate")}
        </Badge>
        {river.meta.seeded && (
          <Badge variant="destructive">{t("rivers.risk.panel.testData")}</Badge>
        )}
      </div>

      <ul className="flex list-disc flex-col gap-1 pl-4 text-sm break-words text-foreground">
        {river.reasons.slice(0, 4).map((reason) => (
          <li key={reason.text}>
            <Reason reason={reason} />
          </li>
        ))}
      </ul>

      <p className="rounded-xl border border-border bg-inset px-3 py-2 text-xs text-muted-foreground">
        {/* The server's note is a fixed sentence (riverrisk.Note); it is translated here. */}
        {locale === "en" ? river.note : t("rivers.risk.note")}
      </p>

      <section className="flex flex-col gap-1.5">
        <h3 className="text-xs font-medium text-muted-foreground">
          {t("rivers.risk.panel.stretches", { coverage: coverageText(river) })}
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
            {t("rivers.risk.panel.areas")}
          </h3>
          {river.affected
            .filter((stretch) => atRisk(stretch.level))
            .map((stretch) => (
              <div
                key={stretch.reachFid}
                className="flex flex-col gap-1.5 rounded-xl border border-border px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="min-w-0 truncate text-sm" title={stretch.reachName}>
                    {stretch.reachName}
                  </span>
                  <Badge variant={riskBadge[stretch.level]} className="shrink-0">
                    {riskLabels[stretch.level]}
                  </Badge>
                </div>
                {stretch.districts.length > 0 && (
                  <p className="text-xs break-words text-muted-foreground">
                    <Template
                      text={t("rivers.risk.panel.districts")}
                      values={{
                        names: stretch.districts.map((region, index) => (
                          <span key={region.id}>
                            {index > 0 && ", "}
                            <button
                              type="button"
                              className="text-foreground underline underline-offset-2"
                              onClick={() => onSelectRegion(region.id)}
                            >
                              {localName(region, locale)}
                            </button>
                          </span>
                        )),
                      }}
                    />
                  </p>
                )}
                {stretch.dsDivisions.length > 0 && (
                  <p className="text-xs break-words text-muted-foreground">
                    {t("rivers.risk.panel.dsDivisions", {
                      names: stretch.dsDivisions
                        .map((region) => localName(region, locale))
                        .join(", "),
                    })}
                  </p>
                )}
              </div>
            ))}
        </section>
      )}

      {gauges.slice(0, chartedGauges).map((gauge) => {
        const lines = stationThresholds(gauge)
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
              <Badge variant={gauge.status} className="shrink-0">
                {statusLabels[gauge.status]}
              </Badge>
            </div>
            <p className="numeric text-xs text-muted-foreground">
              {t(gauge.stale ? "rivers.risk.panel.gaugeLineStale" : "rivers.risk.panel.gaugeLine", {
                level: formatLevel(gauge.waterLevel, gauge.unit),
                age: formatAge(gauge.observedAt),
              })}
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
      <GaugeList title={t("rivers.risk.panel.otherGauges")} items={others} />

      {river.forecasts.length > 0 && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">
            {t("rivers.risk.panel.forecastTitle")}
          </h3>
          <LabeledRows>
            {river.forecasts.map((forecast) => (
              <LabeledRow
                key={forecast.gaugeId}
                label={
                  <span className="flex flex-wrap items-center gap-2">
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
            {t("rivers.risk.panel.forecastIssued", {
              time: formatDayTime(river.forecasts[0]!.issuedAt),
            })}
          </p>
        </section>
      )}

      {river.rain && (
        <section className="flex flex-col gap-1.5">
          <h3 className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
            <CloudRainIcon aria-hidden="true" className="size-3.5" />
            {t("rivers.risk.panel.rainTitle")}
          </h3>
          <LabeledRows>
            <LabeledRow
              label={t("rivers.risk.panel.next24h")}
              value={formatRain(river.rain.next24hMm)}
            />
            <LabeledRow
              label={t("rivers.risk.panel.next72h")}
              value={formatRain(river.rain.next72hMm)}
            />
            <LabeledRow
              label={t("rivers.risk.panel.catchment")}
              value={`${formatInteger(river.rain.catchmentKm2)} km²`}
            />
          </LabeledRows>
          <p className="text-xs text-muted-foreground">
            {[
              t("rivers.risk.panel.rainNote"),
              river.rain.stale ? t("rivers.risk.panel.rainStale") : null,
              t("rivers.risk.panel.rainCredit"),
            ]
              .filter(Boolean)
              .join(" ")}
          </p>
        </section>
      )}

      <GaugeList title={t("rivers.risk.panel.connected")} items={related} />
    </Panel>
  )
}
