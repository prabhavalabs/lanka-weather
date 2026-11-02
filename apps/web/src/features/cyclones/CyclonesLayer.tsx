import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { polygonSdf, type Point } from "../map/sdf"
import { slotBefore, styleReady } from "../map/slots"
import { alertTokens } from "./present"
import type { CycloneCollection } from "./types"

const SOURCE = "cyclones"
/** The storm symbols: what clicks and hovers hit. */
export const CYCLONE_POSITIONS = "cyclones-positions"
const EYE = "cyclones-eye"
const POSITION_LABELS = "cyclones-position-labels"
const BUFFERS = "cyclones-buffers"
const BUFFER_LINES = "cyclones-buffer-lines"
const CONE = "cyclones-cone"
const TRACK = "cyclones-track"
const FORECAST = "cyclones-forecast"
const POINTS = "cyclones-track-points"
const SWIRL = "cyclone-swirl"

const kind = (value: string): ExpressionSpecification => ["==", ["get", "kind"], value]
const selected = ["boolean", ["feature-state", "selected"], false] as never

/** A two-armed swirl, the storm symbol on weather charts. */
function swirl(size: number): Point[] {
  const half = size / 2
  const points: Point[] = []
  const steps = 96
  for (let index = 0; index < steps; index += 1) {
    const angle = (index / steps) * Math.PI * 2
    const phase = (angle * 2) % (Math.PI * 2)
    const radius = half * (0.42 + 0.5 * (phase / (Math.PI * 2)) ** 2.2)
    points.push([half + radius * Math.cos(angle), half + radius * Math.sin(angle)])
  }
  return points
}

function readPalette() {
  return readTokens({
    green: alertTokens.Green!,
    orange: alertTokens.Orange!,
    red: alertTokens.Red!,
    stroke: "--status-stroke",
    strokeSelected: "--status-stroke-selected",
    label: "--map-label-strong",
    halo: "--map-label-halo",
    cone: "--map-label",
    eye: "--map-label-halo",
  })
}

function addLayers(map: MapLibreMap, data: CycloneCollection) {
  const palette = readPalette()
  const alertColor = [
    "match",
    ["get", "alertLevel"],
    "Green",
    palette.green,
    "Orange",
    palette.orange,
    palette.red,
  ] as never
  const bandColor = [
    "match",
    ["get", "band"],
    "green",
    palette.green,
    "orange",
    palette.orange,
    palette.red,
  ] as never
  if (!map.hasImage(SWIRL)) {
    map.addImage(SWIRL, polygonSdf(56, swirl(56)), { sdf: true, pixelRatio: 2 })
  }
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      promoteId: "eventId",
      attribution: "Cyclones: GDACS, gdacs.org",
    })
  }
  const areas = slotBefore(map, "areas")
  const top = slotBefore(map, "points")
  if (!map.getLayer(BUFFERS)) {
    map.addLayer(
      {
        id: BUFFERS,
        type: "fill",
        source: SOURCE,
        filter: kind("wind-buffer"),
        paint: { "fill-color": bandColor, "fill-opacity": 0.14 },
      },
      areas,
    )
  }
  if (!map.getLayer(BUFFER_LINES)) {
    map.addLayer(
      {
        id: BUFFER_LINES,
        type: "line",
        source: SOURCE,
        filter: kind("wind-buffer"),
        paint: { "line-color": bandColor, "line-width": 1, "line-opacity": 0.7 },
      },
      areas,
    )
  }
  if (!map.getLayer(CONE)) {
    map.addLayer(
      {
        id: CONE,
        type: "line",
        source: SOURCE,
        filter: kind("uncertainty-cone"),
        paint: { "line-color": palette.cone, "line-width": 1.25, "line-dasharray": [1, 2] },
      },
      areas,
    )
  }
  if (!map.getLayer(TRACK)) {
    map.addLayer(
      {
        id: TRACK,
        type: "line",
        source: SOURCE,
        filter: kind("track"),
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": palette.label, "line-width": 2.25 },
      },
      top,
    )
  }
  if (!map.getLayer(FORECAST)) {
    map.addLayer(
      {
        id: FORECAST,
        type: "line",
        source: SOURCE,
        filter: kind("forecast-track"),
        layout: { "line-join": "round" },
        paint: { "line-color": palette.label, "line-width": 2, "line-dasharray": [2, 2] },
      },
      top,
    )
  }
  if (!map.getLayer(POINTS)) {
    map.addLayer(
      {
        id: POINTS,
        type: "circle",
        source: SOURCE,
        filter: kind("track-point"),
        paint: {
          "circle-radius": 3,
          "circle-color": [
            "case",
            ["boolean", ["get", "forecast"], false],
            palette.halo,
            palette.label,
          ],
          "circle-stroke-width": 1.25,
          "circle-stroke-color": palette.label,
        },
      },
      top,
    )
  }
  if (!map.getLayer(CYCLONE_POSITIONS)) {
    map.addLayer(
      {
        id: CYCLONE_POSITIONS,
        type: "symbol",
        source: SOURCE,
        filter: kind("position"),
        layout: {
          "icon-image": SWIRL,
          "icon-size": ["interpolate", ["linear"], ["zoom"], 4, 0.8, 8, 1.2],
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
        },
        paint: {
          "icon-color": alertColor,
          "icon-opacity": ["case", ["get", "stale"], 0.5, 1],
          "icon-halo-color": ["case", selected, palette.strokeSelected, palette.stroke],
          "icon-halo-width": ["case", selected, 3, 1.25],
        },
      },
      top,
    )
  }
  if (!map.getLayer(EYE)) {
    map.addLayer(
      {
        id: EYE,
        type: "circle",
        source: SOURCE,
        filter: kind("position"),
        paint: {
          "circle-radius": ["interpolate", ["linear"], ["zoom"], 4, 2.5, 8, 4],
          "circle-color": palette.eye,
          "circle-stroke-width": 1,
          "circle-stroke-color": palette.stroke,
        },
      },
      top,
    )
  }
  if (!map.getLayer(POSITION_LABELS)) {
    map.addLayer(
      {
        id: POSITION_LABELS,
        type: "symbol",
        source: SOURCE,
        filter: kind("position"),
        layout: {
          "text-field": ["concat", ["get", "name"], " · ", ["get", "alertLevel"]],
          "text-font": ["Noto Sans Regular"],
          "text-size": 12,
          "text-offset": [0, 1.7],
          "text-anchor": "top",
          "text-allow-overlap": true,
        },
        paint: {
          "text-color": palette.label,
          "text-halo-color": palette.halo,
          "text-halo-width": 1.75,
        },
      },
      top,
    )
  }
}

const allLayers = [
  POSITION_LABELS,
  EYE,
  CYCLONE_POSITIONS,
  POINTS,
  FORECAST,
  TRACK,
  CONE,
  BUFFER_LINES,
  BUFFERS,
]

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of allLayers) if (map.getLayer(id)) map.removeLayer(id)
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

const empty = { type: "FeatureCollection", features: [], meta: {} } as unknown as CycloneCollection

interface CyclonesLayerProps {
  data: CycloneCollection | undefined
  selectedId: string | null
}

/**
 * Current systems: wind buffers and the uncertainty cone under the labels, the
 * track solid and the forecast dashed, and the storm symbol in its GDACS alert
 * colour, with the alert's word beside it.
 */
export function CyclonesLayer({ data, selectedId }: CyclonesLayerProps) {
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
