import { RainfallLegend } from "@/components/weather/RainfallLegend"
import { RampLegend } from "@/components/weather/RampLegend"
import { TemperatureLegend } from "@/components/weather/TemperatureLegend"
import { WindLegend } from "@/components/weather/WindLegend"
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
  switch (variable) {
    case "temperature":
      return <TemperatureLegend title="Temperature now" bare={bare} className={className} />
    case "precipitation":
      return (
        <RainfallLegend
          title="Rain, this hour"
          unit="mm/h"
          scale={rainRateScale}
          names={{ 0: "Light", 3: "Heavy", 5: "Violent" }}
          bare={bare}
          className={className}
        />
      )
    case "humidity":
      return (
        <RampLegend
          title="Relative humidity"
          unit="%"
          scale={humidityScale}
          openSpan={10}
          ends={["sand", "blue"]}
          bare={bare}
          className={className}
        />
      )
    case "cloud":
      return (
        <RampLegend
          title="Cloud cover"
          unit="%"
          scale={cloudScale}
          ends={["clear", "overcast"]}
          bare={bare}
          className={className}
        />
      )
    case "wind":
      return <WindLegend title="Wind speed now" bare={bare} className={className} />
  }
}
