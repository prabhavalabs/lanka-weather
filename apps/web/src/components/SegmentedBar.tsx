import type * as React from "react"

import { cn } from "@/lib/utils"

export interface BarSegment {
  /** Stable key and spoken label, e.g. "Major flood". */
  label: string
  value: number
  /** CSS colour, e.g. "var(--status-major-flood)". */
  color: string
}

interface SegmentedBarProps extends Omit<React.ComponentProps<"div">, "children"> {
  segments: BarSegment[]
  /** Total the segments are a share of; the rest is hatched. Defaults to their sum. */
  total?: number
  /** Accessible name of the whole bar. */
  label: string
}

/**
 * A stacked bar of rounded segments with gaps, and a hatched remainder.
 * Zero-value segments are left out.
 */
export function SegmentedBar({ segments, total, label, className, ...props }: SegmentedBarProps) {
  const sum = segments.reduce((acc, segment) => acc + Math.max(0, segment.value), 0)
  const whole = Math.max(total ?? sum, sum, 0)
  const shown = segments.filter((segment) => segment.value > 0)
  const rest = whole - sum
  const description = shown.map((segment) => `${segment.label} ${segment.value}`).join(", ")

  return (
    <div
      role="img"
      aria-label={`${label}: ${description || "none"}${rest > 0 ? `, ${rest} remaining` : ""}`}
      data-slot="segmented-bar"
      className={cn("flex h-2.5 w-full items-center gap-1", className)}
      {...props}
    >
      {whole === 0 && <div className="hatched h-full flex-1 rounded-full" />}
      {shown.map((segment) => (
        <div
          key={segment.label}
          className="h-full min-w-1.5 rounded-full"
          style={{ flexGrow: segment.value, flexBasis: 0, backgroundColor: segment.color }}
        />
      ))}
      {rest > 0 && (
        <div
          className="hatched h-full min-w-1.5 rounded-full"
          style={{ flexGrow: rest, flexBasis: 0 }}
        />
      )}
    </div>
  )
}
