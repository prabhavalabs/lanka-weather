import { useQuery } from "@tanstack/react-query"
import { apiGet, apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { Division, LandslideLayerData, WarningsOverview } from "./types"

export interface WarningsFilter {
  type?: string
  district?: string
}

function overviewPath(filter: WarningsFilter) {
  const params = new URLSearchParams()
  if (filter.type) params.set("type", filter.type)
  if (filter.district) params.set("district", filter.district)
  const query = params.toString()
  return `/v1/warnings${query ? `?${query}` : ""}`
}

/**
 * Bulletins, what is in force, the banner and the districts to highlight. The
 * banner and the count are not tied to any layer being on, so this always runs.
 */
export function useWarnings(layer: Layer | undefined, filter: WarningsFilter = {}) {
  return useQuery({
    queryKey: ["warnings", "overview", filter.type ?? null, filter.district ?? null],
    queryFn: ({ signal }) => apiGet<WarningsOverview>(overviewPath(filter), signal),
    enabled: Boolean(layer?.dataPath),
    ...(layer ? layerQueryOptions(layer) : {}),
    meta: { label: "official warnings" },
  })
}

/** The landslide-warnings layer. */
export function useLandslides(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["warnings", "landslides"],
    queryFn: ({ signal }) =>
      apiGetBare<LandslideLayerData>(layer!.dataPath!, signal, "application/geo+json"),
    enabled: enabled && Boolean(layer?.dataPath),
    ...(layer ? layerQueryOptions(layer) : {}),
    meta: { label: "landslide warnings" },
  })
}

/** One DS division's level and recent history. */
export function useDivision(id: string | null) {
  return useQuery({
    queryKey: ["warnings", "division", id],
    queryFn: ({ signal }) =>
      apiGet<Division>(`/v1/warnings/landslides/${encodeURIComponent(id!)}`, signal),
    enabled: id !== null,
    staleTime: 5 * 60_000,
    meta: { label: "the division's landslide warnings" },
  })
}
