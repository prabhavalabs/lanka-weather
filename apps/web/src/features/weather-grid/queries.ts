import { useQuery } from "@tanstack/react-query"
import { apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { CurrentGrid } from "./types"

/** Current conditions on the weather grid (21 KB, 4 KB gzipped); polled at the layer's cadence. */
export function useCurrentGrid(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["weather", "current"],
    queryFn: ({ signal }) =>
      apiGetBare<CurrentGrid>(layer?.dataPath ?? "/v1/weather/current", signal),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions(layer ?? { refreshSeconds: 3600 }),
    meta: { label: "app.data.grid" },
  })
}
