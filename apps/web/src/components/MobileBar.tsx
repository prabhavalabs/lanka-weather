import type * as React from "react"
import { LayersIcon } from "lucide-react"

import { AppLogo } from "@/components/AppLogo"
import { ThemeToggle } from "@/components/ThemeToggle"
import { Button } from "@/components/ui/button"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer"

interface MobileBarProps {
  /** Layer list, shown in a bottom sheet. */
  layers: React.ReactNode
  /** Number of layers switched on, shown on the button. */
  activeCount?: number
  /** Small print at the end of the sheet. */
  footer?: React.ReactNode
  /** Beside the logo: room for a compact search button. */
  actions?: React.ReactNode
}

/** The phone header: logo, and a button that opens the layers in a bottom sheet. */
export function MobileBar({ layers, activeCount = 0, footer, actions }: MobileBarProps) {
  return (
    <div className="pointer-events-auto flex items-center gap-2 rounded-2xl border border-border bg-card/95 py-2 pr-2 pl-3 shadow-panel backdrop-blur-md">
      <AppLogo className="min-w-0 flex-1" />
      {actions}
      <Drawer>
        <DrawerTrigger asChild>
          <Button variant="secondary" size="sm" aria-label={`Layers, ${activeCount} on`}>
            <LayersIcon />
            {/* The bar also holds search and warnings: the word goes where it would squeeze the name. */}
            <span className="max-[29.99rem]:sr-only">Layers</span>
            {activeCount > 0 && (
              <span className="numeric rounded-full bg-brand px-1.5 text-[0.6875rem] leading-4 text-brand-foreground">
                {activeCount}
              </span>
            )}
          </Button>
        </DrawerTrigger>
        <DrawerContent className="max-h-[85dvh]">
          <DrawerHeader className="flex-row items-center justify-between gap-3 px-5 group-data-[vaul-drawer-direction=bottom]/drawer-content:text-left">
            <div>
              <DrawerTitle>Layers</DrawerTitle>
              <DrawerDescription>Choose what the map shows.</DrawerDescription>
            </div>
          </DrawerHeader>
          <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-4 [scrollbar-width:thin]">
            {layers}
          </div>
          <div className="flex flex-col gap-3 border-t border-border px-5 py-4 text-xs text-muted-foreground">
            {footer}
            <ThemeToggle showLabels />
          </div>
        </DrawerContent>
      </Drawer>
    </div>
  )
}
