import { useT } from "@/lib/i18n"
import type { ScaleStep } from "@/lib/tokens"

import { MapLegend } from "./MapLegend"

interface RampLegendProps {
  title: string
  unit: string
  scale: ScaleStep[]
  /** Where an open first step ("below 21") is labelled, below the next bound. */
  openSpan?: number
  /** Words for the two ends, for screen readers, e.g. ["blue", "red"]. */
  ends?: [string, string]
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
  className?: string
}

/** A continuous colour ramp with its stops labelled underneath. */
export function RampLegend({
  title,
  unit,
  scale,
  openSpan = 3,
  ends,
  bare,
  className,
}: RampLegendProps) {
  const t = useT()
  const stops = scale.map(
    (step, index) => `var(${step.token}) ${((index / (scale.length - 1)) * 100).toFixed(1)}%`,
  )
  const labels = scale.map((step, index) =>
    step.from === null ? `≤${(scale[index + 1]?.from ?? 0) - openSpan}` : String(step.from),
  )
  const range = { from: labels[0] ?? "", to: labels.at(-1) ?? "", unit }
  const description = ends
    ? t("weather.legend.rampEnds", { ...range, start: ends[0], end: ends[1] })
    : t("weather.legend.ramp", range)
  return (
    <MapLegend title={title} unit={unit} bare={bare} className={className}>
      <div
        role="img"
        aria-label={description}
        className="h-2.5 rounded-full ring-1 ring-black/10 ring-inset dark:ring-white/10"
        style={{ backgroundImage: `linear-gradient(to right, ${stops.join(", ")})` }}
      />
      <div
        className="numeric mt-1.5 flex justify-between text-[0.6875rem] text-muted-foreground"
        aria-hidden="true"
      >
        {labels.map((label, index) => (
          <span key={label}>{index === labels.length - 1 ? `${label}+` : label}</span>
        ))}
      </div>
    </MapLegend>
  )
}
