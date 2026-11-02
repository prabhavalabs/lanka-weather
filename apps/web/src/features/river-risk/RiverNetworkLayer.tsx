import type { ExpressionSpecification, Map as MapLibreMap } from "maplibre-gl"
import { useEffect, useRef } from "react"
import type { Layer } from "@/lib/layers"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { polygonSdf, type Point } from "../map/sdf"
import { slotBefore, styleReady } from "../map/slots"
import { reachIdsByLevel, riskExtraWidth, riskTokens } from "./present"
import type { RiskLevel, RiverRisk } from "./types"

const SOURCE = "rivers"
const STREAMS = "rivers-streams"
const CASING = "rivers-casing"
const GAUGED = "rivers-gauged"
const UNGAUGED = "rivers-ungauged"
const HIGHLIGHT = "rivers-highlight"
const MARKS = "rivers-marks"
const LABELS = "rivers-labels"
/** What clicks and hovers hit, topmost first. */
export const RIVER_LINES = [GAUGED, UNGAUGED, STREAMS]

const MARK_WATCH = "river-mark-watch"
const MARK_HIGH = "river-mark-high"
const MARK_SEVERE = "river-mark-severe"

/** Line width by the tile's width class (upstream area), at zoom 6 and zoom 11. */
const classWidths: Record<number, [number, number]> = {
  1: [0.7, 1.6],
  2: [1, 2.2],
  3: [1.5, 3],
  4: [2.2, 4.2],
  5: [3, 5.6],
}

function byClass(index: 0 | 1): ExpressionSpecification {
  return [
    "match",
    ["get", "width"],
    1,
    classWidths[1]![index],
    2,
    classWidths[2]![index],
    3,
    classWidths[3]![index],
    4,
    classWidths[4]![index],
    classWidths[5]![index],
  ]
}

/** A value per level, read from the stretch's id. */
function byLevel(
  ids: Record<RiskLevel, number[]>,
  values: Record<RiskLevel, string | number | ExpressionSpecification>,
): ExpressionSpecification {
  const expression: unknown[] = ["match", ["id"]]
  for (const level of ["severe", "high", "watch", "normal"] as const) {
    if (ids[level].length > 0) expression.push(ids[level], values[level])
  }
  expression.push(values.unknown)
  // A match needs at least one label; with no stretches yet, everything is unknown.
  return (expression.length > 3 ? expression : values.unknown) as ExpressionSpecification
}

/**
 * A stretch's width at any zoom: by size, plus more the higher its risk, plus
 * `outset` for a casing or highlight drawn under it. Zoom must stay the outer
 * expression, so the outset goes inside each stop.
 */
function width(ids: Record<RiskLevel, number[]>, outset = 0): ExpressionSpecification {
  const at = (index: 0 | 1, scale: number) =>
    byLevel(ids, {
      severe: ["+", byClass(index), riskExtraWidth.severe * scale + outset],
      high: ["+", byClass(index), riskExtraWidth.high * scale + outset],
      watch: ["+", byClass(index), riskExtraWidth.watch * scale + outset],
      normal: ["+", byClass(index), riskExtraWidth.normal * scale + outset],
      unknown: ["+", byClass(index), riskExtraWidth.unknown * scale + outset],
    })
  return ["interpolate", ["linear"], ["zoom"], 6, at(0, 1), 11, at(1, 1.6)]
}

function addIcons(map: MapLibreMap) {
  const size = 32
  const half = size / 2
  const dot: Point[] = Array.from({ length: 16 }, (_, index) => {
    const angle = (index / 16) * Math.PI * 2
    return [half + Math.cos(angle) * 5, half + Math.sin(angle) * 5] as Point
  })
  // Pointing along the river: a triangle for high, a double chevron for severe.
  const triangle: Point[] = [
    [half + 8, half],
    [half - 6, half - 7],
    [half - 6, half + 7],
  ]
  const chevrons: Point[] = [
    [half - 9, half - 8],
    [half - 1, half],
    [half - 9, half + 8],
    [half - 5, half + 8],
    [half + 3, half],
    [half - 5, half - 8],
  ]
  const images: [string, Point[]][] = [
    [MARK_WATCH, dot],
    [MARK_HIGH, triangle],
    [MARK_SEVERE, chevrons],
  ]
  for (const [name, shape] of images) {
    if (!map.hasImage(name)) {
      map.addImage(name, polygonSdf(size, shape), { sdf: true, pixelRatio: 2 })
    }
  }
}

function absolute(path: string) {
  return /^https?:/.test(path) ? path : `${window.location.origin}${path}`
}

interface LayerState {
  version: string
  ids: Record<RiskLevel, number[]>
  selected: string | null
}

function colours() {
  return readTokens({
    ...riskTokens,
    stream: "--river-stream",
    casing: "--river-casing",
    label: "--river-label",
    halo: "--map-label-halo",
    selected: "--status-stroke-selected",
    stroke: "--status-stroke",
  })
}

function riskIds(ids: Record<RiskLevel, number[]>) {
  return [...ids.severe, ...ids.high, ...ids.watch]
}

function addLayers(map: MapLibreMap, layer: Layer, state: LayerState) {
  if (!state.version) return
  const tokens = colours()
  addIcons(map)
  const before = slotBefore(map, "rivers")
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "vector",
      // The version changes whenever a rebuild changes stretch ids, so tiles
      // and the risk document always agree.
      tiles: [
        absolute(
          `${layer.tilePath ?? "/v1/tiles/rivers/{z}/{x}/{y}.pbf"}?v=${encodeURIComponent(state.version)}`,
        ),
      ],
      minzoom: layer.minZoom ?? 0,
      maxzoom: layer.maxZoom ?? 14,
      attribution: "Rivers: HydroRIVERS (HydroSHEDS), Irrigation Department of Sri Lanka",
    })
  }
  if (!map.getLayer(STREAMS)) {
    map.addLayer(
      {
        id: STREAMS,
        type: "line",
        source: SOURCE,
        "source-layer": "streams",
        minzoom: 7,
        layout: { "line-join": "round", "line-cap": "round" },
        paint: {
          "line-color": tokens.stream,
          "line-width": ["interpolate", ["linear"], ["zoom"], 7, 0.5, 12, 1.4],
        },
      },
      before,
    )
  }
  if (!map.getLayer(CASING)) {
    map.addLayer(
      {
        id: CASING,
        type: "line",
        source: SOURCE,
        "source-layer": "reaches",
        // Only what a gauge measures is lifted off the map: ungauged stretches stay quiet.
        filter: gaugedFilter(state.ids),
        layout: { "line-join": "round", "line-cap": "round" },
        paint: {
          "line-color": tokens.casing,
          "line-opacity": 0.85,
          "line-width": width(state.ids, 2),
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      before,
    )
  }
  if (!map.getLayer(HIGHLIGHT)) {
    map.addLayer(
      {
        id: HIGHLIGHT,
        type: "line",
        source: SOURCE,
        "source-layer": "reaches",
        filter: ["==", ["get", "river"], state.selected ?? ""],
        layout: { "line-join": "round", "line-cap": "round" },
        paint: {
          "line-color": tokens.selected,
          "line-width": width(state.ids, 5),
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      before,
    )
  }
  // A stretch nothing measures is dashed, so it can never read as measured and
  // calm. Dashes cannot vary per feature, so this is a layer of its own.
  if (!map.getLayer(UNGAUGED)) {
    map.addLayer(
      {
        id: UNGAUGED,
        type: "line",
        source: SOURCE,
        "source-layer": "reaches",
        filter: ungaugedFilter(state.ids),
        layout: { "line-join": "round" },
        paint: {
          "line-color": tokens.unknown,
          "line-width": width(state.ids),
          "line-opacity": 0.8,
          "line-dasharray": [2.5, 2.5],
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      before,
    )
  }
  if (!map.getLayer(GAUGED)) {
    map.addLayer(
      {
        id: GAUGED,
        type: "line",
        source: SOURCE,
        "source-layer": "reaches",
        filter: gaugedFilter(state.ids),
        layout: { "line-join": "round", "line-cap": "round" },
        paint: {
          "line-color": byLevel(state.ids, {
            severe: tokens.severe,
            high: tokens.high,
            watch: tokens.watch,
            normal: tokens.normal,
            unknown: tokens.unknown,
          }),
          "line-width": width(state.ids),
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      before,
    )
  }
  if (!map.getLayer(MARKS)) {
    map.addLayer(
      {
        id: MARKS,
        type: "symbol",
        source: SOURCE,
        "source-layer": "reaches",
        minzoom: 6.5,
        filter: marksFilter(state.ids),
        layout: {
          "symbol-placement": "line",
          "symbol-spacing": 70,
          "icon-image": byLevel(state.ids, {
            severe: MARK_SEVERE,
            high: MARK_HIGH,
            watch: MARK_WATCH,
            normal: MARK_WATCH,
            unknown: MARK_WATCH,
          }),
          "icon-size": ["interpolate", ["linear"], ["zoom"], 6, 0.65, 11, 1],
          "icon-rotation-alignment": "map",
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
        },
        paint: {
          "icon-color": tokens.casing,
          "icon-halo-color": tokens.stroke,
          "icon-halo-width": 0.6,
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      before,
    )
  }
  if (!map.getLayer(LABELS)) {
    map.addLayer(
      {
        id: LABELS,
        type: "symbol",
        source: SOURCE,
        "source-layer": "labels",
        minzoom: 6.5,
        // The largest rivers at country scale; the rest as they fit.
        filter: [
          "step",
          ["zoom"],
          ["all", ["get", "main"], [">=", ["get", "upland"], 900]],
          8,
          ["any", ["get", "main"], [">=", ["get", "upland"], 300]],
          9.5,
          true,
        ],
        layout: {
          "symbol-placement": "line",
          "symbol-spacing": 280,
          "text-field": ["get", "name"],
          "text-font": ["Noto Sans Italic"],
          "text-size": ["interpolate", ["linear"], ["zoom"], 7, 11, 11, 14],
          "text-letter-spacing": 0.04,
          "text-max-angle": 60,
          "text-padding": 8,
          "text-optional": true,
        },
        paint: {
          "text-color": tokens.label,
          "text-halo-color": tokens.halo,
          "text-halo-width": 1.6,
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      slotBefore(map, "points"),
    )
  }
}

function ungaugedFilter(ids: Record<RiskLevel, number[]>): ExpressionSpecification {
  return ["in", ["id"], ["literal", ids.unknown]]
}

function gaugedFilter(ids: Record<RiskLevel, number[]>): ExpressionSpecification {
  return ["in", ["id"], ["literal", [...ids.normal, ...riskIds(ids)]]]
}

function marksFilter(ids: Record<RiskLevel, number[]>): ExpressionSpecification {
  return ["in", ["id"], ["literal", riskIds(ids)]]
}

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of [LABELS, MARKS, GAUGED, UNGAUGED, HIGHLIGHT, CASING, STREAMS]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

function applyRisk(map: MapLibreMap, state: LayerState) {
  if (!map.getLayer(GAUGED)) return
  const tokens = colours()
  map.setFilter(GAUGED, gaugedFilter(state.ids))
  map.setFilter(CASING, gaugedFilter(state.ids))
  map.setFilter(UNGAUGED, ungaugedFilter(state.ids))
  map.setFilter(MARKS, marksFilter(state.ids))
  map.setPaintProperty(
    GAUGED,
    "line-color",
    byLevel(state.ids, {
      severe: tokens.severe,
      high: tokens.high,
      watch: tokens.watch,
      normal: tokens.normal,
      unknown: tokens.unknown,
    }),
  )
  map.setPaintProperty(GAUGED, "line-width", width(state.ids))
  map.setPaintProperty(UNGAUGED, "line-width", width(state.ids))
  map.setPaintProperty(CASING, "line-width", width(state.ids, 2))
  map.setPaintProperty(HIGHLIGHT, "line-width", width(state.ids, 5))
  map.setLayoutProperty(
    MARKS,
    "icon-image",
    byLevel(state.ids, {
      severe: MARK_SEVERE,
      high: MARK_HIGH,
      watch: MARK_WATCH,
      normal: MARK_WATCH,
      unknown: MARK_WATCH,
    }),
  )
}

interface RiverNetworkLayerProps {
  /** The `rivers` catalogue entry: tile path and zooms. */
  layer: Layer
  /** The rivers and their estimated risk; drawn once these have arrived. */
  rivers: RiverRisk[] | undefined
  /** Changes when a rebuild changes stretch ids. */
  version: string | undefined
  selectedId: string | null
}

/**
 * The river network from vector tiles: named rivers always, more tributaries
 * as you zoom in, wider downstream, and each stretch coloured by its estimated
 * risk. Risk also shows without colour: stretches at risk are wider and carry
 * marks (dots, triangles, chevrons), and a stretch with no gauge is dashed.
 * Colours are kept in expressions on stretch ids, so new readings recolour the
 * map without reloading tiles.
 */
export function RiverNetworkLayer({ layer, rivers, version, selectedId }: RiverNetworkLayerProps) {
  const map = useMap()
  const state = useRef<LayerState>({
    version: version ?? "",
    ids: reachIdsByLevel(rivers ?? []),
    selected: selectedId,
  })
  const layerRef = useRef(layer)

  useEffect(() => {
    if (!map) return
    const add = () => addLayers(map, layerRef.current, state.current)
    add()
    map.on("style.load", add)
    return () => {
      map.off("style.load", add)
      removeLayers(map)
    }
  }, [map])

  // A new network means new tiles; the first version also draws the layer.
  useEffect(() => {
    if (!map || !version || state.current.version === version) return
    state.current = { ...state.current, version }
    removeLayers(map)
    addLayers(map, layerRef.current, state.current)
  }, [map, version])

  useEffect(() => {
    state.current = { ...state.current, ids: reachIdsByLevel(rivers ?? []) }
    if (map && styleReady(map)) applyRisk(map, state.current)
  }, [map, rivers])

  useEffect(() => {
    state.current = { ...state.current, selected: selectedId }
    if (map && map.getLayer(HIGHLIGHT)) {
      map.setFilter(HIGHLIGHT, ["==", ["get", "river"], selectedId ?? ""])
    }
  }, [map, selectedId])

  return null
}
