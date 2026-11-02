import type * as React from "react"
import { ChevronRightIcon } from "lucide-react"

import { Badge, badgeVariants } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import type { VariantProps } from "class-variance-authority"

type BadgeVariant = NonNullable<VariantProps<typeof badgeVariants>["variant"]>

export interface GaugeListItem {
  id: string
  title: string
  /** Second line, e.g. "Mahaweli Ganga · 4.8 km"; numbers in a `numeric` span. */
  subtitle?: React.ReactNode
  /** Status chip; always a word, never colour alone. */
  badge?: { variant: BadgeVariant; label: string }
  /** Monospaced value on the right, e.g. "3.64 m ↑". */
  value?: React.ReactNode
  /** Opens the item, e.g. its station panel. */
  onSelect?: () => void
}

interface GaugeListProps {
  title: string
  items: GaugeListItem[]
  /** Shown when there are no items; omit to hide the section. */
  emptyText?: string
  className?: string
}

/** A short list of nearby gauges or areas, each with a status chip, opening its details. */
export function GaugeList({ title, items, emptyText, className }: GaugeListProps) {
  if (items.length === 0 && !emptyText) return null
  return (
    <section className={cn("flex flex-col gap-1.5", className)}>
      <h3 className="text-xs font-medium text-muted-foreground">{title}</h3>
      {items.length === 0 ? (
        <p className="rounded-xl border border-border bg-inset px-3 py-2.5 text-sm text-muted-foreground">
          {emptyText}
        </p>
      ) : (
        <ul className="flex flex-col divide-y divide-border overflow-hidden rounded-xl border border-border">
          {items.map((item) => {
            const content = (
              <>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm text-foreground">{item.title}</span>
                  {item.subtitle && (
                    <span className="block truncate text-xs text-muted-foreground">
                      {item.subtitle}
                    </span>
                  )}
                </span>
                {item.value !== undefined && (
                  <span className="numeric shrink-0 text-sm text-foreground">{item.value}</span>
                )}
                {item.badge && (
                  <Badge variant={item.badge.variant} className="shrink-0">
                    {item.badge.label}
                  </Badge>
                )}
                {item.onSelect && (
                  <ChevronRightIcon
                    className="size-4 shrink-0 text-muted-foreground"
                    aria-hidden="true"
                  />
                )}
              </>
            )
            return (
              <li key={item.id}>
                {item.onSelect ? (
                  <button
                    type="button"
                    onClick={item.onSelect}
                    className="flex w-full items-center gap-3 px-3 py-2.5 text-left outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
                  >
                    {content}
                  </button>
                ) : (
                  <div className="flex items-center gap-3 px-3 py-2.5">{content}</div>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}
