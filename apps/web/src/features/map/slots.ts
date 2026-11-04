import type { LayerSpecification, Map as MapLibreMap } from "maplibre-gl"

/**
 * Where data layers sit in the basemap's layer stack, bottom to top:
 *
 * - `fills`: the weather field and region colours, over land and landcover
 *   but under a copy of the basemap's water, so they are clipped to the
 *   coast at every zoom;
 * - `basins`: flood basin tints, over those colours and also under the water;
 * - `lines`: region boundaries and basin outlines, over roads;
 * - `past`: past flood extents, under anything current;
 * - `areas`: flood polygons;
 * - `rivers`: the river network, over flood areas so a river stays readable;
 * - `particles`: the animated wind;
 * - basemap labels;
 * - `points`: river gauges, flood gauges and their labels, on top.
 *
 * Each slot is an invisible anchor layer; a data layer is added just under
 * its slot's anchor, so the order holds whichever layer is added first.
 * MapLibre reloads the whole style on a theme change, so layers call
 * `slotBefore` on every style load.
 */
export type Slot =
  "fills" | "basins" | "lines" | "past" | "areas" | "rivers" | "particles" | "points"

const anchors: Record<Slot, string> = {
  fills: "slot:fills",
  basins: "slot:basins",
  lines: "slot:lines",
  past: "slot:past",
  areas: "slot:areas",
  rivers: "slot:rivers",
  particles: "slot:particles",
  points: "slot:points",
}

/** The copy of the basemap's water that clips the fills. */
export const waterMaskId = "slot:water-mask"

type LayerLike = Pick<LayerSpecification, "id" | "type"> & { "source-layer"?: string }

function sourceLayer(layer: LayerLike): string | undefined {
  return layer["source-layer"]
}

/**
 * Where the anchors go in a basemap's layers: the fills before the first
 * layer drawn over the ground (the first line or symbol), the overlays
 * before the labels that follow the last line or fill. Undefined means
 * "on top". `water` is the basemap's water fill, copied as the mask.
 */
export function slotPlan(layers: readonly LayerLike[]) {
  const firstOverGround = layers.find(
    (layer) => layer.type !== "background" && layer.type !== "fill",
  )
  let lastShape = -1
  layers.forEach((layer, index) => {
    if (layer.type !== "symbol") lastShape = index
  })
  const water = layers.find((layer) => layer.type === "fill" && sourceLayer(layer) === "water")
  return {
    fillsBefore: firstOverGround?.id,
    overlaysBefore: layers[lastShape + 1]?.id,
    waterId: water?.id,
  }
}

function anchor(id: string): LayerSpecification {
  return { id, type: "background", layout: { visibility: "none" } }
}

/** Adds the slot anchors and the water mask to the current style, once per style load. */
export function ensureSlots(map: MapLibreMap) {
  if (map.getLayer(anchors.points)) return
  const layers = map.getStyle().layers
  const plan = slotPlan(layers)

  map.addLayer(anchor(anchors.fills), plan.fillsBefore)
  map.addLayer(anchor(anchors.basins), plan.fillsBefore)
  const water = layers.find((layer) => layer.id === plan.waterId)
  if (water) map.addLayer({ ...water, id: waterMaskId } as LayerSpecification, plan.fillsBefore)

  for (const slot of ["lines", "past", "areas", "rivers", "particles"] as const) {
    map.addLayer(anchor(anchors[slot]), plan.overlaysBefore)
  }
  map.addLayer(anchor(anchors.points))
}

/** The layer id to insert a layer before so that it sits in `slot`. */
export function slotBefore(map: MapLibreMap, slot: Slot): string {
  ensureSlots(map)
  return anchors[slot]
}

/** Whether the map's style is loaded and accepts layers (not torn down or mid style change). */
export function styleReady(map: MapLibreMap): boolean {
  try {
    return Boolean(map.getStyle())
  } catch {
    return false
  }
}
