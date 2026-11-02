import { rainfallScale, type ScaleStep } from "@/lib/tokens"

import { StepLegend } from "./StepLegend"

/** Rainfall in mm, from light to extreme: over 24 hours by default, or another scale. */
export function RainfallLegend({
  title = "Rainfall, 24 h",
  unit = "mm",
  scale = rainfallScale,
  names = { 0: "Light", 3: "Heavy", 5: "Extreme" },
  bare,
  className,
}: {
  title?: string
  unit?: string
  scale?: ScaleStep[]
  names?: Record<number, string>
  bare?: boolean
  className?: string
}) {
  return (
    <StepLegend
      title={title}
      unit={unit}
      scale={scale}
      names={names}
      bare={bare}
      className={className}
    />
  )
}
