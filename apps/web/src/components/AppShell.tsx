import type * as React from "react"

import { cn } from "@/lib/utils"

interface AppShellProps {
  /** The full-screen map. */
  map: React.ReactNode
  /** Left sidebar on tablets and desktops (md and up). */
  sidebar: React.ReactNode
  /** True when the sidebar is collapsed to its small card. */
  sidebarCollapsed?: boolean
  /** Top bar on phones: logo, layers button and, later, search. */
  mobileBar: React.ReactNode
  /**
   * Top of the map on md and up, right of the sidebar: room for the search
   * box (SearchTrigger) and the boundary level selector (AdminLevelSelector).
   */
  toolbar?: React.ReactNode
  /**
   * The detail column: on md and up a right-hand column of cards, on phones
   * a sheet over the bottom of the map. Holds the station panel today and
   * the location weather panel later.
   */
  detail?: React.ReactNode
  /** Map legends, bottom left over the map (md and up). */
  legends?: React.ReactNode
  className?: string
}

/**
 * The page layout: a full-screen map with floating cards around it. Each
 * region is a slot so features can be added without reworking the layout.
 */
export function AppShell({
  map,
  sidebar,
  sidebarCollapsed = false,
  mobileBar,
  toolbar,
  detail,
  legends,
  className,
}: AppShellProps) {
  return (
    <main
      className={cn(
        "relative h-dvh w-full overflow-hidden bg-background text-foreground",
        // Width of the sidebar column, which the toolbar and legends sit beside.
        sidebarCollapsed ? "[--sidebar-w:4.5rem]" : "[--sidebar-w:22.75rem]",
        "[--detail-w:24.5rem]",
        // MapLibre's bottom-right controls move left of an open detail column (styles/map.css).
        detail && "md:[--map-controls-right:var(--detail-w)]",
        className,
      )}
    >
      {map}

      <div className="pointer-events-none absolute inset-y-0 left-0 z-20 hidden p-3 md:flex">
        {sidebar}
      </div>

      <div className="pointer-events-none absolute inset-x-0 top-0 z-20 p-3 pt-[max(0.75rem,env(safe-area-inset-top))] md:hidden">
        {mobileBar}
      </div>

      {toolbar && (
        <div
          className={cn(
            "pointer-events-none absolute top-3 right-3 left-[calc(var(--sidebar-w)+0.75rem)] z-10 hidden justify-center md:flex",
            detail && "right-[calc(var(--detail-w)+0.75rem)]",
          )}
        >
          {toolbar}
        </div>
      )}

      {detail && (
        <div className="pointer-events-none absolute inset-x-0 bottom-0 z-30 flex max-h-[62dvh] flex-col p-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] md:inset-x-auto md:top-0 md:right-0 md:bottom-0 md:max-h-none md:w-(--detail-w) md:p-3">
          <div className="pointer-events-auto min-h-0 overflow-y-auto overscroll-contain rounded-3xl [scrollbar-width:none] md:[scrollbar-width:thin]">
            {detail}
          </div>
        </div>
      )}

      {legends && (
        <div
          className={cn(
            "pointer-events-none absolute bottom-3 left-[calc(var(--sidebar-w)+0.75rem)] z-10 hidden max-w-[calc(100%-var(--sidebar-w)-12rem)] flex-wrap items-end gap-3 md:flex",
            detail && "max-w-[calc(100%-var(--sidebar-w)-var(--detail-w)-4rem)]",
          )}
        >
          {legends}
        </div>
      )}
    </main>
  )
}
