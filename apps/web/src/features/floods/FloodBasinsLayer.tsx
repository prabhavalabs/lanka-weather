import type { ExpressionSpecification, Map as MapLibreMap } from "maplibre-gl"
import { useEffect, useRef } from "react"
import type { Layer } from "@/lib/layers"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import type { BasinsDocument, BasinState } from "../river-risk/types"
import {
  basinFillOpacity,
  basinIdsByState,
  basinLineOpacity,
  basinTokens,
  subbasinIdsByState,
} from "./basins"

const SOURCE = "flood-basins"
/** The official river basins: what clicks and hovers hit at country scale. */
export const BASIN_FILL = "flood-basin-fill"
/** HydroBASINS sub-basins, shown when zoomed in. */
export const SUBBASIN_FILL = "flood-subbasin-fill"
const BASIN_LINE = "flood-basin-line"
const BASIN_LINE_NONE = "flood-basin-line-none"
const BASIN_SELECTED = "flood-basin-selected"
const SUBBASIN_LINE = "flood-subbasin-line"

/** The zoom where sub-basins take over from whole basins. */
const subbasinZoom = 9

type Ids = Record<BasinState, number[]>

function colours() {
  return readTokens({
    ...basinTokens,
    outline: "--basin-outline",
    selected: "--status-stroke-selected",
  })
}

/** A value per state, read from the feature's id; anything unlisted has no forecast. */
function byState<T>(ids: Ids, values: Record<BasinState, T>): ExpressionSpecification {
  const expression: unknown[] = ["match", ["id"]]
  for (const state of ["extreme", "severe", "above_normal", "no_flooding"] as const) {
    if (ids[state].length > 0) expression.push(ids[state], values[state])
  }
  expression.push(values.no_forecast)
  return (expression.length > 3 ? expression : values.no_forecast) as ExpressionSpecification
}

function fillColor(ids: Ids) {
  const tokens = colours()
  return byState(ids, {
    extreme: tokens.extreme,
    severe: tokens.severe,
    above_normal: tokens.above_normal,
    no_flooding: tokens.no_flooding,
    no_forecast: tokens.no_forecast,
  })
}

function lineColor(ids: Ids) {
  const tokens = colours()
  return byState(ids, {
    extreme: tokens.extreme,
    severe: tokens.severe,
    above_normal: tokens.above_normal,
    no_flooding: tokens.outline,
    no_forecast: tokens.outline,
  })
}

function lineWidth(ids: Ids, low: number, high: number): ExpressionSpecification {
  const stop = (base: number) =>
    byState(ids, {
      extreme: base + 2,
      severe: base + 1.5,
      above_normal: base + 1,
      no_flooding: base,
      no_forecast: base,
    })
  return ["interpolate", ["linear"], ["zoom"], 6, stop(low), 11, stop(high)]
}

function absolute(path: string) {
  return /^https?:/.test(path) ? path : `${window.location.origin}${path}`
}

interface State {
  basins: Ids
  subbasins: Ids
  selected: number | null
}

/** Basins with a forecast get a solid outline, those with none a dashed one. */
function hasForecast(ids: Ids): ExpressionSpecification {
  return [
    "in",
    ["id"],
    ["literal", [...ids.extreme, ...ids.severe, ...ids.above_normal, ...ids.no_flooding]],
  ]
}

function addLayers(map: MapLibreMap, layer: Layer, state: State) {
  const tokens = colours()
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "vector",
      tiles: [absolute(layer.tilePath ?? "/v1/tiles/river-basins/{z}/{x}/{y}.pbf")],
      minzoom: 0,
      maxzoom: layer.maxZoom ?? 14,
      attribution: "Basins: Irrigation Department of Sri Lanka, HydroBASINS (HydroSHEDS)",
    })
  }
  const fills = slotBefore(map, "basins")
  if (!map.getLayer(BASIN_FILL)) {
    map.addLayer(
      {
        id: BASIN_FILL,
        type: "fill",
        source: SOURCE,
        "source-layer": "basins",
        maxzoom: subbasinZoom,
        paint: {
          "fill-color": fillColor(state.basins),
          "fill-opacity": byState(state.basins, basinFillOpacity),
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      fills,
    )
  }
  if (!map.getLayer(SUBBASIN_FILL)) {
    map.addLayer(
      {
        id: SUBBASIN_FILL,
        type: "fill",
        source: SOURCE,
        "source-layer": "subbasins",
        minzoom: subbasinZoom,
        paint: {
          "fill-color": fillColor(state.subbasins),
          "fill-opacity": byState(state.subbasins, basinFillOpacity),
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      fills,
    )
  }
  const lines = slotBefore(map, "lines")
  if (!map.getLayer(SUBBASIN_LINE)) {
    map.addLayer(
      {
        id: SUBBASIN_LINE,
        type: "line",
        source: SOURCE,
        "source-layer": "subbasins",
        minzoom: subbasinZoom,
        layout: { "line-join": "round" },
        paint: {
          "line-color": tokens.outline,
          "line-opacity": 0.3,
          "line-width": ["interpolate", ["linear"], ["zoom"], 9, 0.4, 13, 0.8],
        },
      },
      lines,
    )
  }
  if (!map.getLayer(BASIN_LINE_NONE)) {
    map.addLayer(
      {
        id: BASIN_LINE_NONE,
        type: "line",
        source: SOURCE,
        "source-layer": "basins",
        filter: ["!", hasForecast(state.basins)],
        layout: { "line-join": "round" },
        paint: {
          "line-color": tokens.outline,
          "line-opacity": basinLineOpacity.no_forecast,
          "line-width": ["interpolate", ["linear"], ["zoom"], 6, 0.5, 11, 1],
          "line-dasharray": [2, 2],
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      lines,
    )
  }
  if (!map.getLayer(BASIN_LINE)) {
    map.addLayer(
      {
        id: BASIN_LINE,
        type: "line",
        source: SOURCE,
        "source-layer": "basins",
        filter: hasForecast(state.basins),
        layout: { "line-join": "round" },
        paint: {
          "line-color": lineColor(state.basins),
          "line-opacity": byState(state.basins, basinLineOpacity),
          "line-width": lineWidth(state.basins, 0.5, 1),
          // Dashed, so a basin is never taken for one of Google's solid flood areas.
          "line-dasharray": [3, 2],
        },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      lines,
    )
  }
  if (!map.getLayer(BASIN_SELECTED)) {
    map.addLayer(
      {
        id: BASIN_SELECTED,
        type: "line",
        source: SOURCE,
        "source-layer": "basins",
        filter: ["==", ["id"], state.selected ?? -1],
        layout: { "line-join": "round" },
        paint: { "line-color": tokens.selected, "line-width": 3 },
      } as Parameters<MapLibreMap["addLayer"]>[0],
      lines,
    )
  }
}

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of [
    BASIN_SELECTED,
    BASIN_LINE,
    BASIN_LINE_NONE,
    SUBBASIN_LINE,
    SUBBASIN_FILL,
    BASIN_FILL,
  ]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

function applyStates(map: MapLibreMap, state: State) {
  if (!map.getLayer(BASIN_FILL)) return
  map.setPaintProperty(BASIN_FILL, "fill-color", fillColor(state.basins))
  map.setPaintProperty(BASIN_FILL, "fill-opacity", byState(state.basins, basinFillOpacity))
  map.setPaintProperty(SUBBASIN_FILL, "fill-color", fillColor(state.subbasins))
  map.setPaintProperty(SUBBASIN_FILL, "fill-opacity", byState(state.subbasins, basinFillOpacity))
  map.setFilter(BASIN_LINE, hasForecast(state.basins))
  map.setFilter(BASIN_LINE_NONE, ["!", hasForecast(state.basins)])
  map.setPaintProperty(BASIN_LINE, "line-color", lineColor(state.basins))
  map.setPaintProperty(BASIN_LINE, "line-opacity", byState(state.basins, basinLineOpacity))
  map.setPaintProperty(BASIN_LINE, "line-width", lineWidth(state.basins, 0.5, 1))
}

interface FloodBasinsLayerProps {
  /** The `flood-forecast` catalogue entry, for the tile path. */
  layer: Layer
  data: BasinsDocument | undefined
  selectedFid: number | null
}

/**
 * The sections of the country a flood forecast applies to, always shown while
 * the flood layer is on: official river basins, and finer sub-basins from zoom 9,
 * each outlined and tinted by the worst current forecast or gauge in it. Calm basins
 * have barely a tint and a thin solid outline, so a calm country looks like Flood
 * Hub's. A basin nothing covers has no tint and a dashed outline: nothing known,
 * which is not the same as calm. Tints only grow for a warning or above.
 */
export function FloodBasinsLayer({ layer, data, selectedFid }: FloodBasinsLayerProps) {
  const map = useMap()
  const state = useRef<State>({
    basins: basinIdsByState(data?.basins ?? []),
    subbasins: subbasinIdsByState(data?.subbasins ?? []),
    selected: selectedFid,
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

  useEffect(() => {
    state.current = {
      ...state.current,
      basins: basinIdsByState(data?.basins ?? []),
      subbasins: subbasinIdsByState(data?.subbasins ?? []),
    }
    if (map && styleReady(map)) applyStates(map, state.current)
  }, [map, data])

  useEffect(() => {
    state.current = { ...state.current, selected: selectedFid }
    if (map && map.getLayer(BASIN_SELECTED)) {
      map.setFilter(BASIN_SELECTED, ["==", ["id"], selectedFid ?? -1])
    }
  }, [map, selectedFid])

  return null
}
