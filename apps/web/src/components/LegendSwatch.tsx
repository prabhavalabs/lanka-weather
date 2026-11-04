import type * as React from "react"

import { cn } from "@/lib/utils"

interface LegendSwatchProps extends Omit<React.ComponentProps<"span">, "children"> {
  /** CSS colour, e.g. "var(--status-alert)". */
  color: string
  label?: React.ReactNode
  /** Monospaced value after the label, e.g. a count. */
  value?: React.ReactNode
  shape?: "square" | "circle"
  /** Draws the swatch as an outline, e.g. for "out of date". */
  hollow?: boolean
}

/**
 * A small square (or dot) of colour with a label. The swatch has a faint
 * outline so light colours stay visible on light cards; the label carries
 * the meaning, never the colour alone.
 */
export function LegendSwatch({
  color,
  label,
  value,
  shape = "square",
  hollow = false,
  className,
  ...props
}: LegendSwatchProps) {
  return (
    <span
      data-slot="legend-swatch"
      className={cn("inline-flex items-center gap-2 text-xs", className)}
      {...props}
    >
      <span
        aria-hidden="true"
        className={cn(
          "inline-block size-2.5 shrink-0 ring-1 ring-black/25 ring-inset dark:ring-white/10",
          shape === "square" ? "rounded-[3px]" : "rounded-full",
          hollow && "bg-transparent!",
        )}
        style={hollow ? { boxShadow: `inset 0 0 0 2px ${color}` } : { backgroundColor: color }}
      />
      {label && <span className="text-muted-foreground">{label}</span>}
      {value !== undefined && <span className="numeric text-foreground">{value}</span>}
    </span>
  )
}
