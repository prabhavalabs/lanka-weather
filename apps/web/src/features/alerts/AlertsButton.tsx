import { BellPlusIcon, BellRingIcon } from "lucide-react"
import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { useT } from "@/lib/i18n"
import { desktopQuery, useMediaQuery } from "@/lib/useMediaQuery"
import { cn } from "@/lib/utils"
import { AlertsPanel } from "./AlertsPanel"
import { start, useAlerts } from "./store"

interface AlertsButtonProps {
  /** Icon only, for the phone bar. */
  compact?: boolean
  className?: string
}

/**
 * The bell that opens alert settings: a side sheet on a wide screen, a bottom
 * drawer on a phone. Reading the device's state on load asks nothing of the
 * person; permission is asked only when they turn alerts on.
 */
export function AlertsButton({ compact = false, className }: AlertsButtonProps) {
  const t = useT()
  const alerts = useAlerts()
  const desktop = useMediaQuery(desktopQuery, true)
  const [open, setOpen] = useState(false)

  useEffect(() => {
    void start()
  }, [])

  const on = alerts.status === "on"
  const label = on ? t("alerts.buttonOn") : t("alerts.buttonLabel")
  const trigger = (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      onClick={() => setOpen(true)}
      aria-label={label}
      title={label}
      className={cn(!compact && "rounded-full bg-card/95 shadow-panel backdrop-blur-md", className)}
    >
      {on ? <BellRingIcon className="text-brand" /> : <BellPlusIcon />}
      {!compact && t("alerts.button")}
    </Button>
  )

  const body = (
    <div className="px-5 pb-6">
      <AlertsPanel />
    </div>
  )

  return (
    <>
      {trigger}
      {desktop ? (
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetContent
            side="right"
            className="gap-0 data-[side=right]:w-full data-[side=right]:sm:max-w-md"
          >
            <SheetHeader className="px-5 pt-5">
              <SheetTitle>{t("alerts.title")}</SheetTitle>
              <SheetDescription>{t("alerts.description")}</SheetDescription>
            </SheetHeader>
            <ScrollArea className="min-h-0 flex-1 [--panel-bg:var(--color-popover)]">
              {body}
            </ScrollArea>
          </SheetContent>
        </Sheet>
      ) : (
        <Drawer open={open} onOpenChange={setOpen}>
          <DrawerContent className="max-h-[90dvh]">
            <DrawerHeader className="px-5 group-data-[vaul-drawer-direction=bottom]/drawer-content:text-left">
              <DrawerTitle>{t("alerts.title")}</DrawerTitle>
              <DrawerDescription>{t("alerts.description")}</DrawerDescription>
            </DrawerHeader>
            <ScrollArea
              className="flex min-h-0 flex-1 flex-col [--panel-bg:var(--color-popover)]"
              viewportClassName="min-h-0 flex-1"
            >
              {body}
            </ScrollArea>
          </DrawerContent>
        </Drawer>
      )}
    </>
  )
}
