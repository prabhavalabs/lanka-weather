import type * as React from "react"

import { cn } from "@/lib/utils"

interface MapLegendProps extends Omit<React.ComponentProps<"section">, "title"> {
  title: React.ReactNode
  /** Unit or note after the title, e.g. "°C" or "km/h". */
  unit?: React.ReactNode
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
}

/** The small card a map legend sits in, over the map. */
export function MapLegend({
  title,
  unit,
  bare = false,
  className,
  children,
  ...props
}: MapLegendProps) {
  return (
    <section
      className={cn(
        bare
          ? "w-full text-card-foreground"
          : "pointer-events-auto w-64 rounded-2xl border border-border bg-card/95 px-3.5 py-3 text-card-foreground shadow-panel backdrop-blur-md",
        className,
      )}
      {...props}
    >
      <header className="mb-2.5 flex items-baseline justify-between gap-2">
        <h2 className="min-w-0 text-xs font-medium break-words">{title}</h2>
        {unit && (
          <span className="numeric shrink-0 text-[0.6875rem] text-muted-foreground">{unit}</span>
        )}
      </header>
      {children}
    </section>
  )
}
