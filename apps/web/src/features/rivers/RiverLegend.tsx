import { LegendSwatch } from "@/components/LegendSwatch"
import { MapLegend } from "@/components/weather/MapLegend"
import { cn } from "@/lib/utils"
import { statusColors, statusLabels, statusOrder } from "./present"
import type { RiverStatus } from "./types"

interface RiverLegendProps {
  /** Stations per status, shown beside each entry. */
  counts?: Record<RiverStatus, number>
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
  className?: string
}

/** Key to the river-levels symbols. */
export function RiverLegend({ counts, bare = false, className }: RiverLegendProps) {
  const entries = (
    <ul className="grid grid-cols-2 gap-x-4 gap-y-1.5">
      {statusOrder.map((status) => (
        <li key={status} className="flex items-center justify-between gap-2">
          <LegendSwatch shape="circle" color={statusColors[status]} label={statusLabels[status]} />
          {counts && <span className="numeric text-xs text-foreground">{counts[status]}</span>}
        </li>
      ))}
      <li className="flex items-center gap-2 text-xs text-muted-foreground">
        <span
          aria-hidden="true"
          className="inline-block size-2.5 rounded-full bg-status-normal ring-4 ring-status-normal/30"
        />
        Rising
      </li>
      <li>
        <LegendSwatch shape="circle" hollow color="var(--status-unknown)" label="Out of date" />
      </li>
    </ul>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend title="River levels" unit="status" className={cn("w-72", className)}>
      {entries}
    </MapLegend>
  )
}
