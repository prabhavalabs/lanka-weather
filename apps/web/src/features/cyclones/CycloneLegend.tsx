import { MapLegend } from "@/components/weather/MapLegend"
import { cn } from "@/lib/utils"

interface CycloneLegendProps {
  bare?: boolean
  className?: string
}

/** Key to a system's lines and areas. */
export function CycloneLegend({ bare = false, className }: CycloneLegendProps) {
  const entries = (
    <ul className="grid grid-cols-2 gap-x-3 gap-y-1.5 text-xs text-muted-foreground">
      <li className="flex items-center gap-2">
        <span aria-hidden="true" className="inline-block h-0 w-5 border-t-2 border-foreground" />
        Track so far
      </li>
      <li className="flex items-center gap-2">
        <span
          aria-hidden="true"
          className="inline-block h-0 w-5 border-t-2 border-dashed border-foreground"
        />
        Forecast
      </li>
      <li className="flex items-center gap-2">
        <span
          aria-hidden="true"
          className="inline-block size-3 rounded-sm border border-status-minor-flood bg-status-minor-flood/20"
        />
        Wind buffer
      </li>
      <li className="flex items-center gap-2">
        <span
          aria-hidden="true"
          className="inline-block h-0 w-5 border-t border-dotted border-foreground"
        />
        Uncertainty
      </li>
    </ul>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend title="Cyclones" unit="GDACS" className={cn("w-64", className)}>
      {entries}
    </MapLegend>
  )
}
