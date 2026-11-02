import type * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const iconTileVariants = cva(
  "inline-flex shrink-0 items-center justify-center rounded-xl border text-foreground [&_svg]:shrink-0",
  {
    variants: {
      size: {
        sm: "size-8 rounded-lg [&_svg:not([class*='size-'])]:size-4",
        default: "size-10 [&_svg:not([class*='size-'])]:size-[1.125rem]",
        lg: "size-12 rounded-2xl [&_svg:not([class*='size-'])]:size-5",
      },
      tone: {
        default: "border-border bg-transparent",
        muted: "border-border bg-muted",
        brand: "border-brand bg-brand-soft text-brand-soft-foreground",
      },
    },
    defaultVariants: { size: "default", tone: "default" },
  },
)

/** A rounded-square outline holding a line icon; sits beside a title and subtitle. */
export function IconTile({
  className,
  size,
  tone,
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof iconTileVariants>) {
  return (
    <span
      aria-hidden="true"
      data-slot="icon-tile"
      className={cn(iconTileVariants({ size, tone }), className)}
      {...props}
    />
  )
}
