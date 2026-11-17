import { BellIcon, BellPlusIcon, BellRingIcon, SettingsIcon } from "lucide-react"
import { useCallback, useEffect } from "react"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { registerAlertSettingsOpener } from "@/features/install"
import { NotificationsInbox } from "@/features/notifications/NotificationsInbox"
import { badgeText } from "@/features/notifications/present"
import {
  closeInbox,
  type InboxTab,
  openInbox,
  showTab,
  useInboxView,
} from "@/features/notifications/state"
import { useInbox } from "@/features/notifications/useInbox"
import { useT } from "@/lib/i18n"
import { desktopQuery, useMediaQuery } from "@/lib/useMediaQuery"
import { cn } from "@/lib/utils"
import { AlertsPanel } from "./AlertsPanel"
import { start, useAlerts } from "./store"

interface AlertsButtonProps {
  /** Icon only, for the phone bar. */
  compact?: boolean
  className?: string
  /** Opens a notice's map link (a same-origin path) in the app. */
  onShowOnMap: (url: string) => void
}

/**
 * The bell: the alerts this device was sent, newest first, with the alert
 * settings on a second tab. A side sheet on a wide screen, a bottom drawer
 * on a phone. The bell carries the number of alerts not seen yet.
 *
 * Reading the device's state on load asks nothing of the person; permission
 * is asked only when they turn alerts on. The bell is mounted in both the
 * phone bar and the desktop toolbar; the panel's state is shared, and the
 * bell on screen draws it.
 */
export function AlertsButton({ compact = false, className, onShowOnMap }: AlertsButtonProps) {
  const t = useT()
  const alerts = useAlerts()
  const inbox = useInbox()
  const view = useInboxView()
  const desktop = useMediaQuery(desktopQuery, true)

  useEffect(() => {
    void start()
  }, [])

  // The install guide's "installed" view links to the settings.
  const shown = compact !== desktop
  useEffect(() => {
    if (!shown) return
    return registerAlertSettingsOpener(() => openInbox({ tab: "settings" }))
  }, [shown])

  const showOnMap = useCallback(
    (url: string) => {
      // Closing first takes `notice` out of the address the map state is written to.
      closeInbox()
      onShowOnMap(url)
    },
    [onShowOnMap],
  )

  const on = alerts.status === "on"
  const badge = badgeText(inbox.unread)
  const label =
    inbox.unread > 0
      ? t("notifications.buttonUnread", { count: inbox.unread })
      : on
        ? t("alerts.buttonOn")
        : t("notifications.title")
  const trigger = (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      onClick={() => openInbox()}
      aria-label={label}
      title={label}
      className={cn(
        "relative",
        !compact && "rounded-full bg-card/95 shadow-panel backdrop-blur-md",
        className,
      )}
    >
      {on ? (
        <BellRingIcon className="text-brand" />
      ) : inbox.all.length ? (
        <BellIcon />
      ) : (
        <BellPlusIcon />
      )}
      {!compact && t("alerts.button")}
      {badge && (
        <span
          aria-hidden="true"
          className="numeric absolute -top-1 -right-1 flex h-4.5 min-w-4.5 items-center justify-center rounded-full bg-destructive px-1 text-[0.625rem] leading-none font-semibold text-white ring-2 ring-background"
        >
          {badge}
        </span>
      )}
    </Button>
  )

  const onOpenChange = (open: boolean) => (open ? openInbox() : closeInbox())
  const body = (
    <Tabs
      value={view.tab}
      onValueChange={(value) => showTab(value as InboxTab)}
      className="min-h-0 flex-1 gap-0"
    >
      <div className="px-5 pb-3">
        <TabsList className="w-full">
          <TabsTrigger value="list">
            <BellIcon />
            {t("notifications.tabs.list")}
            {inbox.unread > 0 && (
              <span className="numeric rounded-full bg-destructive px-1.5 text-[0.625rem] leading-4 font-semibold text-white">
                {badge}
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="settings">
            <SettingsIcon />
            {t("notifications.tabs.settings")}
          </TabsTrigger>
        </TabsList>
      </div>
      <ScrollArea
        className="flex min-h-0 flex-1 flex-col [--panel-bg:var(--color-popover)]"
        viewportClassName="min-h-0 flex-1"
      >
        <div className="px-5 pb-6">
          <TabsContent value="list">
            <NotificationsInbox inbox={inbox} onShowOnMap={showOnMap} />
          </TabsContent>
          <TabsContent value="settings" className="flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">{t("alerts.description")}</p>
            <AlertsPanel />
          </TabsContent>
        </div>
      </ScrollArea>
    </Tabs>
  )

  return (
    <>
      {trigger}
      {shown &&
        (desktop ? (
          <Sheet open={view.open} onOpenChange={onOpenChange}>
            <SheetContent
              side="right"
              className="gap-0 data-[side=right]:w-full data-[side=right]:sm:max-w-md"
            >
              <SheetHeader className="px-5 pt-5">
                <SheetTitle>{t("notifications.title")}</SheetTitle>
                <SheetDescription>{t("notifications.description")}</SheetDescription>
              </SheetHeader>
              {body}
            </SheetContent>
          </Sheet>
        ) : (
          <Drawer open={view.open} onOpenChange={onOpenChange}>
            <DrawerContent className="max-h-[90dvh]">
              <DrawerHeader className="px-5 group-data-[vaul-drawer-direction=bottom]/drawer-content:text-left">
                <DrawerTitle>{t("notifications.title")}</DrawerTitle>
                <DrawerDescription>{t("notifications.description")}</DrawerDescription>
              </DrawerHeader>
              {body}
            </DrawerContent>
          </Drawer>
        ))}
    </>
  )
}
