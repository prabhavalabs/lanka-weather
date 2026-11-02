import { RainfallLegend } from "@/components/weather/RainfallLegend"
import { StepLegend } from "@/components/weather/StepLegend"
import { WindLegend } from "@/components/weather/WindLegend"
import { temperatureScale } from "@/lib/tokens"
import { levelWords } from "../location/present"
import type { RegionVariable } from "./present"
import type { RegionLevel } from "./types"

const by: Record<RegionLevel, string> = {
  province: "by province",
  district: "by district",
  ds: "by DS division",
  gn: "by GN division",
}

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
  const each = by[level] ?? levelWords[level].toLowerCase()
  switch (variable) {
    case "temperature":
      return (
        <StepLegend
          title={`Temperature ${each}`}
          unit="°C"
          scale={temperatureScale}
          bare={bare}
          className={className}
        />
      )
    case "rain":
      return (
        <RainfallLegend title={`Rain today ${each}`} unit="mm" bare={bare} className={className} />
      )
    case "wind":
      return <WindLegend title={`Max wind ${each}`} unit="km/h" bare={bare} className={className} />
  }
}
