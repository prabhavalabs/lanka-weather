import type { GeoJSONSource, Map as MapLibreMap } from "maplibre-gl"
import type { FeatureCollection } from "geojson"
import { useEffect, useMemo, useRef } from "react"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import { ageOpacity, magnitudeRadius, withAge } from "./present"
import type { EarthquakeCollection } from "./types"

const SOURCE = "earthquakes"
/** The circles: what clicks and hovers hit. */
export const EARTHQUAKES = "earthquakes-circles"
const TSUNAMI = "earthquakes-tsunami"
const LABELS = "earthquakes-labels"

const empty = {
  type: "FeatureCollection",
  features: [],
  meta: {},
} as unknown as EarthquakeCollection

function radius(extra: number) {
  // Radius grows with magnitude, and a little with zoom.
  const at = (magnitude: number) => magnitudeRadius(magnitude) + extra
  const stops: (number | unknown[])[] = []
  for (const zoom of [4, 8]) {
    stops.push(zoom, [
      "interpolate",
      ["linear"],
      ["get", "magnitude"],
      2.5,
      at(2.5) * (zoom === 4 ? 0.8 : 1.2),
      8.5,
      at(8.5) * (zoom === 4 ? 0.8 : 1.2),
    ])
  }
  return ["interpolate", ["linear"], ["zoom"], ...stops] as never
}

function addLayers(map: MapLibreMap, data: EarthquakeCollection) {
  const palette = readTokens({
    brand: "--brand",
    stroke: "--status-stroke",
    strokeSelected: "--status-stroke-selected",
    tsunami: "--status-major-flood",
    label: "--map-label-strong",
    halo: "--map-label-halo",
  })
  const selected = ["boolean", ["feature-state", "selected"], false] as never
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "geojson",
      data: data as unknown as FeatureCollection,
      promoteId: "id",
      attribution: "Earthquakes: U.S. Geological Survey",
    })
  }
  if (!map.getLayer(EARTHQUAKES)) {
    map.addLayer(
      {
        id: EARTHQUAKES,
        type: "circle",
        source: SOURCE,
        layout: { "circle-sort-key": ["get", "magnitude"] },
        paint: {
          "circle-radius": radius(0),
          "circle-color": palette.brand,
          "circle-opacity": [
            "match",
            ["get", "age"],
            "day",
            ageOpacity.day,
            "week",
            ageOpacity.week,
            ageOpacity.month,
          ],
          "circle-stroke-width": ["case", selected, 3, 1],
          "circle-stroke-color": ["case", selected, palette.strokeSelected, palette.stroke],
        },
      },
      slotBefore(map, "points"),
    )
  }
  if (!map.getLayer(TSUNAMI)) {
    map.addLayer(
      {
        id: TSUNAMI,
        type: "circle",
        source: SOURCE,
        filter: ["==", ["get", "tsunami"], true],
        paint: {
          "circle-radius": radius(4),
          "circle-color": "rgba(0, 0, 0, 0)",
          "circle-stroke-width": 2,
          "circle-stroke-color": palette.tsunami,
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
        filter: [">=", ["get", "magnitude"], 5.5],
        layout: {
          "text-field": [
            "concat",
            "M",
            [
              "number-format",
              ["get", "magnitude"],
              { "min-fraction-digits": 1, "max-fraction-digits": 1 },
            ],
          ],
          "text-font": ["Noto Sans Regular"],
          "text-size": 11,
          "text-offset": [0, 1.2],
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
  for (const id of [LABELS, TSUNAMI, EARTHQUAKES]) if (map.getLayer(id)) map.removeLayer(id)
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface EarthquakesLayerProps {
  data: EarthquakeCollection | undefined
  selectedId: string | null
}

/**
 * Earthquakes as circles sized by magnitude and shaded by age, with a ring
 * where USGS has set its tsunami flag. Clicks are handled by the map's click
 * handler.
 */
export function EarthquakesLayer({ data, selectedId }: EarthquakesLayerProps) {
  const map = useMap()
  const shown = useMemo(() => withAge(data ?? empty), [data])
  const current = useRef({ shown, selectedId })

  useEffect(() => {
    if (!map) return
    const add = () => {
      addLayers(map, current.current.shown)
      if (current.current.selectedId) {
        map.setFeatureState({ source: SOURCE, id: current.current.selectedId }, { selected: true })
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
    current.current = { ...current.current, shown }
    const source = map?.getSource(SOURCE) as GeoJSONSource | undefined
    source?.setData(shown as unknown as FeatureCollection)
  }, [map, shown])

  useEffect(() => {
    const previous = current.current.selectedId
    current.current = { ...current.current, selectedId }
    if (!map || !map.getSource(SOURCE)) return
    if (previous) map.setFeatureState({ source: SOURCE, id: previous }, { selected: false })
    if (selectedId) map.setFeatureState({ source: SOURCE, id: selectedId }, { selected: true })
  }, [map, selectedId])

  return null
}
