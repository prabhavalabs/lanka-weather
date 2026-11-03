import { RainfallLegend } from "@/components/weather/RainfallLegend"
import { StepLegend } from "@/components/weather/StepLegend"
import { WindLegend } from "@/components/weather/WindLegend"
import { useT } from "@/lib/i18n"
import { temperatureScale } from "@/lib/tokens"
import type { RegionVariable } from "./present"
import type { RegionLevel } from "./types"

/** The legend of the region colours: the variable's steps, per region of the level. */
export function RegionLegend({
  variable,
  level,
  bare,
  className,
}: {
  variable: RegionVariable
  level: RegionLevel
  bare?: boolean
  className?: string
}) {
  const t = useT()
  const by = t(`regions.by.${level}`)
  switch (variable) {
    case "temperature":
      return (
        <StepLegend
          title={t("regions.legend.temperature", { by })}
          unit="°C"
          scale={temperatureScale}
          bare={bare}
          className={className}
        />
      )
    case "rain":
      return (
        <RainfallLegend
          title={t("regions.legend.rain", { by })}
          unit="mm"
          bare={bare}
          className={className}
        />
      )
    case "wind":
      return (
        <WindLegend
          title={t("regions.legend.wind", { by })}
          unit="km/h"
          bare={bare}
          className={className}
        />
      )
  }
}
