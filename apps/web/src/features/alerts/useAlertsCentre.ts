import { useEffect } from "react"
import { registerAlertSettingsOpener } from "@/features/install"
import type { InboxTab } from "@/features/notifications/state"
import { start } from "./store"

/**
 * Starts the alerts on load (reading this device's state asks nothing of the
 * person) and lets the install guide open the alert settings.
 */
export function useAlertsCentre(open: (tab?: InboxTab) => void) {
  useEffect(() => {
    void start()
  }, [])
  useEffect(() => registerAlertSettingsOpener(() => open("settings")), [open])
}
