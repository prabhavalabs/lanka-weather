import { MapIcon, MapPinIcon, XIcon } from "lucide-react"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { formatAge } from "@/lib/format"
import { cn } from "@/lib/utils"
import { formatLevel } from "../rivers/present"
import type { BasinGauge, BasinsDocument, BasinStatus } from "../river-risk/types"
import { basinBadge, basinStateLabels } from "./basins"
import { floodAreaLabels, floodTrendArrows } from "./present"
import type { FloodAreaKind } from "./types"

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
  gauge: BasinGauge,
  onSelectStation: (id: string) => void,
  onSelectGauge: (id: string) => void,
): GaugeListItem {
  const notes = [gauge.stale ? "Out of date" : formatAge(gauge.observedAt)]
  if (gauge.kind === "google") notes.push("Google")
  if (!gauge.verified) notes.push("not verified by Google")
  return {
    id: `${gauge.kind}:${gauge.id}`,
    title: gauge.name,
    subtitle: notes.join(" · "),
    value:
      gauge.waterLevel !== undefined && gauge.unit
        ? formatLevel(gauge.waterLevel, gauge.unit as "m" | "ft")
        : floodTrendArrows[gauge.trend as keyof typeof floodTrendArrows] || undefined,
    badge: gauge.state
      ? { variant: basinBadge[gauge.state], label: basinStateLabels[gauge.state] }
      : { variant: "unknown", label: "Not used" },
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
  const gauges = basin.gauges.map((gauge) => gaugeItem(gauge, onSelectStation, onSelectGauge))
  const areas: GaugeListItem[] = basin.areas.map((area) => ({
    id: area.id,
    title: floodAreaLabels[area.kind as FloodAreaKind] ?? area.kind,
    badge: { variant: basinBadge[area.state], label: basinStateLabels[area.state] },
    onSelect: () => onSelectArea(area.id),
  }))
  return (
    <Panel
      icon={<MapIcon />}
      title={basin.name}
      subtitle={
        <>
          {basin.named ? "River basin" : "Coast outside any named basin"} ·{" "}
          <span className="numeric">{Math.round(basin.areaKm2).toLocaleString("en-GB")} km²</span>
        </>
      }
      className={cn("w-full", className)}
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label="Close basin details">
          <XIcon />
        </Button>
      }
      footer={
        <p>
          Boundary: {meta.attribution}. Official flood warnings come from the Irrigation Department
          and the DMC.
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={basinBadge[basin.state]}>{basinStateLabels[basin.state]}</Badge>
        {basin.onlyUnverified && (
          <Badge variant="chip" className="text-foreground">
            Unverified gauges only
          </Badge>
        )}
        {meta.seeded && (
          <Badge variant="destructive" className="uppercase">
            Test data
          </Badge>
        )}
      </div>

      <p className="text-sm text-foreground">
        {basin.state === "no_forecast"
          ? "No current gauge or forecast covers this basin, so nothing is known about flooding here. That does not mean it is safe."
          : basin.state === "no_flooding"
            ? "No gauge or forecast in this basin is above normal."
            : `The worst gauge or forecast in this basin reads ${basinStateLabels[basin.state].toLowerCase()}.`}
      </p>
      {basin.observedAt && (
        <p className="numeric text-xs text-muted-foreground">
          Newest reading {formatAge(basin.observedAt)}
        </p>
      )}

      <GaugeList title="Gauges and forecasts here" items={gauges} />
      <GaugeList title="Google flood areas touching this basin" items={areas} />

      <div className="flex flex-wrap gap-2">
        {basin.riverId && (
          <Button variant="secondary" size="sm" onClick={() => onSelectRiver(basin.riverId!)}>
            Open the river
          </Button>
        )}
        {clickedAt && (
          <Button variant="secondary" size="sm" onClick={() => onWeatherAt(clickedAt)}>
            <MapPinIcon />
            Weather at this place
          </Button>
        )}
      </div>
    </Panel>
  )
}
