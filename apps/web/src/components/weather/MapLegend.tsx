import type * as React from "react"
import { ChevronDownIcon } from "lucide-react"
import { useId, useState } from "react"

import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

interface MapLegendProps extends Omit<React.ComponentProps<"section">, "title"> {
  title: React.ReactNode
  /** Unit or note after the title, e.g. "°C" or "km/h". */
  unit?: React.ReactNode
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
}

/**
 * The small card a map legend sits in, over the map. Its title folds it to
 * a single line, so a legend read once stops covering the map it explains.
 */
export function MapLegend({
  title,
  unit,
  bare = false,
  className,
  children,
  ...props
}: MapLegendProps) {
  const t = useT()
  const bodyId = useId()
  const [folded, setFolded] = useState(false)

  if (bare) {
    return (
      <section className={cn("w-full text-card-foreground", className)} {...props}>
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

  return (
    <section
      className={cn(
        "pointer-events-auto w-64 rounded-2xl border border-border bg-card/95 text-card-foreground shadow-panel backdrop-blur-md",
        className,
        folded && "w-auto max-w-64",
      )}
      {...props}
    >
      <h2 className="text-xs font-medium">
        <button
          type="button"
          aria-expanded={!folded}
          aria-controls={folded ? undefined : bodyId}
          title={t(folded ? "weather.legendDock.unfold" : "weather.legendDock.fold")}
          onClick={() => setFolded((value) => !value)}
          className={cn(
            "flex w-full items-baseline gap-2 rounded-2xl px-3.5 pt-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring",
            folded ? "pb-3" : "pb-2.5",
          )}
        >
          <span className="min-w-0 flex-1 break-words">{title}</span>
          {unit && (
            <span className="numeric shrink-0 text-[0.6875rem] font-normal text-muted-foreground">
              {unit}
            </span>
          )}
          <ChevronDownIcon
            aria-hidden="true"
            className={cn(
              "size-3.5 shrink-0 self-center text-muted-foreground transition-transform",
              folded && "-rotate-90",
            )}
          />
        </button>
      </h2>
      {!folded && (
        <div id={bodyId} className="px-3.5 pb-3">
          {children}
        </div>
      )}
    </section>
  )
}
