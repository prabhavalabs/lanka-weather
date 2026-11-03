import { useQuery } from "@tanstack/react-query"
import { apiGet } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { BasinsDocument, PastFloodsDocument, RiskDocument, RiverDetail } from "./types"

/** Every river's estimated risk; polled at the layer's cadence. */
export function useRiverRisk(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["rivers", "risk"],
    queryFn: ({ signal }) => apiGet<RiskDocument>(layer?.dataPath ?? "/v1/rivers/risk", signal),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions(layer ?? { refreshSeconds: 900 }),
    meta: { label: "app.data.riverRisk" },
  })
}

/** One river in full: gauges with readings, forecasts, rain and areas. */
export function useRiverDetail(id: string | null) {
  return useQuery({
    queryKey: ["rivers", "detail", id],
    queryFn: ({ signal }) => apiGet<RiverDetail>(`/v1/rivers/${encodeURIComponent(id!)}`, signal),
    enabled: id !== null,
    staleTime: 60_000,
    refetchInterval: 15 * 60_000,
    // Readings are large, and only of use while the panel is open.
    meta: { label: "app.data.riverDetail", persist: false },
  })
}

/** Every flood basin's state. */
export function useFloodBasins(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["floods", "basins"],
    queryFn: ({ signal }) => apiGet<BasinsDocument>("/v1/floods/basins", signal),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions({ refreshSeconds: 900 }),
    meta: { label: "app.data.basins" },
  })
}

/** Mapped past floods. They change a few times a year. */
export function usePastFloods(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["floods", "past"],
    queryFn: ({ signal }) =>
      apiGet<PastFloodsDocument>(layer?.dataPath ?? "/v1/floods/past", signal),
    enabled: enabled && Boolean(layer),
    staleTime: 60 * 60_000,
    meta: { label: "app.data.pastFloods" },
  })
}
