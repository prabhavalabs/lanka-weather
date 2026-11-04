import { useQuery } from "@tanstack/react-query"
import { apiGet } from "@/lib/api"
import { layerQueryOptions } from "@/lib/layers"
import type { CanalsDocument } from "./types"

/** Colombo's canal levels; collected every 30 minutes. */
export function useCanals(enabled: boolean) {
  return useQuery({
    queryKey: ["rivers", "canals"],
    queryFn: ({ signal }) => apiGet<CanalsDocument>("/v1/rivers/canals", signal),
    enabled,
    ...layerQueryOptions({ refreshSeconds: 1800 }),
    meta: { label: "app.data.canals" },
  })
}
