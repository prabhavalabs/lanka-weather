import { keepPreviousData, useQuery, type QueryClient } from "@tanstack/react-query"
import { apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import { decodeWind, type WindResponse } from "./field"

const defaultPath = "/v1/weather/wind"

function windQuery(layer: Layer | undefined, time: number) {
  const iso = new Date(time).toISOString()
  const path = layer?.dataPath ?? defaultPath
  return {
    queryKey: ["wind", iso] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      apiGetBare<WindResponse>(`${path}?t=${encodeURIComponent(iso)}`, signal),
    ...layerQueryOptions(layer ?? { refreshSeconds: 3600 }),
    // Fields are 39 KB each and change with every run: kept in memory only (ADR 0002).
    meta: { persist: false, label: "the wind forecast" },
  }
}

/**
 * The wind field valid at `time` (a 3-hour step), decoded. While another
 * step loads, the previous field stays on the map.
 */
export function useWindField(layer: Layer | undefined, time: number, enabled: boolean) {
  return useQuery({
    ...windQuery(layer, time),
    enabled,
    placeholderData: keepPreviousData,
    select: decodeWind,
  })
}

/** Loads a step ahead of time, e.g. the next one while the forecast plays. */
export function prefetchWindField(client: QueryClient, layer: Layer | undefined, time: number) {
  return client.prefetchQuery(windQuery(layer, time))
}
