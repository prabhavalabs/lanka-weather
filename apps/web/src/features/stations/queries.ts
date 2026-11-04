import { useQuery } from "@tanstack/react-query"
import { apiGet, apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { MetStationCollection, MetStationDetail } from "./types"

/** The weather-stations layer; polled at the cadence the catalogue declares. */
export function useMetStations(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["stations", "layer"],
    queryFn: ({ signal }) =>
      apiGetBare<MetStationCollection>(
        layer?.dataPath ?? "/v1/stations",
        signal,
        "application/geo+json",
      ),
    enabled: enabled && Boolean(layer),
    ...layerQueryOptions(layer ?? { refreshSeconds: 3600 }),
    meta: { label: "app.data.stations" },
  })
}

/** One station's latest reading and history. */
export function useMetStation(id: string | null) {
  return useQuery({
    queryKey: ["stations", "detail", id],
    queryFn: ({ signal }) =>
      apiGet<MetStationDetail>(`/v1/stations/${encodeURIComponent(id!)}?days=7`, signal),
    enabled: id !== null,
    staleTime: 5 * 60_000,
    refetchInterval: 30 * 60_000,
    meta: { label: "app.data.station" },
  })
}
