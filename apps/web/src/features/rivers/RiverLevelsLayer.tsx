import type {
  ExpressionSpecification,
  GeoJSONSource,
  MapLayerMouseEvent,
  Map as MapLibreMap,
} from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useRef } from "react"
import { useMap } from "../map/MapContext"
import { statusColors } from "./present"
import type { RiverLevelsCollection } from "./types"

const SOURCE = "river-levels"
const HALO = "river-levels-rising"
const CIRCLES = "river-levels-circles"
const LABELS = "river-levels-labels"

const empty: RiverLevelsCollection = {
  type: "FeatureCollection",
  features: [],
  meta: { source: "", attribution: "", fetchedAt: null, generatedAt: "" },
}

const statusColor: ExpressionSpecification = [
  "match",
  ["get", "status"],
  "alert",
  statusColors.alert,
  "minor_flood",
  statusColors.minor_flood,
  "major_flood",
  statusColors.major_flood,
  "unknown",
  statusColors.unknown,
  statusColors.normal,
]

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
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      promoteId: "id",
      attribution: "River levels: Irrigation Department of Sri Lanka",
    })
  }
  if (!map.getLayer(HALO)) {
    map.addLayer({
      id: HALO,
      type: "circle",
      source: SOURCE,
      filter: ["all", ["==", ["get", "trend"], "rising"], ["!", ["get", "stale"]]],
      paint: {
        "circle-radius": ["interpolate", ["linear"], ["zoom"], 6, 10, 10, 16],
        "circle-color": statusColor,
        "circle-opacity": 0.28,
      },
    })
  }
  if (!map.getLayer(CIRCLES)) {
    map.addLayer({
      id: CIRCLES,
      type: "circle",
      source: SOURCE,
      layout: { "circle-sort-key": severity },
      paint: {
        "circle-radius": ["interpolate", ["linear"], ["zoom"], 6, 5, 10, 9],
        "circle-color": statusColor,
        "circle-opacity": ["case", ["get", "stale"], 0.4, 1],
        "circle-stroke-width": ["case", ["boolean", ["feature-state", "selected"], false], 3, 1.5],
        "circle-stroke-color": [
          "case",
          ["boolean", ["feature-state", "selected"], false],
          "#0f172a",
          "#ffffff",
        ],
      },
    })
  }
  if (!map.getLayer(LABELS)) {
    map.addLayer({
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
      paint: { "text-color": "#0f172a", "text-halo-color": "#ffffff", "text-halo-width": 1.5 },
    })
  }
}

function removeLayers(map: MapLibreMap) {
  // The map may already be torn down or mid style change.
  if (!map.getStyle()) return
  for (const id of [LABELS, CIRCLES, HALO]) if (map.getLayer(id)) map.removeLayer(id)
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface RiverLevelsLayerProps {
  data: RiverLevelsCollection | undefined
  selectedId: string | null
  onSelect: (id: string | null) => void
}

/** River gauging stations, coloured by flood status, with a halo when rising. */
export function RiverLevelsLayer({ data, selectedId, onSelect }: RiverLevelsLayerProps) {
  const map = useMap()
  const dataRef = useRef(data ?? empty)
  const selectedRef = useRef(selectedId)
  const onSelectRef = useRef(onSelect)

  useEffect(() => {
    onSelectRef.current = onSelect
  }, [onSelect])

  useEffect(() => {
    if (!map) return
    const add = () => {
      addLayers(map, dataRef.current)
      if (selectedRef.current) {
        map.setFeatureState({ source: SOURCE, id: selectedRef.current }, { selected: true })
      }
    }
    const click = (event: MapLayerMouseEvent) => {
      const id = event.features?.[0]?.properties?.id
      if (typeof id === "string") onSelectRef.current(id)
    }
    const enter = () => (map.getCanvas().style.cursor = "pointer")
    const leave = () => (map.getCanvas().style.cursor = "")

    add()
    map.on("style.load", add)
    map.on("click", CIRCLES, click)
    map.on("mouseenter", CIRCLES, enter)
    map.on("mouseleave", CIRCLES, leave)
    return () => {
      map.off("style.load", add)
      map.off("click", CIRCLES, click)
      map.off("mouseenter", CIRCLES, enter)
      map.off("mouseleave", CIRCLES, leave)
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
