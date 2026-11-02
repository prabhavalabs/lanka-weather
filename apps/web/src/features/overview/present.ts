/**
 * How the overview reads: pure, so it is tested without a browser. The words of
 * official warnings and the sentence at the top come from the API.
 */
import { inPointBounds, roundCoordinate, type Selection } from "../location/selection"
import type { RegionVariable } from "../regions/present"
import type { OverviewGroup, OverviewSection, OverviewTarget, OverviewTone } from "./types"

export const groupOrder: OverviewGroup[] = ["official", "water", "hazards", "now"]

export const groupLabels: Record<OverviewGroup, string> = {
  official: "Official",
  water: "Rivers and reservoirs",
  hazards: "Cyclones and earthquakes",
  now: "Right now",
}

/** Status badges for a line's tone; the badge's word is the API's. */
export const toneBadge = {
  neutral: "chip",
  watch: "alert",
  high: "minor_flood",
  severe: "major_flood",
} as const satisfies Record<OverviewTone, string>

/** Sections show at most this many lines, and offer the rest through their target. */
const maxItems: Record<string, number> = { reservoirs: 5, gauges: 5 }

export function shownItems(section: OverviewSection) {
  const limit = maxItems[section.id] ?? section.items.length
  return { items: section.items.slice(0, limit), more: Math.max(0, section.items.length - limit) }
}

/** Only rivers and landslide levels have a tally worth a line; zero landslides say nothing. */
export function shownCounts(section: OverviewSection) {
  if (section.id === "rivers") return section.counts
  if (section.id === "landslides" && section.counts.some((count) => count.value > 0)) {
    return section.counts
  }
  return []
}

/** The tag beside a section's title that says what kind of thing it is. */
export function sectionTag(section: OverviewSection): string | null {
  if (section.official) return "Official"
  if (section.estimate) return "Estimate"
  return null
}

/** What a click on a line does: layers to switch on, then what to open or show. */
export interface OverviewAction {
  enable: string[]
  selection?: Selection
  /** Where the camera goes for the selection. */
  camera?: "fit" | "reveal"
  /** A place to bring into view when nothing is selected. */
  reveal?: [number, number]
  fill?: RegionVariable
  /** Regions to show districts. */
  level?: "district"
}

export function overviewAction(target: OverviewTarget): OverviewAction {
  const enable = target.layer ? [target.layer] : []
  const id = target.id ?? ""
  switch (target.kind) {
    case "warnings":
      return { enable, selection: { kind: "warnings", ...(id ? { type: id } : {}) } }
    case "landslide":
      return {
        enable: [...enable, "landslide-warnings"],
        selection: { kind: "landslide", id },
      }
    case "river":
      return {
        enable: [...enable, "rivers"],
        selection:
          target.reach !== undefined
            ? { kind: "river", id, reach: target.reach }
            : { kind: "river", id },
        camera: "fit",
      }
    case "station":
      return {
        enable: [...enable, "river-levels"],
        selection: { kind: "station", id },
        camera: "reveal",
      }
    case "raingauge":
      return {
        enable: [...enable, "rain-gauges"],
        selection: { kind: "raingauge", id },
        camera: "reveal",
      }
    case "reservoir":
      return {
        enable: [...enable, "reservoirs"],
        selection: { kind: "reservoir", id },
        camera: "reveal",
      }
    case "reservoirs":
      return { enable: [...enable, "reservoirs"], selection: { kind: "reservoirs" } }
    case "cyclone":
      return { enable: [...enable, "cyclones"], selection: { kind: "cyclone", id } }
    case "quake":
      return { enable: [...enable, "earthquakes"], selection: { kind: "quake", id } }
    case "region":
      return {
        enable,
        selection: { kind: "region", id },
        camera: "fit",
        fill: target.fill,
        level: "district",
      }
    case "place":
    case "layer": {
      const place =
        target.lon !== undefined && target.lat !== undefined
          ? ([target.lon, target.lat] as [number, number])
          : undefined
      if (target.kind === "place" && place && inPointBounds(place[1], place[0])) {
        return {
          enable,
          selection: {
            kind: "point",
            lat: roundCoordinate(place[1]),
            lon: roundCoordinate(place[0]),
          },
          camera: "reveal",
        }
      }
      return { enable, reveal: place, fill: target.fill }
    }
  }
}
