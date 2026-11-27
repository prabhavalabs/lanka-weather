import type * as React from "react"

import { DetailSheet } from "@/components/DetailSheet"
import { ScrollArea } from "@/components/ui/scroll-area"
import { useT } from "@/lib/i18n"
import { desktopQuery, useMediaQuery } from "@/lib/useMediaQuery"
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
  /** Closes the detail; on phones the sheet can be dragged down to close. */
  onDetailClose?: () => void
  /** Accessible name of the detail sheet on phones. */
  detailLabel?: string
  /** On phones, opens the detail sheet at full height, for a list to read. */
  detailFull?: boolean
  /** On phones, opens the detail sheet low, so more of the map stays in view. */
  detailSmall?: boolean
  /** Map legends, bottom left over the map (md and up). */
  legends?: React.ReactNode
  /** Map controls beside the legends, e.g. the wind and rain timelines (md and up). */
  controls?: React.ReactNode
  /** A short notice centred under the toolbar or the phone bar, e.g. "Zoom in to see GN divisions". */
  notice?: React.ReactNode
  /** Phones only: a control along the bottom of the map, e.g. the wind forecast time; hidden under an open sheet. */
  timeline?: React.ReactNode
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
  onDetailClose,
  detailLabel,
  detailFull = false,
  detailSmall = false,
  legends,
  controls,
  notice,
  timeline,
  className,
}: AppShellProps) {
  const desktop = useMediaQuery(desktopQuery, true)
  const t = useT()
  return (
    <main
      className={cn(
        "relative h-dvh w-full overflow-hidden bg-background text-foreground",
        // Width of the sidebar column, which the toolbar and legends sit beside.
        sidebarCollapsed ? "[--sidebar-w:4.5rem]" : "[--sidebar-w:22.75rem]",
        "[--detail-w:24.5rem]",
        // MapLibre's bottom-right controls move left of an open detail column (styles/map.css),
        // and on phones above the timeline.
        detail && "md:[--map-controls-right:var(--detail-w)]",
        timeline && !detail && "max-md:[--map-controls-bottom:8.5rem]",
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

      {notice && (
        <div
          className={cn(
            "pointer-events-none absolute inset-x-0 top-[calc(max(0.75rem,env(safe-area-inset-top))+4.25rem)] z-10 flex justify-center px-3 md:top-16 md:right-3 md:left-[calc(var(--sidebar-w)+0.75rem)]",
            detail && "md:right-[calc(var(--detail-w)+0.75rem)]",
          )}
        >
          {notice}
        </div>
      )}

      {detail && desktop && (
        <div className="pointer-events-none absolute top-0 right-0 bottom-0 z-30 flex w-(--detail-w) flex-col p-3">
          <ScrollArea
            type="auto"
            className="pointer-events-auto flex min-h-0 flex-col rounded-3xl"
            viewportClassName="min-h-0 flex-1"
          >
            {detail}
          </ScrollArea>
        </div>
      )}

      {!desktop && (
        <DetailSheet
          open={Boolean(detail)}
          onClose={() => onDetailClose?.()}
          label={detailLabel ?? t("common.details")}
          openFull={detailFull}
          openSmall={detailSmall}
        >
          {detail}
        </DetailSheet>
      )}

      {timeline && !detail && (
        <div className="pointer-events-none absolute inset-x-0 bottom-0 z-10 flex justify-center p-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] md:hidden">
          {timeline}
        </div>
      )}

      {(legends || controls) && (
        <div
          className={cn(
            "pointer-events-none absolute bottom-3 left-[calc(var(--sidebar-w)+0.75rem)] z-10 hidden max-w-[calc(100%-var(--sidebar-w)-12rem)] flex-wrap items-end gap-3 md:flex",
            detail && "max-w-[calc(100%-var(--sidebar-w)-var(--detail-w)-4rem)]",
          )}
        >
          {controls}
          {legends}
        </div>
      )}
    </main>
  )
}
