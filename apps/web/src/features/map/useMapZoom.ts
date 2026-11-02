import type { Map as MapLibreMap } from "maplibre-gl"
import { useEffect, useState } from "react"

/** The map's zoom to a tenth, updated when a move ends. */
export function useMapZoom(map: MapLibreMap | null): number {
  const [zoom, setZoom] = useState(() => (map ? map.getZoom() : 7))
  useEffect(() => {
    if (!map) return
    const update = () => setZoom(Math.round(map.getZoom() * 10) / 10)
    update()
    map.on("moveend", update)
    return () => {
      map.off("moveend", update)
    }
  }, [map])
  return zoom
}
