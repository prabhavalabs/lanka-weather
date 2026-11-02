import { useQueries, useQuery } from "@tanstack/react-query"
import { apiGet } from "@/lib/api"
import { layerQueryOptions, type Layer } from "@/lib/layers"
import { ancestorId, regionLevelOf } from "./present"
import type { RegionDetail, RegionLevel, RegionWeather, RegionWeatherResponse } from "./types"

/** Region weather follows the weather grid's cadence (hourly); this is the fallback. */
const regionWeatherTimings = { refreshSeconds: 3600 }

function regionWeatherPath(level: RegionLevel, parent?: string) {
  const query = new URLSearchParams({ level })
  if (parent) query.set("parent", parent)
  return `/v1/weather/regions?${query}`
}

function regionWeatherQuery(
  level: RegionLevel,
  parent: string | undefined,
  timings: Pick<Layer, "refreshSeconds">,
) {
  return {
    queryKey: ["weather", "regions", level, parent ?? null],
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      apiGet<RegionWeatherResponse>(regionWeatherPath(level, parent), signal),
    ...layerQueryOptions(timings),
    // GN divisions are many: kept in memory only.
    meta: { persist: level !== "gn", label: "weather by region" },
  }
}

/** Weather for every region of a level, or of the regions within `parent`. */
export function useRegionWeather(
  level: RegionLevel,
  options: { parent?: string; enabled?: boolean; timings?: Pick<Layer, "refreshSeconds"> } = {},
) {
  return useQuery({
    ...regionWeatherQuery(level, options.parent, options.timings ?? regionWeatherTimings),
    enabled: options.enabled ?? true,
  })
}

interface GnResult {
  data?: RegionWeatherResponse
  isPending: boolean
  fetchStatus: string
  error: Error | null
}

/** Merges the districts' GN weather; module-level so TanStack Query can memoise it. */
function mergeGn(results: GnResult[]) {
  const regions: RegionWeather[] = []
  let validAt: string | undefined
  for (const result of results) {
    if (!result.data) continue
    regions.push(...result.data.regions)
    validAt ??= result.data.validAt
  }
  return {
    regions,
    validAt,
    pending: results.some((result) => result.isPending && result.fetchStatus !== "idle"),
    error: results.find((result) => result.error)?.error ?? null,
  }
}

/**
 * GN division weather for several districts at once, merged; districts
 * already loaded come from the cache.
 */
export function useGnWeather(
  districts: string[],
  enabled: boolean,
  timings: Pick<Layer, "refreshSeconds"> = regionWeatherTimings,
) {
  return useQueries({
    queries: districts.map((district) => ({
      ...regionWeatherQuery("gn", district, timings),
      enabled,
    })),
    combine: mergeGn,
  })
}

/**
 * One region's weather: from the whole level for provinces, districts and
 * DS divisions (small payloads), and from its DS division for a GN division.
 */
export function useRegionAggregate(id: string | null) {
  const level = id ? regionLevelOf(id) : null
  const parent = id && level === "gn" ? (ancestorId(id, "ds") ?? undefined) : undefined
  const query = useRegionWeather(level ?? "district", { parent, enabled: level !== null })
  const region = query.data?.regions.find((candidate) => candidate.id === id)
  return { ...query, region, level }
}

/** A region's names, hierarchy, label point and bounding box. */
export function useRegionDetail(id: string | null) {
  return useQuery({
    queryKey: ["regions", "detail", id],
    queryFn: ({ signal }) => apiGet<RegionDetail>(`/v1/regions/${encodeURIComponent(id!)}`, signal),
    enabled: id !== null,
    // Boundaries change a few times a year.
    staleTime: 24 * 60 * 60 * 1000,
    meta: { label: "the region" },
  })
}
