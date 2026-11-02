import type { Map as MapLibreMap } from "maplibre-gl"
import { createContext, useContext } from "react"

/** The map instance once its first style has loaded; null before. */
export const MapContext = createContext<MapLibreMap | null>(null)

export function useMap() {
  return useContext(MapContext)
}
