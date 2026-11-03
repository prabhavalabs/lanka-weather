/**
 * What a click or a hover makes of these layers' points on the map. Pure, so it
 * is tested without a browser.
 */
import type { MapGeoJSONFeature } from "maplibre-gl"
import { formatAge, formatNumber } from "@/lib/format"
import { translate } from "@/lib/i18n"
import { RAIN_GAUGE_LAYERS } from "../raingauges/RainGaugesLayer"
import { bandLabel } from "../raingauges/present"
import type { RainBand } from "../raingauges/types"
import {
  RAIN_GAUGE_POINTS as RIVER_RAIN_POINTS,
  RAIN_MET_POINTS,
} from "../rainfall/ObservedRainLayer"
import { methodLabel } from "../rainfall/present"
import type { ObservedRainMethod } from "../rainfall/types"
import { MET_STATIONS } from "../stations/MetStationsLayer"
import { formatRain } from "../stations/present"

/** Point layers these layers draw, for clicks and hovers. */
export const observedPointLayers = [
  ...RAIN_GAUGE_LAYERS,
  MET_STATIONS,
  RIVER_RAIN_POINTS,
  RAIN_MET_POINTS,
]

type Hit = Pick<MapGeoJSONFeature, "properties"> & { layer: Pick<MapGeoJSONFeature["layer"], "id"> }

const bands: RainBand[] = ["very_heavy", "heavy", "moderate", "light", "none", "unknown"]

/**
 * What a hover chip says about one of these layers' points; null for another's.
 * In the current language, so the chip rebuilds it when the language changes.
 */
export function observedChip(point: Hit): { title: string; lines: string[] } | null {
  const properties = point.properties
  const name = typeof properties.name === "string" && properties.name ? properties.name : null
  if (point.layer.id === MET_STATIONS) {
    const temperature = formatNumber(Number(properties.shown), 1)
    const age = formatAge(typeof properties.observedAt === "string" ? properties.observedAt : null)
    return {
      title: name ?? translate("stations.chip.title"),
      lines: [
        properties.reading === "daily"
          ? translate("stations.chip.dailyMaximum", { temperature })
          : translate("stations.chip.observed", { temperature }),
        properties.stale ? translate("stations.chip.ageStale", { age }) : age,
      ],
    }
  }
  if (RAIN_GAUGE_LAYERS.includes(point.layer.id)) {
    const observedAt = typeof properties.observedAt === "string" ? properties.observedAt : null
    const rain = typeof properties.rain24hMm === "number" ? properties.rain24hMm : null
    const band = bands.includes(properties.band as RainBand) ? (properties.band as RainBand) : null
    const institution = typeof properties.institution === "string" ? properties.institution : ""
    const age = formatAge(observedAt)
    return {
      title: name ?? translate("rain.chip.gauge"),
      lines: [
        rain === null
          ? translate("rain.chip.noTotal")
          : band
            ? translate("rain.chip.totalBand", { rain: formatRain(rain), band: bandLabel(band) })
            : translate("rain.chip.total", { rain: formatRain(rain) }),
        institution ? translate("rain.chip.runBy", { institution, age }) : age,
      ],
    }
  }
  if (point.layer.id === RIVER_RAIN_POINTS || point.layer.id === RAIN_MET_POINTS) {
    const windowEnd = typeof properties.windowEnd === "string" ? properties.windowEnd : null
    const age = formatAge(windowEnd)
    return {
      title: name ?? translate("rain.chip.rainfall"),
      lines: [
        translate("rain.chip.total", { rain: formatRain(Number(properties.rainMm)) }),
        methodLabel(properties.method as ObservedRainMethod),
        properties.stale
          ? translate("rain.chip.endingStale", { age })
          : translate("rain.chip.ending", { age }),
      ].filter(Boolean),
    }
  }
  return null
}

/** What a click on one of these layers' points opens; null for another's. */
export function observedTarget(
  hit: Hit,
): { kind: "met" | "station" | "raingauge"; id: string } | null {
  if (RAIN_GAUGE_LAYERS.includes(hit.layer.id)) {
    return { kind: "raingauge", id: String(hit.properties.id) }
  }
  if (hit.layer.id === MET_STATIONS) return { kind: "met", id: String(hit.properties.id) }
  if (hit.layer.id === RAIN_MET_POINTS) return { kind: "met", id: String(hit.properties.stationId) }
  if (hit.layer.id === RIVER_RAIN_POINTS) {
    return { kind: "station", id: String(hit.properties.stationId) }
  }
  return null
}
