import type { Map as MapLibreMap, MapMouseEvent } from "maplibre-gl"
import { useEffect, useRef } from "react"

/** Calls `handler` for every click on the map; the latest handler is used. */
export function useMapClick(map: MapLibreMap | null, handler: (event: MapMouseEvent) => void) {
  const ref = useRef(handler)
  useEffect(() => {
    ref.current = handler
  }, [handler])
  useEffect(() => {
    if (!map) return
    const click = (event: MapMouseEvent) => ref.current(event)
    map.on("click", click)
    return () => {
      map.off("click", click)
    }
  }, [map])
}
