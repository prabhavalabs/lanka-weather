import { useQuery } from "@tanstack/react-query"
import { apiGet, apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { RainGaugeCollection, RainGaugeDetail } from "./types"

/** The live rain gauges; polled at the layer's cadence. */
export function useRainGauges(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["rain", "gauges"],
    queryFn: ({ signal }) =>
      apiGetBare<RainGaugeCollection>(
        layer?.dataPath ?? "/v1/rain/gauges",
        signal,
        "application/geo+json",
      ),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions(layer ?? { refreshSeconds: 1800 }),
    meta: { label: "app.data.rainGauges" },
  })
}

/** One gauge with up to 48 hours of its rolling total. */
export function useRainGaugeDetail(id: string) {
  return useQuery({
    queryKey: ["rain", "gauges", id],
    queryFn: ({ signal }) =>
      apiGet<RainGaugeDetail>(`/v1/rain/gauges/${encodeURIComponent(id)}`, signal),
    ...layerQueryOptions({ refreshSeconds: 1800 }),
    meta: { label: "app.data.rainGauge" },
  })
}
