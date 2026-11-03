import { formatNumber } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
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
  const t = useT()
  const cells = [
    {
      key: "meanNow",
      label: t("regions.aggregate.meanNow"),
      value: value(region.temperatureC, 1, "°C"),
    },
    {
      key: "today",
      label: t("regions.aggregate.today"),
      value:
        region.temperatureMinC === null || region.temperatureMaxC === null
          ? "–"
          : `${formatNumber(region.temperatureMinC)}–${formatNumber(region.temperatureMaxC)}°C`,
    },
    {
      key: "maxWind",
      label: t("regions.aggregate.maxWind"),
      value: value(region.windSpeedMaxKmh, 0, " km/h"),
    },
    {
      key: "rainNow",
      label: t("regions.aggregate.rainNow"),
      value: value(region.precipitationMm, 1, " mm"),
    },
    {
      key: "rainToday",
      label: t("regions.aggregate.rainToday"),
      value: value(region.precipitationTodayMm, 1, " mm"),
    },
    {
      key: "maxGusts",
      label: t("regions.aggregate.maxGusts"),
      value: value(region.windGustsMaxKmh, 0, " km/h"),
    },
  ]
  const method =
    region.method === "cells"
      ? t("regions.aggregate.cells", { count: region.cells })
      : t("regions.aggregate.point")
  return (
    <section className={cn("flex flex-col gap-1.5", className)}>
      <h3 className="text-xs font-medium text-muted-foreground">
        {t("regions.aggregate.title", { level: t(`terms.adminLevel.${level}`) })}
      </h3>
      <dl className="grid grid-cols-3 gap-px overflow-hidden rounded-xl border border-border bg-border">
        {cells.map((cell) => (
          <div key={cell.key} className="flex min-w-0 flex-col gap-0.5 bg-card px-2.5 py-2">
            <dt className="truncate text-[0.6875rem] text-muted-foreground" title={cell.label}>
              {cell.label}
            </dt>
            <dd className="numeric truncate text-sm text-foreground">{cell.value}</dd>
          </div>
        ))}
      </dl>
      <p className="text-[0.6875rem] text-muted-foreground">{method}</p>
    </section>
  )
}
