import { useQuery } from "@tanstack/react-query"
import { apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { CycloneCollection } from "./types"

/** The cyclones layer, polled at the catalogue's cadence. */
export function useCyclones(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["hazards", "cyclones"],
    queryFn: ({ signal }) =>
      apiGetBare<CycloneCollection>(layer!.dataPath!, signal, "application/geo+json"),
    enabled: enabled && Boolean(layer?.dataPath),
    ...(layer ? layerQueryOptions(layer) : {}),
    meta: { label: "app.data.cyclones" },
  })
}
