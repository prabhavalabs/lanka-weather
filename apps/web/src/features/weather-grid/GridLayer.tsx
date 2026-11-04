import type { ImageSource, Map as MapLibreMap } from "maplibre-gl"
import { useEffect, useRef } from "react"
import { colourTable, rampFromScale, type RampStop } from "@/lib/colour"
import { readToken } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import { gridBounds, imageSize, paintField, rasterise } from "./field"
import type { CurrentGrid } from "./types"
import { gridVariables, type GridVariable } from "./variables"

const SOURCE = "weather-grid"
const LAYER = "weather-grid-field"

interface GridLayerProps {
  grid: CurrentGrid | undefined
  variable: GridVariable
  opacity?: number
}

/** The variable's colour ramp from the current theme's tokens. */
function rampFor(variable: GridVariable): RampStop[] {
  const spec = gridVariables[variable]
  const colours = spec.scale.map((step) => readToken(step.token))
  const ramp = rampFromScale(spec.scale, colours, spec.openSpan)
  if (spec.fadeBelow && ramp[0]) {
    // Below the first step (a trace of rain) the colour fades out rather than stopping.
    const first = ramp[0]
    ramp.unshift({ value: 0, color: [first.color[0], first.color[1], first.color[2], 0] })
  }
  return ramp
}

/** The field as a PNG data URL, or null without values. */
function render(grid: CurrentGrid, variable: GridVariable): string | null {
  const values = grid.values[gridVariables[variable].key]
  if (!values) return null
  const [width, height] = imageSize(grid.grid)
  const field = rasterise(grid.grid, values, width, height)
  const canvas = document.createElement("canvas")
  canvas.width = width
  canvas.height = height
  const context = canvas.getContext("2d")
  if (!context) return null
  const image = context.createImageData(width, height)
  paintField(field, colourTable(rampFor(variable), 512), image.data)
  context.putImageData(image, 0, 0)
  return canvas.toDataURL("image/png")
}

function coordinates(
  grid: CurrentGrid,
): [[number, number], [number, number], [number, number], [number, number]] {
  const [west, south, east, north] = gridBounds(grid.grid)
  return [
    [west, north],
    [east, north],
    [east, south],
    [west, south],
  ]
}

function show(map: MapLibreMap, grid: CurrentGrid, url: string, opacity: number) {
  const source = map.getSource(SOURCE) as ImageSource | undefined
  if (source) {
    source.updateImage({ url, coordinates: coordinates(grid) })
  } else {
    map.addSource(SOURCE, {
      type: "image",
      url,
      coordinates: coordinates(grid),
    })
  }
  if (!map.getLayer(LAYER)) {
    map.addLayer(
      {
        id: LAYER,
        type: "raster",
        source: SOURCE,
        paint: {
          "raster-opacity": opacity,
          "raster-resampling": "linear",
          "raster-fade-duration": 0,
        },
      },
      slotBefore(map, "fills"),
    )
  }
}

function hide(map: MapLibreMap) {
  if (!styleReady(map)) return
  if (map.getLayer(LAYER)) map.removeLayer(LAYER)
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

/**
 * The weather grid as a smooth colour field: bilinear between the 0.2°
 * nodes, painted with the variable's ramp from the theme, and laid under the
 * basemap's water so it stops at the coast.
 */
export function GridLayer({ grid, variable, opacity = 0.72 }: GridLayerProps) {
  const map = useMap()
  const current = useRef<{ grid?: CurrentGrid; variable: GridVariable }>({ grid, variable })

  useEffect(() => {
    if (!map) return
    // Re-render on every style load: a theme change brings other colours.
    const draw = () => {
      const { grid: data, variable: shown } = current.current
      if (!data) return
      const url = render(data, shown)
      if (url) show(map, data, url, opacity)
    }
    map.on("style.load", draw)
    return () => {
      map.off("style.load", draw)
      hide(map)
    }
  }, [map, opacity])

  useEffect(() => {
    current.current = { grid, variable }
    if (!map || !grid || !styleReady(map)) return
    const url = render(grid, variable)
    if (url) show(map, grid, url, opacity)
    else hide(map)
  }, [map, grid, variable, opacity])

  return null
}
