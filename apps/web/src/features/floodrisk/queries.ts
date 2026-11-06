import { useQuery } from "@tanstack/react-query"
import { apiGet } from "@/lib/api"
import { layerQueryOptions } from "@/lib/layers"
import type { FloodRiskDocument } from "./types"

/** The server works the estimate out every minute; two minutes is close enough here. */
const timings = { refreshSeconds: 120 }

/** Every district's flood-risk estimate, one request for all of them. */
export function useFloodRisk(enabled: boolean) {
  return useQuery({
    queryKey: ["flood-risk"],
    queryFn: ({ signal }) => apiGet<FloodRiskDocument>("/v1/flood-risk", signal),
    ...layerQueryOptions(timings),
    enabled,
    meta: { label: "app.data.floodRisk" },
  })
}
