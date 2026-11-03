import { useQuery } from "@tanstack/react-query"
import { apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { ObservedRainCollection, RainSteps } from "./types"

/** The satellite images there are; polled at the layer's cadence. */
export function useRainSteps(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["rain", "steps"],
    queryFn: ({ signal }) => apiGetBare<RainSteps>(layer?.dataPath ?? "/v1/rain/steps", signal),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions(layer ?? { refreshSeconds: 1800 }),
    meta: { label: "app.data.rainImages" },
  })
}

/** 24-hour totals at stations and gauges. Gauges report about hourly. */
export function useObservedRain(enabled: boolean) {
  return useQuery({
    queryKey: ["rain", "observed"],
    queryFn: ({ signal }) =>
      apiGetBare<ObservedRainCollection>("/v1/rain/observed", signal, "application/geo+json"),
    enabled,
    ...layerQueryOptions({ refreshSeconds: 900 }),
    meta: { label: "app.data.observedRain" },
  })
}
