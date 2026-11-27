import type { GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { Feature, FeatureCollection } from "geojson"
import { useEffect, useMemo, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { focusVeil, type FocusGeometry } from "./focus"
import { useMap } from "./MapContext"
import { slotBefore, styleReady } from "./slots"

const VEIL_SOURCE = "focus-veil"
const SHAPE_SOURCE = "focus-shape"
const VEIL = "focus-veil-fill"
const HALO = "focus-shape-halo"
const LINE = "focus-shape-line"

const empty: FeatureCollection = { type: "FeatureCollection", features: [] }

function collection(feature: Feature | null): FeatureCollection {
  return feature ? { type: "FeatureCollection", features: [feature] } : empty
}

interface Shapes {
  veil: FeatureCollection
  shape: FeatureCollection
}

function addLayers(map: MapLibreMap, shapes: Shapes) {
  const palette = readTokens({ veil: "--map-focus-veil", line: "--foreground", halo: "--card" })
  if (!map.getSource(VEIL_SOURCE))
    map.addSource(VEIL_SOURCE, { type: "geojson", data: shapes.veil })
  if (!map.getSource(SHAPE_SOURCE)) {
    map.addSource(SHAPE_SOURCE, { type: "geojson", data: shapes.shape })
  }
  // Over the flood areas, divisions and rivers; under the basemap labels and
  // the gauges, so place names and readings stay legible.
  if (!map.getLayer(VEIL)) {
    map.addLayer(
      { id: VEIL, type: "fill", source: VEIL_SOURCE, paint: { "fill-color": palette.veil } },
      slotBefore(map, "particles"),
    )
  }
  if (!map.getLayer(HALO)) {
    map.addLayer(
      {
        id: HALO,
        type: "line",
        source: SHAPE_SOURCE,
        layout: { "line-join": "round" },
        paint: { "line-color": palette.halo, "line-width": 5, "line-opacity": 0.9 },
      },
      slotBefore(map, "particles"),
    )
  }
  if (!map.getLayer(LINE)) {
    map.addLayer(
      {
        id: LINE,
        type: "line",
        source: SHAPE_SOURCE,
        layout: { "line-join": "round" },
        paint: { "line-color": palette.line, "line-width": 2 },
      },
      slotBefore(map, "particles"),
    )
  }
}

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of [LINE, HALO, VEIL]) if (map.getLayer(id)) map.removeLayer(id)
  for (const id of [SHAPE_SOURCE, VEIL_SOURCE]) if (map.getSource(id)) map.removeSource(id)
}

interface FocusMaskProps {
  /** The selected area; null draws nothing. */
  geometry: FocusGeometry | null
}

/**
 * Greys out the map around the selected area and outlines it in the text
 * colour on a halo, so it reads on any fill beneath (a red division's own
 * red outline does not). Clicks pass through: the veil is not a hit layer.
 */
export function FocusMask({ geometry }: FocusMaskProps) {
  const map = useMap()
  const shapes = useMemo<Shapes>(
    () => ({
      veil: collection(geometry ? focusVeil(geometry) : null),
      shape: collection(geometry ? { type: "Feature", properties: {}, geometry } : null),
    }),
    [geometry],
  )
  const current = useRef(shapes)

  useEffect(() => {
    if (!map) return
    const add = () => addLayers(map, current.current)
    add()
    map.on("style.load", add)
    return () => {
      map.off("style.load", add)
      removeLayers(map)
    }
  }, [map])

  useEffect(() => {
    current.current = shapes
    ;(map?.getSource(VEIL_SOURCE) as GeoJSONSource | undefined)?.setData(shapes.veil)
    ;(map?.getSource(SHAPE_SOURCE) as GeoJSONSource | undefined)?.setData(shapes.shape)
  }, [map, shapes])

  return null
}
