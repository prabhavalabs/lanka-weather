import { MapLegend } from "@/components/weather/MapLegend"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { bandLabel, bandOrder, bandRange, bandTokens } from "./present"
import type { RainBand } from "./types"

function Drop({ band }: { band: RainBand }) {
  const hollow = band === "unknown"
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 12 14"
      className={cn("h-3.5 w-3 shrink-0", band === "none" && "opacity-70")}
    >
      <path
        d="M6 0.8C6 0.8 1 6.2 1 9a5 5 0 0 0 10 0C11 6.2 6 0.8 6 0.8Z"
        fill={hollow ? "none" : `var(${bandTokens[band]})`}
        stroke={hollow ? "var(--status-unknown)" : "var(--status-stroke)"}
        strokeWidth={hollow ? 1.5 : 0.5}
      />
    </svg>
  )
}

/** Key to the rain gauges: what each band means, and how many gauges read it. */
export function RainGaugeLegend({
  counts,
  bare = false,
  className,
}: {
  counts?: Record<RainBand, number>
  bare?: boolean
  className?: string
}) {
  const t = useT()
  const entries = (
    <ul className="flex flex-col gap-1.5 text-xs">
      {bandOrder.map((band) => (
        <li key={band} className="flex items-center justify-between gap-2">
          <span className="flex min-w-0 items-center gap-2">
            <Drop band={band} />
            <span className="flex min-w-0 flex-wrap items-baseline gap-x-2">
              <span className="text-foreground">{bandLabel(band)}</span>
              <span className="text-muted-foreground">{bandRange(band)}</span>
            </span>
          </span>
          {counts && <span className="numeric text-foreground">{counts[band]}</span>}
        </li>
      ))}
    </ul>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend
      title={t("rain.gauges.legendTitle")}
      unit={t("rain.gauges.legendUnit")}
      className={cn("w-72", className)}
    >
      {entries}
    </MapLegend>
  )
}
