import { formatNumber } from "@/lib/format"
import { cn } from "@/lib/utils"
import { levelWords } from "../location/present"
import type { RegionLevel, RegionWeather } from "./types"

interface RegionAggregateProps {
  region: RegionWeather
  level: RegionLevel
  className?: string
}

const value = (number: number | null, decimals: number, unit: string) =>
  number === null ? "–" : `${formatNumber(number, decimals)}${unit}`

/**
 * A region's weather as a whole: its mean temperature now and today's
 * range, rain now and today, and its strongest wind, with how it was
 * computed (over grid cells or at a point inside it).
 */
export function RegionAggregate({ region, level, className }: RegionAggregateProps) {
  const cells = [
    { label: "Mean now", value: value(region.temperatureC, 1, "°C") },
    {
      label: "Today",
      value:
        region.temperatureMinC === null || region.temperatureMaxC === null
          ? "–"
          : `${formatNumber(region.temperatureMinC)}–${formatNumber(region.temperatureMaxC)}°C`,
    },
    { label: "Max wind", value: value(region.windSpeedMaxKmh, 0, " km/h") },
    { label: "Rain now", value: value(region.precipitationMm, 1, " mm") },
    { label: "Rain today", value: value(region.precipitationTodayMm, 1, " mm") },
    { label: "Max gusts", value: value(region.windGustsMaxKmh, 0, " km/h") },
  ]
  const method =
    region.method === "cells"
      ? `Averaged over ${region.cells} grid ${region.cells === 1 ? "cell" : "cells"}, weighted by area`
      : "Read at a point inside the region"
  return (
    <section className={cn("flex flex-col gap-1.5", className)}>
      <h3 className="text-xs font-medium text-muted-foreground">{levelWords[level]} as a whole</h3>
      <dl className="grid grid-cols-3 gap-px overflow-hidden rounded-xl border border-border bg-border">
        {cells.map((cell) => (
          <div key={cell.label} className="flex min-w-0 flex-col gap-0.5 bg-card px-2.5 py-2">
            <dt className="truncate text-[0.6875rem] text-muted-foreground">{cell.label}</dt>
            <dd className="numeric truncate text-sm text-foreground">{cell.value}</dd>
          </div>
        ))}
      </dl>
      <p className="text-[0.6875rem] text-muted-foreground">{method}.</p>
    </section>
  )
}
