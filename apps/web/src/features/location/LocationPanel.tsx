/**
 * The location panel with its data: weather at a point or a region's label
 * point, the regions containing it, nearby gauges and alerts. Loaded on
 * demand (React.lazy), so it is not in the first download.
 */
import { useMemo } from "react"
import { AlertList } from "@/components/weather/AlertList"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { LocationWeatherPanel } from "@/components/weather/LocationWeatherPanel"
import { Badge } from "@/components/ui/badge"
import { ApiError, isNotReady } from "@/lib/api"
import { formatAge } from "@/lib/format"
import {
  areaBadge,
  areaLikelihood,
  floodAreaLabels,
  floodSeverityBadge,
  floodSeverityLabels,
  floodTrendArrows,
  formatWindow,
  gaugeName,
} from "../floods/present"
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
}

function unavailableText(error: unknown, outside: boolean) {
  if (outside || (error instanceof ApiError && error.code === "outside_grid")) {
    return "Weather is available over Sri Lanka and its coastal waters only."
  }
  if (isNotReady(error) || !error) {
    return "Weather data is still loading. It appears here once the weather collector has run."
  }
  return "Weather could not be loaded. It will be retried shortly."
}

export default function LocationPanel({
  selection,
  onClose,
  onSelectStation,
  onSelectGauge,
  onSelectRiver,
  onSelectMetStation,
  onSelectReservoir,
}: LocationPanelProps) {
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
    }),
    [point.data],
  )

  const loading = location === null || point.isPending
  const status = loading ? "loading" : current ? "ready" : "unavailable"
  const area = place.name
  const generatedAt = summary.data?.generatedAt ?? new Date().toISOString()
  const alerts = (summary.data?.alerts ?? []).map((alert, index) =>
    estimateAlert(alert, index, area, generatedAt),
  )
  const official = officialSlot(summary.data?.official, summary.data?.regions?.district?.name)

  const nearbyRivers: GaugeListItem[] = (summary.data?.nearbyRivers ?? []).map((river) => ({
    id: river.id,
    title: river.name,
    subtitle: (
      <>
        <span className="numeric">{river.distanceKm.toFixed(1)} km</span> · Estimate ·{" "}
        {river.reachName}
      </>
    ),
    badge: { variant: riskBadge[river.level], label: riskLabels[river.level] },
    onSelect: onSelectRiver ? () => onSelectRiver(river.id) : undefined,
  }))

  const rivers: GaugeListItem[] = (summary.data?.rivers ?? []).map((river) => ({
    id: river.id,
    title: river.name,
    subtitle: (
      <>
        {river.basin} · <span className="numeric">{river.distanceKm.toFixed(1)} km</span>
        {river.stale && " · out of date"}
      </>
    ),
    value: `${formatLevel(river.waterLevel, river.unit)} ${trendArrows[river.trend]}`.trim(),
    badge: { variant: river.status, label: statusLabels[river.status] },
    onSelect: () => onSelectStation(river.id),
  }))

  const floodGauges: GaugeListItem[] = (summary.data?.floodForecast?.gauges ?? []).map((gauge) => ({
    id: gauge.gaugeId,
    title: gaugeName(gauge),
    subtitle: (
      <>
        {gauge.distanceKm !== undefined && (
          <span className="numeric">{gauge.distanceKm.toFixed(1)} km · </span>
        )}
        forecast for{" "}
        <span className="numeric">{formatWindow(gauge.forecastStart, gauge.forecastEnd)}</span>
      </>
    ),
    value: floodTrendArrows[gauge.trend] || undefined,
    badge: {
      variant: floodSeverityBadge[gauge.severity],
      label: floodSeverityLabels[gauge.severity],
    },
    onSelect: () => onSelectGauge(gauge.gaugeId),
  }))

  const floodAreas: GaugeListItem[] = (summary.data?.floodForecast?.areas ?? []).map(
    (floodArea, index) => ({
      id: floodArea.id ?? `${floodArea.kind}-${index}`,
      title: floodAreaLabels[floodArea.kind as keyof typeof floodAreaLabels] ?? floodArea.kind,
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
    subtitle: (
      <>
        <span className="numeric">{reservoir.distanceKm.toFixed(1)} km</span>
        {reservoir.flow && ` · ${flowShort(reservoir.flow)}`}
      </>
    ),
    value: reservoir.storagePercent !== null ? `${reservoir.storagePercent}%` : undefined,
    badge: { variant: reservoir.status, label: reservoirStatusLabels[reservoir.status] },
    onSelect: () => onSelectReservoir(reservoir.id),
  }))

  const meta = point.data?.meta
  const weatherAge = meta?.currentFetchedAt ?? current?.observedAt ?? null
  const riversShown = rivers.length > 0
  const floodsShown = floodGauges.length > 0 || floodAreas.length > 0

  const sources = (
    <ul className="flex flex-col gap-1">
      {meta && (
        <li className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
          <span>
            Weather:{" "}
            <a
              href="https://open-meteo.com/"
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-2 hover:text-foreground"
            >
              Open-Meteo.com
            </a>{" "}
            (CC BY 4.0), 0.2° model grid · <span className="numeric">{formatAge(weatherAge)}</span>
          </span>
          {(meta.currentStale || meta.forecastStale) && (
            <Badge variant="chip" className="h-5 text-foreground">
              Out of date
            </Badge>
          )}
        </li>
      )}
      {summary.data?.observed && <li>Observed: Department of Meteorology, Sri Lanka</li>}
      {summary.data?.airQuality && (
        <li>
          Air quality: Open-Meteo.com (CC BY 4.0), CAMS, generated using Copernicus information
        </li>
      )}
      {summary.data?.sea && <li>Sea: Open-Meteo.com (CC BY 4.0)</li>}
      {riversShown && (
        <li>
          River levels:{" "}
          {[...new Set((summary.data?.rivers ?? []).map((river) => river.attribution))].join("; ")}
        </li>
      )}
      {reservoirs.length > 0 && (
        <li>
          Reservoirs: Irrigation Department and Mahaweli Authority via Sri Lanka Water Levels
          (waterlevels.lk). Not an official warning.
        </li>
      )}
      {floodsShown && <li>Flood forecasts: Google Flood Hub (Flood Forecasting API)</li>}
      {summary.data?.official.fetchedAt && (
        <li className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
          <span>
            Official warnings: NBRO and DMC ·{" "}
            <span className="numeric">{formatAge(summary.data.official.fetchedAt)}</span>
          </span>
          {summary.data.official.stale && (
            <Badge variant="chip" className="h-5 text-foreground">
              Out of date
            </Badge>
          )}
        </li>
      )}
      {selection.kind === "region" && region.data && (
        <li>Boundaries: Survey Department and Department of Census and Statistics</li>
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
      officialNote={official.note ?? "Official warnings could not be read just now."}
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
        </>
      }
      sources={sources}
      onClose={onClose}
    >
      <GaugeList title="Rivers nearby" items={nearbyRivers} />
      {status === "unavailable" && (
        <section className="flex flex-col gap-2">
          <h3 className="text-xs font-medium text-muted-foreground">Official warnings</h3>
          <AlertList
            alerts={official.alerts}
            label="Official warnings"
            emptyText={official.note ?? "Official warnings could not be read just now."}
            emptyIcon="info"
          />
        </section>
      )}
      <GaugeList title="River gauges nearby" items={rivers} />
      <GaugeList title="Flood forecast gauges nearby" items={floodGauges} />
      <GaugeList title="Flood areas here" items={floodAreas} />
      <GaugeList title="Reservoirs spilling or releasing nearby" items={reservoirs} />
    </LocationWeatherPanel>
  )
}
