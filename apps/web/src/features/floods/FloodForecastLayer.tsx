import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { polygonSdf, type Point } from "../map/sdf"
import { slotBefore, styleReady } from "../map/slots"
import { areaKinds, areaStyle } from "./basins"
import { floodSeverityTokens } from "./present"
import type { FloodForecastCollection } from "./types"

const SOURCE = "flood-forecast"
/** The gauges: what clicks and hovers hit. */
export const FLOOD_GAUGES = "flood-forecast-gauges"
const TRENDS = "flood-forecast-trends"
/** Google's flood polygons: what clicks and hovers hit. */
export const FLOOD_AREAS = "flood-forecast-areas"
const AREAS = FLOOD_AREAS
const AREA_LINES = "flood-forecast-area-lines"
const DIAMOND = "flood-diamond"
const RISE = "flood-trend-rise"
const FALL = "flood-trend-fall"

const empty: FloodForecastCollection = {
  type: "FeatureCollection",
  features: [],
  meta: {
    source: "",
    attribution: "",
    region: "LK",
    configured: true,
    fetchedAt: null,
    generatedAt: "",
    gauges: 0,
    areas: 0,
  },
}

/** Icons drawn at twice their size for sharp edges on any screen. */
const ICON_RATIO = 2

function addIcons(map: MapLibreMap) {
  const size = 40
  const half = size / 2
  const r = 14
  const diamond: Point[] = [
    [half, half - r],
    [half + r, half],
    [half, half + r],
    [half - r, half],
  ]
  const up: Point[] = [
    [half, half - 9],
    [half + 8, half + 6],
    [half - 8, half + 6],
  ]
  const down = up.map(([x, y]) => [x, size - y] as Point)
  const images: [string, Point[]][] = [
    [DIAMOND, diamond],
    [RISE, up],
    [FALL, down],
  ]
  for (const [name, shape] of images) {
    if (!map.hasImage(name)) {
      map.addImage(name, polygonSdf(size, shape), { sdf: true, pixelRatio: ICON_RATIO })
    }
  }
}

function areaValue<T>(
  pick: (kind: (typeof areaKinds)[number]) => T,
  fallback: T,
): ExpressionSpecification {
  const expression: unknown[] = ["match", ["get", "kind"]]
  for (const kind of areaKinds) expression.push(kind, pick(kind))
  expression.push(fallback)
  return expression as ExpressionSpecification
}

function readPalette() {
  return readTokens({
    ...floodSeverityTokens,
    ...Object.fromEntries(areaKinds.map((kind) => [`area:${kind}`, areaStyle[kind].token])),
    stroke: "--status-stroke",
    strokeSelected: "--status-stroke-selected",
    label: "--map-label-strong",
    halo: "--map-label-halo",
  })
}

type Palette = ReturnType<typeof readPalette>

function severityColor(palette: Palette): ExpressionSpecification {
  return [
    "match",
    ["get", "severity"],
    "extreme",
    palette.extreme,
    "severe",
    palette.severe,
    "above_normal",
    palette.above_normal,
    "no_flooding",
    palette.no_flooding,
    palette.unknown,
  ]
}

/** Draws severe gauges above calm ones. */
const severityRank: ExpressionSpecification = [
  "match",
  ["get", "severity"],
  "extreme",
  4,
  "severe",
  3,
  "above_normal",
  2,
  "no_flooding",
  1,
  0,
]

const isGauge: ExpressionSpecification = ["==", ["get", "kind"], "gauge"]
const selected: ExpressionSpecification = ["boolean", ["feature-state", "selected"], false]

function addLayers(map: MapLibreMap, data: FloodForecastCollection) {
  const palette = readPalette()
  addIcons(map)
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      promoteId: "gaugeId",
      attribution: "Flood forecasts: Google Flood Hub",
    })
  }
  if (!map.getLayer(AREAS)) {
    map.addLayer(
      {
        id: AREAS,
        type: "fill",
        source: SOURCE,
        filter: ["!", isGauge],
        layout: { "fill-sort-key": areaValue((kind) => areaStyle[kind].rank, 0) },
        paint: {
          "fill-color": areaValue(
            (kind) => (palette as Record<string, string>)[`area:${kind}`]!,
            palette.unknown,
          ),
          "fill-opacity": areaValue((kind) => areaStyle[kind].opacity, 0.3),
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
        source: SOURCE,
        filter: ["!", isGauge],
        layout: {
          "line-join": "round",
          "line-sort-key": areaValue((kind) => areaStyle[kind].rank, 0),
        },
        paint: {
          "line-color": areaValue(
            (kind) => (palette as Record<string, string>)[`area:${kind}`]!,
            palette.unknown,
          ),
          "line-width": areaValue((kind) => areaStyle[kind].width, 1.5),
        },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(FLOOD_GAUGES)) {
    map.addLayer(
      {
        id: FLOOD_GAUGES,
        type: "symbol",
        source: SOURCE,
        filter: isGauge,
        layout: {
          "icon-image": DIAMOND,
          // Calm forecasts stay small until there is room; a raised one never does.
          "icon-size": [
            "interpolate",
            ["linear"],
            ["zoom"],
            4,
            ["case", ["==", ["get", "severity"], "no_flooding"], 0.2, 0.6],
            6,
            ["case", ["==", ["get", "severity"], "no_flooding"], 0.42, 0.75],
            8,
            ["case", ["==", ["get", "severity"], "no_flooding"], 0.58, 0.85],
            10,
            1,
          ],
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
          "symbol-sort-key": severityRank,
        },
        paint: {
          "icon-color": severityColor(palette),
          "icon-opacity": ["case", ["get", "stale"], 0.45, 1],
          "icon-halo-color": ["case", selected, palette.strokeSelected, palette.stroke],
          "icon-halo-width": ["case", selected, 3, 1.25],
        },
      },
      slotBefore(map, "points"),
    )
  }
  if (!map.getLayer(TRENDS)) {
    map.addLayer(
      {
        id: TRENDS,
        type: "symbol",
        source: SOURCE,
        filter: ["all", isGauge, ["in", ["get", "trend"], ["literal", ["rise", "fall"]]]],
        minzoom: 7,
        layout: {
          "icon-image": ["match", ["get", "trend"], "rise", RISE, FALL],
          "icon-size": 0.5,
          "icon-offset": [22, -18],
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
        },
        paint: {
          "icon-color": palette.label,
          "icon-halo-color": palette.halo,
          "icon-halo-width": 1.5,
        },
      },
      slotBefore(map, "points"),
    )
  }
}

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of [TRENDS, FLOOD_GAUGES, AREA_LINES, AREAS]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface FloodForecastLayerProps {
  data: FloodForecastCollection | undefined
  selectedId: string | null
}

/**
 * Google flood forecast gauges as diamonds coloured by severity (the river
 * status colours), with an arrow when the forecast rises or falls, and
 * forecast flood areas as solid fills with strong outlines: inundation graded
 * by how likely the water is, the likeliest drawn on top.
 */
export function FloodForecastLayer({ data, selectedId }: FloodForecastLayerProps) {
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
