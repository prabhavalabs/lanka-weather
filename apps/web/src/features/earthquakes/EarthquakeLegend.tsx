import { MapLegend } from "@/components/weather/MapLegend"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { ageLabel, ageOpacity, type AgeBand } from "./present"

const ages: AgeBand[] = ["day", "week", "month"]
const sizes = [4.5, 6, 7.5]

interface EarthquakeLegendProps {
  bare?: boolean
  className?: string
}

/** Key to the circles: size is magnitude, shade is age, a ring is USGS's tsunami flag. */
export function EarthquakeLegend({ bare = false, className }: EarthquakeLegendProps) {
  const t = useT()
  const entries = (
    <div className="flex flex-col gap-2 text-xs text-muted-foreground">
      <ul className="flex items-end gap-3" aria-label={t("earthquakes.legend.size")}>
        {sizes.map((magnitude) => (
          <li key={magnitude} className="flex items-center gap-1.5">
            <span
              aria-hidden="true"
              className="inline-block rounded-full bg-brand ring-1 ring-black/50"
              style={{ width: (magnitude - 2.5) * 3.2, height: (magnitude - 2.5) * 3.2 }}
            />
            <span className="numeric">M{magnitude}</span>
          </li>
        ))}
      </ul>
      <ul className="flex flex-col gap-1">
        {ages.map((age) => (
          <li key={age} className="flex items-center gap-2">
            <span
              aria-hidden="true"
              className="inline-block size-2.5 shrink-0 rounded-full bg-brand ring-1 ring-black/50"
              style={{ opacity: ageOpacity[age] }}
            />
            {ageLabel(age)}
          </li>
        ))}
        <li className="flex items-center gap-2">
          <span
            aria-hidden="true"
            className="inline-block size-2.5 shrink-0 rounded-full border-2 border-status-major-flood"
          />
          {t("earthquakes.legend.tsunami")}
        </li>
      </ul>
    </div>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend title={t("earthquakes.title")} unit="USGS" className={cn("w-60", className)}>
      {entries}
    </MapLegend>
  )
}
