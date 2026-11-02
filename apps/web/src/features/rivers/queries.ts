import { useQuery } from "@tanstack/react-query"
import { apiGet } from "../../lib/api"
import { queryTimings, type Layer } from "../../lib/layers"
import type { RiverLevelsCollection, StationHistory } from "./types"

/** The river-levels layer; polled at the cadence the layer catalogue declares. */
export function useRiverLevels(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["rivers", "levels"],
    queryFn: async ({ signal }) => {
      const response = await fetch(layer!.dataPath!, {
        signal,
        headers: { Accept: "application/geo+json" },
      })
      if (!response.ok) throw new Error(`River levels failed with status ${response.status}`)
      return (await response.json()) as RiverLevelsCollection
    },
    enabled: enabled && Boolean(layer?.dataPath),
    ...(layer ? queryTimings(layer) : {}),
  })
}

/** One station's readings over the last `hours` hours. */
export function useStationHistory(stationId: string | null, hours = 72) {
  return useQuery({
    queryKey: ["rivers", "station", stationId, hours],
    queryFn: ({ signal }) =>
      apiGet<StationHistory>(
        `/v1/rivers/stations/${encodeURIComponent(stationId!)}/readings?hours=${hours}`,
        signal,
      ),
    enabled: stationId !== null,
    staleTime: 5 * 60_000,
    refetchInterval: 15 * 60_000,
  })
}
