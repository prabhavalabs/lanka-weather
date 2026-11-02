import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import { statusTokens } from "./present"
import type { RiverLevelsCollection } from "./types"

const SOURCE = "river-levels"
const HALO = "river-levels-rising"
/** The stations: what clicks and hovers hit. */
export const RIVER_STATIONS = "river-levels-circles"
const CIRCLES = RIVER_STATIONS
const LABELS = "river-levels-labels"

const empty: RiverLevelsCollection = {
  type: "FeatureCollection",
  features: [],
  meta: { source: "", attribution: "", fetchedAt: null, generatedAt: "" },
}

/** Theme colours for the layer, read when the layers are added (on every style load). */
function readPalette() {
  return readTokens({
    ...statusTokens,
    stroke: "--status-stroke",
    strokeSelected: "--status-stroke-selected",
    label: "--map-label-strong",
    halo: "--map-label-halo",
  })
}

type Palette = ReturnType<typeof readPalette>

function statusColor(palette: Palette): ExpressionSpecification {
  return [
    "match",
    ["get", "status"],
    "alert",
    palette.alert,
    "minor_flood",
    palette.minor_flood,
    "major_flood",
    palette.major_flood,
    "unknown",
    palette.unknown,
    palette.normal,
  ]
}

/** Draws severe stations above calm ones. */
const severity: ExpressionSpecification = [
  "match",
  ["get", "status"],
  "major_flood",
  4,
  "minor_flood",
  3,
  "alert",
  2,
  "normal",
  1,
  0,
]

function addLayers(map: MapLibreMap, data: RiverLevelsCollection) {
  const palette = readPalette()
  const selected: ExpressionSpecification = ["boolean", ["feature-state", "selected"], false]
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      promoteId: "id",
      attribution: "River levels: Irrigation Dept; RiverNet (DMC) via waterlevels.lk",
    })
  }
  if (!map.getLayer(HALO)) {
    map.addLayer(
      {
        id: HALO,
        type: "circle",
        source: SOURCE,
        filter: ["all", ["==", ["get", "trend"], "rising"], ["!", ["get", "stale"]]],
        paint: {
          "circle-radius": ["interpolate", ["linear"], ["zoom"], 6, 10, 10, 16],
          "circle-color": statusColor(palette),
          "circle-opacity": 0.3,
        },
      },
      slotBefore(map, "points"),
    )
  }
  if (!map.getLayer(CIRCLES)) {
    map.addLayer(
      {
        id: CIRCLES,
        type: "circle",
        source: SOURCE,
        layout: { "circle-sort-key": severity },
        paint: {
          "circle-radius": [
            "interpolate",
            ["linear"],
            ["zoom"],
            6,
            ["case", selected, 7, 5],
            10,
            ["case", selected, 11, 9],
          ],
          "circle-color": statusColor(palette),
          "circle-opacity": ["case", ["get", "stale"], 0.4, 1],
          "circle-stroke-width": ["case", selected, 3, 1.25],
          "circle-stroke-color": ["case", selected, palette.strokeSelected, palette.stroke],
        },
      },
      slotBefore(map, "points"),
    )
  }
  if (!map.getLayer(LABELS)) {
    map.addLayer(
      {
        id: LABELS,
        type: "symbol",
        source: SOURCE,
        minzoom: 8.5,
        layout: {
          "text-field": ["get", "name"],
          "text-font": ["Noto Sans Regular"],
          "text-size": 12,
          "text-offset": [0, 1.2],
          "text-anchor": "top",
          "text-optional": true,
        },
        paint: {
          "text-color": palette.label,
          "text-halo-color": palette.halo,
          "text-halo-width": 1.5,
        },
      },
      slotBefore(map, "points"),
    )
  }
}

function removeLayers(map: MapLibreMap) {
  // The map may already be torn down or mid style change.
  if (!styleReady(map)) return
  for (const id of [LABELS, CIRCLES, HALO]) if (map.getLayer(id)) map.removeLayer(id)
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface RiverLevelsLayerProps {
  data: RiverLevelsCollection | undefined
  selectedId: string | null
}

/**
 * River gauging stations, coloured by flood status, with a halo when rising.
 * Clicks are handled by the map's click handler (App), which knows which
 * layer is on top.
 */
export function RiverLevelsLayer({ data, selectedId }: RiverLevelsLayerProps) {
  const map = useMap()
  const dataRef = useRef(data ?? empty)
  const selectedRef = useRef(selectedId)

  useEffect(() => {
    if (!map) return
    const add = () => {
      addLayers(map, dataRef.current)
      if (selectedRef.current) {
        map.setFeatureState({ source: SOURCE, id: selectedRef.current }, { selected: true })
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
    dataRef.current = data ?? empty
    const source = map?.getSource(SOURCE) as GeoJSONSource | undefined
    source?.setData(dataRef.current as unknown as FeatureCollection)
  }, [map, data])

  useEffect(() => {
    const previous = selectedRef.current
    selectedRef.current = selectedId
    if (!map || !map.getSource(SOURCE)) return
    if (previous) map.setFeatureState({ source: SOURCE, id: previous }, { selected: false })
    if (selectedId) map.setFeatureState({ source: SOURCE, id: selectedId }, { selected: true })
  }, [map, selectedId])

  return null
}
