import { windScale } from "@/lib/tokens"

import { StepLegend } from "./StepLegend"

/** Wind speed steps in km/h, from calm to a very severe cyclonic storm. */
export function WindLegend({
  title = "Wind speed",
  unit = "km/h",
  bare,
  className,
}: {
  title?: string
  unit?: string
  bare?: boolean
  className?: string
}) {
  return (
    <StepLegend
      title={title}
      unit={unit}
      scale={windScale}
      names={{ 0: "Calm", 3: "Strong", 6: "Cyclone" }}
      bare={bare}
      className={className}
    />
  )
}
