import { useQuery } from "@tanstack/react-query"
import { apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { FloodForecastCollection } from "./types"

/** Google flood forecasts for Sri Lanka; polled at the layer's cadence (hourly). */
export function useFloodForecast(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["floods", "forecast"],
    queryFn: ({ signal }) =>
      apiGetBare<FloodForecastCollection>(
        layer?.dataPath ?? "/v1/floods/forecast",
        signal,
        "application/geo+json",
      ),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions(layer ?? { refreshSeconds: 3600 }),
    meta: { label: "app.data.floods" },
  })
}
