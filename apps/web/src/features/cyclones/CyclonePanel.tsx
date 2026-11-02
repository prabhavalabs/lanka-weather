import { ExternalLinkIcon, TornadoIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatAge, formatDayTime, formatNumber } from "@/lib/format"
import { formatDistance } from "../earthquakes/present"
import { alertBadge, evaluationNote } from "./present"
import type { CyclonePosition } from "./types"

interface CyclonePanelProps {
  system: CyclonePosition
  /** The layer has not been checked recently. */
  stale?: boolean
  onClose: () => void
}

/** One cyclone: alert, position, distance from Sri Lanka and the link to GDACS. */
export function CyclonePanel({ system, stale = false, onClose }: CyclonePanelProps) {
  return (
    <Panel
      icon={<TornadoIcon />}
      title={system.name}
      subtitle="Tropical cyclone · North Indian Ocean"
      className="w-full"
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label="Close cyclone details">
          <XIcon />
        </Button>
      }
      footer={
        <p>
          Source: Global Disaster Alert and Coordination System (GDACS), gdacs.org. Fetched{" "}
          <span className="numeric">{formatAge(system.fetchedAt)}</span>.
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={alertBadge[system.alertLevel] ?? "unknown"}>
          GDACS {system.alertLevel} alert
        </Badge>
        {(stale || system.stale) && (
          <Badge variant="chip" className="text-foreground">
            Out of date
          </Badge>
        )}
      </div>
      {system.severityText && <p className="text-sm text-foreground">{system.severityText}</p>}
      <LabeledRows>
        <LabeledRow label="From Sri Lanka's coast" value={formatDistance(system.distanceKm)} />
        {system.windKmh !== null && (
          <LabeledRow label="Maximum wind" value={`${formatNumber(system.windKmh)} km/h`} />
        )}
        <LabeledRow
          label="Position"
          value={`${Math.abs(system.lat).toFixed(1)}° ${system.lat >= 0 ? "N" : "S"}, ${Math.abs(system.lon).toFixed(1)}° E`}
        />
        <LabeledRow
          label="Advisory"
          value={`${formatDayTime(system.latestAt)} · ${formatAge(system.latestAt)}`}
          tone={system.stale ? "warning" : "default"}
        />
        <LabeledRow label="First seen" value={formatDayTime(system.startedAt)} />
        {system.country && (
          <LabeledRow label="GDACS lists" value={system.country} numeric={false} />
        )}
        {system.advisorySource && (
          <LabeledRow label="Track from" value={system.advisorySource} numeric={false} />
        )}
      </LabeledRows>
      <a
        href={system.reportUrl}
        target="_blank"
        rel="noreferrer"
        className="inline-flex items-center gap-1.5 self-start rounded-sm text-sm font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
      >
        GDACS report
        <ExternalLinkIcon className="size-3.5" aria-hidden="true" />
        <span className="sr-only">(opens in a new tab)</span>
      </a>
      <p className="text-xs text-muted-foreground">
        GDACS&apos;s alert rates likely impact on the countries it lists; it is not a warning for
        Sri Lanka. Official warnings come from the Department of Meteorology and the DMC.{" "}
        {evaluationNote}
      </p>
    </Panel>
  )
}
