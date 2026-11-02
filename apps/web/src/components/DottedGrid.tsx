import type * as React from "react"

import { cn } from "@/lib/utils"

/** An inset panel with a subtle dotted grid, for charts and illustrations. */
export function DottedGrid({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="dotted-grid"
      className={cn("dot-grid relative rounded-2xl border border-border bg-inset", className)}
      {...props}
    />
  )
}
