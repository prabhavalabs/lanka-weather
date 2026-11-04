/**
 * Signed-distance icons for MapLibre symbols, drawn in code so they can be
 * coloured per feature with `icon-color` and outlined with `icon-halo-*`.
 * MapLibre reads an SDF image's alpha as distance, as TinySDF writes glyphs:
 * 0.75 on the edge, falling by 1/radius per pixel outward.
 */

export type Point = [number, number]

function segmentDistance(px: number, py: number, [ax, ay]: Point, [bx, by]: Point) {
  const dx = bx - ax
  const dy = by - ay
  const length = dx * dx + dy * dy
  const t = length === 0 ? 0 : Math.max(0, Math.min(1, ((px - ax) * dx + (py - ay) * dy) / length))
  return Math.hypot(px - (ax + t * dx), py - (ay + t * dy))
}

function inside(px: number, py: number, polygon: Point[]) {
  let result = false
  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i, i += 1) {
    const [xi, yi] = polygon[i]!
    const [xj, yj] = polygon[j]!
    if (yi > py !== yj > py && px < ((xj - xi) * (py - yi)) / (yj - yi) + xi) result = !result
  }
  return result
}

/** Signed distance from a point to a polygon's edge: negative inside. */
export function polygonDistance(px: number, py: number, polygon: Point[]): number {
  let distance = Infinity
  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i, i += 1) {
    distance = Math.min(distance, segmentDistance(px, py, polygon[j]!, polygon[i]!))
  }
  return inside(px, py, polygon) ? -distance : distance
}

/** The SDF alpha for a signed distance in pixels (TinySDF's encoding, cutoff 0.25). */
export function sdfAlpha(distance: number, radius = 8): number {
  return Math.round(Math.max(0, Math.min(255, 255 * (0.75 - distance / radius))))
}

/**
 * An SDF image `size` pixels square of a polygon given in pixels, for
 * `map.addImage(name, image, { sdf: true, pixelRatio })`.
 */
export function polygonSdf(size: number, polygon: Point[], radius = 8) {
  const data = new Uint8Array(size * size * 4)
  for (let y = 0; y < size; y += 1) {
    for (let x = 0; x < size; x += 1) {
      const alpha = sdfAlpha(polygonDistance(x + 0.5, y + 0.5, polygon), radius)
      const at = (y * size + x) * 4
      data[at] = 255
      data[at + 1] = 255
      data[at + 2] = 255
      data[at + 3] = alpha
    }
  }
  return { width: size, height: size, data }
}
