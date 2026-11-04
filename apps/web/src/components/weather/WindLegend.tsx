import { useT } from "@/lib/i18n"
import { windScale } from "@/lib/tokens"

import { StepLegend } from "./StepLegend"

/** Wind speed steps in km/h, from calm to a very severe cyclonic storm. */
export function WindLegend({
  title,
  unit = "km/h",
  bare,
  className,
}: {
  title?: string
  unit?: string
  bare?: boolean
  className?: string
}) {
  const t = useT()
  return (
    <StepLegend
      title={title ?? t("weather.legend.wind")}
      unit={unit}
      scale={windScale}
      names={{
        0: t("weather.legend.calm"),
        3: t("weather.legend.strong"),
        6: t("terms.weatherSystem.cyclone"),
      }}
      bare={bare}
      className={className}
    />
  )
}
