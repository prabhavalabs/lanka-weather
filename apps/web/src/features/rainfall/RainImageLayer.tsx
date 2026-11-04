import type { Map as MapLibreMap, RasterTileSource } from "maplibre-gl"
import { useEffect, useRef } from "react"
import type { ResolvedTheme } from "@/lib/theme"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import { rainImageLift, tilesInBounds, tileUrl } from "./present"
import type { RainStep, RainSteps } from "./types"

const SOURCE = "rain-image"
const LAYER = "rain-image-raster"

interface RainImageLayerProps {
  steps: RainSteps | undefined
  step: RainStep | undefined
  /** The step shown next while playing, loaded ahead. */
  nextStep?: RainStep
  theme: ResolvedTheme
}

function template(steps: RainSteps, step: RainStep) {
  return steps.tilePath.replace("{time}", step.time)
}

function add(map: MapLibreMap, steps: RainSteps, step: RainStep, theme: ResolvedTheme) {
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "raster",
      tiles: [template(steps, step)],
      tileSize: steps.tileSize,
      minzoom: steps.minZoom,
      maxzoom: steps.maxZoom,
      bounds: steps.bounds,
      attribution: steps.attribution,
    })
  }
  if (!map.getLayer(LAYER)) {
    map.addLayer(
      {
        id: LAYER,
        type: "raster",
        source: SOURCE,
        paint: {
          "raster-opacity": 0.82,
          "raster-resampling": "linear",
          "raster-fade-duration": 0,
          "raster-brightness-min": rainImageLift(theme),
        },
      },
      slotBefore(map, "particles"),
    )
  }
}

function remove(map: MapLibreMap) {
  if (!styleReady(map)) return
  if (map.getLayer(LAYER)) map.removeLayer(LAYER)
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

/**
 * NASA's IMERG image for the chosen half hour, from our API, over land and
 * sea under the labels. Rain below 0.1 mm/h is transparent in the tiles.
 */
export function RainImageLayer({ steps, step, nextStep, theme }: RainImageLayerProps) {
  const map = useMap()
  const current = useRef({ steps, step, theme })

  useEffect(() => {
    current.current = { steps, step, theme }
  }, [steps, step, theme])

  useEffect(() => {
    if (!map) return
    const draw = () => {
      const { steps: data, step: shown, theme: colours } = current.current
      if (data && shown) add(map, data, shown, colours)
    }
    draw()
    map.on("style.load", draw)
    return () => {
      map.off("style.load", draw)
      remove(map)
    }
  }, [map])

  const url = steps && step ? template(steps, step) : null
  useEffect(() => {
    if (!map || !steps || !step || !url || !styleReady(map)) return
    const source = map.getSource(SOURCE) as RasterTileSource | undefined
    if (!source) add(map, steps, step, theme)
    else if (source.tiles?.[0] !== url) source.setTiles([url])
  }, [map, steps, step, url, theme])

  useEffect(() => {
    if (!map || !styleReady(map) || !map.getLayer(LAYER)) return
    map.setPaintProperty(LAYER, "raster-brightness-min", rainImageLift(theme))
  }, [map, theme])

  // Load the next image ahead, so stepping through time does not flash.
  useEffect(() => {
    if (!steps || !nextStep) return
    for (const tile of tilesInBounds(steps.bounds, steps.maxZoom)) {
      const image = new Image()
      image.src = tileUrl(steps.tilePath, nextStep.time, tile.z, tile.x, tile.y)
    }
  }, [steps, nextStep])

  return null
}
