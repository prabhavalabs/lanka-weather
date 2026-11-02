import * as maplibregl from "maplibre-gl"
import "maplibre-gl/dist/maplibre-gl.css"
import { useEffect, useRef, useState, type ReactNode } from "react"
import { mapStyles, panLimit, sriLankaBounds, type MapTheme } from "./config"
import { MapContext } from "./MapContext"

interface MapViewProps {
  theme: MapTheme
  /** Data layers; they read the map from MapContext. */
  children?: ReactNode
}

/** The base map of Sri Lanka. Data layers are rendered as children. */
export function MapView({ theme, children }: MapViewProps) {
  const container = useRef<HTMLDivElement>(null)
  const [map, setMap] = useState<maplibregl.Map | null>(null)

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
    instance.once("load", () => setMap(instance))
    return () => {
      setMap(null)
      instance.remove()
    }
    // The map is created once; theme changes are applied below without recreating it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    // Data layers re-add themselves on "style.load" after a style change.
    map?.setStyle(mapStyles[theme])
  }, [map, theme])

  return (
    <>
      <div
        ref={container}
        className="absolute inset-0"
        aria-label="Map of Sri Lanka"
        role="region"
      />
      <MapContext.Provider value={map}>{children}</MapContext.Provider>
    </>
  )
}
