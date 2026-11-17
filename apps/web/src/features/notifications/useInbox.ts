/**
 * The inbox: this device's copies (IndexedDB, written by the service
 * worker) merged with the server's list for this subscription, which also
 * holds notices sent before the device kept copies, or while it was away.
 * Both are TanStack queries; the server's is persisted with the rest of the
 * cache, so the list works offline from what was last seen.
 */
import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query"
import { useCallback, useEffect, useMemo } from "react"
import { getInbox } from "@/features/alerts/api"
import { subscriptionJSON, useAlerts } from "@/features/alerts/store"
import { inboxMessage } from "@/features/pwa/push"
import { ApiError } from "@/lib/api"
import { type Locale, useLocale } from "@/lib/i18n"
import { deleteLocalNotices, type LocalNotice, readLocalNotices } from "./db"
import {
  clearAll,
  clearOne,
  expiredLocal,
  type InboxItem,
  type InboxNotice,
  mergeInbox,
  parseInbox,
  unreadCount,
  visibleItems,
} from "./present"
import { setMarks, useInboxMarks } from "./state"

const inboxKey = ["notifications"] as const
const localKey = [...inboxKey, "local"] as const

/** The device's copies, without those older than the server keeps (which are deleted). */
async function loadLocal(): Promise<LocalNotice[]> {
  try {
    const all = await readLocalNotices()
    const expired = new Set(expiredLocal(all, Date.now()))
    if (expired.size > 0) await deleteLocalNotices([...expired]).catch(() => undefined)
    return all.filter((notice) => !expired.has(notice.id))
  } catch {
    // No IndexedDB (a private window, an old browser): the server's list still shows.
    return []
  }
}

async function loadServer(locale: Locale): Promise<InboxNotice[]> {
  const subscription = await subscriptionJSON()
  if (!subscription?.endpoint || !subscription.keys?.auth) return []
  try {
    const data = await getInbox(subscription, locale)
    return parseInbox(data)
  } catch (error) {
    // The server no longer knows this subscription: nothing to list from it.
    if (error instanceof ApiError && (error.status === 404 || error.status === 403)) return []
    throw error
  }
}

let listening = false

/** The service worker says when a notice arrives, so an open list shows it at once. */
function listen(client: QueryClient) {
  if (listening || typeof navigator === "undefined" || !("serviceWorker" in navigator)) return
  listening = true
  navigator.serviceWorker.addEventListener("message", (event: MessageEvent) => {
    if ((event.data as { type?: unknown } | null)?.type === inboxMessage) {
      void client.invalidateQueries({ queryKey: inboxKey })
    }
  })
}

export interface Inbox {
  /** Every notice known, cleared ones included: a link can still open one. */
  all: InboxItem[]
  /** What the list shows. */
  visible: InboxItem[]
  unread: number
  /** The server's list is loading for the first time. */
  pending: boolean
  /** The server's list could not be loaded; the list shows the device's copies. */
  failed: boolean
  dismiss: (id: string) => void
  dismissAll: () => void
}

export function useInbox(): Inbox {
  const client = useQueryClient()
  const locale = useLocale()
  const alerts = useAlerts()
  const marks = useInboxMarks()

  useEffect(() => listen(client), [client])

  const local = useQuery({
    queryKey: localKey,
    queryFn: loadLocal,
    // IndexedDB answers offline too.
    networkMode: "always",
    staleTime: 0,
    meta: { persist: false },
  })
  const server = useQuery({
    queryKey: [...inboxKey, "server", locale],
    queryFn: () => loadServer(locale),
    enabled: alerts.status === "on",
    staleTime: 60_000,
    refetchOnWindowFocus: true,
  })

  const all = useMemo(
    () => mergeInbox(local.data ?? [], server.data ?? []),
    [local.data, server.data],
  )
  // Clearing changes the list; reading does not.
  const { clearedBefore, cleared } = marks
  const visible = useMemo(
    () => visibleItems(all, { seenAt: null, clearedBefore, cleared }),
    [all, clearedBefore, cleared],
  )

  const dismiss = useCallback(
    (id: string) => {
      setMarks((current) => clearOne(current, id, Date.now()))
      void deleteLocalNotices([id])
        .catch(() => undefined)
        .then(() => client.invalidateQueries({ queryKey: localKey }))
    },
    [client],
  )
  const dismissAll = useCallback(() => {
    setMarks((current) => clearAll(current, all, Date.now()))
    const ids = all.filter((item) => item.local).map((item) => item.id)
    void deleteLocalNotices(ids)
      .catch(() => undefined)
      .then(() => client.invalidateQueries({ queryKey: localKey }))
  }, [all, client])

  return {
    all,
    visible,
    unread: unreadCount(visible, marks.seenAt),
    pending: server.isPending && server.fetchStatus !== "idle",
    failed: server.isError,
    dismiss,
    dismissAll,
  }
}
