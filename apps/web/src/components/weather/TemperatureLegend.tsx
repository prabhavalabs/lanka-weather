import { temperatureScale } from "@/lib/tokens"

import { RampLegend } from "./RampLegend"

interface TemperatureLegendProps {
  /** Heading, e.g. "Temperature" or "Feels like". */
  title?: string
  bare?: boolean
  className?: string
}

/** A continuous ramp from cool to hot with its °C stops. */
export function TemperatureLegend({
  title = "Air temperature",
  bare,
  className,
}: TemperatureLegendProps) {
  return (
    <RampLegend
      title={title}
      unit="°C"
      scale={temperatureScale}
      ends={["blue", "red"]}
      bare={bare}
      className={className}
    />
  )
}
