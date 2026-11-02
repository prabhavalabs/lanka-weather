import * as maplibregl from "maplibre-gl"
import "maplibre-gl/dist/maplibre-gl.css"
import workerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?url"
import { useEffect, useRef, useState, type ReactNode } from "react"
import { readTokens } from "@/lib/tokens"
import { basemapTokens, tuneBasemap } from "./basemap"
import { mapStyles, panLimit, sriLankaBounds, type MapTheme } from "./config"
import { MapContext } from "./MapContext"

// MapLibre finds its worker next to its own module. Vite pre-bundles the
// module into its dependency cache (and the production build into assets/),
// where no worker file exists, so the worker is emitted and named explicitly.
maplibregl.setWorkerUrl(workerUrl)

/**
 * Tunes each fetched basemap to the theme in effect. The `dark` class is set
 * before a theme change reaches React, so the tokens read here already
 * belong to the new theme.
 */
const transformStyle: maplibregl.TransformStyleFunction = (_previous, next) =>
  tuneBasemap(next, readTokens(basemapTokens))

interface MapViewProps {
  theme: MapTheme
  /** Called with the map once its first style has loaded, and with null when it goes. */
  onReady?: (map: maplibregl.Map | null) => void
  /** Data layers; they read the map from MapContext. */
  children?: ReactNode
}

/** The base map of Sri Lanka. Data layers are rendered as children. */
export function MapView({ theme, onReady, children }: MapViewProps) {
  const container = useRef<HTMLDivElement>(null)
  const [map, setMap] = useState<maplibregl.Map | null>(null)
  const ready = useRef(onReady)

  useEffect(() => {
    ready.current = onReady
  }, [onReady])

  useEffect(() => {
    ready.current?.(map)
  }, [map])

  useEffect(() => {
    if (!container.current) return
    const instance = new maplibregl.Map({
      container: container.current,
      style: mapStyles[theme],
      transformStyle,
      bounds: sriLankaBounds,
      fitBoundsOptions: { padding: 24 },
      maxBounds: panLimit,
      minZoom: 5,
      maxZoom: 16,
      attributionControl: { compact: true },
    })
    instance.addControl(new maplibregl.NavigationControl({ visualizePitch: false }), "bottom-right")
    instance.addControl(new maplibregl.ScaleControl({ unit: "metric" }), "bottom-right")
    instance.addControl(
      new maplibregl.GeolocateControl({ positionOptions: { enableHighAccuracy: false } }),
      "bottom-right",
    )
    instance.once("load", () => {
      // On phones the attribution, open at first, would cover much of the map: start it
      // folded behind its button, as MapLibre does after the first drag.
      if (instance.getCanvasContainer().offsetWidth <= 640) {
        container.current
          ?.querySelector(".maplibregl-ctrl-attrib.maplibregl-compact-show")
          ?.classList.remove("maplibregl-compact-show")
      }
      setMap(instance)
    })
    return () => {
      setMap(null)
      instance.remove()
    }
    // The map is created once; theme changes are applied below without recreating it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadedTheme = useRef(theme)
  useEffect(() => {
    if (!map || loadedTheme.current === theme) return
    loadedTheme.current = theme
    // A full reload rather than a diff, so "style.load" fires and data layers
    // re-add themselves with the new theme's colours.
    map.setStyle(mapStyles[theme], { diff: false, transformStyle })
  }, [map, theme])

  return (
    <>
      {/* MapLibre's stylesheet gives its container `position: relative`, which
          outranks Tailwind's layered utilities, so the positioning lives on a
          wrapper and the map container only fills it. */}
      <div className="absolute inset-0">
        <div
          ref={container}
          className="h-full w-full"
          aria-label="Map of Sri Lanka"
          role="region"
        />
      </div>
      <MapContext.Provider value={map}>{children}</MapContext.Provider>
    </>
  )
}
