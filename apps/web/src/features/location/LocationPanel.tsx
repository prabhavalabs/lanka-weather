/**
 * The location panel with its data: weather at a point or a region's label
 * point, the regions containing it, nearby gauges and alerts. Loaded on
 * demand (React.lazy), so it is not in the first download.
 */
import { useMemo, type ReactNode } from "react"
import { AlertList } from "@/components/weather/AlertList"
import { fillNodes } from "@/components/weather/fillNodes"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { LocationWeatherPanel } from "@/components/weather/LocationWeatherPanel"
import { Badge } from "@/components/ui/badge"
import { ApiError, isNotReady } from "@/lib/api"
import { formatAge } from "@/lib/format"
import { localName, translate, useLocale, useT } from "@/lib/i18n"
import {
  areaBadge,
  areaLikelihood,
  floodAreaLabel,
  floodSeverityBadge,
  floodSeverityLabel,
  floodTrendArrows,
  formatWindow,
  gaugeName,
} from "../floods/present"
import { bandBadge, bandLabel } from "../raingauges/present"
import { riskBadge, riskLabels } from "../river-risk/present"
import { flowShort, reservoirStatusLabels } from "../reservoirs/present"
import { formatLevel, statusLabels, trendArrows } from "../rivers/present"
import { useRegionAggregate, useRegionDetail } from "../regions/queries"
import { RegionAggregate } from "../regions/RegionAggregate"
import { officialSlot } from "../warnings/present"
import {
  currentFrom,
  daysFrom,
  estimateAlert,
  hoursFrom,
  localDate,
  placeFor,
  windFrom,
} from "./present"
import { ObservedSummary } from "../stations/ObservedSummary"
import { usePointForecast, useSelectionLocation, useSummary } from "./queries"
import type { Selection } from "./selection"

export interface LocationPanelProps {
  selection: Extract<Selection, { kind: "point" } | { kind: "region" }>
  onClose: () => void
  onSelectStation: (id: string) => void
  onSelectGauge: (id: string) => void
  onSelectRiver?: (id: string) => void
  onSelectMetStation: (id: string) => void
  onSelectReservoir: (id: string) => void
  onSelectRainGauge?: (id: string) => void
  onOpenCanals?: () => void
}

function unavailableText(error: unknown, outside: boolean) {
  if (outside || (error instanceof ApiError && error.code === "outside_grid")) {
    return translate("location.unavailable.outside")
  }
  if (isNotReady(error) || !error) {
    return translate("location.unavailable.loading")
  }
  return translate("location.unavailable.failed")
}

/** A distance in a `numeric` span: "4.8 km". */
function Distance({ km }: { km: number }) {
  const t = useT()
  return (
    <span className="numeric">{t("location.lists.distance", { distance: km.toFixed(1) })}</span>
  )
}

/** Parts of a list item's second line, joined by the separator. */
function joined(parts: ReactNode[], separator: string): ReactNode {
  return parts
    .filter((part) => part !== null && part !== undefined && part !== false && part !== "")
    .map((part, index) => (
      <span key={index}>
        {index > 0 && separator}
        {part}
      </span>
    ))
}

export default function LocationPanel({
  selection,
  onClose,
  onSelectStation,
  onSelectGauge,
  onSelectRiver,
  onSelectMetStation,
  onSelectReservoir,
  onSelectRainGauge,
  onOpenCanals,
}: LocationPanelProps) {
  const t = useT()
  // Names, words and the alerts are rebuilt in the reader's language.
  const locale = useLocale()
  const separator = t("common.separator")
  const regionId = selection.kind === "region" ? selection.id : null
  const region = useRegionDetail(regionId)
  const aggregate = useRegionAggregate(regionId)
  const location = useSelectionLocation(selection)
  const summary = useSummary(location)
  const point = usePointForecast(location)

  const place = placeFor(selection, location ?? { lat: 0, lon: 0 }, summary.data, region.data)
  const current = currentFrom(point.data)
  const view = useMemo(
    () => ({
      hours: hoursFrom(point.data).slice(0, 72),
      days: daysFrom(point.data),
      locale,
    }),
    [point.data, locale],
  )

  const loading = location === null || point.isPending
  const status = loading ? "loading" : current ? "ready" : "unavailable"
  const area = place.name
  const generatedAt = summary.data?.generatedAt ?? new Date().toISOString()
  const alerts = (summary.data?.alerts ?? []).map((alert, index) =>
    estimateAlert(alert, index, area, generatedAt),
  )
  const district = summary.data?.regions?.district
  const official = officialSlot(summary.data?.official, district ? localName(district) : null)
  const officialNote = official.note ?? t("location.officialUnread")

  const nearbyRivers: GaugeListItem[] = (summary.data?.nearbyRivers ?? []).map((river) => ({
    id: river.id,
    title: river.name,
    subtitle: joined(
      [<Distance km={river.distanceKm} />, t("terms.estimate"), river.reachName],
      separator,
    ),
    badge: { variant: riskBadge[river.level], label: riskLabels[river.level] },
    onSelect: onSelectRiver ? () => onSelectRiver(river.id) : undefined,
  }))

  const rivers: GaugeListItem[] = (summary.data?.rivers ?? []).map((river) => ({
    id: river.id,
    title: river.name,
    subtitle: joined(
      [river.basin, <Distance km={river.distanceKm} />, river.stale && t("common.outOfDate")],
      separator,
    ),
    value: `${formatLevel(river.waterLevel, river.unit)} ${trendArrows[river.trend]}`.trim(),
    badge: { variant: river.status, label: statusLabels[river.status] },
    onSelect: () => onSelectStation(river.id),
  }))

  const floodGauges: GaugeListItem[] = (summary.data?.floodForecast?.gauges ?? []).map((gauge) => ({
    id: gauge.gaugeId,
    title: gaugeName(gauge),
    subtitle: joined(
      [
        gauge.distanceKm !== undefined && <Distance km={gauge.distanceKm} />,
        fillNodes(t("location.lists.forecastFor"), {
          window: (
            <span className="numeric">{formatWindow(gauge.forecastStart, gauge.forecastEnd)}</span>
          ),
        }),
      ],
      separator,
    ),
    value: floodTrendArrows[gauge.trend] || undefined,
    badge: {
      variant: floodSeverityBadge[gauge.severity],
      label: floodSeverityLabel(gauge.severity),
    },
    onSelect: () => onSelectGauge(gauge.gaugeId),
  }))

  const floodAreas: GaugeListItem[] = (summary.data?.floodForecast?.areas ?? []).map(
    (floodArea, index) => ({
      id: floodArea.id ?? `${floodArea.kind}-${index}`,
      title: floodAreaLabel(floodArea.kind),
      subtitle: floodArea.validFrom ? (
        <span className="numeric">{formatWindow(floodArea.validFrom, floodArea.validUntil)}</span>
      ) : undefined,
      badge: {
        variant: areaBadge(floodArea as Parameters<typeof areaBadge>[0]),
        label: areaLikelihood(floodArea as Parameters<typeof areaLikelihood>[0]).label,
      },
    }),
  )

  const reservoirs: GaugeListItem[] = (summary.data?.reservoirs ?? []).map((reservoir) => ({
    id: reservoir.id,
    title: reservoir.name,
    subtitle: joined(
      [<Distance km={reservoir.distanceKm} />, reservoir.flow && flowShort(reservoir.flow)],
      separator,
    ),
    value: reservoir.storagePercent !== null ? `${reservoir.storagePercent}%` : undefined,
    badge: { variant: reservoir.status, label: reservoirStatusLabels[reservoir.status] },
    onSelect: () => onSelectReservoir(reservoir.id),
  }))

  const nearRainGauge = summary.data?.rainGauge
  const liveRain: GaugeListItem[] = nearRainGauge
    ? [
        {
          id: nearRainGauge.id,
          title: nearRainGauge.name,
          subtitle: joined(
            [
              <Distance km={nearRainGauge.distanceKm} />,
              nearRainGauge.institution,
              <span className="numeric">{formatAge(nearRainGauge.observedAt)}</span>,
            ],
            separator,
          ),
          value:
            nearRainGauge.rain24hMm !== null
              ? `${nearRainGauge.rain24hMm.toFixed(1)} mm`
              : undefined,
          badge: { variant: bandBadge[nearRainGauge.band], label: bandLabel(nearRainGauge.band) },
          onSelect: onSelectRainGauge ? () => onSelectRainGauge(nearRainGauge.id) : undefined,
        },
      ]
    : []
  const nearCanals = summary.data?.canals
  const canalRows: GaugeListItem[] = nearCanals
    ? [
        {
          id: "colombo-canals",
          title: t("location.lists.colomboCanals"),
          subtitle: joined(
            [
              t("location.lists.canalsReporting", {
                reporting: nearCanals.reporting,
                total: nearCanals.total,
              }),
              t("location.lists.canalsContext"),
            ],
            separator,
          ),
          badge: nearCanals.stale
            ? { variant: "chip" as const, label: t("common.outOfDate") }
            : nearCanals.raised > 0
              ? {
                  variant: "alert" as const,
                  label: t("location.lists.raised", { count: nearCanals.raised }),
                }
              : { variant: "normal" as const, label: t("location.lists.noneRaised") },
          onSelect: onOpenCanals,
        },
      ]
    : []

  const meta = point.data?.meta
  const weatherAge = meta?.currentFetchedAt ?? current?.observedAt ?? null
  const riversShown = rivers.length > 0
  const floodsShown = floodGauges.length > 0 || floodAreas.length > 0

  const source = (label: string, credit: ReactNode) =>
    fillNodes(t("location.sources.item"), { label, credit })
  const outOfDate = (
    <Badge variant="chip" className="h-5 text-foreground">
      {t("common.outOfDate")}
    </Badge>
  )
  const sources = (
    <ul className="flex flex-col gap-1">
      {meta && (
        <li className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
          <span>
            {source(
              t("location.sources.weather"),
              fillNodes(t("location.sources.weatherCredit"), {
                source: (
                  <a
                    href="https://open-meteo.com/"
                    target="_blank"
                    rel="noreferrer"
                    className="underline underline-offset-2 hover:text-foreground"
                  >
                    Open-Meteo.com
                  </a>
                ),
              }),
            )}
            {separator}
            <span className="numeric">{formatAge(weatherAge)}</span>
          </span>
          {(meta.currentStale || meta.forecastStale) && outOfDate}
        </li>
      )}
      {summary.data?.observed && (
        <li>{source(t("location.sources.observed"), t("location.sources.observedCredit"))}</li>
      )}
      {summary.data?.airQuality && (
        <li>
          {source(
            t("location.sources.airQuality"),
            <span lang="en">
              Open-Meteo.com (CC BY 4.0), CAMS, generated using Copernicus information
            </span>,
          )}
        </li>
      )}
      {summary.data?.sea && (
        <li>{source(t("location.sources.sea"), "Open-Meteo.com (CC BY 4.0)")}</li>
      )}
      {riversShown && (
        <li>
          {source(
            t("location.sources.riverLevels"),
            <span lang="en">
              {[...new Set((summary.data?.rivers ?? []).map((river) => river.attribution))].join(
                "; ",
              )}
            </span>,
          )}
        </li>
      )}
      {reservoirs.length > 0 && (
        <li>{source(t("location.sources.reservoirs"), t("location.sources.reservoirsCredit"))}</li>
      )}
      {floodsShown && (
        <li>{source(t("location.sources.floods"), "Google Flood Hub (Flood Forecasting API)")}</li>
      )}
      {summary.data?.official.fetchedAt && (
        <li className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
          <span>
            {source(t("location.sources.official"), t("location.sources.officialCredit"))}
            {separator}
            <span className="numeric">{formatAge(summary.data.official.fetchedAt)}</span>
          </span>
          {summary.data.official.stale && outOfDate}
        </li>
      )}
      {selection.kind === "region" && region.data && (
        <li>{source(t("location.sources.boundaries"), t("location.sources.boundariesCredit"))}</li>
      )}
    </ul>
  )

  return (
    <LocationWeatherPanel
      place={place}
      current={current}
      forecast={view.hours}
      daily={view.days}
      today={localDate()}
      wind={windFrom(point.data)}
      alerts={alerts}
      officialAlerts={official.alerts}
      officialNote={officialNote}
      officialBulletins={official.bulletins}
      status={status}
      unavailableText={unavailableText(point.error, summary.data?.weather === null && !point.error)}
      highlight={
        <>
          {aggregate.region && aggregate.level && (
            <RegionAggregate region={aggregate.region} level={aggregate.level} />
          )}
          <ObservedSummary
            observed={summary.data?.observed}
            air={summary.data?.airQuality}
            sea={summary.data?.sea}
            onSelectStation={onSelectMetStation}
          />
          <GaugeList title={t("location.lists.rainGauge")} items={liveRain} />
        </>
      }
      sources={sources}
      onClose={onClose}
    >
      <GaugeList title={t("location.lists.rivers")} items={nearbyRivers} />
      {status === "unavailable" && (
        <section className="flex flex-col gap-2">
          <h3 className="text-xs font-medium text-muted-foreground">
            {t("weather.panel.officialWarnings")}
          </h3>
          <AlertList
            alerts={official.alerts}
            label={t("weather.panel.officialWarnings")}
            emptyText={officialNote}
            emptyIcon="info"
          />
        </section>
      )}
      <GaugeList title={t("location.lists.riverGauges")} items={rivers} />
      <GaugeList title={t("location.lists.canals")} items={canalRows} />
      <GaugeList title={t("location.lists.floodGauges")} items={floodGauges} />
      <GaugeList title={t("location.lists.floodAreas")} items={floodAreas} />
      <GaugeList title={t("location.lists.reservoirs")} items={reservoirs} />
    </LocationWeatherPanel>
  )
}
