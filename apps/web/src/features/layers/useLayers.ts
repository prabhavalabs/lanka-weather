import { useQuery } from "@tanstack/react-query"
import { apiGet } from "../../lib/api"
import type { Layer } from "../../lib/layers"

/** The layer catalogue. It changes only with deployments, so it is cached for an hour. */
export function useLayers() {
  return useQuery({
    queryKey: ["layers"],
    queryFn: ({ signal }) => apiGet<Layer[]>("/v1/layers", signal),
    staleTime: 60 * 60 * 1000,
  })
}
