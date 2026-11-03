import { useT } from "@/lib/i18n"
import { temperatureScale } from "@/lib/tokens"

import { RampLegend } from "./RampLegend"

interface TemperatureLegendProps {
  /** Heading, e.g. "Temperature" or "Feels like". */
  title?: string
  bare?: boolean
  className?: string
}

/** A continuous ramp from cool to hot with its °C stops. */
export function TemperatureLegend({ title, bare, className }: TemperatureLegendProps) {
  const t = useT()
  return (
    <RampLegend
      title={title ?? t("weather.legend.temperature")}
      unit="°C"
      scale={temperatureScale}
      ends={[t("weather.legend.cool"), t("weather.legend.hot")]}
      bare={bare}
      className={className}
    />
  )
}
