import { BellIcon, BellRingIcon, SettingsIcon, XIcon } from "lucide-react"
import { useCallback, type ReactNode } from "react"
import { Panel } from "@/components/Panel"
import { Button } from "@/components/ui/button"
import { NotificationsInbox } from "@/features/notifications/NotificationsInbox"
import { closeInbox, showTab, useInboxView } from "@/features/notifications/state"
import type { useInbox } from "@/features/notifications/useInbox"
import { useT } from "@/lib/i18n"
import { AlertsPanel } from "./AlertsPanel"
import { useAlerts } from "./store"

interface AlertsInboxPanelProps {
  inbox: ReturnType<typeof useInbox>
  /** The official / alerts switch over the panel. */
  tabs: ReactNode
  /** Opens a notice's map link (a same-origin path) in the app. */
  onShowOnMap: (url: string) => void
}

/**
 * The alerts this device was sent, newest first, beside the official
 * warnings in the warnings panel. The gear swaps the list for the settings:
 * where, what and how serious.
 */
export function AlertsInboxPanel({ inbox, tabs, onShowOnMap }: AlertsInboxPanelProps) {
  const t = useT()
  const alerts = useAlerts()
  const view = useInboxView()
  const settings = view.tab === "settings"

  const showOnMap = useCallback(
    (url: string) => {
      // Closing first takes `notice` out of the address the map state is written to.
      closeInbox()
      onShowOnMap(url)
    },
    [onShowOnMap],
  )

  return (
    <Panel
      icon={alerts.status === "on" ? <BellRingIcon className="text-brand" /> : <BellIcon />}
      title={settings ? t("alerts.buttonLabel") : t("notifications.title")}
      subtitle={settings ? t("alerts.title") : t("notifications.description")}
      className="w-full"
      action={
        <>
          <Button
            variant="outline"
            size="icon"
            aria-pressed={settings}
            aria-label={t("alerts.buttonLabel")}
            title={t("alerts.buttonLabel")}
            onClick={() => showTab(settings ? "list" : "settings")}
            className="aria-pressed:bg-foreground aria-pressed:text-background aria-pressed:hover:bg-foreground/90"
          >
            <SettingsIcon />
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={closeInbox}
            aria-label={t("notifications.close")}
          >
            <XIcon />
          </Button>
        </>
      }
    >
      {tabs}
      {settings ? (
        <div className="flex flex-col gap-4">
          <p className="text-sm text-muted-foreground">{t("alerts.description")}</p>
          <AlertsPanel />
        </div>
      ) : (
        <NotificationsInbox inbox={inbox} onShowOnMap={showOnMap} />
      )}
    </Panel>
  )
}
