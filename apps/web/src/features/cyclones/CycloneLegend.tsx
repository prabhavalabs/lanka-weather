import { MapLegend } from "@/components/weather/MapLegend"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

interface CycloneLegendProps {
  bare?: boolean
  className?: string
}

/** Key to a system's lines and areas. */
export function CycloneLegend({ bare = false, className }: CycloneLegendProps) {
  const t = useT()
  const entries = (
    <ul className="grid grid-cols-2 gap-x-3 gap-y-1.5 text-xs text-muted-foreground">
      <li className="flex min-w-0 items-center gap-2">
        <span
          aria-hidden="true"
          className="inline-block h-0 w-5 shrink-0 border-t-2 border-foreground"
        />
        {t("cyclones.legend.track")}
      </li>
      <li className="flex min-w-0 items-center gap-2">
        <span
          aria-hidden="true"
          className="inline-block h-0 w-5 shrink-0 border-t-2 border-dashed border-foreground"
        />
        {t("cyclones.legend.forecast")}
      </li>
      <li className="flex min-w-0 items-center gap-2">
        <span
          aria-hidden="true"
          className="inline-block size-3 shrink-0 rounded-sm border border-status-minor-flood bg-status-minor-flood/20"
        />
        {t("cyclones.legend.buffer")}
      </li>
      <li className="flex min-w-0 items-center gap-2">
        <span
          aria-hidden="true"
          className="inline-block h-0 w-5 shrink-0 border-t border-dotted border-foreground"
        />
        {t("cyclones.legend.uncertainty")}
      </li>
    </ul>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend title={t("cyclones.title")} unit="GDACS" className={cn("w-64", className)}>
      {entries}
    </MapLegend>
  )
}
