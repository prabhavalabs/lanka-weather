import type { ExpressionSpecification, Map as MapLibreMap } from "maplibre-gl"
import { useEffect, useMemo, useRef } from "react"
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
/** The focus mask for the warnings panel: grey over the other districts, an outline round these. */
const FOCUS_VEIL = "official-districts-focus-veil"
const FOCUS_HALO = "official-districts-focus-halo"
const FOCUS_LINE = "official-districts-focus-line"

const noFocus: string[] = []

function absolute(path: string) {
  return /^https?:/.test(path) ? path : `${window.location.origin}${path}`
}

function idFilter(ids: number[]): ExpressionSpecification {
  return ["in", ["id"], ["literal", ids]] as unknown as ExpressionSpecification
}

/** Every district but the focused ones; nothing at all when none is focused. */
function veilFilter(ids: number[]): ExpressionSpecification {
  if (ids.length === 0) return ["boolean", false] as unknown as ExpressionSpecification
  return ["!", idFilter(ids)] as unknown as ExpressionSpecification
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

function addLayers(
  map: MapLibreMap,
  layer: Layer,
  districts: DistrictHighlight[],
  focus: number[],
) {
  const palette = readTokens({
    brand: "--brand",
    stroke: "--status-stroke",
    veil: "--map-focus-veil",
    line: "--foreground",
    halo: "--card",
  })
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
  // Like features/map/FocusMask, over areas and rivers and under labels and gauges.
  // Antialiasing off, so neighbouring districts meet without darker seams.
  if (!map.getLayer(FOCUS_VEIL)) {
    map.addLayer(
      {
        id: FOCUS_VEIL,
        type: "fill",
        source: SOURCE,
        "source-layer": "district",
        filter: veilFilter(focus),
        paint: { "fill-color": palette.veil, "fill-antialias": false },
      },
      slotBefore(map, "particles"),
    )
  }
  for (const [id, paint] of [
    [FOCUS_HALO, { "line-color": palette.halo, "line-width": 5, "line-opacity": 0.9 }],
    [FOCUS_LINE, { "line-color": palette.line, "line-width": 2 }],
  ] as const) {
    if (map.getLayer(id)) continue
    map.addLayer(
      {
        id,
        type: "line",
        source: SOURCE,
        "source-layer": "district",
        filter: idFilter(focus),
        layout: { "line-join": "round" },
        paint,
      },
      slotBefore(map, "particles"),
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
  for (const id of [
    FOCUS_LINE,
    FOCUS_HALO,
    FOCUS_VEIL,
    LANDSLIDE_LINE,
    MENTION_LINE,
    OFFICIAL_FILL,
  ]) {
    if (map.getLayer(id)) map.removeLayer(id)
  }
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface OfficialDistrictsLayerProps {
  /** The `admin-regions` catalogue entry: tile path and zooms. */
  tiles: Layer
  districts: DistrictHighlight[]
  /** Districts the open warnings are about: the map goes grey around them. Empty for none. */
  focus?: string[]
}

/**
 * Districts a structured source supports (solid outline, a landslide level in
 * force) or a warning or advisory in force names by exact word (dashed
 * outline, "Mentioned in bulletin"). Nothing else is highlighted.
 */
export function OfficialDistrictsLayer({
  tiles,
  districts,
  focus = noFocus,
}: OfficialDistrictsLayerProps) {
  const map = useMap()
  const focusFids = useMemo(() => focus.map(regionFid), [focus])
  const current = useRef({ tiles, districts, focusFids })

  useEffect(() => {
    if (!map) return
    const add = () =>
      addLayers(map, current.current.tiles, current.current.districts, current.current.focusFids)
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

  useEffect(() => {
    current.current = { ...current.current, focusFids }
    if (!map || !map.getLayer(FOCUS_VEIL)) return
    map.setFilter(FOCUS_VEIL, veilFilter(focusFids))
    map.setFilter(FOCUS_HALO, idFilter(focusFids))
    map.setFilter(FOCUS_LINE, idFilter(focusFids))
  }, [map, focusFids])

  return null
}
