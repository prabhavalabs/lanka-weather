import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { polygonSdf, type Point } from "../map/sdf"
import { slotBefore, styleReady } from "../map/slots"
import { bandMinZoom, bandOrder, bandTokens } from "./present"
import type { RainBand, RainGaugeCollection } from "./types"

const SOURCE = "rain-gauges"
/** The gauges: what clicks and hovers hit. */
export const RAIN_GAUGE_POINTS = "rain-gauge-points"
const LABELS = "rain-gauge-labels"
const DROPLET = "rain-gauge-droplet"

const empty: RainGaugeCollection = {
  type: "FeatureCollection",
  features: [],
  meta: {
    source: "",
    attribution: "",
    note: "",
    fetchedAt: null,
    sourceTime: null,
    generatedAt: "",
    staleAfterSeconds: 0,
    total: 0,
    fresh: 0,
    stale: 0,
    wet: 0,
    institutions: null,
  },
}

/** A drop: no other layer's point has this shape, so a rain gauge is never read as a river gauge. */
function droplet(): Point[] {
  const size = 40
  const centre: Point = [size / 2, 25]
  const radius = 10
  const points: Point[] = [[size / 2, 5]]
  for (let step = 0; step <= 16; step += 1) {
    const angle = -Math.PI / 3 + (step / 16) * ((5 * Math.PI) / 3)
    points.push([centre[0] + radius * Math.cos(angle), centre[1] + radius * Math.sin(angle)])
  }
  return points
}

function readPalette() {
  return readTokens({
    ...(Object.fromEntries(bandOrder.map((band) => [band, bandTokens[band]])) as Record<
      RainBand,
      `--${string}`
    >),
    stroke: "--status-stroke",
    strokeSelected: "--status-stroke-selected",
    label: "--map-label-strong",
    halo: "--map-label-halo",
    hollow: "--map-label-halo",
  })
}

const band: ExpressionSpecification = ["get", "band"]
const selected: ExpressionSpecification = ["boolean", ["feature-state", "selected"], false]
const rain: ExpressionSpecification = ["coalesce", ["get", "rain24hMm"], -1]

/**
 * A gauge's size at a zoom, for what it says. A size of zero is not drawn and cannot
 * be clicked, so a gauge first appears at its band's zoom (`bandMinZoom`).
 */
function size(at: {
  heavy: number
  moderate: number
  light: number
  quiet: number
}): ExpressionSpecification {
  return [
    "match",
    band,
    ["very_heavy", "heavy"],
    at.heavy,
    "moderate",
    at.moderate,
    "light",
    at.light,
    at.quiet,
  ] as ExpressionSpecification
}

const iconSize: ExpressionSpecification = [
  "interpolate",
  ["linear"],
  ["zoom"],
  5,
  size({ heavy: 0.6, moderate: 0.32, light: 0, quiet: 0 }),
  7,
  size({ heavy: 0.72, moderate: 0.4, light: 0, quiet: 0 }),
  bandMinZoom.light,
  size({ heavy: 0.86, moderate: 0.58, light: 0, quiet: 0 }),
  bandMinZoom.light + 0.25,
  size({ heavy: 0.86, moderate: 0.58, light: 0.42, quiet: 0 }),
  bandMinZoom.none,
  size({ heavy: 1, moderate: 0.78, light: 0.55, quiet: 0 }),
  bandMinZoom.none + 0.25,
  size({ heavy: 1, moderate: 0.78, light: 0.55, quiet: 0.36 }),
  12,
  size({ heavy: 1.2, moderate: 1, light: 0.8, quiet: 0.55 }),
] as ExpressionSpecification

const labelText: ExpressionSpecification = [
  "concat",
  ["to-string", ["round", ["get", "rain24hMm"]]],
  " mm",
]

function addLayers(map: MapLibreMap, data: RainGaugeCollection) {
  const palette = readPalette()
  if (!map.hasImage(DROPLET)) {
    map.addImage(DROPLET, polygonSdf(40, droplet()), { sdf: true, pixelRatio: 2 })
  }
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      promoteId: "id",
      attribution: "Rain gauges: HMIS, via Sri Lanka Water Levels (waterlevels.lk)",
    })
  }
  if (!map.getLayer(RAIN_GAUGE_POINTS)) {
    map.addLayer(
      {
        id: RAIN_GAUGE_POINTS,
        type: "symbol",
        source: SOURCE,
        layout: {
          "icon-image": DROPLET,
          "icon-size": iconSize,
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
          // The wettest are drawn last, so on top.
          "symbol-sort-key": rain,
        },
        paint: {
          "icon-color": [
            "match",
            band,
            "very_heavy",
            palette.very_heavy,
            "heavy",
            palette.heavy,
            "moderate",
            palette.moderate,
            "light",
            palette.light,
            "none",
            palette.none,
            palette.hollow,
          ],
          "icon-opacity": ["match", band, "none", 0.7, 1],
          "icon-halo-color": [
            "case",
            selected,
            palette.strokeSelected,
            ["==", band, "unknown"],
            palette.unknown,
            palette.stroke,
          ],
          "icon-halo-width": [
            "case",
            selected,
            3,
            ["in", band, ["literal", ["none", "moderate", "light"]]],
            0.5,
            1,
          ],
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
        // An old total gets no number that could read as what is happening now.
        filter: ["!", ["get", "stale"]],
        minzoom: 6,
        layout: {
          "text-field": [
            "step",
            ["zoom"],
            ["case", [">=", rain, 100], labelText, ""],
            8,
            ["case", [">=", rain, 50], labelText, ""],
            9,
            ["case", [">=", rain, 10], labelText, ""],
            10.5,
            ["case", [">", rain, 0], labelText, ""],
          ],
          "text-font": ["Noto Sans Regular"],
          "text-size": 11,
          "text-offset": [0, 0.35],
          "text-anchor": "top",
          "text-optional": true,
          // The wettest are placed first, so they keep their numbers.
          "symbol-sort-key": ["-", 0, rain],
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
  if (!styleReady(map)) return
  for (const id of [LABELS, RAIN_GAUGE_POINTS]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

/**
 * The live rain gauges as drops, coloured and sized by their rolling 24-hour total.
 * At country zoom only meaningful rain shows; light rain and then dry and out-of-date
 * gauges follow as you zoom in, small and quiet. Clicks are handled by App.
 */
export function RainGaugesLayer({
  data,
  selectedId,
}: {
  data: RainGaugeCollection | undefined
  selectedId: string | null
}) {
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
