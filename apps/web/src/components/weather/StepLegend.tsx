import type { ScaleStep } from "@/lib/tokens"

import { MapLegend } from "./MapLegend"

interface StepLegendProps {
  title: string
  unit: string
  scale: ScaleStep[]
  /** Optional words under some steps, keyed by step index. */
  names?: Record<number, string>
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
  className?: string
}

/** Discrete colour steps with their lower bounds, as small squares. */
export function StepLegend({ title, unit, scale, names = {}, bare, className }: StepLegendProps) {
  return (
    <MapLegend title={title} unit={unit} bare={bare} className={className}>
      <ol className="grid gap-1" style={{ gridTemplateColumns: `repeat(${scale.length}, 1fr)` }}>
        {scale.map((step, index) => {
          const next = scale[index + 1]?.from
          const range =
            step.from === null
              ? `under ${next}`
              : next === undefined
                ? `${step.from} and above`
                : `${step.from} to ${next}`
          return (
            <li
              key={step.token}
              className="flex flex-col gap-1"
              aria-label={`${range} ${unit}${names[index] ? `, ${names[index]}` : ""}`}
            >
              <span
                aria-hidden="true"
                className="h-2.5 rounded-[3px] ring-1 ring-black/10 ring-inset dark:ring-white/10"
                style={{ backgroundColor: `var(${step.token})` }}
              />
              <span aria-hidden="true" className="numeric text-[0.6875rem] text-muted-foreground">
                {step.from === null ? `<${next}` : next === undefined ? `${step.from}+` : step.from}
              </span>
            </li>
          )
        })}
      </ol>
      {Object.keys(names).length > 0 && (
        <p
          className="mt-1 grid text-[0.6875rem] text-muted-foreground"
          style={{ gridTemplateColumns: `repeat(${scale.length}, 1fr)` }}
          aria-hidden="true"
        >
          {Object.entries(names).map(([index, name]) => {
            const column = Number(index)
            const last = column >= scale.length - 2
            return (
              <span
                key={index}
                className={last ? "text-right whitespace-nowrap" : "whitespace-nowrap"}
                style={{
                  gridColumn: last ? `${column} / span 2` : `${column + 1} / span 2`,
                  gridRow: 1,
                }}
              >
                {name}
              </span>
            )
          })}
        </p>
      )}
    </MapLegend>
  )
}
