import { MapIcon, MapPinIcon, XIcon } from "lucide-react"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { formatAge, formatInteger } from "@/lib/format"
import { type Translate, useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { formatLevel } from "../rivers/present"
import type { BasinGauge, BasinsDocument, BasinStatus } from "../river-risk/types"
import { basinBadge, basinStateLabel } from "./basins"
import { floodAreaLabel, floodTrendArrows } from "./present"

interface BasinPanelProps {
  basin: BasinStatus
  meta: BasinsDocument["meta"]
  /** Where on the map it was clicked, to offer the weather and area there. */
  clickedAt?: { lat: number; lon: number } | null
  onClose: () => void
  onSelectStation: (id: string) => void
  onSelectGauge: (id: string) => void
  onSelectRiver: (id: string) => void
  onSelectArea: (id: string) => void
  onWeatherAt: (at: { lat: number; lon: number }) => void
  className?: string
}

function gaugeItem(
  t: Translate,
  gauge: BasinGauge,
  onSelectStation: (id: string) => void,
  onSelectGauge: (id: string) => void,
): GaugeListItem {
  const notes = [gauge.stale ? t("common.outOfDate") : formatAge(gauge.observedAt)]
  if (gauge.kind === "google") notes.push("Google")
  if (!gauge.verified) notes.push(t("floods.basin.notVerified"))
  return {
    id: `${gauge.kind}:${gauge.id}`,
    title: gauge.name,
    subtitle: notes.join(t("common.separator")),
    value:
      gauge.waterLevel !== undefined && gauge.unit
        ? formatLevel(gauge.waterLevel, gauge.unit as "m" | "ft")
        : floodTrendArrows[gauge.trend as keyof typeof floodTrendArrows] || undefined,
    badge: gauge.state
      ? { variant: basinBadge[gauge.state], label: basinStateLabel(gauge.state) }
      : { variant: "unknown", label: t("floods.basin.notUsed") },
    onSelect: () => (gauge.kind === "google" ? onSelectGauge(gauge.id) : onSelectStation(gauge.id)),
  }
}

/** One river basin: its forecast state and what that state comes from. */
export function BasinPanel({
  basin,
  meta,
  clickedAt,
  onClose,
  onSelectStation,
  onSelectGauge,
  onSelectRiver,
  onSelectArea,
  onWeatherAt,
  className,
}: BasinPanelProps) {
  const t = useT()
  const gauges = basin.gauges.map((gauge) => gaugeItem(t, gauge, onSelectStation, onSelectGauge))
  const areas: GaugeListItem[] = basin.areas.map((area) => ({
    id: area.id,
    title: floodAreaLabel(area.kind),
    badge: { variant: basinBadge[area.state], label: basinStateLabel(area.state) },
    onSelect: () => onSelectArea(area.id),
  }))
  return (
    <Panel
      icon={<MapIcon />}
      title={basin.name}
      subtitle={
        <>
          {basin.named ? t("floods.basin.river") : t("floods.basin.coast")}
          {t("common.separator")}
          <span className="numeric">{formatInteger(Math.round(basin.areaKm2))} km²</span>
        </>
      }
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("floods.basin.close")}
        >
          <XIcon />
        </Button>
      }
      footer={<p>{t("floods.basin.footer", { attribution: meta.attribution })}</p>}
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={basinBadge[basin.state]}>{basinStateLabel(basin.state)}</Badge>
        {basin.onlyUnverified && (
          <Badge variant="chip" className="text-foreground">
            {t("floods.basin.unverifiedOnly")}
          </Badge>
        )}
        {meta.seeded && (
          <Badge variant="destructive" className="uppercase">
            {t("floods.testData")}
          </Badge>
        )}
      </div>

      <p className="text-sm text-foreground">
        {basin.state === "no_forecast"
          ? t("floods.basin.noForecast")
          : basin.state === "no_flooding"
            ? t("floods.basin.calm")
            : t("floods.basin.worst", { state: basinStateLabel(basin.state).toLowerCase() })}
      </p>
      {basin.observedAt && (
        <p className="numeric text-xs text-muted-foreground">
          {t("floods.basin.newest", { age: formatAge(basin.observedAt) })}
        </p>
      )}

      <GaugeList title={t("floods.basin.gauges")} items={gauges} />
      <GaugeList title={t("floods.basin.areas")} items={areas} />

      <div className="flex flex-wrap gap-2">
        {basin.riverId && (
          <Button variant="secondary" size="sm" onClick={() => onSelectRiver(basin.riverId!)}>
            {t("floods.basin.openRiver")}
          </Button>
        )}
        {clickedAt && (
          <Button variant="secondary" size="sm" onClick={() => onWeatherAt(clickedAt)}>
            <MapPinIcon />
            {t("floods.basin.weatherHere")}
          </Button>
        )}
      </div>
    </Panel>
  )
}
