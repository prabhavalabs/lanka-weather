import { rainfallScale } from "@/lib/tokens"

import { StepLegend } from "./StepLegend"

/** Rainfall over 24 hours in mm, from light to extreme. */
export function RainfallLegend({
  title = "Rainfall, 24 h",
  className,
}: {
  title?: string
  className?: string
}) {
  return (
    <StepLegend
      title={title}
      unit="mm"
      scale={rainfallScale}
      names={{ 0: "Light", 3: "Heavy", 5: "Extreme" }}
      className={className}
    />
  )
}
