import * as maplibregl from "maplibre-gl"
import "maplibre-gl/dist/maplibre-gl.css"
import workerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?url"
import sinhalaFontUrl from "@fontsource-variable/noto-sans-sinhala/files/noto-sans-sinhala-sinhala-wght-normal.woff2?url"
import tamilFontUrl from "@fontsource-variable/noto-sans-tamil/files/noto-sans-tamil-tamil-wght-normal.woff2?url"
import { useEffect, useRef, useState, type ReactNode } from "react"
import { getLocale, translateIn, useLocale, useT, type Locale } from "@/lib/i18n"
import { readTokens } from "@/lib/tokens"
import { basemapTokens, labelFields, localisedLabel, tuneBasemap } from "./basemap"
import { mapStyles, minimumZoom, panLimit, sriLankaBounds, type MapTheme } from "./config"
import { MapContext } from "./MapContext"

// MapLibre finds its worker next to its own module. Vite pre-bundles the
// module into its dependency cache (and the production build into assets/),
// where no worker file exists, so the worker is emitted and named explicitly.
maplibregl.setWorkerUrl(workerUrl)

/**
 * The app's own Sinhala and Tamil fonts, for map text in those scripts
 * (basemap.ts). Absolute, because MapLibre loads them outside the page.
 */
const scriptFonts = {
  sinhala: new URL(sinhalaFontUrl, window.location.href).href,
  tamil: new URL(tamilFontUrl, window.location.href).href,
}

/**
 * Tunes each fetched basemap to the theme and language in effect. The `dark`
 * class is set before a theme change reaches React, so the tokens read here
 * already belong to the new theme.
 */
const transformStyle: maplibregl.TransformStyleFunction = (_previous, next) =>
  tuneBasemap(next, readTokens(basemapTokens), { locale: getLocale(), fonts: scriptFonts })

/** MapLibre's own button labels, in a language. */
function controlStrings(locale: Locale): Record<string, string> {
  const t = (key: Parameters<typeof translateIn>[1]) => translateIn(locale, key)
  return {
    "NavigationControl.ZoomIn": t("map.controls.zoomIn"),
    "NavigationControl.ZoomOut": t("map.controls.zoomOut"),
    "NavigationControl.ResetBearing": t("map.controls.resetBearing"),
    "GeolocateControl.FindMyLocation": t("map.controls.findMyLocation"),
    "GeolocateControl.LocationNotAvailable": t("map.controls.locationNotAvailable"),
    "AttributionControl.ToggleAttribution": t("map.controls.toggleAttribution"),
    "AttributionControl.MapFeedback": t("map.controls.mapFeedback"),
  }
}

/** The control buttons MapLibre has already drawn, by the string that labels them. */
const controlButtons: Record<string, string> = {
  "NavigationControl.ZoomIn": ".maplibregl-ctrl-zoom-in",
  "NavigationControl.ZoomOut": ".maplibregl-ctrl-zoom-out",
  "NavigationControl.ResetBearing": ".maplibregl-ctrl-compass",
  "GeolocateControl.FindMyLocation": ".maplibregl-ctrl-geolocate",
  "AttributionControl.ToggleAttribution": ".maplibregl-ctrl-attrib-button",
}

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
      minZoom: minimumZoom,
      maxZoom: 16,
      attributionControl: { compact: true },
      locale: controlStrings(getLocale()),
    })
    instance.addControl(new maplibregl.NavigationControl({ visualizePitch: false }), "bottom-right")
    instance.addControl(new maplibregl.ScaleControl({ unit: "metric" }), "bottom-right")
    instance.addControl(
      new maplibregl.GeolocateControl({ positionOptions: { enableHighAccuracy: false } }),
      "bottom-right",
    )
    instance.once("load", () => {
      // The attribution names every source on the map, open it would run under the
      // cards: start it folded behind its button, as MapLibre does after the first drag.
      container.current
        ?.querySelector(".maplibregl-ctrl-attrib.maplibregl-compact-show")
        ?.classList.remove("maplibregl-compact-show")
      setMap(instance)
    })
    return () => {
      setMap(null)
      instance.remove()
    }
    // The map is created once; theme changes are applied below without recreating it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // A language change relabels the basemap in place; the style is not reloaded.
  const locale = useLocale()
  useEffect(() => {
    if (!map) return
    for (const { id, original } of labelFields(map.getStyle().layers)) {
      map.setLayoutProperty(
        id,
        "text-field",
        localisedLabel(original, locale) as maplibregl.ExpressionSpecification,
      )
    }
    // MapLibre labels its buttons once, when it draws them: relabel them in place.
    const strings = controlStrings(locale)
    for (const [key, selector] of Object.entries(controlButtons)) {
      for (const button of map.getContainer().querySelectorAll<HTMLElement>(selector)) {
        if (key === "GeolocateControl.FindMyLocation" && button.hasAttribute("disabled")) continue
        button.title = strings[key]!
        button.setAttribute("aria-label", strings[key]!)
      }
    }
  }, [map, locale])

  const t = useT()
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
          aria-label={t("app.mapLabel")}
          role="region"
        />
      </div>
      <MapContext.Provider value={map}>{children}</MapContext.Provider>
    </>
  )
}
