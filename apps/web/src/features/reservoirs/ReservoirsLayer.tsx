import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useMemo, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { polygonSdf } from "../map/sdf"
import { slotBefore, styleReady } from "../map/slots"
import { reservoirShapes, reservoirTokens } from "./present"
import type { ReservoirAreas, ReservoirFeature, ReservoirsCollection } from "./types"

const POINTS_SOURCE = "reservoirs"
const AREAS_SOURCE = "reservoir-areas"
/** The markers: what clicks and hovers hit first. */
export const RESERVOIR_POINTS = "reservoirs-points"
/** The water bodies: clicks and hovers land on these otherwise. */
export const RESERVOIR_AREAS = "reservoirs-areas"
const AREA_LINES = "reservoirs-area-lines"
const LABELS_ACTIVE = "reservoirs-labels-active"
const LABELS_QUIET = "reservoirs-labels-quiet"

const DROP = "reservoir-drop"
const TRIANGLE = "reservoir-triangle"
const SQUARE = "reservoir-square"

/** Icons drawn at twice their size for sharp edges on any screen. */
const ICON_RATIO = 2
const ICON_SIZE = 40

const emptyPoints: FeatureCollection = { type: "FeatureCollection", features: [] }

function addIcons(map: MapLibreMap) {
  const images: [string, [number, number][]][] = [
    [DROP, reservoirShapes.spilling],
    [TRIANGLE, reservoirShapes.releasing],
    [SQUARE, reservoirShapes.holding],
  ]
  for (const [name, shape] of images) {
    if (!map.hasImage(name)) {
      map.addImage(name, polygonSdf(ICON_SIZE, shape), { sdf: true, pixelRatio: ICON_RATIO })
    }
  }
}

function readPalette() {
  return readTokens({
    ...reservoirTokens,
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
    "spilling",
    palette.spilling,
    "releasing",
    palette.releasing,
    "holding",
    palette.holding,
    palette.nodata,
  ]
}

/** Draws reservoirs needing attention above the others. */
const attention: ExpressionSpecification = [
  "match",
  ["get", "status"],
  "spilling",
  3,
  "releasing",
  2,
  "holding",
  1,
  0,
]

const isActive: ExpressionSpecification = [
  "match",
  ["get", "status"],
  ["spilling", "releasing"],
  ["!", ["get", "stale"]],
  false,
]

const selected: ExpressionSpecification = ["boolean", ["feature-state", "selected"], false]

function addLayers(map: MapLibreMap, points: FeatureCollection, areas: FeatureCollection) {
  const palette = readPalette()
  addIcons(map)
  if (!map.getSource(AREAS_SOURCE)) {
    map.addSource(AREAS_SOURCE, { type: "geojson", data: areas, promoteId: "id" })
  }
  if (!map.getSource(POINTS_SOURCE)) {
    map.addSource(POINTS_SOURCE, {
      type: "geojson",
      data: points,
      promoteId: "id",
      attribution: "Reservoirs: Irrigation Dept, Mahaweli Authority via waterlevels.lk",
    })
  }
  if (!map.getLayer(RESERVOIR_AREAS)) {
    map.addLayer(
      {
        id: RESERVOIR_AREAS,
        type: "fill",
        source: AREAS_SOURCE,
        paint: {
          "fill-color": statusColor(palette),
          "fill-opacity": [
            "case",
            selected,
            0.6,
            ["get", "stale"],
            0.12,
            ["==", ["get", "status"], "nodata"],
            0.12,
            ["==", ["get", "status"], "holding"],
            0.3,
            0.55,
          ],
        },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(AREA_LINES)) {
    map.addLayer(
      {
        id: AREA_LINES,
        type: "line",
        source: AREAS_SOURCE,
        paint: {
          "line-color": ["case", selected, palette.strokeSelected, statusColor(palette)],
          "line-width": ["case", selected, 2.5, 1],
          "line-opacity": ["case", ["get", "stale"], 0.5, 0.9],
        },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(RESERVOIR_POINTS)) {
    map.addLayer(
      {
        id: RESERVOIR_POINTS,
        type: "symbol",
        source: POINTS_SOURCE,
        layout: {
          "symbol-sort-key": attention,
          "icon-image": [
            "match",
            ["get", "status"],
            "spilling",
            DROP,
            "releasing",
            TRIANGLE,
            SQUARE,
          ],
          // Quiet reservoirs are small dots until the map is close enough to
          // have room for them; those needing attention are always clear.
          "icon-size": [
            "interpolate",
            ["linear"],
            ["zoom"],
            6,
            ["case", isActive, 0.62, 0.2],
            8,
            ["case", isActive, 0.72, 0.32],
            11,
            ["case", isActive, 0.95, 0.55],
          ],
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
        },
        paint: {
          // A square with no fill leaves only its outline: no data, or out of date.
          "icon-color": [
            "case",
            ["any", ["get", "stale"], ["==", ["get", "status"], "nodata"]],
            "rgba(0, 0, 0, 0)",
            statusColor(palette),
          ],
          "icon-halo-color": [
            "case",
            selected,
            palette.strokeSelected,
            ["any", ["get", "stale"], ["==", ["get", "status"], "nodata"]],
            statusColor(palette),
            palette.stroke,
          ],
          "icon-halo-width": ["case", selected, 3, 1.25],
        },
      },
      slotBefore(map, "points"),
    )
  }
  const label = (id: string, minzoom: number, filter: ExpressionSpecification) => {
    if (map.getLayer(id)) return
    map.addLayer(
      {
        id,
        type: "symbol",
        source: POINTS_SOURCE,
        minzoom,
        filter,
        layout: {
          "text-field": ["get", "name"],
          "text-font": ["Noto Sans Regular"],
          "text-size": 12,
          "text-offset": [0, 0.9],
          "text-anchor": "top",
          "text-optional": true,
          "symbol-sort-key": ["-", 0, attention],
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
  label(LABELS_ACTIVE, 7.6, isActive)
  label(LABELS_QUIET, 9.2, ["!", isActive])
}

function removeLayers(map: MapLibreMap) {
  // The map may already be torn down or mid style change.
  if (!styleReady(map)) return
  for (const id of [LABELS_QUIET, LABELS_ACTIVE, RESERVOIR_POINTS, AREA_LINES, RESERVOIR_AREAS]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  for (const id of [POINTS_SOURCE, AREAS_SOURCE]) if (map.getSource(id)) map.removeSource(id)
}

/** Only reservoirs with a position reach the map. */
function mappedPoints(data: ReservoirsCollection | undefined): FeatureCollection {
  if (!data) return emptyPoints
  return {
    type: "FeatureCollection",
    features: data.features.filter(
      (feature: ReservoirFeature) => feature.geometry !== null,
    ) as unknown as FeatureCollection["features"],
  }
}

interface ReservoirsLayerProps {
  data: ReservoirsCollection | undefined
  areas: ReservoirAreas | undefined
  selectedId: string | null
}

/**
 * Reservoirs with a known place: their water bodies coloured by status, under
 * labels, and a marker whose shape says the status too. Reservoirs not matched
 * to the map are in the list only. Clicks go through the map's click handler.
 */
export function ReservoirsLayer({ data, areas, selectedId }: ReservoirsLayerProps) {
  const map = useMap()
  const points = useMemo(() => mappedPoints(data), [data])
  const polygons = useMemo(() => (areas ?? emptyPoints) as unknown as FeatureCollection, [areas])
  const latest = useRef({ points, polygons, selectedId })

  useEffect(() => {
    if (!map) return
    const add = () => {
      addLayers(map, latest.current.points, latest.current.polygons)
      const id = latest.current.selectedId
      if (id) {
        map.setFeatureState({ source: POINTS_SOURCE, id }, { selected: true })
        map.setFeatureState({ source: AREAS_SOURCE, id }, { selected: true })
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
    latest.current.points = points
    ;(map?.getSource(POINTS_SOURCE) as GeoJSONSource | undefined)?.setData(points)
  }, [map, points])

  useEffect(() => {
    latest.current.polygons = polygons
    ;(map?.getSource(AREAS_SOURCE) as GeoJSONSource | undefined)?.setData(polygons)
  }, [map, polygons])

  useEffect(() => {
    const previous = latest.current.selectedId
    latest.current.selectedId = selectedId
    if (!map || !map.getSource(POINTS_SOURCE)) return
    for (const source of [POINTS_SOURCE, AREAS_SOURCE]) {
      if (previous) map.setFeatureState({ source, id: previous }, { selected: false })
      if (selectedId) map.setFeatureState({ source, id: selectedId }, { selected: true })
    }
  }, [map, selectedId])

  return null
}
