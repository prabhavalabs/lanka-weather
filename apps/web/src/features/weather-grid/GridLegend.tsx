import { RainfallLegend } from "@/components/weather/RainfallLegend"
import { RampLegend } from "@/components/weather/RampLegend"
import { TemperatureLegend } from "@/components/weather/TemperatureLegend"
import { WindLegend } from "@/components/weather/WindLegend"
import { useT } from "@/lib/i18n"
import { cloudScale, humidityScale, rainRateScale } from "@/lib/tokens"
import type { GridVariable } from "./variables"

/** The legend of the weather field for the variable it shows. */
export function GridLegend({
  variable,
  bare,
  className,
}: {
  variable: GridVariable
  bare?: boolean
  className?: string
}) {
  const t = useT()
  switch (variable) {
    case "temperature":
      return (
        <TemperatureLegend title={t("grid.legend.temperature")} bare={bare} className={className} />
      )
    case "precipitation":
      return (
        <RainfallLegend
          title={t("grid.legend.precipitation")}
          unit="mm/h"
          scale={rainRateScale}
          names={{
            0: t("grid.legend.light"),
            3: t("grid.legend.heavy"),
            5: t("grid.legend.violent"),
          }}
          bare={bare}
          className={className}
        />
      )
    case "humidity":
      return (
        <RampLegend
          title={t("grid.legend.humidity")}
          unit="%"
          scale={humidityScale}
          openSpan={10}
          ends={[t("grid.legend.humidityLow"), t("grid.legend.humidityHigh")]}
          bare={bare}
          className={className}
        />
      )
    case "cloud":
      return (
        <RampLegend
          title={t("grid.legend.cloud")}
          unit="%"
          scale={cloudScale}
          ends={[t("grid.legend.cloudLow"), t("grid.legend.cloudHigh")]}
          bare={bare}
          className={className}
        />
      )
    case "wind":
      return <WindLegend title={t("grid.legend.wind")} bare={bare} className={className} />
  }
}
