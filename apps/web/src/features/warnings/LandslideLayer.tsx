import type { GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection, Point } from "geojson"
import { useEffect, useMemo, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import type { LandslideLayerData } from "./types"

const SOURCE = "landslide-warnings"
const LABEL_SOURCE = "landslide-warnings-labels"
/** The fill: what clicks and hovers hit. */
export const LANDSLIDE_FILL = "landslide-warnings-fill"
const HATCH = "landslide-warnings-hatch"
const LINE = "landslide-warnings-line"
const SELECTED = "landslide-warnings-selected"
const LABELS = "landslide-warnings-labels"
const HATCH_IMAGE = (level: number) => `landslide-hatch-${level}`

const selected = ["boolean", ["feature-state", "selected"], false] as const

/**
 * The cue that does not rely on colour: dots for level 1, widely spaced lines
 * for level 2 and a dense cross-hatch for level 3, matched by the legend.
 */
function hatchImage(level: number, colour: [number, number, number]) {
  const size = 16
  const data = new Uint8ClampedArray(size * size * 4)
  const paint = (x: number, y: number) => {
    const at = (((y + size) % size) * size + ((x + size) % size)) * 4
    data[at] = colour[0]
    data[at + 1] = colour[1]
    data[at + 2] = colour[2]
    data[at + 3] = 255
  }
  for (let i = 0; i < size; i += 1) {
    if (level === 1) {
      if (i < 2) for (let j = 0; j < 2; j += 1) paint(i + 7, j + 7)
    } else if (level === 2) {
      paint(i, i)
      paint(i, i + 1)
    } else {
      paint(i, i)
      paint(i, i + 1)
      paint(i, i + 8)
      paint(i, i + 9)
      paint(i, size - i)
      paint(i, size - i + 1)
      paint(i, size - i + 8)
      paint(i, size - i + 9)
    }
  }
  return { width: size, height: size, data }
}

function rgb(hex: string): [number, number, number] {
  const value = hex.replace("#", "")
  return [
    Number.parseInt(value.slice(0, 2), 16),
    Number.parseInt(value.slice(2, 4), 16),
    Number.parseInt(value.slice(4, 6), 16),
  ]
}

function readPalette() {
  return readTokens({
    level1: "--status-alert",
    level2: "--status-minor-flood",
    level3: "--status-major-flood",
    stroke: "--status-stroke",
    strokeSelected: "--status-stroke-selected",
    label: "--map-label-strong",
    halo: "--map-label-halo",
  })
}

function addLayers(map: MapLibreMap, data: LandslideLayerData, labels: FeatureCollection) {
  const palette = readPalette()
  const colour = [
    "match",
    ["get", "level"],
    1,
    palette.level1,
    2,
    palette.level2,
    palette.level3,
  ] as const
  for (const level of [1, 2, 3]) {
    const name = HATCH_IMAGE(level)
    if (map.hasImage(name)) map.removeImage(name)
    map.addImage(name, hatchImage(level, rgb(palette.stroke)), { pixelRatio: 2 })
  }
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      promoteId: "id",
      attribution: "Landslide warnings: NBRO, DMC",
    })
  }
  if (!map.getSource(LABEL_SOURCE)) {
    map.addSource(LABEL_SOURCE, { type: "geojson", data: labels })
  }
  if (!map.getLayer(LANDSLIDE_FILL)) {
    map.addLayer(
      {
        id: LANDSLIDE_FILL,
        type: "fill",
        source: SOURCE,
        paint: {
          "fill-color": colour as never,
          "fill-opacity": ["interpolate", ["linear"], ["zoom"], 7, 0.9, 12, 0.5],
        },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(HATCH)) {
    map.addLayer(
      {
        id: HATCH,
        type: "fill",
        source: SOURCE,
        paint: {
          "fill-pattern": [
            "match",
            ["get", "level"],
            1,
            HATCH_IMAGE(1),
            2,
            HATCH_IMAGE(2),
            HATCH_IMAGE(3),
          ],
          "fill-opacity": 0.32,
        },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(LINE)) {
    map.addLayer(
      {
        id: LINE,
        type: "line",
        source: SOURCE,
        layout: { "line-join": "round" },
        paint: {
          "line-color": palette.stroke,
          "line-width": ["match", ["get", "level"], 1, 0.75, 2, 1.25, 1.75],
          "line-opacity": 0.8,
        },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(SELECTED)) {
    map.addLayer(
      {
        id: SELECTED,
        type: "line",
        source: SOURCE,
        layout: { "line-join": "round" },
        paint: {
          "line-color": palette.strokeSelected,
          "line-width": ["case", selected as never, 3, 0],
        },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(LABELS)) {
    map.addLayer(
      {
        id: LABELS,
        type: "symbol",
        source: LABEL_SOURCE,
        minzoom: 7,
        layout: {
          "text-field": ["get", "text"],
          "text-font": ["Noto Sans Regular"],
          "text-size": 12,
          "text-allow-overlap": false,
          "text-optional": true,
        },
        paint: {
          "text-color": palette.label,
          "text-halo-color": palette.halo,
          "text-halo-width": 1.75,
        },
      },
      slotBefore(map, "points"),
    )
  }
}

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of [LABELS, SELECTED, LINE, HATCH, LANDSLIDE_FILL]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  for (const id of [SOURCE, LABEL_SOURCE]) if (map.getSource(id)) map.removeSource(id)
}

const emptyData = {
  type: "FeatureCollection",
  features: [],
  meta: {},
} as unknown as LandslideLayerData

function labelCollection(data: LandslideLayerData): FeatureCollection<Point> {
  return {
    type: "FeatureCollection",
    features: data.features.map((feature) => ({
      type: "Feature",
      properties: { text: `L${feature.properties.level} ${feature.properties.name}` },
      geometry: {
        type: "Point",
        coordinates: [feature.properties.labelLon, feature.properties.labelLat],
      },
    })),
  }
}

interface LandslideLayerProps {
  data: LandslideLayerData | undefined
  selectedId: string | null
}

/**
 * DS divisions coloured by NBRO's level, with a hatch and an "L2" label so the
 * level is not told by colour alone. The API lists a division only while the
 * report's levels belong on the map, so an empty list draws nothing.
 */
export function LandslideLayer({ data, selectedId }: LandslideLayerProps) {
  const map = useMap()
  const shown = data ?? emptyData
  const labels = useMemo(() => labelCollection(shown), [shown])
  const current = useRef({ shown, labels, selectedId })

  useEffect(() => {
    if (!map) return
    const add = () => {
      addLayers(map, current.current.shown, current.current.labels)
      if (current.current.selectedId) {
        map.setFeatureState({ source: SOURCE, id: current.current.selectedId }, { selected: true })
      }
    }
    add()
    map.on("style.load", add)
    return () => {
      map.off("style.load", add)
      removeLayers(map)
    }
  }, [map])

  useEffect(() => {
    current.current = { ...current.current, shown, labels }
    const source = map?.getSource(SOURCE) as GeoJSONSource | undefined
    source?.setData(shown as unknown as FeatureCollection)
    const labelSource = map?.getSource(LABEL_SOURCE) as GeoJSONSource | undefined
    labelSource?.setData(labels)
  }, [map, shown, labels])

  useEffect(() => {
    const previous = current.current.selectedId
    current.current = { ...current.current, selectedId }
    if (!map || !map.getSource(SOURCE)) return
    if (previous) map.setFeatureState({ source: SOURCE, id: previous }, { selected: false })
    if (selectedId) map.setFeatureState({ source: SOURCE, id: selectedId }, { selected: true })
  }, [map, selectedId])

  return null
}
