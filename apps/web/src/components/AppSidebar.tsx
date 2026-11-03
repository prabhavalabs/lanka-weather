import type * as React from "react"
import { PanelLeftCloseIcon, PanelLeftOpenIcon } from "lucide-react"

import { AppLogo } from "@/components/AppLogo"
import { ThemeToggle } from "@/components/ThemeToggle"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { InstallButton } from "@/features/install"
import { cn } from "@/lib/utils"

interface AppSidebarProps {
  collapsed: boolean
  onCollapsedChange: (collapsed: boolean) => void
  /** Under the logo: room for the search box. */
  top?: React.ReactNode
  /** The scrolling body, e.g. the layer list. */
  children: React.ReactNode
  /** Small print above the theme switch, e.g. update time and sources. */
  footer?: React.ReactNode
}

/** The left sidebar on md and up: logo, layers and settings. Collapses to a small card. */
export function AppSidebar({
  collapsed,
  onCollapsedChange,
  top,
  children,
  footer,
}: AppSidebarProps) {
  if (collapsed) {
    return (
      <div className="pointer-events-auto flex h-fit flex-col items-center gap-2 rounded-2xl border border-sidebar-border bg-sidebar p-2 text-sidebar-foreground shadow-panel">
        <AppLogo variant="mark" className="px-1.5 py-1" />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onCollapsedChange(false)}
              aria-label="Show layers"
              aria-expanded={false}
            >
              <PanelLeftOpenIcon />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="right">Show layers</TooltipContent>
        </Tooltip>
      </div>
    )
  }

  return (
    <aside
      aria-label="Layers and settings"
      className="pointer-events-auto flex w-[22rem] flex-col overflow-hidden rounded-3xl border border-sidebar-border bg-sidebar text-sidebar-foreground shadow-panel"
    >
      <header className="flex items-center gap-2 px-5 pt-5 pb-4">
        <AppLogo tagline="Weather and hazard map of Sri Lanka" className="min-w-0 flex-1" />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="outline"
              size="icon"
              onClick={() => onCollapsedChange(true)}
              aria-label="Hide layers"
              aria-expanded={true}
            >
              <PanelLeftCloseIcon />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="right">Hide layers</TooltipContent>
        </Tooltip>
      </header>
      {top && <div className="px-5 pb-4">{top}</div>}
      <div className="h-px bg-sidebar-border" />
      {/* Group headings stick to the top of the list on this colour. */}
      <ScrollArea type="auto" className="min-h-0 flex-1 [--panel-bg:var(--color-sidebar)]">
        <div className="px-4 pt-2 pb-4">{children}</div>
      </ScrollArea>
      <footer
        className={cn(
          "flex flex-col gap-3 border-t border-sidebar-border px-5 py-4 text-xs text-muted-foreground",
        )}
      >
        {footer}
        <InstallButton />
        <div className="flex items-center justify-between gap-3">
          <span>Appearance</span>
          <ThemeToggle />
        </div>
      </footer>
    </aside>
  )
}
