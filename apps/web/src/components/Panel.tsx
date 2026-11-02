import * as React from "react"

import { IconTile } from "@/components/IconTile"
import { cn } from "@/lib/utils"

interface PanelProps extends Omit<React.ComponentProps<"section">, "title"> {
  /** A line icon, shown in an outlined tile. */
  icon?: React.ReactNode
  title: React.ReactNode
  subtitle?: React.ReactNode
  /** Beside the title: a round icon button, a switch, a segmented control or a chip. */
  action?: React.ReactNode
  /** Under the body, label left and value right, e.g. a source or an update time. */
  footer?: React.ReactNode
  /** Heading level of the title, so a panel fits the page outline. */
  headingLevel?: 2 | 3
  /** Drops the card chrome (border, background, shadow), e.g. inside a sheet. */
  bare?: boolean
}

/**
 * The card shell of the reference designs: a near-black (or white) card with
 * a large radius, an icon tile, a title and muted subtitle, and an action.
 */
export function Panel({
  icon,
  title,
  subtitle,
  action,
  footer,
  headingLevel = 2,
  bare = false,
  className,
  children,
  ...props
}: PanelProps) {
  const titleId = React.useId()
  const Heading = `h${headingLevel}` as const

  return (
    <section
      aria-labelledby={titleId}
      data-slot="panel"
      className={cn(
        "flex flex-col gap-4 text-card-foreground",
        !bare && "rounded-2xl border border-border bg-card p-4 shadow-panel sm:p-5",
        className,
      )}
      {...props}
    >
      <header className="flex items-center gap-3">
        {icon && <IconTile>{icon}</IconTile>}
        <div className="min-w-0 flex-1">
          <Heading id={titleId} className="truncate text-[0.9375rem] leading-tight font-medium">
            {title}
          </Heading>
          {subtitle && (
            <p className="mt-0.5 truncate text-sm leading-tight text-muted-foreground">
              {subtitle}
            </p>
          )}
        </div>
        {action && <div className="flex shrink-0 items-center gap-2">{action}</div>}
      </header>
      {children}
      {footer && (
        <footer className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
          {footer}
        </footer>
      )}
    </section>
  )
}
