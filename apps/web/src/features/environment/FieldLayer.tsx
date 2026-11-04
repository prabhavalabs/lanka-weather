import type { ExpressionSpecification, ImageSource, Map as MapLibreMap } from "maplibre-gl"
import { useEffect, useRef } from "react"
import { colourTable, type RampStop } from "@/lib/colour"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady, type Slot } from "../map/slots"
import { gridBounds, imageSize, paintField, rasterise } from "../weather-grid/field"
import type { GridSpec } from "../location/types"
import { fadeEdges, ringToPixels } from "./present"

interface FieldLayerProps {
  /** Names the source and layer. */
  id: string
  grid: GridSpec | undefined
  values: (number | null)[] | undefined
  /** Read on every draw, so colours follow the theme. */
  ramp: () => RampStop[]
  /** Changes when what `ramp` gives does, other than by theme. */
  rampKey: string
  slot: Slot
  /** A number, or an expression that changes it with zoom. */
  opacity?: number | ExpressionSpecification
  /** Land rings to cut out, for a field that belongs on the water. */
  land?: [number, number][][]
}

/** How much finer a clipped field is drawn than it is sampled, so the coast is crisp. */
const clipScale = 3
const fadePixels = 45

/** The field as a PNG data URL, or null where the browser gives no canvas. */
function render(
  grid: GridSpec,
  values: (number | null)[],
  ramp: RampStop[],
  land: [number, number][][] | undefined,
): string | null {
  const [width, height] = imageSize(grid)
  const canvas = document.createElement("canvas")
  canvas.width = width
  canvas.height = height
  const context = canvas.getContext("2d")
  if (!context) return null
  const image = context.createImageData(width, height)
  paintField(rasterise(grid, values, width, height), colourTable(ramp, 1024), image.data)
  if (land) fadeEdges(image.data, width, height, fadePixels)
  context.putImageData(image, 0, 0)
  if (!land) return canvas.toDataURL("image/png")

  const clipped = document.createElement("canvas")
  clipped.width = width * clipScale
  clipped.height = height * clipScale
  const fine = clipped.getContext("2d")
  if (!fine) return null
  fine.imageSmoothingEnabled = true
  fine.drawImage(canvas, 0, 0, clipped.width, clipped.height)
  fine.globalCompositeOperation = "destination-out"
  fine.beginPath()
  for (const ring of land) {
    ringToPixels(ring, grid, clipped.width, clipped.height).forEach(([x, y], index) => {
      if (index === 0) fine.moveTo(x, y)
      else fine.lineTo(x, y)
    })
    fine.closePath()
  }
  fine.fill()
  return clipped.toDataURL("image/png")
}

function corners(
  grid: GridSpec,
): [[number, number], [number, number], [number, number], [number, number]] {
  const [west, south, east, north] = gridBounds(grid)
  return [
    [west, north],
    [east, north],
    [east, south],
    [west, south],
  ]
}

function show(
  map: MapLibreMap,
  id: string,
  grid: GridSpec,
  url: string,
  slot: Slot,
  opacity: number | ExpressionSpecification,
) {
  const sourceId = `${id}-field`
  const layerId = `${id}-field-image`
  const source = map.getSource(sourceId) as ImageSource | undefined
  if (source) source.updateImage({ url, coordinates: corners(grid) })
  else map.addSource(sourceId, { type: "image", url, coordinates: corners(grid) })
  if (!map.getLayer(layerId)) {
    map.addLayer(
      {
        id: layerId,
        type: "raster",
        source: sourceId,
        paint: {
          "raster-opacity": opacity,
          "raster-resampling": "linear",
          "raster-fade-duration": 0,
        },
      },
      slotBefore(map, slot),
    )
  }
}

function hide(map: MapLibreMap, id: string) {
  if (!styleReady(map)) return
  if (map.getLayer(`${id}-field-image`)) map.removeLayer(`${id}-field-image`)
  if (map.getSource(`${id}-field`)) map.removeSource(`${id}-field`)
}

/**
 * A smooth colour field from values on a grid, as the weather field draws
 * itself, for layers on grids of their own. A field with land rings stops at
 * the coast and fades out at its outer edges.
 */
export function FieldLayer({
  id,
  grid,
  values,
  ramp,
  rampKey,
  slot,
  opacity = 0.72,
  land,
}: FieldLayerProps) {
  const map = useMap()
  const current = useRef({ grid, values, ramp, land })

  useEffect(() => {
    current.current = { grid, values, ramp, land }
  }, [grid, values, ramp, land])

  useEffect(() => {
    if (!map) return
    const draw = () => {
      const { grid: spec, values: data, ramp: colours, land: rings } = current.current
      if (!spec || !data) return
      const url = render(spec, data, colours(), rings)
      if (url) show(map, id, spec, url, slot, opacity)
    }
    map.on("style.load", draw)
    return () => {
      map.off("style.load", draw)
      hide(map, id)
    }
  }, [map, id, slot, opacity])

  useEffect(() => {
    if (!map || !grid || !values || !styleReady(map)) return
    const url = render(grid, values, current.current.ramp(), land)
    if (url) show(map, id, grid, url, slot, opacity)
    else hide(map, id)
  }, [map, id, grid, values, rampKey, land, slot, opacity])

  return null
}
