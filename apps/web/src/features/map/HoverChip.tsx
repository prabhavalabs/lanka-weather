import type { MapGeoJSONFeature, MapMouseEvent } from "maplibre-gl"
import { useEffect, useRef, useState } from "react"
import { formatNumber } from "@/lib/format"
import { useMediaQuery } from "@/lib/useMediaQuery"
import { floodSeverityLabels, gaugeName } from "../floods/present"
import { FLOOD_GAUGES } from "../floods/FloodForecastLayer"
import type { FloodSeverity } from "../floods/types"
import { regionLabel } from "../location/present"
import type { RegionLevel } from "../location/types"
import { statusLabels } from "../rivers/present"
import { RIVER_STATIONS } from "../rivers/RiverLevelsLayer"
import type { RiverStatus } from "../rivers/types"
import { regionVariables, type RegionVariable } from "../regions/present"
import { REGIONS_FILL, REGIONS_SOURCE } from "../regions/RegionsLayer"
import type { RegionWeather } from "../regions/types"
import { sampleWind, windText, type WindField } from "../wind/field"
import { useMap } from "./MapContext"

interface HoverChipProps {
  /** Region weather by region id, and what the choropleth shows. */
  regions?: { variable: RegionVariable | null; lookup: Map<string, RegionWeather> } | null
  /** The wind field on the map, read under the pointer. */
  wind?: WindField | null
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
export function HoverChip({ regions, wind }: HoverChipProps) {
  const map = useMap()
  const canHover = useMediaQuery("(hover: hover) and (pointer: fine)")
  const [chip, setChip] = useState<Chip | null>(null)
  const props = useRef({ regions, wind })

  useEffect(() => {
    props.current = { regions, wind }
  }, [regions, wind])

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
      const { regions: regionInfo, wind: field } = props.current
      const lines: string[] = []

      const pointLayers = [RIVER_STATIONS, FLOOD_GAUGES].filter((id) => map.getLayer(id))
      const point = pointLayers.length
        ? map.queryRenderedFeatures(
            [
              [x - pointSlack, y - pointSlack],
              [x + pointSlack, y + pointSlack],
            ],
            { layers: pointLayers },
          )[0]
        : undefined
      const region: MapGeoJSONFeature | undefined =
        !point && map.getLayer(REGIONS_FILL)
          ? map.queryRenderedFeatures(event.point, { layers: [REGIONS_FILL] })[0]
          : undefined

      setHovered(
        region && typeof region.id === "number" && region.sourceLayer
          ? { sourceLayer: region.sourceLayer, id: region.id }
          : null,
      )
      map.getCanvas().style.cursor = point || region ? "pointer" : ""

      let title = ""
      let local: string | undefined
      if (point?.layer.id === RIVER_STATIONS) {
        title = String(point.properties.name ?? "River gauge")
        const status = point.properties.status as RiverStatus
        const level = point.properties.waterLevel
        lines.push(
          `${statusLabels[status] ?? "River gauge"}` +
            (typeof level === "number" ? ` · ${level.toFixed(2)} ${point.properties.unit}` : ""),
        )
      } else if (point?.layer.id === FLOOD_GAUGES) {
        title = gaugeName({
          gaugeId: String(point.properties.gaugeId),
          siteName: point.properties.siteName as string | undefined,
          river: point.properties.river as string | undefined,
        })
        lines.push(
          `Flood forecast: ${floodSeverityLabels[point.properties.severity as FloodSeverity] ?? "unknown"}`,
        )
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
