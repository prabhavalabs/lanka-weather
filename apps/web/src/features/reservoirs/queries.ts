import { useQuery } from "@tanstack/react-query"
import { apiGet } from "@/lib/api"
import { queryTimings, type Layer } from "@/lib/layers"
import type { ReservoirAreas, ReservoirHistory, ReservoirsCollection } from "./types"

/** The reservoirs layer; polled at the cadence the catalogue declares. */
export function useReservoirs(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["reservoirs"],
    queryFn: async ({ signal }) => {
      const response = await fetch(layer!.dataPath!, {
        signal,
        headers: { Accept: "application/geo+json" },
      })
      if (!response.ok) throw new Error(`Reservoirs failed with status ${response.status}`)
      return (await response.json()) as ReservoirsCollection
    },
    enabled: enabled && Boolean(layer?.dataPath),
    ...(layer ? queryTimings(layer) : {}),
    meta: { label: "app.data.reservoirs" },
  })
}

/** The polygons of mapped reservoirs, coloured by the status they carry. */
export function useReservoirAreas(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["reservoirs", "areas"],
    queryFn: async ({ signal }) => {
      const response = await fetch("/v1/reservoirs/areas", {
        signal,
        headers: { Accept: "application/geo+json" },
      })
      if (!response.ok) throw new Error(`Reservoir areas failed with status ${response.status}`)
      return (await response.json()) as ReservoirAreas
    },
    enabled: enabled && Boolean(layer),
    ...(layer ? queryTimings(layer) : {}),
    meta: { label: "app.data.reservoirAreas" },
  })
}

/** One reservoir's readings over the last `days` days. */
export function useReservoirHistory(id: string | null, days = 60) {
  return useQuery({
    queryKey: ["reservoirs", "history", id, days],
    queryFn: ({ signal }) =>
      apiGet<ReservoirHistory>(
        `/v1/reservoirs/${encodeURIComponent(id!)}/readings?days=${days}`,
        signal,
      ),
    enabled: id !== null,
    staleTime: 60 * 60_000,
  })
}
