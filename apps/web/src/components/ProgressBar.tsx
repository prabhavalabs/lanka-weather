import type * as React from "react"

import { cn } from "@/lib/utils"

interface ProgressBarProps extends Omit<React.ComponentProps<"div">, "children"> {
  /** Progress from 0 to 1. */
  value: number
  /** Accessible name, e.g. "Level towards the alert threshold". */
  label: string
  /** Spoken value, e.g. "64 percent"; defaults to the percentage. */
  valueText?: string
  /** Fill colour as a CSS colour; defaults to the brand colour. */
  color?: string
  size?: "sm" | "default"
}

/** A filled bar whose remainder is hatched, as in the reference cards. */
export function ProgressBar({
  value,
  label,
  valueText,
  color,
  size = "default",
  className,
  ...props
}: ProgressBarProps) {
  const fraction = Math.min(1, Math.max(0, Number.isFinite(value) ? value : 0))
  const percent = Math.round(fraction * 100)
  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={percent}
      aria-valuetext={valueText ?? `${percent} percent`}
      data-slot="progress-bar"
      className={cn("flex w-full items-center gap-1", size === "sm" ? "h-1.5" : "h-2.5", className)}
      {...props}
    >
      {fraction > 0 && (
        <div
          className="h-full rounded-full bg-brand transition-[width] duration-500 ease-out"
          style={{ width: `${fraction * 100}%`, backgroundColor: color }}
        />
      )}
      {fraction < 1 && <div className="hatched h-full min-w-1 flex-1 rounded-full" />}
    </div>
  )
}
