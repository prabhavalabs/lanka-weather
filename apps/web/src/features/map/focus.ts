/**
 * The focus mask: a grey veil over everything but the selected shape, so a
 * division or flood area chosen from a list stands out on a busy map. Pure,
 * so it is tested without a browser.
 */
import type { Feature, MultiPolygon, Polygon, Position } from "geojson"

export type FocusGeometry = Polygon | MultiPolygon

/** The whole world in Web Mercator's range: the veil's outer ring. */
const world: Position[] = [
  [-180, -85],
  [180, -85],
  [180, 85],
  [-180, 85],
  [-180, -85],
]

/** Twice the signed area of a ring: positive when it runs anticlockwise. */
export function signedArea(ring: Position[]): number {
  let sum = 0
  for (let i = 0; i < ring.length - 1; i += 1) {
    const [x1 = 0, y1 = 0] = ring[i]!
    const [x2 = 0, y2 = 0] = ring[i + 1]!
    sum += x1 * y2 - x2 * y1
  }
  return sum
}

function clockwise(ring: Position[]): Position[] {
  return signedArea(ring) > 0 ? [...ring].reverse() : ring
}

/** The outer ring of each part; a part's own holes stay under the veil. */
function outerRings(geometry: FocusGeometry): Position[][] {
  const parts = geometry.type === "Polygon" ? [geometry.coordinates] : geometry.coordinates
  return parts.map((part) => part[0]).filter((ring): ring is Position[] => (ring?.length ?? 0) >= 4)
}

/**
 * The world with the selected shape cut out. Rings follow RFC 7946 (outer
 * anticlockwise, holes clockwise): MapLibre tells a hole from a new polygon
 * by its winding, so a hole drawn the wrong way would be filled in.
 */
export function focusVeil(geometry: FocusGeometry): Feature<Polygon> | null {
  const holes = outerRings(geometry).map(clockwise)
  if (holes.length === 0) return null
  return {
    type: "Feature",
    properties: {},
    geometry: { type: "Polygon", coordinates: [world, ...holes] },
  }
}

/** West, south, east, north of the shape's outer rings, or null for an empty shape. */
export function focusBounds(geometry: FocusGeometry): [number, number, number, number] | null {
  let box: [number, number, number, number] | null = null
  for (const ring of outerRings(geometry)) {
    for (const [lon = 0, lat = 0] of ring) {
      box = box
        ? [
            Math.min(box[0], lon),
            Math.min(box[1], lat),
            Math.max(box[2], lon),
            Math.max(box[3], lat),
          ]
        : [lon, lat, lon, lat]
    }
  }
  return box
}

/** Bounds grown by a share of their size on every side, so a shape is seen with its surroundings. */
export function padBounds(
  [west, south, east, north]: [number, number, number, number],
  share = 0.25,
): [number, number, number, number] {
  const x = (east - west) * share
  const y = (north - south) * share
  return [west - x, south - y, east + x, north + y]
}

/** Polygon or MultiPolygon from a feature whose geometry type is loosely typed. */
export function asFocusGeometry(
  geometry: { type: string; coordinates: unknown } | null | undefined,
): FocusGeometry | null {
  if (!geometry) return null
  if (geometry.type === "Polygon" || geometry.type === "MultiPolygon") {
    return geometry as FocusGeometry
  }
  return null
}
