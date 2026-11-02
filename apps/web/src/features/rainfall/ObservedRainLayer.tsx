import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useRef } from "react"
import { stepExpression } from "@/lib/colour"
import { rainfallScale, readToken, readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import { addStationIcon, STATION_ICON } from "../stations/icon"
import type { ObservedRainCollection } from "./types"

const SOURCE = "observed-rain"
/** River gauges are circles. */
export const RAIN_GAUGE_POINTS = "observed-rain-gauges"
/** Met stations are squares, as on the stations layer. */
export const RAIN_MET_POINTS = "observed-rain-met"
const LABELS = "observed-rain-labels"

const empty: ObservedRainCollection = {
  type: "FeatureCollection",
  features: [],
  meta: { attribution: "", generatedAt: "", observed: true, fresh: 0, stale: 0 },
}

function paints() {
  const palette = readTokens({
    dry: "--status-unknown",
    stroke: "--status-stroke",
    label: "--map-label-strong",
    halo: "--map-label-halo",
    hollow: "--map-label-halo",
  })
  const colours = rainfallScale.map((step) => readToken(step.token))
  const colour = stepExpression(
    ["get", "rainMm"],
    rainfallScale,
    colours,
    palette.dry,
  ) as unknown as ExpressionSpecification
  return { palette, colour }
}

function addLayers(map: MapLibreMap, data: ObservedRainCollection) {
  const { palette, colour } = paints()
  addStationIcon(map)
  const stale: ExpressionSpecification = ["get", "stale"]
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      attribution: "Rainfall: Department of Meteorology, Irrigation Department of Sri Lanka",
    })
  }
  if (!map.getLayer(RAIN_GAUGE_POINTS)) {
    map.addLayer(
      {
        id: RAIN_GAUGE_POINTS,
        type: "circle",
        source: SOURCE,
        filter: ["==", ["get", "kind"], "gauge"],
        paint: {
          "circle-radius": [
            "interpolate",
            ["linear"],
            ["zoom"],
            6,
            ["case", stale, 3, 4],
            10,
            ["case", stale, 5, 7],
          ],
          "circle-color": ["case", stale, palette.hollow, colour],
          "circle-stroke-width": ["case", stale, 2, 1.25],
          "circle-stroke-color": ["case", stale, palette.dry, palette.stroke],
        },
      },
      slotBefore(map, "points"),
    )
  }
  if (!map.getLayer(RAIN_MET_POINTS)) {
    map.addLayer(
      {
        id: RAIN_MET_POINTS,
        type: "symbol",
        source: SOURCE,
        filter: ["==", ["get", "kind"], "met"],
        layout: {
          "icon-image": STATION_ICON,
          "icon-size": [
            "interpolate",
            ["linear"],
            ["zoom"],
            6,
            ["case", stale, 0.4, 0.55],
            10,
            ["case", stale, 0.6, 0.8],
          ],
          "icon-allow-overlap": true,
        },
        paint: {
          "icon-color": ["case", stale, palette.hollow, colour],
          "icon-halo-color": ["case", stale, palette.dry, palette.stroke],
          "icon-halo-width": 1.25,
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
        // An old total keeps its point, marked hollow, but not a number that
        // reads as what is happening now.
        filter: ["!", ["get", "stale"]],
        layout: {
          "text-field": ["concat", ["to-string", ["round", ["get", "rainMm"]]], " mm"],
          "text-font": ["Noto Sans Regular"],
          "text-size": 11,
          "text-offset": [0, 0.9],
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
  if (!styleReady(map)) return
  for (const id of [LABELS, RAIN_MET_POINTS, RAIN_GAUGE_POINTS]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

/**
 * Rain measured over the last 24 hours, labelled in mm: circles at river
 * gauges and squares at Met stations. Clicks are handled by App.
 */
export function ObservedRainLayer({ data }: { data: ObservedRainCollection | undefined }) {
  const map = useMap()
  const dataRef = useRef(data ?? empty)

  useEffect(() => {
    if (!map) return
    const add = () => addLayers(map, dataRef.current)
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

  return null
}
