import type { Map as MapLibreMap, MapMouseEvent, PointLike } from "maplibre-gl"
import { MapPinIcon, WavesIcon, ZoomInIcon } from "lucide-react"
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react"
import { toast } from "sonner"
import { AppShell } from "@/components/AppShell"
import { AppSidebar } from "@/components/AppSidebar"
import { DetailPlaceholder } from "@/components/DetailPlaceholder"
import { MobileBar } from "@/components/MobileBar"
import { SegmentedBar } from "@/components/SegmentedBar"
import { Button } from "@/components/ui/button"
import { Toaster } from "@/components/ui/sonner"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { TooltipProvider } from "@/components/ui/tooltip"
import { AdminLevelSelector } from "@/components/weather/AdminLevelSelector"
import { SearchTrigger } from "@/components/weather/SearchTrigger"
import { isNotReady } from "@/lib/api"
import type { Layer } from "@/lib/layers"
import { desktopQuery, useMediaQuery } from "@/lib/useMediaQuery"
import { useTheme } from "@/lib/theme"
import { useLocalStorage } from "@/lib/useLocalStorage"
import { FloodForecastLayer, FLOOD_GAUGES } from "./features/floods/FloodForecastLayer"
import { FloodGaugePanel } from "./features/floods/FloodGaugePanel"
import { FloodLegend } from "./features/floods/FloodLegend"
import { countBySeverity, floodSummary, isGauge } from "./features/floods/present"
import { useFloodForecast } from "./features/floods/queries"
import type { FloodAreaFeature } from "./features/floods/types"
import { hazardDefaults, useHazards } from "./features/hazards/useHazards"
import { LayerPanel } from "./features/layers/LayerPanel"
import { count, layerStatus, type LayerStatus } from "./features/layers/status"
import { useLayers } from "./features/layers/useLayers"
import { LocationMarker } from "./features/location/LocationMarker"
import { useSelectionLocation } from "./features/location/queries"
import {
  inPointBounds,
  roundCoordinate,
  sameSelection,
  type Selection,
} from "./features/location/selection"
import { useUrlSelection } from "./features/location/useUrlSelection"
import { HoverChip } from "./features/map/HoverChip"
import { MapView } from "./features/map/MapView"
import { clearCentreOffset, clearPadding, inClearArea, layoutInsets } from "./features/map/reveal"
import { useMapClick } from "./features/map/useMapClick"
import { useMapZoom } from "./features/map/useMapZoom"
import {
  levelMinZoom,
  regionLevelOf,
  regionVariables,
  type RegionVariable,
} from "./features/regions/present"
import { useGnWeather, useRegionDetail, useRegionWeather } from "./features/regions/queries"
import { RegionLegend } from "./features/regions/RegionLegend"
import { RegionsLayer, REGIONS_FILL } from "./features/regions/RegionsLayer"
import type { RegionLevel } from "./features/regions/types"
import { countByStatus, statusColors, statusLabels, statusOrder } from "./features/rivers/present"
import { useRiverLevels } from "./features/rivers/queries"
import { RiverLegend } from "./features/rivers/RiverLegend"
import { RiverLevelsLayer, RIVER_STATIONS } from "./features/rivers/RiverLevelsLayer"
import { StationPanel } from "./features/rivers/StationPanel"
import { placeZoom } from "./features/search/present"
import type { SearchChoice } from "./features/search/types"
import { countValues } from "./features/weather-grid/field"
import { GridLayer } from "./features/weather-grid/GridLayer"
import { GridLegend } from "./features/weather-grid/GridLegend"
import { useCurrentGrid } from "./features/weather-grid/queries"
import {
  gridVariableOrder,
  gridVariables,
  type GridVariable,
} from "./features/weather-grid/variables"
import { useWindForecast } from "./features/wind/useWindForecast"
import { WindTimeline } from "./features/wind/WindTimeline"

// Loaded when first needed, to keep the first download small.
const loadSearch = () => import("./features/search/SearchPalette")
const SearchPalette = lazy(loadSearch)
const LocationPanel = lazy(() => import("./features/location/LocationPanel"))
const WindLayer = lazy(() => import("./features/wind/WindLayer"))

const RIVER_LEVELS = "river-levels"
const FLOOD_FORECAST = "flood-forecast"
const TEMPERATURE = "temperature"
const WIND = "wind"
const ADMIN_REGIONS = "admin-regions"

/** Layers on at the first visit. */
const defaultLayers = [ADMIN_REGIONS, RIVER_LEVELS, FLOOD_FORECAST, WIND, ...hazardDefaults]

type RegionFill = RegionVariable | "none"

const levelPlural: Record<RegionLevel, string> = {
  province: "province",
  district: "district",
  ds: "DS division",
  gn: "GN division",
}

/** Gauges are small: clicks and hovers count within a few pixels. */
const pointSlack = 6

export function App() {
  const [enabled, setEnabled] = useLocalStorage<string[]>("lanka-weather:layers", defaultLayers)
  const [sidebarCollapsed, setSidebarCollapsed] = useLocalStorage(
    "lanka-weather:sidebar-collapsed",
    false,
  )
  const [level, setLevel] = useLocalStorage<RegionLevel>("lanka-weather:region-level", "district")
  const [regionFill, setRegionFill] = useLocalStorage<RegionFill>(
    "lanka-weather:region-fill",
    "temperature",
  )
  const [gridVariable, setGridVariable] = useLocalStorage<GridVariable>(
    "lanka-weather:grid-variable",
    "temperature",
  )
  const [selection, setSelection] = useUrlSelection()
  const [map, setMap] = useState<MapLibreMap | null>(null)
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchLoaded, setSearchLoaded] = useState(false)
  const [gnDistricts, setGnDistricts] = useState<string[]>([])
  const zoom = useMapZoom(map)
  const desktop = useMediaQuery(desktopQuery, true)
  const { resolved: theme } = useTheme()

  // The catalogue says which layers exist and how often their data changes.
  const catalogue = useLayers()
  const available = useCallback(
    (id: string): Layer | undefined =>
      catalogue.data?.find((layer) => layer.id === id && layer.status === "available"),
    [catalogue.data],
  )
  const riverLayer = available(RIVER_LEVELS)
  const floodLayer = available(FLOOD_FORECAST)
  const gridLayer = available(TEMPERATURE)
  const windLayer = available(WIND)
  const regionsLayer = available(ADMIN_REGIONS)
  const on = (id: string) => enabled.includes(id) && available(id) !== undefined
  const riversOn = on(RIVER_LEVELS)
  const floodsOn = on(FLOOD_FORECAST)
  const gridOn = on(TEMPERATURE)
  const windOn = on(WIND)
  const regionsOn = on(ADMIN_REGIONS)

  // Region colours give way to the weather field, which shows the same weather in more detail.
  const fillVariable: RegionVariable | null =
    regionsOn && regionFill !== "none" && !gridOn ? regionFill : null

  const rivers = useRiverLevels(
    riverLayer,
    riversOn || selection?.kind === "station" || searchLoaded,
  )
  const floods = useFloodForecast(floodLayer, floodsOn || selection?.kind === "gauge")
  const grid = useCurrentGrid(gridLayer, gridOn)
  const wind = useWindForecast(windLayer, windOn)
  const levelWeather = useRegionWeather(level, {
    enabled: regionsOn && level !== "gn",
    timings: gridLayer,
  })
  const gnWeather = useGnWeather(
    gnDistricts,
    regionsOn && level === "gn" && zoom >= levelMinZoom.gn,
    gridLayer,
  )
  const regionList = useMemo(
    () => (level === "gn" ? gnWeather.regions : (levelWeather.data?.regions ?? [])),
    [level, gnWeather.regions, levelWeather.data],
  )
  const regionLookup = useMemo(
    () => new Map(regionList.map((region) => [region.id, region])),
    [regionList],
  )
  const regionValidAt = level === "gn" ? gnWeather.validAt : levelWeather.data?.validAt

  const addGnDistricts = useCallback((districts: string[]) => {
    setGnDistricts((current) => {
      const next = [...new Set([...current, ...districts])].sort()
      return next.length === current.length ? current : next
    })
  }, [])

  // What is selected, and where.
  const regionId = selection?.kind === "region" ? selection.id : null
  const regionDetail = useRegionDetail(regionId)
  const location = useSelectionLocation(selection)
  const station =
    selection?.kind === "station"
      ? rivers.data?.features.find((feature) => feature.id === selection.id)
      : undefined
  const gauge =
    selection?.kind === "gauge"
      ? floods.data?.features.filter(isGauge).find((feature) => feature.id === selection.id)
      : undefined

  const enable = useCallback(
    (id: string) => setEnabled((current) => (current.includes(id) ? current : [...current, id])),
    [setEnabled],
  )

  const toggle = (id: string) => {
    const turningOn = !enabled.includes(id)
    if (!turningOn && id === RIVER_LEVELS && selection?.kind === "station") setSelection(null)
    if (!turningOn && id === FLOOD_FORECAST && selection?.kind === "gauge") setSelection(null)
    setEnabled((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    )
  }

  // Camera: keep the selected place clear of the cards floating over the map.
  const insets = useCallback(
    () =>
      layoutInsets({
        desktop,
        sidebarCollapsed,
        detailOpen: true,
        viewportHeight: window.innerHeight,
      }),
    [desktop, sidebarCollapsed],
  )
  const reveal = useCallback(
    (lngLat: [number, number], options: { zoom?: number } = {}) => {
      if (!map) return
      const canvas = map.getCanvas()
      const width = canvas.clientWidth
      const height = canvas.clientHeight
      const clear = insets()
      const offset = clearCentreOffset(width, height, clear)
      if (options.zoom !== undefined) {
        map.flyTo({
          center: lngLat,
          zoom: Math.max(map.getZoom(), options.zoom),
          offset,
          duration: 1600,
        })
      } else if (!inClearArea(map.project(lngLat), width, height, clear)) {
        map.easeTo({ center: lngLat, offset, duration: 600 })
      }
    },
    [map, insets],
  )
  const fitBox = useCallback(
    (bbox: [number, number, number, number]) => {
      map?.fitBounds(
        [
          [bbox[0], bbox[1]],
          [bbox[2], bbox[3]],
        ],
        { padding: clearPadding(insets()), duration: 1400, maxZoom: 13 },
      )
    },
    [map, insets],
  )

  // A camera move waiting for its target's position (a region's label point, a gauge).
  const pending = useRef<{ selection: Selection; target: "reveal" | "zoom" | "fit" } | null>(
    selection ? { selection, target: "fit" } : null,
  )
  const pendingCamera = useCallback((next: Selection, target: "reveal" | "zoom" | "fit") => {
    pending.current = { selection: next, target }
  }, [])

  useEffect(() => {
    const move = pending.current
    if (!map || !move || !selection || !sameSelection(move.selection, selection)) return
    let lngLat: [number, number] | null = null
    let bbox: [number, number, number, number] | null = null
    if (selection.kind === "point") lngLat = [selection.lon, selection.lat]
    else if (selection.kind === "region") {
      if (!regionDetail.data) return
      lngLat = regionDetail.data.labelPoint
      bbox = regionDetail.data.bbox
    } else if (selection.kind === "station") {
      if (!station) return
      lngLat = station.geometry.coordinates
    } else if (selection.kind === "gauge") {
      if (!gauge) return
      lngLat = gauge.geometry.coordinates
    }
    pending.current = null
    if (!lngLat) return
    if (move.target === "fit" && bbox) fitBox(bbox)
    else if (move.target === "fit" || move.target === "zoom") reveal(lngLat, { zoom: 10 })
    else reveal(lngLat)
  }, [map, selection, regionDetail.data, station, gauge, reveal, fitBox])

  const openStation = useCallback(
    (id: string) => {
      enable(RIVER_LEVELS)
      const next: Selection = { kind: "station", id }
      pendingCamera(next, "reveal")
      setSelection(next)
    },
    [enable, pendingCamera, setSelection],
  )
  const openGauge = useCallback(
    (id: string) => {
      enable(FLOOD_FORECAST)
      const next: Selection = { kind: "gauge", id }
      pendingCamera(next, "reveal")
      setSelection(next)
    },
    [enable, pendingCamera, setSelection],
  )

  const revealPoint = useCallback((lngLat: [number, number]) => reveal(lngLat), [reveal])
  const hazards = useHazards({
    map,
    available,
    isOn: on,
    enable,
    selection,
    setSelection,
    fitBox,
    reveal: revealPoint,
  })

  // Clicks: a gauge, else a hazard, else a region (weather at its label point), else the point itself.
  useMapClick(map, (event: MapMouseEvent) => {
    if (!map) return
    const { x, y } = event.point
    const box: [PointLike, PointLike] = [
      [x - pointSlack, y - pointSlack],
      [x + pointSlack, y + pointSlack],
    ]
    const pointLayers = [RIVER_STATIONS, FLOOD_GAUGES].filter((id) => map.getLayer(id))
    const hit = pointLayers.length
      ? map.queryRenderedFeatures(box, { layers: pointLayers })[0]
      : null
    if (hit?.layer.id === RIVER_STATIONS) return openStation(String(hit.properties.id))
    if (hit?.layer.id === FLOOD_GAUGES) return openGauge(String(hit.properties.gaugeId))
    if (hazards.handleClick(event)) return

    const region = map.getLayer(REGIONS_FILL)
      ? map.queryRenderedFeatures(event.point, { layers: [REGIONS_FILL] })[0]
      : null
    if (region && typeof region.properties.id === "string") {
      const next: Selection = { kind: "region", id: region.properties.id }
      pendingCamera(next, "reveal")
      return setSelection(next)
    }

    const { lat, lng } = event.lngLat
    if (!inPointBounds(lat, lng)) {
      toast("Weather is available over Sri Lanka and its coastal waters only.", {
        id: "outside",
      })
      return
    }
    const next: Selection = { kind: "point", lat: roundCoordinate(lat), lon: roundCoordinate(lng) }
    pendingCamera(next, "reveal")
    setSelection(next)
  })

  const choose = useCallback(
    (choice: SearchChoice) => {
      if (choice.type === "place") {
        const next: Selection = {
          kind: "point",
          lat: roundCoordinate(choice.lat),
          lon: roundCoordinate(choice.lon),
          name: choice.name,
        }
        pending.current = null
        setSelection(next)
        reveal([choice.lon, choice.lat], { zoom: placeZoom[choice.kind] ?? 12 })
      } else if (choice.type === "region") {
        if (enabled.includes(ADMIN_REGIONS)) setLevel(choice.level)
        const next: Selection = { kind: "region", id: choice.id }
        setSelection(next)
        if (choice.bbox) {
          pending.current = null
          fitBox(choice.bbox)
        } else pendingCamera(next, "fit")
      } else {
        enable(RIVER_LEVELS)
        pending.current = null
        setSelection({ kind: "station", id: choice.id })
        reveal([choice.lon, choice.lat], { zoom: 11 })
      }
    },
    [enabled, enable, fitBox, pendingCamera, reveal, setLevel, setSelection],
  )

  // Search: ⌘K or Ctrl+K anywhere; the palette's code loads on first use.
  const openSearch = useCallback(() => {
    setSearchLoaded(true)
    setSearchOpen(true)
  }, [])
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() === "k" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault()
        setSearchLoaded(true)
        setSearchOpen((open) => !open)
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [])

  // Escape closes the detail panel when no dialog is open.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || !selection) return
      if (document.querySelector('[role="dialog"][data-state="open"]')) return
      setSelection(null)
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [selection, setSelection])

  const close = useCallback(() => setSelection(null), [setSelection])

  // Layer list: one-line status, details under layers that are on, stale marks.
  const statuses: Record<string, LayerStatus> = { ...hazards.statuses }
  const details: Record<string, ReactNode> = { ...hazards.details }

  const riverCounts = rivers.data
    ? countByStatus(rivers.data.features.map((feature) => feature.properties))
    : undefined
  if (riverLayer) {
    const raised = riverCounts
      ? riverCounts.alert + riverCounts.minor_flood + riverCounts.major_flood
      : 0
    statuses[RIVER_LEVELS] = layerStatus({
      layer: riverLayer,
      parts: rivers.data
        ? [
            count(rivers.data.features.length, "station"),
            raised === 0 ? "none above alert" : `${raised} above alert`,
          ]
        : [],
      observedAt: rivers.data?.meta.fetchedAt,
      error: rivers.error,
      pending: rivers.isPending,
    })
  }
  if (riverCounts) {
    details[RIVER_LEVELS] = (
      <div className="flex flex-col gap-3">
        <SegmentedBar
          label="Stations by flood status"
          segments={statusOrder.map((status) => ({
            label: statusLabels[status],
            value: riverCounts[status],
            color: statusColors[status],
          }))}
        />
        <RiverLegend bare counts={riverCounts} />
      </div>
    )
  }

  const floodGauges = floods.data?.features.filter(isGauge) ?? []
  const floodCounts = floods.data ? countBySeverity(floodGauges) : undefined
  if (floodLayer) {
    const issued = floodGauges
      .map((feature) => feature.properties.issuedAt)
      .sort()
      .at(-1)
    statuses[FLOOD_FORECAST] = layerStatus({
      layer: floodLayer,
      parts: floods.data
        ? floods.data.meta.configured
          ? [floodSummary(floods.data)]
          : ["Not configured on this server"]
        : [],
      observedAt: issued ?? floods.data?.meta.fetchedAt,
      error: floods.error,
      pending: floods.isPending,
    })
    if (floods.data && floodGauges.length === 0 && floods.data.meta.configured) {
      statuses[FLOOD_FORECAST] = {
        state: "waiting",
        summary: "No forecasts yet · collected hourly",
        stale: false,
      }
    }
  }
  if (floodCounts) details[FLOOD_FORECAST] = <FloodLegend bare counts={floodCounts} />

  if (gridLayer) {
    const values = grid.data?.values[gridVariables[gridVariable].key]
    statuses[TEMPERATURE] = layerStatus({
      layer: gridLayer,
      parts: grid.data
        ? [count(countValues(values), "grid point"), gridVariables[gridVariable].label]
        : [],
      observedAt: grid.data?.validAt,
      error: grid.error,
      pending: grid.isPending,
    })
  }
  details[TEMPERATURE] = (
    <div className="flex flex-col gap-3">
      <ToggleGroup
        type="single"
        size="sm"
        value={gridVariable}
        onValueChange={(value) => {
          if (value) setGridVariable(value as GridVariable)
        }}
        aria-label="Weather field shows"
        className="w-full"
      >
        {gridVariableOrder.map((variable) => (
          <ToggleGroupItem
            key={variable}
            value={variable}
            aria-label={gridVariables[variable].label}
            title={gridVariables[variable].label}
            className="flex-1"
          >
            {gridVariables[variable].short}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      <GridLegend variable={gridVariable} bare className="md:hidden" />
      <p className="text-xs text-muted-foreground">
        Model values on a 0.2° grid (about 22 km), smoothed between grid points. Weather data by
        Open-Meteo.com (CC BY 4.0).
      </p>
    </div>
  )

  if (windLayer) {
    const field = wind.field
    statuses[WIND] = layerStatus({
      layer: windLayer,
      parts: field ? ["NOAA GFS, 10 m", `${wind.steps.length} forecast times`] : [],
      observedAt: field?.runAt,
      ageLabel: "run",
      error: wind.query.error,
      pending: wind.query.isPending,
    })
  }
  const windTimeline = windOn && (
    <WindTimeline
      steps={wind.steps}
      index={wind.index}
      onIndexChange={wind.setIndex}
      playing={wind.playing}
      onPlayingChange={wind.setPlaying}
      runAt={wind.field?.runAt}
      loading={wind.loading}
      now={wind.now}
    />
  )
  details[WIND] = (
    <p className="text-xs text-muted-foreground">
      Particles follow the forecast wind 10 m above ground; choose the time below the map. Speeds
      can differ a little from the weather panel, which uses Open-Meteo.
    </p>
  )

  if (regionsLayer) {
    const levelCount = regionList.length
    statuses[ADMIN_REGIONS] = layerStatus({
      layer: gridLayer ?? regionsLayer,
      parts:
        levelCount > 0
          ? [
              count(levelCount, levelPlural[level]),
              fillVariable ? regionVariables[fillVariable].label.toLowerCase() : "boundaries",
            ]
          : [level === "gn" && zoom < levelMinZoom.gn ? "Zoom in for GN divisions" : "Boundaries"],
      observedAt: regionValidAt,
      error: level === "gn" ? gnWeather.error : levelWeather.error,
      pending: false,
    })
  }
  details[ADMIN_REGIONS] = (
    <div className="flex flex-col gap-3">
      <AdminLevelSelector level={level} onLevelChange={setLevel} className="md:hidden" />
      <div className="flex flex-col gap-1.5">
        <span className="text-xs text-muted-foreground">Colour regions by</span>
        <ToggleGroup
          type="single"
          size="sm"
          value={regionFill}
          onValueChange={(value) => {
            if (value) setRegionFill(value as RegionFill)
          }}
          aria-label="Colour regions by"
          className="w-full"
          disabled={gridOn}
        >
          <ToggleGroupItem value="none" className="flex-1">
            None
          </ToggleGroupItem>
          {(Object.keys(regionVariables) as RegionVariable[]).map((variable) => (
            <ToggleGroupItem
              key={variable}
              value={variable}
              aria-label={regionVariables[variable].label}
              className="flex-1"
            >
              {regionVariables[variable].short}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        {gridOn && (
          <p className="text-xs text-muted-foreground">
            Region colours are hidden while the weather field is on.
          </p>
        )}
      </div>
      {fillVariable && (
        <RegionLegend variable={fillVariable} level={level} bare className="md:hidden" />
      )}
    </div>
  )

  const summaries = Object.fromEntries(
    Object.entries(statuses).map(([id, status]) => [id, status.summary]),
  )
  const stale = Object.fromEntries(
    Object.entries(statuses).map(([id, status]) => [id, status.stale]),
  )
  const activeCount = enabled.filter((id) => available(id)).length

  const layerList = (
    <LayerPanel
      enabled={enabled}
      onToggle={toggle}
      summaries={summaries}
      details={details}
      stale={stale}
    />
  )
  const smallPrint = (
    <div className="flex flex-col gap-1">
      <span>
        Official warnings come from the Department of Meteorology and the DMC. Region and point
        alerts here are estimates.
      </span>
    </div>
  )

  // The notice under the toolbar: a level that needs a closer zoom, or data still on its way.
  const waiting = [
    gridOn && isNotReady(grid.error),
    windOn && isNotReady(wind.query.error),
    regionsOn && isNotReady(level === "gn" ? gnWeather.error : levelWeather.error),
  ].some(Boolean)
  let notice: ReactNode = null
  if (regionsOn && zoom < levelMinZoom[level] && map && !hazards.farAway && zoom >= 4.5) {
    notice = (
      <div className="pointer-events-auto flex items-center gap-2 rounded-full border border-border bg-card/95 py-1 pr-1 pl-3.5 text-sm shadow-panel backdrop-blur-md">
        <span>
          Zoom in to see {level === "gn" ? "GN" : level === "ds" ? "DS" : levelPlural[level]}{" "}
          divisions
        </span>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => map.easeTo({ zoom: levelMinZoom[level] + 0.3, duration: 800 })}
        >
          <ZoomInIcon />
          Zoom in
        </Button>
      </div>
    )
  } else if (waiting) {
    notice = (
      <p
        role="status"
        className="pointer-events-auto rounded-full border border-border bg-card/95 px-3.5 py-1.5 text-sm text-muted-foreground shadow-panel backdrop-blur-md"
      >
        Weather data is still loading. Layers fill in as the collectors run.
      </p>
    )
  }

  // The detail panel for the selection.
  let detail: ReactNode = hazards.detail
  let detailLabel = hazards.detailLabel ?? "Details"
  if (selection?.kind === "point" || selection?.kind === "region") {
    detailLabel = "Weather here"
    detail = (
      <Suspense
        fallback={<DetailPlaceholder icon={<MapPinIcon />} title="Weather" onClose={close} />}
      >
        <LocationPanel
          selection={selection}
          onClose={close}
          onSelectStation={openStation}
          onSelectGauge={openGauge}
        />
      </Suspense>
    )
  } else if (selection?.kind === "station") {
    detailLabel = "River gauge"
    detail = station ? (
      <StationPanel station={station.properties} onClose={close} />
    ) : (
      <DetailPlaceholder
        icon={<WavesIcon />}
        title="River gauge"
        message={
          rivers.isPending ? undefined : "This gauge is not in the latest river level readings."
        }
        onClose={close}
      />
    )
  } else if (selection?.kind === "gauge") {
    detailLabel = "Flood forecast gauge"
    const areas = (floods.data?.features ?? []).filter(
      (feature): feature is FloodAreaFeature =>
        !isGauge(feature) && feature.properties.gaugeId === selection.id,
    )
    detail = gauge ? (
      <FloodGaugePanel gauge={gauge.properties} areas={areas} onClose={close} />
    ) : (
      <DetailPlaceholder
        icon={<WavesIcon />}
        title="Flood forecast gauge"
        message={floods.isPending ? undefined : "This gauge is not in the latest flood forecast."}
        onClose={close}
      />
    )
  }

  // Legends for the layers that are on (wide screens; phones show them in the layer sheet).
  // Gauge legends are in the sidebar while it is open.
  const legendClass = "w-60"
  const legends = (
    <>
      {windTimeline}
      {gridOn && <GridLegend variable={gridVariable} className={legendClass} />}
      {fillVariable && (
        <RegionLegend variable={fillVariable} level={level} className={legendClass} />
      )}
      {sidebarCollapsed && floodsOn && floodGauges.length > 0 && (
        <FloodLegend counts={floodCounts} />
      )}
      {sidebarCollapsed && riversOn && <RiverLegend counts={riverCounts} />}
      {sidebarCollapsed && hazards.legends}
    </>
  )

  const markerAt =
    (selection?.kind === "point" || selection?.kind === "region") && location ? location : null

  return (
    <TooltipProvider>
      <AppShell
        sidebarCollapsed={sidebarCollapsed}
        map={
          <MapView theme={theme} onReady={setMap}>
            {gridOn && <GridLayer grid={grid.data} variable={gridVariable} />}
            {regionsOn && regionsLayer && (
              <RegionsLayer
                layer={regionsLayer}
                level={level}
                variable={fillVariable}
                weather={regionList}
                selectedId={regionId && regionLevelOf(regionId) === level ? regionId : null}
                onVisibleDistricts={addGnDistricts}
              />
            )}
            {hazards.fills}
            {floodsOn && (
              <FloodForecastLayer
                data={floods.data}
                selectedId={selection?.kind === "gauge" ? selection.id : null}
              />
            )}
            {windOn && (
              <Suspense fallback={null}>
                <WindLayer field={wind.field} />
              </Suspense>
            )}
            {riversOn && (
              <RiverLevelsLayer
                data={rivers.data}
                selectedId={selection?.kind === "station" ? selection.id : null}
              />
            )}
            {hazards.points}
            {markerAt && (
              <LocationMarker lat={markerAt.lat} lon={markerAt.lon} label={detailLabel} />
            )}
            <HoverChip
              regions={regionsOn ? { variable: fillVariable, lookup: regionLookup } : null}
              wind={windOn ? wind.field : null}
              hazards={hazards.hover}
            />
          </MapView>
        }
        sidebar={
          <AppSidebar
            collapsed={sidebarCollapsed}
            onCollapsedChange={setSidebarCollapsed}
            footer={smallPrint}
          >
            {layerList}
          </AppSidebar>
        }
        mobileBar={
          <MobileBar
            layers={layerList}
            activeCount={activeCount}
            footer={smallPrint}
            actions={
              <>
                <SearchTrigger compact onClick={openSearch} onPointerEnter={loadSearch} />
                {hazards.button(true)}
              </>
            }
          />
        }
        toolbar={
          <div className="flex w-full max-w-2xl items-center justify-center gap-2">
            <SearchTrigger
              className="pointer-events-auto max-w-sm flex-1"
              onClick={openSearch}
              onPointerEnter={loadSearch}
              onFocus={loadSearch}
            />
            {hazards.button(false)}
            {regionsOn && (
              <div className="pointer-events-auto rounded-full border border-border bg-card/95 p-1 shadow-panel backdrop-blur-md">
                <AdminLevelSelector level={level} onLevelChange={setLevel} />
              </div>
            )}
          </div>
        }
        notice={
          hazards.banner || notice ? (
            <div className="flex w-full flex-col items-center gap-2">
              {hazards.banner}
              {notice}
            </div>
          ) : null
        }
        detail={detail}
        detailLabel={detailLabel}
        detailFull={hazards.detailFull}
        onDetailClose={close}
        legends={legends}
        timeline={windTimeline}
      />
      {searchLoaded && (
        <Suspense fallback={null}>
          <SearchPalette
            open={searchOpen}
            onOpenChange={setSearchOpen}
            onChoose={choose}
            rivers={rivers.data}
          />
        </Suspense>
      )}
      <Toaster position={desktop ? "bottom-center" : "top-center"} />
    </TooltipProvider>
  )
}
