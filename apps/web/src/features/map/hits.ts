import type { MapGeoJSONFeature, Map as MapLibreMap, PointLike } from "maplibre-gl"

/**
 * What a click or a hover lands on, when layers overlap. Click and hover read
 * the same list, so what the chip names is what a click opens:
 *
 * 1. points, which are small and drawn on top;
 * 2. rivers, thin lines that would otherwise be lost under any area;
 * 3. a reservoir's water, a small shape with a place of its own;
 * 4. flood, landslide and official areas, current and in that order;
 * 5. past floods;
 * 6. basins;
 * 7. regions.
 */
export type HitGroup = "point" | "river" | "water" | "area" | "past" | "basin" | "region"

export interface HitLayers {
  group: HitGroup
  layers: string[]
  /** Points and lines are hit with a little slack; areas exactly under the pointer. */
  near: boolean
}

export interface Hit<Feature = MapGeoJSONFeature> {
  group: HitGroup
  feature: Feature
}

/** Which layers there are, by group. Every group is optional in `order`. */
export interface HitLayerIds {
  points: string[]
  rivers: string[]
  water: string[]
  /** Listed most current first: each is asked in turn, not only the topmost drawn. */
  areas: string[]
  past: string[]
  basins: string[]
  regions: string[]
}

export function hitOrder(ids: HitLayerIds): HitLayers[] {
  return [
    { group: "point", layers: ids.points, near: true },
    { group: "river", layers: ids.rivers, near: true },
    { group: "water", layers: ids.water, near: false },
    // Each its own step, so the order among areas does not depend on drawing order.
    ...ids.areas.map((id) => ({ group: "area" as const, layers: [id], near: false })),
    { group: "past", layers: ids.past, near: false },
    { group: "basin", layers: ids.basins, near: false },
    { group: "region", layers: ids.regions, near: false },
  ]
}

/** Features of some layers under the pointer, topmost first. */
export type HitQuery<Feature = MapGeoJSONFeature> = (layers: string[], near: boolean) => Feature[]

/**
 * Everything under the pointer, in order and asked for lazily: most events
 * stop at the first, and a query per layer group is not free on every mouse
 * move. A click may pass over a hit it cannot use.
 */
export function* orderedHits<Feature>(
  order: HitLayers[],
  query: HitQuery<Feature>,
): Generator<Hit<Feature>> {
  for (const { group, layers, near } of order) {
    if (layers.length === 0) continue
    for (const feature of query(layers, near)) yield { group, feature }
  }
}

/** The hit a hover names: the first in order. */
export function firstHit<Feature>(
  order: HitLayers[],
  query: HitQuery<Feature>,
): Hit<Feature> | undefined {
  for (const hit of orderedHits(order, query)) return hit
  return undefined
}

/** Asks the map, within `slack` pixels for points and lines. */
export function queryMap(map: MapLibreMap, point: { x: number; y: number }, slack: number) {
  const box: [PointLike, PointLike] = [
    [point.x - slack, point.y - slack],
    [point.x + slack, point.y + slack],
  ]
  return (layers: string[], near: boolean): MapGeoJSONFeature[] => {
    const present = layers.filter((id) => map.getLayer(id))
    if (present.length === 0) return []
    return map.queryRenderedFeatures(near ? box : ([point.x, point.y] as PointLike), {
      layers: present,
    })
  }
}
