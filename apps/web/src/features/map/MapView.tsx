import * as maplibregl from "maplibre-gl"
import "maplibre-gl/dist/maplibre-gl.css"
import { useEffect, useRef } from "react"
import { mapStyles, panLimit, sriLankaBounds, type MapTheme } from "./config"

interface MapViewProps {
  theme: MapTheme
}

/** The base map of Sri Lanka. Data layers are added on top of it. */
export function MapView({ theme }: MapViewProps) {
  const container = useRef<HTMLDivElement>(null)
  const map = useRef<maplibregl.Map | null>(null)

  useEffect(() => {
    if (!container.current) return
    const instance = new maplibregl.Map({
      container: container.current,
      style: mapStyles[theme],
      bounds: sriLankaBounds,
      fitBoundsOptions: { padding: 24 },
      maxBounds: panLimit,
      minZoom: 5,
      maxZoom: 16,
      attributionControl: { compact: true },
    })
    instance.addControl(new maplibregl.NavigationControl({ visualizePitch: false }), "bottom-right")
    instance.addControl(new maplibregl.ScaleControl({ unit: "metric" }), "bottom-left")
    instance.addControl(
      new maplibregl.GeolocateControl({ positionOptions: { enableHighAccuracy: false } }),
      "bottom-right",
    )
    map.current = instance
    return () => {
      instance.remove()
      map.current = null
    }
    // The map is created once; theme changes are applied below without recreating it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    map.current?.setStyle(mapStyles[theme])
  }, [theme])

  return (
    <div ref={container} className="absolute inset-0" aria-label="Map of Sri Lanka" role="region" />
  )
}
