import { useT } from "@/lib/i18n"
import { rainfallScale, type ScaleStep } from "@/lib/tokens"

import { StepLegend } from "./StepLegend"

/** Rainfall in mm, from light to extreme: over 24 hours by default, or another scale. */
export function RainfallLegend({
  title,
  unit = "mm",
  scale = rainfallScale,
  names,
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
  const t = useT()
  return (
    <StepLegend
      title={title ?? t("weather.legend.rainfall")}
      unit={unit}
      scale={scale}
      names={
        names ?? {
          0: t("weather.legend.rainLight"),
          3: t("weather.legend.rainHeavy"),
          5: t("weather.legend.rainExtreme"),
        }
      }
      bare={bare}
      className={className}
    />
  )
}
