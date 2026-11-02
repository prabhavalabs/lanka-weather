import { createAsyncStoragePersister } from "@tanstack/query-async-storage-persister"
import { QueryClient } from "@tanstack/react-query"
import type { PersistQueryClientProviderProps } from "@tanstack/react-query-persist-client"
import { del, get, set } from "idb-keyval"

const day = 24 * 60 * 60 * 1000

/**
 * Query client. Cached data is kept for a day so the persisted copy can be
 * restored on the next visit; every layer query sets its own staleTime from
 * the layer catalogue.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      gcTime: day,
      staleTime: 60_000,
      retry: 2,
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
