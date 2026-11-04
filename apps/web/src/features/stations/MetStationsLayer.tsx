import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useRef } from "react"
import { stepExpression } from "@/lib/colour"
import { readToken, readTokens, temperatureScale } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import { addStationIcon, STATION_ICON } from "./icon"
import { markerTemperature } from "./present"
import type { MetStationCollection } from "./types"

const SOURCE = "met-stations"
/** The stations: what clicks and hovers hit. */
export const MET_STATIONS = "met-stations-tags"
const NAMES = "met-stations-names"

const empty: MetStationCollection = {
  type: "FeatureCollection",
  features: [],
  meta: {
    attribution: "",
    sourceUrl: "",
    generatedAt: "",
    observed: true,
    threeHourly: { source: "", newestObservedAt: null, fetchedAt: null, stale: true },
    daily: { source: "", newestObservedAt: null, fetchedAt: null, stale: true },
    stale: true,
  },
}

/** Only stations with a temperature to draw, which the map reads as one property. */
function mapData(data: MetStationCollection) {
  return {
    type: "FeatureCollection",
    features: data.features
      .filter((feature) => markerTemperature(feature.properties) !== null)
      .map((feature) => ({
        ...feature,
        properties: { ...feature.properties, shown: markerTemperature(feature.properties) },
      })),
  } as unknown as FeatureCollection
}

function addLayers(map: MapLibreMap, data: MetStationCollection) {
  addStationIcon(map)
  const palette = readTokens({
    stroke: "--status-stroke",
    selected: "--status-stroke-selected",
    label: "--map-label-strong",
    halo: "--map-label-halo",
  })
  const colours = temperatureScale.map((step) => readToken(step.token))
  // The coolest and hottest colours need light text; the others dark.
  const textColours = temperatureScale.map((_, index) =>
    index === 0 || index === temperatureScale.length - 1 ? "#ffffff" : "#0a0a0a",
  )
  const shown: ExpressionSpecification = ["get", "shown"]
  const selected: ExpressionSpecification = ["boolean", ["feature-state", "selected"], false]
  const stale: ExpressionSpecification = ["get", "stale"]
  const colour = stepExpression(
    shown,
    temperatureScale,
    colours,
  ) as unknown as ExpressionSpecification
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: mapData(data),
      promoteId: "id",
      attribution: "Stations: Department of Meteorology, Sri Lanka",
    })
  }
  if (!map.getLayer(MET_STATIONS)) {
    map.addLayer(
      {
        id: MET_STATIONS,
        type: "symbol",
        source: SOURCE,
        layout: {
          "icon-image": STATION_ICON,
          "icon-size": ["interpolate", ["linear"], ["zoom"], 6, 0.85, 10, 1.1],
          "icon-allow-overlap": true,
          "icon-ignore-placement": true,
          "text-field": ["to-string", ["round", shown]],
          "text-font": ["Noto Sans Regular"],
          "text-size": ["interpolate", ["linear"], ["zoom"], 6, 11, 10, 13],
          "text-allow-overlap": true,
          "text-ignore-placement": true,
        },
        paint: {
          // Out of date: a hollow square, its colour only in the outline, so it does not
          // read as the temperature now and stays legible over the map.
          "icon-color": ["case", stale, palette.halo, colour],
          "icon-halo-color": ["case", selected, palette.selected, stale, colour, palette.stroke],
          "icon-halo-width": ["case", selected, 3, stale, 2, 1.25],
          "text-color": [
            "case",
            stale,
            palette.label,
            stepExpression(shown, temperatureScale, textColours),
          ] as unknown as ExpressionSpecification,
        },
      },
      slotBefore(map, "points"),
    )
  }
  if (!map.getLayer(NAMES)) {
    map.addLayer(
      {
        id: NAMES,
        type: "symbol",
        source: SOURCE,
        minzoom: 8.5,
        layout: {
          "text-field": ["get", "name"],
          "text-font": ["Noto Sans Regular"],
          "text-size": 12,
          "text-offset": [0, 1.5],
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
  if (map.getLayer(NAMES)) map.removeLayer(NAMES)
  if (map.getLayer(MET_STATIONS)) map.removeLayer(MET_STATIONS)
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface MetStationsLayerProps {
  data: MetStationCollection | undefined
  selectedId: string | null
}

/**
 * Met stations: squares with the temperature in them, coloured by the same
 * scale as the weather field, and faded when out of date. Clicks are handled
 * by App.
 */
export function MetStationsLayer({ data, selectedId }: MetStationsLayerProps) {
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
    source?.setData(mapData(dataRef.current))
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
