import * as React from "react"

import { Drawer, DrawerContent, DrawerTitle } from "@/components/ui/drawer"
import { cn } from "@/lib/utils"
import { phoneTopInset, sheetPeekHeight } from "@/features/map/reveal"

interface DetailSheetProps {
  open: boolean
  onClose: () => void
  /** Accessible name of the sheet. */
  label: string
  children: React.ReactNode
}

function useViewportHeight() {
  const [height, setHeight] = React.useState(() => window.innerHeight)
  React.useEffect(() => {
    const update = () => setHeight(window.innerHeight)
    window.addEventListener("resize", update)
    return () => window.removeEventListener("resize", update)
  }, [])
  return height
}

/**
 * The detail panel on phones: a sheet over the bottom of the map that
 * opens part way (the map above stays usable) and can be dragged up to
 * fill the screen below the top bar, or down to close. It is not modal.
 */
export function DetailSheet({ open, onClose, label, children }: DetailSheetProps) {
  const height = useViewportHeight()
  // The sheet is as tall as the screen below the top bar; vaul measures snap
  // points from the bottom of the window, so the peek adds the bar's height.
  const peek = `${sheetPeekHeight(height) + phoneTopInset}px`
  const snapPoints = React.useMemo(() => [peek, 1], [peek])
  const [snap, setSnap] = React.useState<number | string | null>(peek)
  const full = snap === 1

  // Each opening starts at the peek height.
  const [wasOpen, setWasOpen] = React.useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) setSnap(peek)
  }

  // Keep the last content while the sheet slides away.
  const [shown, setShown] = React.useState(children)
  if (children && children !== shown) setShown(children)

  return (
    <Drawer
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose()
      }}
      modal={false}
      snapPoints={snapPoints}
      activeSnapPoint={snap}
      setActiveSnapPoint={setSnap}
    >
      <DrawerContent
        aria-describedby={undefined}
        // The map stays in use: opening the sheet does not move focus into it.
        onOpenAutoFocus={(event) => event.preventDefault()}
        className="h-[calc(100dvh-4.75rem)] border-x border-border shadow-panel outline-none data-[vaul-drawer-direction=bottom]:mt-0 data-[vaul-drawer-direction=bottom]:max-h-none md:hidden"
      >
        <DrawerTitle className="sr-only">{label}</DrawerTitle>
        <div
          className={cn(
            "min-h-0 flex-1 overscroll-contain px-1 pt-1 pb-[max(1rem,env(safe-area-inset-bottom))] [&>[data-slot=panel]]:border-0 [&>[data-slot=panel]]:bg-transparent [&>[data-slot=panel]]:shadow-none",
            full ? "overflow-y-auto" : "overflow-hidden",
          )}
        >
          {children ?? shown}
        </div>
      </DrawerContent>
    </Drawer>
  )
}
