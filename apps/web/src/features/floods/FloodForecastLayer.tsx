import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { polygonSdf, type Point } from "../map/sdf"
import { slotBefore, styleReady } from "../map/slots"
import { areaKinds, areaStyle } from "./basins"
import { floodSeverityGlyphs, floodSeverityOrder, floodSeverityTokens } from "./present"
import type { FloodForecastCollection } from "./types"

const SOURCE = "flood-forecast"
/** The gauges: what clicks and hovers hit. */
export const FLOOD_GAUGES = "flood-forecast-gauges"
const SHADOWS = "flood-forecast-gauge-shadows"
const GLYPHS = "flood-forecast-glyphs"
const TRENDS = "flood-forecast-trends"
/** Google's flood polygons: what clicks and hovers hit. */
export const FLOOD_AREAS = "flood-forecast-areas"
const AREAS = FLOOD_AREAS
const AREA_LINES = "flood-forecast-area-lines"
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
  const up: Point[] = [
    [half, half - 9],
    [half + 8, half + 6],
    [half - 8, half + 6],
  ]
  const down = up.map(([x, y]) => [x, size - y] as Point)
  const images: [string, Point[]][] = [
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
    ring: "--flood-ring",
    glyph: "--flood-glyph",
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

/** Draws severe gauges above calm ones, and verified above lower-confidence ones. */
const gaugeOrder: ExpressionSpecification = [
  "+",
  ["match", ["get", "severity"], "extreme", 4, "severe", 3, "above_normal", 2, "no_flooding", 1, 0],
  ["case", ["get", "qualityVerified"], 10, 0],
]

const isGauge: ExpressionSpecification = ["==", ["get", "kind"], "gauge"]
const selected: ExpressionSpecification = ["boolean", ["feature-state", "selected"], false]
const verified: ExpressionSpecification = ["get", "qualityVerified"]
const raised: ExpressionSpecification = [
  "in",
  ["get", "severity"],
  ["literal", ["above_normal", "severe", "extreme"]],
]

/** A gauge's radius at a zoom: raised gauges larger, lower-confidence ones smaller. */
function radius(extra = 0): ExpressionSpecification {
  const at = (base: number) => [
    "case",
    verified,
    ["case", raised, base + 1.5 + extra, base + extra],
    base * 0.72 + extra,
  ]
  return [
    "interpolate",
    ["linear"],
    ["zoom"],
    5,
    at(4.5),
    7,
    at(6),
    9,
    at(7),
    12,
    at(8.5),
  ] as unknown as ExpressionSpecification
}

function glyphText(): ExpressionSpecification {
  const expression: unknown[] = ["match", ["get", "severity"]]
  for (const severity of floodSeverityOrder) {
    expression.push(severity, floodSeverityGlyphs[severity])
  }
  expression.push("")
  return expression as ExpressionSpecification
}

function addLayers(map: MapLibreMap, data: FloodForecastCollection) {
  const palette = readPalette()
  addIcons(map)
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      promoteId: "gaugeId",
      // Keeps the corners of Google's pixel-grid polygons at every zoom.
      tolerance: 0.1,
      attribution: "Flood forecasts: Google Flood Hub (Flood Forecasting API)",
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
          // Square corners: these are grid cells, and a round join would soften them.
          "line-join": "miter",
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
  if (!map.getLayer(SHADOWS)) {
    map.addLayer(
      {
        id: SHADOWS,
        type: "circle",
        source: SOURCE,
        filter: isGauge,
        layout: { "circle-sort-key": gaugeOrder },
        paint: {
          // A soft edge, so the light ring holds on a pale basemap as Flood Hub's does.
          "circle-radius": radius(4),
          "circle-color": "#000000",
          "circle-opacity": 0.22,
          "circle-blur": 0.55,
        },
      },
      slotBefore(map, "points"),
    )
  }
  if (!map.getLayer(FLOOD_GAUGES)) {
    map.addLayer(
      {
        id: FLOOD_GAUGES,
        type: "circle",
        source: SOURCE,
        filter: isGauge,
        layout: { "circle-sort-key": gaugeOrder },
        paint: {
          "circle-radius": radius(),
          // A lower-confidence gauge is hollow: its colour on the ring, not filled.
          "circle-color": ["case", verified, severityColor(palette), palette.ring],
          "circle-opacity": ["case", ["get", "stale"], 0.45, 1],
          "circle-stroke-color": [
            "case",
            selected,
            palette.strokeSelected,
            verified,
            palette.ring,
            severityColor(palette),
          ],
          "circle-stroke-width": ["case", selected, 3, verified, 2.25, 2],
        },
      },
      slotBefore(map, "points"),
    )
  }
  if (!map.getLayer(GLYPHS)) {
    map.addLayer(
      {
        id: GLYPHS,
        type: "symbol",
        source: SOURCE,
        filter: ["all", isGauge, ["!=", ["get", "severity"], "no_flooding"]],
        minzoom: 6,
        layout: {
          "text-field": glyphText(),
          "text-font": ["Noto Sans Regular"],
          "text-size": 10,
          "text-allow-overlap": true,
          "text-ignore-placement": true,
          "symbol-sort-key": gaugeOrder,
        },
        paint: {
          "text-color": [
            "case",
            ["all", verified, ["==", ["get", "severity"], "extreme"]],
            palette.ring,
            palette.glyph,
          ],
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
        // A calm forecast often rises, so at country scale arrows would only be noise.
        minzoom: 8,
        layout: {
          "icon-image": ["match", ["get", "trend"], "rise", RISE, FALL],
          "icon-size": 0.5,
          "icon-offset": [26, -20],
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
  for (const id of [TRENDS, GLYPHS, FLOOD_GAUGES, SHADOWS, AREA_LINES, AREAS]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface FloodForecastLayerProps {
  data: FloodForecastCollection | undefined
  selectedId: string | null
}

/**
 * Google's flood forecast in Flood Hub's language: round gauges with a light ring,
 * coloured by expected severity, with a mark beside the colour and an arrow when the
 * forecast rises or falls; flash-flood areas with a pale fill and a strong outline,
 * highly likely on top; and inundation on a blue ramp. Takes what the view shows.
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
