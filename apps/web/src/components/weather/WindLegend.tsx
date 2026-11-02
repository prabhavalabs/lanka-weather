import { windScale } from "@/lib/tokens"

import { StepLegend } from "./StepLegend"

/** Wind speed steps in km/h, from calm to a very severe cyclonic storm. */
export function WindLegend({
  title = "Wind speed",
  className,
}: {
  title?: string
  className?: string
}) {
  return (
    <StepLegend
      title={title}
      unit="km/h"
      scale={windScale}
      names={{ 0: "Calm", 3: "Strong", 6: "Cyclone" }}
      className={className}
    />
  )
}
