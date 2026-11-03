import { useQuery } from "@tanstack/react-query"
import { apiGet } from "@/lib/api"
import { useRegionDetail } from "../regions/queries"
import { roundCoordinate, type Selection } from "./selection"
import type { PointForecast, Summary } from "./types"

export interface LatLon {
  lat: number
  lon: number
}

function pointQuery(location: LatLon | null) {
  const lat = location ? roundCoordinate(location.lat) : null
  const lon = location ? roundCoordinate(location.lon) : null
  return { lat, lon, params: `lat=${lat}&lon=${lon}` }
}

/**
 * Everything known about a place: its regions, nearby gauges and derived
 * alerts. The API caches it for two minutes.
 */
export function useSummary(location: LatLon | null) {
  const { lat, lon, params } = pointQuery(location)
  return useQuery({
    queryKey: ["summary", lat, lon],
    queryFn: ({ signal }) => apiGet<Summary>(`/v1/summary?${params}`, signal),
    enabled: location !== null,
    staleTime: 2 * 60_000,
    refetchInterval: 10 * 60_000,
    meta: { label: "app.data.locationSummary" },
  })
}

/** The point forecast: 72 hours and 7 days, interpolated from the weather grid. */
export function usePointForecast(location: LatLon | null) {
  const { lat, lon, params } = pointQuery(location)
  return useQuery({
    queryKey: ["weather", "point", lat, lon],
    queryFn: ({ signal }) => apiGet<PointForecast>(`/v1/weather/point?${params}`, signal),
    enabled: location !== null,
    staleTime: 5 * 60_000,
    refetchInterval: 15 * 60_000,
    meta: { label: "app.data.forecast" },
  })
}

/** Where the panel's weather is read: the point, or the region's label point. */
export function useSelectionLocation(selection: Selection | null): LatLon | null {
  const region = useRegionDetail(selection?.kind === "region" ? selection.id : null)
  if (selection?.kind === "point") return { lat: selection.lat, lon: selection.lon }
  if (selection?.kind === "region" && region.data) {
    const [lon, lat] = region.data.labelPoint
    return { lat, lon }
  }
  return null
}
