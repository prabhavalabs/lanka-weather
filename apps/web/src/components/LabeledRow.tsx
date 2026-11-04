import type * as React from "react"

import { cn } from "@/lib/utils"

/** A list of label–value rows divided by hairlines. */
export function LabeledRows({ className, ...props }: React.ComponentProps<"dl">) {
  return (
    <dl
      data-slot="labeled-rows"
      className={cn("flex flex-col divide-y divide-border", className)}
      {...props}
    />
  )
}

interface LabeledRowProps extends Omit<React.ComponentProps<"div">, "children"> {
  label: React.ReactNode
  value: React.ReactNode
  /** A swatch or icon before the label. */
  leading?: React.ReactNode
  /** `signal` shows the value in the brand colour; `muted` dims it. */
  tone?: "default" | "signal" | "muted" | "warning"
  /** Values are monospaced numbers by default; pass false for words. */
  numeric?: boolean
}

/** Label left, value right in mono. Use inside LabeledRows. */
export function LabeledRow({
  label,
  value,
  leading,
  tone = "default",
  numeric = true,
  className,
  ...props
}: LabeledRowProps) {
  return (
    <div
      data-slot="labeled-row"
      className={cn("flex items-center justify-between gap-4 py-2.5 text-sm", className)}
      {...props}
    >
      <dt className="flex min-w-0 items-center gap-2 text-muted-foreground">
        {leading}
        <span className="truncate">{label}</span>
      </dt>
      <dd
        className={cn(
          "text-right",
          numeric && "numeric",
          tone === "default" && "text-foreground",
          tone === "signal" && "text-brand-text",
          tone === "muted" && "text-muted-foreground",
          tone === "warning" &&
            "text-foreground underline decoration-status-alert decoration-2 underline-offset-4",
        )}
      >
        {value}
      </dd>
    </div>
  )
}
