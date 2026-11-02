import { temperatureScale } from "@/lib/tokens"

import { MapLegend } from "./MapLegend"

interface TemperatureLegendProps {
  /** Heading, e.g. "Temperature" or "Feels like". */
  title?: string
  className?: string
}

/** A continuous ramp from cool to hot with its °C stops. */
export function TemperatureLegend({
  title = "Air temperature",
  className,
}: TemperatureLegendProps) {
  const stops = temperatureScale.map(
    (step, index) =>
      `var(${step.token}) ${((index / (temperatureScale.length - 1)) * 100).toFixed(1)}%`,
  )
  const labels = temperatureScale.map((step, index) =>
    step.from === null ? `≤${(temperatureScale[index + 1]?.from ?? 0) - 3}` : String(step.from),
  )
  return (
    <MapLegend title={title} unit="°C" className={className}>
      <div
        role="img"
        aria-label={`Colour ramp from ${labels[0]} °C (blue) to ${labels.at(-1)} °C and above (red)`}
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
