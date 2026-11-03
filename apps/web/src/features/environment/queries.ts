import { useQuery } from "@tanstack/react-query"
import { apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { AirGrid, SeaGrid } from "./types"

/** Air quality on its grid; polled at the layer's cadence. */
export function useAirGrid(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["air", "current"],
    queryFn: ({ signal }) => apiGetBare<AirGrid>(layer?.dataPath ?? "/v1/air/current", signal),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions(layer ?? { refreshSeconds: 10800 }),
    meta: { label: "app.data.air" },
  })
}

/** Sea conditions on their grid; polled at the layer's cadence. */
export function useSeaGrid(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["sea", "current"],
    queryFn: ({ signal }) => apiGetBare<SeaGrid>(layer?.dataPath ?? "/v1/sea/current", signal),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions(layer ?? { refreshSeconds: 10800 }),
    meta: { label: "app.data.sea" },
  })
}
