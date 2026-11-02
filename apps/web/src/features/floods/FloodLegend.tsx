import { MapLegend } from "@/components/weather/MapLegend"
import { cn } from "@/lib/utils"
import { floodSeverityColors, floodSeverityLabels, floodSeverityOrder } from "./present"
import type { FloodSeverity } from "./types"

interface FloodLegendProps {
  /** Gauges per severity, shown beside each entry. */
  counts?: Record<FloodSeverity, number>
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
  className?: string
}

function Diamond({ color }: { color: string }) {
  return (
    <span
      aria-hidden="true"
      className="inline-block size-2.5 shrink-0 rotate-45 rounded-[2px] ring-1 ring-black/40 dark:ring-black/60"
      style={{ backgroundColor: color }}
    />
  )
}

/** Key to the flood forecast symbols: severity colours, trend arrows and areas. */
export function FloodLegend({ counts, bare = false, className }: FloodLegendProps) {
  const entries = (
    <ul className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-xs">
      {floodSeverityOrder.map((severity) => (
        <li key={severity} className="flex items-center justify-between gap-2">
          <span className="flex items-center gap-2">
            <Diamond color={floodSeverityColors[severity]} />
            <span className="text-muted-foreground">{floodSeverityLabels[severity]}</span>
          </span>
          {counts && <span className="numeric text-foreground">{counts[severity]}</span>}
        </li>
      ))}
      <li className="flex items-center gap-2 text-muted-foreground">
        <span aria-hidden="true" className="numeric text-foreground">
          ↑↓
        </span>
        Forecast trend
      </li>
      <li className="flex items-center gap-2 text-muted-foreground">
        <span
          aria-hidden="true"
          className="inline-block h-2.5 w-3.5 rounded-[2px] border border-dashed border-status-minor-flood bg-status-minor-flood/25"
        />
        Flood area
      </li>
    </ul>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend title="Flood forecast" unit="Google" className={cn("w-72", className)}>
      {entries}
    </MapLegend>
  )
}
