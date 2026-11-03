import { CloudOffIcon } from "lucide-react"
import { useEffect, useRef, useSyncExternalStore, type ReactNode } from "react"
import { toast } from "sonner"
import { useT } from "@/lib/i18n"

function subscribe(onChange: () => void) {
  window.addEventListener("online", onChange)
  window.addEventListener("offline", onChange)
  return () => {
    window.removeEventListener("online", onChange)
    window.removeEventListener("offline", onChange)
  }
}

/** Whether the browser says it has a connection. */
export function useOnline(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => navigator.onLine,
    () => true,
  )
}

/**
 * The calm notice for the map's notice slot while the browser is offline:
 * the map keeps the last saved data, and each layer still says how old it
 * is. Null when online. Coming back online says so briefly, once.
 */
export function useOfflineNotice(): ReactNode {
  const online = useOnline()
  const t = useT()
  const wasOffline = useRef(false)

  useEffect(() => {
    if (!online) {
      wasOffline.current = true
      toast.dismiss("pwa-back-online")
    } else if (wasOffline.current) {
      wasOffline.current = false
      toast.success(t("pwa.offline.back"), { id: "pwa-back-online", duration: 3000 })
    }
  }, [online, t])

  if (online) return null
  return (
    <div
      role="status"
      className="pointer-events-auto flex max-w-md items-start gap-2.5 rounded-2xl border border-border bg-card/95 px-3.5 py-2 text-sm shadow-panel backdrop-blur-md"
    >
      <CloudOffIcon aria-hidden className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
      <p className="min-w-0">
        <span className="font-medium">{t("pwa.offline.title")}.</span>{" "}
        <span className="text-muted-foreground">{t("pwa.offline.body")}</span>
      </p>
    </div>
  )
}
