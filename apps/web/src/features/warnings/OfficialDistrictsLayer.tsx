import type { ExpressionSpecification, Map as MapLibreMap } from "maplibre-gl"
import { useEffect, useRef } from "react"
import type { Layer } from "@/lib/layers"
import { readTokens } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { regionFid } from "../regions/present"
import { slotBefore, styleReady } from "../map/slots"
import type { DistrictHighlight } from "./types"

const SOURCE = "official-districts"
/** A faint fill over highlighted districts: what clicks and hovers hit. */
export const OFFICIAL_FILL = "official-districts-fill"
const LANDSLIDE_LINE = "official-districts-landslide"
const MENTION_LINE = "official-districts-mention"

function absolute(path: string) {
  return /^https?:/.test(path) ? path : `${window.location.origin}${path}`
}

function idFilter(ids: number[]): ExpressionSpecification {
  return ["in", ["id"], ["literal", ids]] as unknown as ExpressionSpecification
}

function split(districts: DistrictHighlight[]) {
  return {
    all: districts.map((district) => regionFid(district.id)),
    landslide: districts.filter((d) => d.landslideLevel > 0).map((d) => regionFid(d.id)),
    mention: districts
      .filter((d) => d.landslideLevel === 0 && d.mentioned.length > 0)
      .map((d) => regionFid(d.id)),
  }
}

function addLayers(map: MapLibreMap, layer: Layer, districts: DistrictHighlight[]) {
  const palette = readTokens({ brand: "--brand", stroke: "--status-stroke" })
  const ids = split(districts)
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "vector",
      // MapLibre fetches tiles from its worker, which needs absolute URLs.
      tiles: [absolute(layer.tilePath ?? "/v1/tiles/regions/{z}/{x}/{y}.pbf")],
      minzoom: 0,
      maxzoom: layer.maxZoom ?? 14,
    })
  }
  if (!map.getLayer(OFFICIAL_FILL)) {
    map.addLayer(
      {
        id: OFFICIAL_FILL,
        type: "fill",
        source: SOURCE,
        "source-layer": "district",
        filter: idFilter(ids.all),
        paint: { "fill-color": palette.brand, "fill-opacity": 0.1 },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(MENTION_LINE)) {
    map.addLayer(
      {
        id: MENTION_LINE,
        type: "line",
        source: SOURCE,
        "source-layer": "district",
        filter: idFilter(ids.mention),
        layout: { "line-join": "round" },
        paint: { "line-color": palette.brand, "line-width": 1.75, "line-dasharray": [2.5, 2] },
      },
      slotBefore(map, "areas"),
    )
  }
  if (!map.getLayer(LANDSLIDE_LINE)) {
    map.addLayer(
      {
        id: LANDSLIDE_LINE,
        type: "line",
        source: SOURCE,
        "source-layer": "district",
        filter: idFilter(ids.landslide),
        layout: { "line-join": "round" },
        paint: { "line-color": palette.brand, "line-width": 2.25 },
      },
      slotBefore(map, "areas"),
    )
  }
}

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of [LANDSLIDE_LINE, MENTION_LINE, OFFICIAL_FILL]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface OfficialDistrictsLayerProps {
  /** The `admin-regions` catalogue entry: tile path and zooms. */
  tiles: Layer
  districts: DistrictHighlight[]
}

/**
 * Districts a structured source supports (solid outline, a landslide level in
 * force) or a warning or advisory in force names by exact word (dashed
 * outline, "Mentioned in bulletin"). Nothing else is highlighted.
 */
export function OfficialDistrictsLayer({ tiles, districts }: OfficialDistrictsLayerProps) {
  const map = useMap()
  const current = useRef({ tiles, districts })

  useEffect(() => {
    if (!map) return
    const add = () => addLayers(map, current.current.tiles, current.current.districts)
    add()
    map.on("style.load", add)
    return () => {
      map.off("style.load", add)
      removeLayers(map)
    }
  }, [map])

  useEffect(() => {
    current.current = { ...current.current, districts }
    if (!map || !map.getLayer(OFFICIAL_FILL)) return
    const ids = split(districts)
    map.setFilter(OFFICIAL_FILL, idFilter(ids.all))
    map.setFilter(MENTION_LINE, idFilter(ids.mention))
    map.setFilter(LANDSLIDE_LINE, idFilter(ids.landslide))
  }, [map, districts])

  return null
}
