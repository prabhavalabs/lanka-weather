/**
 * What a click or a hover makes of these layers' points on the map. Pure, so it
 * is tested without a browser.
 */
import type { MapGeoJSONFeature } from "maplibre-gl"
import { formatAge, formatNumber } from "@/lib/format"
import { RAIN_GAUGE_POINTS, RAIN_MET_POINTS } from "../rainfall/ObservedRainLayer"
import { methodLabels } from "../rainfall/present"
import type { ObservedRainMethod } from "../rainfall/types"
import { MET_STATIONS } from "../stations/MetStationsLayer"
import { formatRain } from "../stations/present"

/** Point layers these layers draw, for clicks and hovers. */
export const observedPointLayers = [MET_STATIONS, RAIN_GAUGE_POINTS, RAIN_MET_POINTS]

type Hit = Pick<MapGeoJSONFeature, "properties"> & { layer: Pick<MapGeoJSONFeature["layer"], "id"> }

/** What a hover chip says about one of these layers' points; null for another's. */
export function observedChip(point: Hit): { title: string; lines: string[] } | null {
  const properties = point.properties
  if (point.layer.id === MET_STATIONS) {
    const value = Number(properties.shown)
    const reading = properties.reading === "daily" ? "Observed, daily maximum" : "Observed"
    const age = formatAge(typeof properties.observedAt === "string" ? properties.observedAt : null)
    return {
      title: String(properties.name ?? "Weather station"),
      lines: [
        `${reading}: ${formatNumber(value, 1)} °C`,
        `${age}${properties.stale ? " · out of date" : ""}`,
      ],
    }
  }
  if (point.layer.id === RAIN_GAUGE_POINTS || point.layer.id === RAIN_MET_POINTS) {
    const windowEnd = typeof properties.windowEnd === "string" ? properties.windowEnd : null
    return {
      title: String(properties.name ?? "Rainfall"),
      lines: [
        `Rain, 24 h: ${formatRain(Number(properties.rainMm))}`,
        methodLabels[properties.method as ObservedRainMethod] ?? "",
        `To ${formatAge(windowEnd)}${properties.stale ? " · out of date" : ""}`,
      ].filter(Boolean),
    }
  }
  return null
}

/** What a click on one of these layers' points opens; null for another's. */
export function observedTarget(hit: Hit): { kind: "met" | "station"; id: string } | null {
  if (hit.layer.id === MET_STATIONS) return { kind: "met", id: String(hit.properties.id) }
  if (hit.layer.id === RAIN_MET_POINTS) return { kind: "met", id: String(hit.properties.stationId) }
  if (hit.layer.id === RAIN_GAUGE_POINTS) {
    return { kind: "station", id: String(hit.properties.stationId) }
  }
  return null
}
