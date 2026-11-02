import type { ExpressionSpecification, Map as MapLibreMap } from "maplibre-gl"
import { useEffect, useRef } from "react"
import { stepExpression } from "@/lib/colour"
import type { Layer } from "@/lib/layers"
import { readToken, readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import { districtsOf, levelMinZoom, regionFid, regionLevels, regionVariables } from "./present"
import type { RegionVariable } from "./present"
import type { RegionLevel, RegionWeather } from "./types"

export const REGIONS_SOURCE = "regions"
/** The fill of the chosen level: coloured by weather, and what clicks and hovers hit. */
export const REGIONS_FILL = "regions-fill"
const HIGHLIGHT = "regions-highlight"
const lineId = (level: RegionLevel) => `regions-line-${level}`

/** Boundary widths by level at zoom 6 and 12: coarser levels draw heavier. */
const lineWidths: Record<RegionLevel, [number, number]> = {
  province: [1.4, 2.4],
  district: [0.9, 1.7],
  ds: [0.6, 1.2],
  gn: [0.4, 0.9],
}

const transparent = "rgba(0, 0, 0, 0)"
const hasValue: ExpressionSpecification = ["==", ["typeof", ["feature-state", "value"]], "number"]
const hovered: ExpressionSpecification = ["boolean", ["feature-state", "hover"], false]
const selected: ExpressionSpecification = ["boolean", ["feature-state", "selected"], false]

function absolute(path: string) {
  return /^https?:/.test(path) ? path : `${window.location.origin}${path}`
}

function fillColor(variable: RegionVariable | null): ExpressionSpecification | string {
  if (!variable) return transparent
  const scale = regionVariables[variable].scale
  const colours = scale.map((step) => readToken(step.token))
  return [
    "case",
    hasValue,
    stepExpression(["feature-state", "value"], scale, colours, transparent),
    transparent,
  ] as unknown as ExpressionSpecification
}

interface RegionsLayerProps {
  /** The `admin-regions` catalogue entry: tile path and zooms. */
  layer: Layer
  level: RegionLevel
  /** What the fill shows, or null for boundaries only. */
  variable: RegionVariable | null
  /** Weather of the level's regions (for GN, of the districts loaded). */
  weather: RegionWeather[]
  selectedId: string | null
  /** The districts whose GN divisions are on screen, when the level is GN. */
  onVisibleDistricts?: (districts: string[]) => void
}

interface State {
  level: RegionLevel
  variable: RegionVariable | null
  values: Map<number, number | null>
  selected: number | null
}

function addLayers(map: MapLibreMap, layer: Layer, state: State) {
  const tokens = readTokens({
    boundary: "--map-boundary",
    label: "--map-label-strong",
    brand: "--status-stroke-selected",
  })
  if (!map.getSource(REGIONS_SOURCE)) {
    map.addSource(REGIONS_SOURCE, {
      type: "vector",
      // MapLibre fetches tiles from its worker, which needs absolute URLs.
      tiles: [absolute(layer.tilePath ?? "/v1/tiles/regions/{z}/{x}/{y}.pbf")],
      minzoom: layer.minZoom ?? 0,
      maxzoom: layer.maxZoom ?? 14,
      attribution: "Boundaries: Survey Department of Sri Lanka, DCS (OCHA HDX, CC BY-IGO)",
    })
  }
  if (!map.getLayer(REGIONS_FILL)) {
    map.addLayer(
      {
        id: REGIONS_FILL,
        type: "fill",
        source: REGIONS_SOURCE,
        "source-layer": state.level,
        minzoom: levelMinZoom[state.level],
        paint: {
          "fill-color": fillColor(state.variable),
          // Strong at country scale, lighter close up so streets and places stay readable.
          "fill-opacity": [
            "interpolate",
            ["linear"],
            ["zoom"],
            8,
            ["case", hovered, 0.82, 0.62],
            12,
            ["case", hovered, 0.55, 0.32],
          ],
        },
      },
      slotBefore(map, "fills"),
    )
  }
  // Boundaries of the chosen level and every coarser one.
  const shown = regionLevels.slice(0, regionLevels.indexOf(state.level) + 1)
  for (const level of regionLevels) {
    const id = lineId(level)
    if (!shown.includes(level)) continue
    if (map.getLayer(id)) continue
    const [low, high] = lineWidths[level]
    map.addLayer(
      {
        id,
        type: "line",
        source: REGIONS_SOURCE,
        "source-layer": level,
        minzoom: levelMinZoom[level],
        layout: { "line-join": "round" },
        paint: {
          "line-color": tokens.boundary,
          "line-width": ["interpolate", ["linear"], ["zoom"], 6, low, 12, high],
          "line-opacity": level === state.level ? 0.75 : 0.9,
        },
      },
      slotBefore(map, "lines"),
    )
  }
  if (!map.getLayer(HIGHLIGHT)) {
    map.addLayer(
      {
        id: HIGHLIGHT,
        type: "line",
        source: REGIONS_SOURCE,
        "source-layer": state.level,
        minzoom: levelMinZoom[state.level],
        layout: { "line-join": "round" },
        paint: {
          "line-color": ["case", selected, tokens.brand, tokens.label],
          "line-width": ["case", selected, 2.5, hovered, 1.75, 0],
          "line-opacity": ["case", ["any", selected, hovered], 1, 0],
        },
      },
      slotBefore(map, "lines"),
    )
  }
  applyStates(map, state)
}

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of [HIGHLIGHT, REGIONS_FILL, ...regionLevels.map(lineId)]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(REGIONS_SOURCE)) map.removeSource(REGIONS_SOURCE)
}

function applyStates(map: MapLibreMap, state: State) {
  if (!map.getSource(REGIONS_SOURCE)) return
  for (const [id, value] of state.values) {
    map.setFeatureState({ source: REGIONS_SOURCE, sourceLayer: state.level, id }, { value })
  }
  if (state.selected !== null) {
    map.setFeatureState(
      { source: REGIONS_SOURCE, sourceLayer: state.level, id: state.selected },
      { selected: true },
    )
  }
}

/**
 * Administrative boundaries from the region vector tiles, and the chosen
 * level coloured by its weather. Colours are set per region with feature
 * state (keyed by the tiles' numeric ids), so new weather recolours the map
 * without reloading tiles. Fills sit under the basemap's water, so they
 * stop at the coast.
 */
export function RegionsLayer({
  layer,
  level,
  variable,
  weather,
  selectedId,
  onVisibleDistricts,
}: RegionsLayerProps) {
  const map = useMap()
  const state = useRef<State>({ level, variable, values: new Map(), selected: null })
  const layerRef = useRef(layer)
  const onDistricts = useRef(onVisibleDistricts)

  useEffect(() => {
    onDistricts.current = onVisibleDistricts
  }, [onVisibleDistricts])

  // Add the source and layers, and again after every style load (theme change).
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

  // A new level redraws the fill, highlight and boundaries for it.
  useEffect(() => {
    if (!map || state.current.level === level) return
    state.current = { ...state.current, level, values: new Map(), selected: null }
    if (!styleReady(map)) return
    for (const id of [HIGHLIGHT, REGIONS_FILL, ...regionLevels.map(lineId)]) {
      if (map.getLayer(id)) map.removeLayer(id)
    }
    for (const sourceLayer of regionLevels) {
      map.removeFeatureState({ source: REGIONS_SOURCE, sourceLayer })
    }
    addLayers(map, layerRef.current, state.current)
  }, [map, level])

  // Recolour for another variable.
  useEffect(() => {
    state.current = { ...state.current, variable }
    if (!map || !map.getLayer(REGIONS_FILL)) return
    map.setPaintProperty(REGIONS_FILL, "fill-color", fillColor(variable))
  }, [map, variable])

  // Each region's value as feature state.
  useEffect(() => {
    const values = new Map<number, number | null>()
    const read = variable ? regionVariables[variable].value : null
    for (const region of weather) values.set(regionFid(region.id), read ? read(region) : null)
    state.current = { ...state.current, values }
    if (map && styleReady(map)) applyStates(map, state.current)
  }, [map, weather, variable, level])

  // The selected region's outline.
  useEffect(() => {
    const previous = state.current.selected
    const next = selectedId ? regionFid(selectedId) : null
    state.current = { ...state.current, selected: next }
    if (!map || !map.getSource(REGIONS_SOURCE)) return
    const sourceLayer = state.current.level
    if (previous !== null) {
      map.setFeatureState(
        { source: REGIONS_SOURCE, sourceLayer, id: previous },
        { selected: false },
      )
    }
    if (next !== null) {
      map.setFeatureState({ source: REGIONS_SOURCE, sourceLayer, id: next }, { selected: true })
    }
  }, [map, selectedId, level])

  // Which districts' GN divisions are on screen, as the map moves and tiles load.
  useEffect(() => {
    if (!map || level !== "gn") return
    let timer = 0
    let last = ""
    const report = () => {
      window.clearTimeout(timer)
      timer = window.setTimeout(() => {
        if (!map.getLayer(REGIONS_FILL) || map.getZoom() < levelMinZoom.gn) return
        const districts = districtsOf(map.queryRenderedFeatures({ layers: [REGIONS_FILL] }))
        const key = districts.join(",")
        if (key === last || districts.length === 0) return
        last = key
        onDistricts.current?.(districts)
      }, 200)
    }
    report()
    map.on("moveend", report)
    map.on("sourcedata", report)
    return () => {
      window.clearTimeout(timer)
      map.off("moveend", report)
      map.off("sourcedata", report)
    }
  }, [map, level])

  return null
}
