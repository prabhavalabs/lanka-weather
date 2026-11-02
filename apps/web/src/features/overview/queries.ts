import { useQuery } from "@tanstack/react-query"
import { apiGet } from "@/lib/api"
import type { Overview } from "./types"

/** Sri Lanka now. Gauges change every quarter hour, and the API keeps this a minute. */
export function useOverview(enabled: boolean) {
  return useQuery({
    queryKey: ["overview"],
    queryFn: ({ signal }) => apiGet<Overview>("/v1/overview", signal),
    enabled,
    staleTime: 60_000,
    refetchInterval: 5 * 60_000,
    meta: { label: "the overview" },
  })
}
