import type { Map as MapLibreMap } from "maplibre-gl"
import { polygonSdf, type Point } from "../map/sdf"

/** A rounded square: Met stations, so they cannot be taken for round river gauges. */
export const STATION_ICON = "met-station-square"

const SIZE = 40
const INSET = 8
const CORNER = 5

function roundedSquare(): Point[] {
  const low = INSET
  const high = SIZE - INSET
  const points: Point[] = []
  const arc = (cx: number, cy: number, from: number) => {
    for (let step = 0; step <= 4; step += 1) {
      const angle = from + (step / 4) * (Math.PI / 2)
      points.push([cx + CORNER * Math.cos(angle), cy + CORNER * Math.sin(angle)])
    }
  }
  arc(high - CORNER, low + CORNER, -Math.PI / 2)
  arc(high - CORNER, high - CORNER, 0)
  arc(low + CORNER, high - CORNER, Math.PI / 2)
  arc(low + CORNER, low + CORNER, Math.PI)
  return points
}

/** Adds the icon to the current style; images go with the style on a theme change. */
export function addStationIcon(map: MapLibreMap) {
  if (map.hasImage(STATION_ICON)) return
  map.addImage(STATION_ICON, polygonSdf(SIZE, roundedSquare()), { sdf: true, pixelRatio: 1 })
}
