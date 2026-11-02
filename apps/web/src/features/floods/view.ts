/**
 * What the Flood Forecast layer shows, as Flood Hub's View options has it. Pure, so
 * it is tested without a browser.
 */
import type { BasinsDocument } from "../river-risk/types"
import { isGauge } from "./present"
import type { FloodAreaKind, FloodForecastCollection } from "./types"

export interface FloodView {
  /** Gauges forecast normal; off leaves only those that need attention. */
  normal: boolean
  flash: boolean
  events: boolean
  inundation: boolean
  /** Lower-confidence gauges, which Flood Hub calls Extended coverage. */
  extended: boolean
}

export const defaultFloodView: FloodView = {
  normal: true,
  flash: true,
  events: true,
  inundation: true,
  extended: false,
}

function areaShown(kind: FloodAreaKind, view: FloodView) {
  if (kind === "flash_likely" || kind === "flash_highly_likely") return view.flash
  if (kind === "significant_event") return view.events
  if (kind.startsWith("inundation_")) return view.inundation
  return true
}

/** The features a view shows. The counts in `meta` stay what the API read. */
export function floodsForView(
  collection: FloodForecastCollection,
  view: FloodView,
): FloodForecastCollection {
  return {
    ...collection,
    features: collection.features.filter((feature) => {
      if (isGauge(feature)) {
        const { qualityVerified, severity } = feature.properties
        return (qualityVerified || view.extended) && (severity !== "no_flooding" || view.normal)
      }
      const { kind, lowerConfidence } = feature.properties
      return (!lowerConfidence || view.extended) && areaShown(kind, view)
    }),
  }
}

/**
 * Basins as a view reads them: without Extended coverage they say only what verified
 * evidence says, and lower-confidence gauges and their areas are not listed.
 */
export function basinsForView(document: BasinsDocument, view: FloodView): BasinsDocument {
  if (view.extended) return document
  return {
    ...document,
    basins: document.basins.map((basin) => ({
      ...basin,
      state: basin.verifiedState ?? basin.state,
      onlyUnverified: false,
      gauges: basin.gauges.filter((gauge) => gauge.verified),
      areas: basin.areas.filter((area) => !area.lowerConfidence),
    })),
    subbasins: document.subbasins.map((subbasin) => ({
      ...subbasin,
      state: subbasin.verifiedState ?? subbasin.state,
      onlyUnverified: false,
    })),
  }
}
