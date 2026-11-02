import type { MapGeoJSONFeature, MapMouseEvent } from "maplibre-gl"
import { useEffect, useRef, useState } from "react"
import { formatNumber } from "@/lib/format"
import { useMediaQuery } from "@/lib/useMediaQuery"
import { basinStateLabels, topArea } from "../floods/basins"
import { SUBBASIN_FILL } from "../floods/FloodBasinsLayer"
import { floodAreaLabels, floodSeverityLabels, gaugeName } from "../floods/present"
import { FLOOD_AREAS, FLOOD_GAUGES } from "../floods/FloodForecastLayer"
import type { FloodAreaKind, FloodSeverity } from "../floods/types"
import { riskLabels } from "../river-risk/present"
import type { BasinState, ReachRisk, RiverRisk } from "../river-risk/types"
import { regionLabel } from "../location/present"
import type { RegionLevel } from "../location/types"
import { formatLevel, statusLabels } from "../rivers/present"
import { RIVER_STATIONS } from "../rivers/RiverLevelsLayer"
import type { LevelUnit, RiverStatus } from "../rivers/types"
import { reservoirStatusLabels } from "../reservoirs/present"
import { RESERVOIR_AREAS, RESERVOIR_POINTS } from "../reservoirs/ReservoirsLayer"
import type { ReservoirStatus } from "../reservoirs/types"
import { regionVariables, type RegionVariable } from "../regions/present"
import { REGIONS_SOURCE } from "../regions/RegionsLayer"
import type { RegionWeather } from "../regions/types"
import { sampleWind, windText, type WindField } from "../wind/field"
import { hitLayerIds } from "./hitLayers"
import { firstHit, hitOrder, queryMap } from "./hits"
import { useMap } from "./MapContext"

/** Other layers the chip can name, and how. */
export interface HoverLayers {
  /** Small points, hit with a little slack, drawn over gauges and regions. */
  points: string[]
  /** Areas, hit exactly, drawn over regions. */
  areas: string[]
  describe: (
    feature: MapGeoJSONFeature,
  ) => { title: string; local?: string; lines: string[] } | null
}

interface HoverChipProps {
  /** Region weather by region id, and what the choropleth shows. */
  regions?: { variable: RegionVariable | null; lookup: Map<string, RegionWeather> } | null
  /** The wind field on the map, read under the pointer. */
  wind?: WindField | null
  /** Rivers and stretches with their estimated risk. */
  rivers?: { byId: Map<string, RiverRisk>; reaches: Map<number, ReachRisk> } | null
  /** Basin names and states, by feature id; sub-basin states by feature id. */
  basins?: {
    byFid: Map<number, { name: string; state: BasinState }>
    subbasins: Map<number, BasinState>
  } | null
  hazards?: HoverLayers | null
  /** More layers' points, and lines read from their grids at a place. */
  extra?: {
    layers: string[]
    describe: (point: MapGeoJSONFeature) => { title: string; lines: string[] } | null
    lineAt: (lon: number, lat: number) => string[]
  }
}

interface Chip {
  x: number
  y: number
  title: string
  local?: string
  lines: string[]
}

/** Points are easier to hit with a little slack around the pointer. */
const pointSlack = 5

/**
 * A dark chip by the mouse pointer naming what is under it: a gauge and its
 * status, or a region with its Sinhala name and weather, and the forecast
 * wind there when the wind layer is on. Hovered regions are outlined. Only
 * on devices with a mouse; on touch, a tap opens the panel instead.
 */
export function HoverChip({ regions, wind, rivers, basins, hazards, extra }: HoverChipProps) {
  const map = useMap()
  const canHover = useMediaQuery("(hover: hover) and (pointer: fine)")
  const [chip, setChip] = useState<Chip | null>(null)
  const props = useRef({ regions, wind, rivers, basins, hazards, extra })

  useEffect(() => {
    props.current = { regions, wind, rivers, basins, hazards, extra }
  }, [regions, wind, rivers, basins, hazards, extra])

  useEffect(() => {
    if (!map || !canHover) return
    let frame = 0
    let hovered: { sourceLayer: string; id: number } | null = null
    let pending: MapMouseEvent | null = null

    const setHovered = (next: typeof hovered) => {
      if (hovered && (hovered.id !== next?.id || hovered.sourceLayer !== next?.sourceLayer)) {
        if (map.getSource(REGIONS_SOURCE)) {
          map.setFeatureState({ source: REGIONS_SOURCE, ...hovered }, { hover: false })
        }
      }
      if (next && map.getSource(REGIONS_SOURCE)) {
        map.setFeatureState({ source: REGIONS_SOURCE, ...next }, { hover: true })
      }
      hovered = next
    }

    const clear = () => {
      setHovered(null)
      map.getCanvas().style.cursor = ""
      setChip(null)
    }

    const update = () => {
      frame = 0
      const event = pending
      if (!event) return
      const { x, y } = event.point
      const {
        regions: regionInfo,
        wind: field,
        rivers: riverInfo,
        basins: basinInfo,
        hazards: hazardLayers,
        extra: more,
      } = props.current
      const lines: string[] = []

      // The same order a click follows (features/map/hits).
      const hit = firstHit(
        hitOrder(hitLayerIds({ hazards: hazardLayers, observed: more?.layers ?? [] })),
        queryMap(map, event.point, pointSlack),
      )
      const layer = hit?.feature.layer.id
      const isHazard =
        layer !== undefined &&
        (hazardLayers?.points.includes(layer) || hazardLayers?.areas.includes(layer))
      const hazard = isHazard ? hit?.feature : undefined
      const point = hit?.group === "point" && !isHazard ? hit.feature : undefined
      const reservoir =
        layer === RESERVOIR_POINTS || layer === RESERVOIR_AREAS ? hit?.feature : undefined
      const area = layer === FLOOD_AREAS ? hit?.feature : undefined
      const river = hit?.group === "river" ? hit.feature : undefined
      const past = hit?.group === "past" ? hit.feature : undefined
      const basin = hit?.group === "basin" ? hit.feature : undefined
      const region = hit?.group === "region" ? hit.feature : undefined

      setHovered(
        region && typeof region.id === "number" && region.sourceLayer
          ? { sourceLayer: region.sourceLayer, id: region.id }
          : null,
      )
      map.getCanvas().style.cursor = hit ? "pointer" : ""

      let title = ""
      let local: string | undefined
      if (point?.layer.id === RIVER_STATIONS) {
        title = String(point.properties.name ?? "River gauge")
        const status = point.properties.status as RiverStatus
        const level = point.properties.waterLevel
        lines.push(
          `${statusLabels[status] ?? "River gauge"}` +
            (typeof level === "number"
              ? ` · ${formatLevel(level, point.properties.unit as LevelUnit)}`
              : ""),
        )
      } else if (reservoir) {
        title = String(reservoir.properties.name ?? "Reservoir")
        const status = reservoir.properties.status as ReservoirStatus
        lines.push(`Reservoir: ${(reservoirStatusLabels[status] ?? "No data").toLowerCase()}`)
      } else if (point?.layer.id === FLOOD_GAUGES) {
        title = gaugeName({
          gaugeId: String(point.properties.gaugeId),
          siteName: point.properties.siteName as string | undefined,
          river: point.properties.river as string | undefined,
        })
        lines.push(
          `Flood forecast: ${floodSeverityLabels[point.properties.severity as FloodSeverity] ?? "no data"}`,
        )
        if (point.properties.qualityVerified === false) lines.push("Lower confidence")
      } else if (hazard) {
        const described = hazardLayers?.describe(hazard)
        if (described) {
          title = described.title
          local = described.local
          lines.push(...described.lines)
        }
      } else if (area) {
        const top = topArea(queryMap(map, event.point, pointSlack)([FLOOD_AREAS], false)) ?? area
        title = floodAreaLabels[top.properties.kind as FloodAreaKind] ?? "Flood area"
        lines.push("Google flood forecast")
      } else if (river) {
        const known = riverInfo?.byId.get(String(river.properties.river))
        const reach = typeof river.id === "number" ? riverInfo?.reaches.get(river.id) : undefined
        title = known?.name ?? "Stream"
        if (reach) {
          lines.push(reach.name)
          lines.push(`${riskLabels[reach.level]} · Estimate`)
        } else if (known) {
          lines.push("Tributary, not assessed")
        }
      } else if (past) {
        title = String(past.properties.name ?? "Past flood")
        lines.push("Past event, not a forecast")
      } else if (basin) {
        const fid = typeof basin.id === "number" ? basin.id : -1
        const isSubbasin = basin.layer.id === SUBBASIN_FILL
        const parent = isSubbasin
          ? basinInfo?.byFid.get(Number(basin.properties.basin))
          : basinInfo?.byFid.get(fid)
        title = parent?.name ?? String(basin.properties.name ?? "River basin")
        const state: BasinState = isSubbasin
          ? (basinInfo?.subbasins.get(fid) ?? "no_forecast")
          : (parent?.state ?? "no_forecast")
        lines.push(`${isSubbasin ? "Sub-basin" : "Basin"}: ${basinStateLabels[state]}`)
      } else if (region) {
        const level = region.properties.level as RegionLevel
        title = regionLabel({ level, name: String(region.properties.name ?? "") })
        local = (region.properties.nameSi as string | undefined) ?? undefined
        const weather = regionInfo?.lookup.get(String(region.properties.id))
        const variable = regionInfo?.variable
        if (weather && variable) {
          const spec = regionVariables[variable]
          const value = spec.value(weather)
          if (value !== null) {
            lines.push(`${spec.label}: ${formatNumber(value, spec.decimals)} ${spec.unit}`)
          }
        }
      }
      const described = point ? more?.describe(point) : null
      if (described) {
        title = described.title
        lines.push(...described.lines)
      }
      lines.push(...(more?.lineAt(event.lngLat.lng, event.lngLat.lat) ?? []))
      if (field) {
        const sample = sampleWind(field, event.lngLat.lng, event.lngLat.lat)
        if (sample) lines.push(`Wind ${windText(sample[0], sample[1])}`)
      }
      if (!title && lines.length === 0) {
        setChip(null)
        return
      }
      setChip({ x, y, title, local, lines })
    }

    const move = (event: MapMouseEvent) => {
      pending = event
      if (!frame) frame = requestAnimationFrame(update)
    }
    const leave = () => {
      pending = null
      if (frame) cancelAnimationFrame(frame)
      frame = 0
      clear()
    }

    map.on("mousemove", move)
    map.on("mouseout", leave)
    map.on("movestart", leave)
    return () => {
      map.off("mousemove", move)
      map.off("mouseout", leave)
      map.off("movestart", leave)
      if (frame) cancelAnimationFrame(frame)
      clear()
    }
  }, [map, canHover])

  if (!chip) return null
  const flip = chip.x > window.innerWidth - 280
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none fixed z-20 max-w-64 rounded-xl bg-tooltip px-3 py-2 text-xs text-tooltip-foreground shadow-panel"
      style={{
        left: chip.x,
        top: chip.y,
        transform: `translate(${flip ? "calc(-100% - 14px)" : "14px"}, 14px)`,
      }}
    >
      {chip.title && (
        <p className="font-medium">
          {chip.title}
          {chip.local && (
            <span className="ml-1.5 font-normal opacity-75" lang="si">
              {chip.local}
            </span>
          )}
        </p>
      )}
      {chip.lines.map((line) => (
        <p key={line} className="numeric mt-0.5 opacity-85">
          {line}
        </p>
      ))}
    </div>
  )
}
