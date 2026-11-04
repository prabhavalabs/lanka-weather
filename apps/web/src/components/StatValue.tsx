import type * as React from "react"

import { cn } from "@/lib/utils"

interface StatValueProps extends React.ComponentProps<"div"> {
  /** The number, already formatted, e.g. "2.35" or "31". */
  value: React.ReactNode
  /** Unit after the number, e.g. "m" or "°C". */
  unit?: React.ReactNode
  /** Small label above the number. */
  label?: React.ReactNode
  /** Line under the number, e.g. "of 4.50 m alert level". */
  caption?: React.ReactNode
  /** Change, e.g. "+0.13 m/h"; shown in the signal colour. */
  delta?: React.ReactNode
  /** Words after the delta, e.g. "in the last hour". */
  deltaLabel?: React.ReactNode
  /** Right of the caption, e.g. a percentage. */
  aside?: React.ReactNode
  size?: "default" | "lg" | "sm"
}

/** A big monospaced number with its unit, a delta and a caption. */
export function StatValue({
  value,
  unit,
  label,
  caption,
  delta,
  deltaLabel,
  aside,
  size = "default",
  className,
  ...props
}: StatValueProps) {
  return (
    <div data-slot="stat-value" className={cn("flex flex-col gap-1", className)} {...props}>
      {label && <p className="text-sm text-muted-foreground">{label}</p>}
      <div className="flex items-end justify-between gap-3">
        <p
          className={cn(
            "numeric leading-none font-medium tracking-tight text-foreground",
            size === "lg" && "text-5xl",
            size === "default" && "text-4xl",
            size === "sm" && "text-2xl",
          )}
        >
          {value}
          {unit && (
            <span className="ml-1 align-baseline text-[0.5em] font-normal text-muted-foreground">
              {unit}
            </span>
          )}
        </p>
        {delta && (
          <p className="pb-0.5 text-right text-sm whitespace-nowrap">
            <span className="numeric text-brand-text">{delta}</span>
            {deltaLabel && <span className="ml-1.5 text-muted-foreground">{deltaLabel}</span>}
          </p>
        )}
      </div>
      {(caption || aside) && (
        <div className="flex items-baseline justify-between gap-3 text-sm text-muted-foreground">
          <span>{caption}</span>
          {aside && <span className="numeric text-foreground">{aside}</span>}
        </div>
      )}
    </div>
  )
}
