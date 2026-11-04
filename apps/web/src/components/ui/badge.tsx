import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Slot } from "radix-ui"

import { cn } from "@/lib/utils"

/**
 * Small labels. `chip` is the dark rounded chip of the reference cards
 * ("142 days left", "0 of 10"); the status variants carry flood severity
 * and always pair the colour with a word.
 */
const badgeVariants = cva(
  "group/badge inline-flex h-6 w-fit shrink-0 items-center justify-center gap-1.5 overflow-hidden rounded-full border border-transparent px-2.5 text-xs font-medium whitespace-nowrap transition-colors focus-visible:ring-2 focus-visible:ring-ring [&>svg]:pointer-events-none [&>svg]:size-3.5!",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground",
        secondary: "bg-secondary text-secondary-foreground",
        outline: "border-border text-foreground",
        chip: "rounded-lg border-border bg-muted text-muted-foreground",
        brand: "border-brand bg-brand-soft text-brand-soft-foreground",
        destructive: "bg-destructive/12 text-destructive",
        normal: "bg-status-normal text-status-normal-foreground",
        alert: "bg-status-alert text-status-alert-foreground",
        minor_flood: "bg-status-minor-flood text-status-minor-flood-foreground",
        major_flood: "bg-status-major-flood text-status-major-flood-foreground",
        unknown: "bg-status-unknown text-status-unknown-foreground",
        flood_normal: "bg-[var(--flood-normal)] text-[var(--flood-normal-foreground)]",
        flood_warning: "bg-[var(--flood-warning)] text-[var(--flood-warning-foreground)]",
        flood_danger: "bg-[var(--flood-danger)] text-[var(--flood-danger-foreground)]",
        flood_extreme: "bg-[var(--flood-extreme)] text-[var(--flood-extreme-foreground)]",
        flood_nodata: "bg-[var(--flood-nodata)] text-[var(--flood-nodata-foreground)]",
        spilling: "bg-reservoir-spilling text-reservoir-spilling-foreground",
        releasing: "bg-reservoir-releasing text-reservoir-releasing-foreground",
        holding: "bg-reservoir-holding text-reservoir-holding-foreground",
        nodata: "border-reservoir-nodata bg-transparent text-foreground",
        ghost: "text-muted-foreground hover:bg-accent",
        link: "text-brand-text underline-offset-4 hover:underline",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
)

function Badge({
  className,
  variant = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : "span"

  return (
    <Comp
      data-slot="badge"
      data-variant={variant}
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
