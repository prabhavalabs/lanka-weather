import { createAsyncStoragePersister } from "@tanstack/query-async-storage-persister"
import { QueryCache, QueryClient, type Query } from "@tanstack/react-query"
import type { PersistQueryClientProviderProps } from "@tanstack/react-query-persist-client"
import { del, get, set } from "idb-keyval"
import { toast } from "sonner"
import { isNotReady, isRetryable } from "./api"

const day = 24 * 60 * 60 * 1000

declare module "@tanstack/react-query" {
  interface Register {
    queryMeta: {
      /** False keeps a large payload (wind fields, GN weather) out of IndexedDB. */
      persist?: boolean
      /** What the query loads, for the failure toast, e.g. "river levels". */
      label?: string
    }
  }
}

/**
 * A toast for a failed request that is worth telling about: not an abort,
 * and not data that simply has not been collected yet (the panels show a
 * calm empty state for that). One toast per query, replaced on repeat.
 */
function announceFailure(error: unknown, query: Query<unknown, unknown, unknown>) {
  if (isNotReady(error)) return
  if (error instanceof DOMException && error.name === "AbortError") return
  const label = query.meta?.label
  if (!label) return
  const hasData = query.state.data !== undefined
  toast.error(`Could not update ${label}`, {
    id: query.queryHash,
    description: hasData
      ? "Showing the last data received. It will be retried shortly."
      : "It will be retried shortly.",
  })
}

/**
 * Query client. Cached data is kept for a day so the persisted copy can be
 * restored on the next visit; every layer query sets its own staleTime from
 * the layer catalogue. Data that is not collected yet (503 no_data) is not
 * retried at once: the layer's polling picks it up.
 */
export const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: announceFailure }),
  defaultOptions: {
    queries: {
      gcTime: day,
      staleTime: 60_000,
      retry: (failures, error) => failures < 2 && isRetryable(error),
      refetchOnWindowFocus: true,
    },
  },
})

/** Persists the query cache to IndexedDB, which has a far larger quota than localStorage. */
const persister = createAsyncStoragePersister({
  key: "lanka-weather:query-cache",
  storage: {
    getItem: (key) => get<string>(key).then((value) => value ?? null),
    setItem: (key, value) => set(key, value),
    removeItem: (key) => del(key),
  },
  throttleTime: 2_000,
})

/**
 * Queries opt out of persistence with `meta: { persist: false }`, for large
 * payloads such as wind fields.
 */
export const persistOptions: PersistQueryClientProviderProps["persistOptions"] = {
  persister,
  maxAge: day,
  buster: __BUILD_ID__,
  dehydrateOptions: {
    shouldDehydrateQuery: (query) =>
      query.state.status === "success" && query.meta?.persist !== false,
  },
}
