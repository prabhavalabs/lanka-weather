import { Marker } from "maplibre-gl"
import { useEffect, useRef } from "react"
import { useMap } from "../map/MapContext"

interface LocationMarkerProps {
  lat: number
  lon: number
  /** Accessible name, e.g. the place's name. */
  label: string
}

/** A pin at the place the location panel describes. */
export function LocationMarker({ lat, lon, label }: LocationMarkerProps) {
  const map = useMap()
  const marker = useRef<Marker | null>(null)

  useEffect(() => {
    if (!map) return
    const element = document.createElement("div")
    element.className = "location-pin"
    element.setAttribute("role", "img")
    const instance = new Marker({ element, anchor: "bottom" }).setLngLat([0, 0])
    marker.current = instance
    return () => {
      instance.remove()
      marker.current = null
    }
  }, [map])

  useEffect(() => {
    if (!map || !marker.current) return
    marker.current.setLngLat([lon, lat]).addTo(map)
    marker.current.getElement().setAttribute("aria-label", `Selected place: ${label}`)
  }, [map, lat, lon, label])

  return null
}
