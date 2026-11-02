import { MapLegend } from "@/components/weather/MapLegend"
import { cn } from "@/lib/utils"
import { reservoirStatusLabels, reservoirStatusOrder } from "./present"
import { ReservoirSymbol } from "./ReservoirSymbol"
import type { ReservoirCounts } from "./types"

interface ReservoirLegendProps {
  /** Reservoirs per status, shown beside each entry. */
  counts?: ReservoirCounts
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
  className?: string
}

/** Key to the reservoir symbols. */
export function ReservoirLegend({ counts, bare = false, className }: ReservoirLegendProps) {
  const entries = (
    <ul className="grid grid-cols-2 gap-x-4 gap-y-1.5">
      {reservoirStatusOrder.map((status) => (
        <li key={status} className="flex items-center justify-between gap-2">
          <span className="inline-flex items-center gap-2 text-xs text-muted-foreground">
            <ReservoirSymbol status={status} />
            {reservoirStatusLabels[status]}
          </span>
          {counts && <span className="numeric text-xs text-foreground">{counts[status]}</span>}
        </li>
      ))}
      <li className="col-span-2 text-xs text-muted-foreground">Hollow: no data or out of date</li>
    </ul>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend title="Reservoirs" unit="status" className={cn("w-72", className)}>
      {entries}
    </MapLegend>
  )
}
