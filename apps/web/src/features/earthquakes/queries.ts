import { useQuery } from "@tanstack/react-query"
import { apiGetBare } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import type { EarthquakeCollection } from "./types"

/** The earthquakes layer, polled at the catalogue's cadence. */
export function useEarthquakes(layer: Layer | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["hazards", "earthquakes"],
    queryFn: ({ signal }) =>
      apiGetBare<EarthquakeCollection>(layer!.dataPath!, signal, "application/geo+json"),
    enabled: enabled && Boolean(layer?.dataPath),
    ...(layer ? layerQueryOptions(layer) : {}),
    meta: { label: "app.data.earthquakes" },
  })
}
