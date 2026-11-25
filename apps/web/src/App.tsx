import type { Map as MapLibreMap, MapMouseEvent } from "maplibre-gl"
import { ContainerIcon, ListIcon, MapPinIcon, WavesIcon, ZoomInIcon } from "lucide-react"
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
import { MapKey, type MapKeySection } from "@/components/weather/MapKey"
import { SearchTrigger } from "@/components/weather/SearchTrigger"
import { toolbarPill } from "@/components/weather/toolbar"
import { isNotReady } from "@/lib/api"
import type { Layer } from "@/lib/layers"
import { desktopQuery, useMediaQuery } from "@/lib/useMediaQuery"
import { useT } from "@/lib/i18n"
import { useTheme } from "@/lib/theme"
import { useLocalStorage } from "@/lib/useLocalStorage"
import { anyUnverified, countBasins, topArea } from "./features/floods/basins"
import { BasinPanel } from "./features/floods/BasinPanel"
import { FloodAreaPanel } from "./features/floods/FloodAreaPanel"
import { FloodBasinsLayer, BASIN_FILL, SUBBASIN_FILL } from "./features/floods/FloodBasinsLayer"
import { FloodForecastLayer, FLOOD_AREAS, FLOOD_GAUGES } from "./features/floods/FloodForecastLayer"
import { FloodGaugePanel } from "./features/floods/FloodGaugePanel"
import {
  PAST_FLOOD_FILL,
  PastFloodPanel,
  PastFloodsLayer,
  PastFloodsList,
} from "./features/floods/PastFloods"
import { FloodLegend } from "./features/floods/FloodLegend"
import { countBySeverity, floodSummary, isGauge } from "./features/floods/present"
import { useFloodForecast } from "./features/floods/queries"
import type { FloodAreaFeature } from "./features/floods/types"
import {
  basinsForView,
  defaultFloodView,
  floodsForView,
  type FloodView,
} from "./features/floods/view"
import { hazardDefaults, useHazards } from "./features/hazards/useHazards"
import { LayerPanel } from "./features/layers/LayerPanel"
import { layerStatus, type LayerStatus } from "./features/layers/status"
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
import { homeSelection, type HomePlace } from "./features/home/home"
import { useHome } from "./features/home/useHome"
import { InstallHost } from "./features/install"
import { useOfflineNotice } from "./features/pwa/OfflineNotice"
import { hitLayerIds } from "./features/map/hitLayers"
import { hitOrder, orderedHits, queryMap } from "./features/map/hits"
import { asFocusGeometry } from "./features/map/focus"
import { FocusMask } from "./features/map/FocusMask"
import { HoverChip } from "./features/map/HoverChip"
import { MapView } from "./features/map/MapView"
import { sriLankaBounds } from "./features/map/config"
import {
  clearCentreOffset,
  clearPadding,
  inClearArea,
  layoutInsets,
  openingTopRoom,
} from "./features/map/reveal"
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
import { riversAtRisk } from "./features/river-risk/present"
import { useFloodBasins, usePastFloods, useRiverRisk } from "./features/river-risk/queries"
import { RiverNetworkLayer, RIVER_LINES } from "./features/river-risk/RiverNetworkLayer"
import { RiverRiskLegend } from "./features/river-risk/RiverRiskLegend"
import { RiversAtRisk } from "./features/river-risk/RiversAtRisk"
import { countByStatus, statusColors, statusLabels, statusOrder } from "./features/rivers/present"
import {
  activeReservoirs,
  hasCurrentFigures,
  reservoirColors,
  reservoirStatusLabels,
  reservoirStatusOrder,
  reservoirSummary,
} from "./features/reservoirs/present"
import { useReservoirAreas, useReservoirs } from "./features/reservoirs/queries"
import { ReservoirLegend } from "./features/reservoirs/ReservoirLegend"
import { ReservoirListPanel } from "./features/reservoirs/ReservoirListPanel"
import { ReservoirPanel } from "./features/reservoirs/ReservoirPanel"
import {
  ReservoirsLayer,
  RESERVOIR_AREAS,
  RESERVOIR_POINTS,
} from "./features/reservoirs/ReservoirsLayer"
import { useRiverLevels } from "./features/rivers/queries"
import { RiverLegend } from "./features/rivers/RiverLegend"
import { RiverLevelsLayer, RIVER_STATIONS } from "./features/rivers/RiverLevelsLayer"
import { StationPanel } from "./features/rivers/StationPanel"
import { MetStationPanel } from "./features/stations/MetStationPanel"
import { AlertsButton } from "./features/alerts/AlertsButton"
import { linkTarget } from "./features/notifications/present"
import { OverviewButton } from "./features/overview/OverviewButton"
import { OverviewPanel } from "./features/overview/OverviewPanel"
import { overviewAction } from "./features/overview/present"
import { useOverview } from "./features/overview/queries"
import type { OverviewTarget } from "./features/overview/types"
import { observedChip, observedPointLayers, observedTarget } from "./features/observed/points"
import {
  RAIN_GAUGES,
  useObservedLayers,
  WEATHER_STATIONS,
} from "./features/observed/useObservedLayers"
import { CanalsPanel } from "./features/canals/CanalsPanel"
import { RainGaugePanel } from "./features/raingauges/RainGaugePanel"
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
const RiverPanel = lazy(() => import("./features/river-risk/RiverPanel"))

const RIVER_LEVELS = "river-levels"
const FLOOD_FORECAST = "flood-forecast"
const TEMPERATURE = "temperature"
const WIND = "wind"
const ADMIN_REGIONS = "admin-regions"
const RIVERS = "rivers"
const PAST_FLOODS = "past-floods"
const RESERVOIRS = "reservoirs"

/**
 * Layers on at the first visit: official warnings and hazards first, then the
 * water, with boundaries for where things are. Wind is one switch away: its
 * animation sits over everything and its timeline takes room on a phone.
 */
const defaultLayers = [
  ...hazardDefaults,
  RIVERS,
  RIVER_LEVELS,
  FLOOD_FORECAST,
  RESERVOIRS,
  RAIN_GAUGES,
  ADMIN_REGIONS,
]
/** Default layers added since people may have saved their choice; offered to them once. */
const newDefaultLayers = [RIVERS, RESERVOIRS, RAIN_GAUGES, ...hazardDefaults]

type RegionFill = RegionVariable | "none"

/** Gauges are small: clicks and hovers count within a few pixels. */
const pointSlack = 6

export function App() {
  const [enabled, setEnabled] = useLocalStorage<string[]>("lanka-weather:layers", defaultLayers)
  const [offered, setOffered] = useLocalStorage<string[]>("lanka-weather:layers-offered", [])
  const [sidebarCollapsed, setSidebarCollapsed] = useLocalStorage(
    "lanka-weather:sidebar-collapsed",
    false,
  )
  const [level, setLevel] = useLocalStorage<RegionLevel>("lanka-weather:region-level", "district")
  const [regionFill, setRegionFill] = useLocalStorage<RegionFill>(
    "lanka-weather:region-fill",
    "none",
  )
  const [gridVariable, setGridVariable] = useLocalStorage<GridVariable>(
    "lanka-weather:grid-variable",
    "temperature",
  )
  const [floodView, setFloodView] = useLocalStorage<FloodView>(
    "lanka-weather:flood-view",
    defaultFloodView,
  )
  const [selection, setSelection] = useUrlSelection()
  const [home, setHome] = useHome()
  const [map, setMap] = useState<MapLibreMap | null>(null)
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchLoaded, setSearchLoaded] = useState(false)
  const [gnDistricts, setGnDistricts] = useState<string[]>([])
  const [clickedAt, setClickedAt] = useState<{ lat: number; lon: number } | null>(null)
  // The first thing on load, until dismissed or the map is used instead.
  const [overviewOpen, setOverviewOpen] = useState(true)
  const overview = useOverview(overviewOpen)
  const zoom = useMapZoom(map)
  const desktop = useMediaQuery(desktopQuery, true)
  const { resolved: theme } = useTheme()
  // Subscribes the page to the language: statuses, legends and panels re-render on a switch.
  const t = useT()

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
  const riversNetworkLayer = available(RIVERS)
  const pastFloodsLayer = available(PAST_FLOODS)
  const reservoirLayer = available(RESERVOIRS)
  const on = (id: string) => enabled.includes(id) && available(id) !== undefined
  const riversOn = on(RIVER_LEVELS)
  const floodsOn = on(FLOOD_FORECAST)
  const gridOn = on(TEMPERATURE)
  const windOn = on(WIND)
  const regionsOn = on(ADMIN_REGIONS)
  const riverNetworkOn = on(RIVERS)
  const pastFloodsOn = on(PAST_FLOODS)

  // Someone who saved their layers before a default was added still gets it, once.
  useEffect(() => {
    const unseen = newDefaultLayers.filter((id) => !offered.includes(id))
    if (unseen.length === 0) return
    setOffered((current) => [...current, ...unseen])
    setEnabled((current) => [...new Set([...current, ...unseen])])
  }, [offered, setOffered, setEnabled])

  const observedLayers = useObservedLayers({
    available,
    enabled,
    selectedMetId: selection?.kind === "met" ? selection.id : null,
    selectedRainGaugeId: selection?.kind === "raingauge" ? selection.id : null,
    theme,
    onShowModelRain: () => {
      setGridVariable("precipitation")
      setEnabled((current) => (current.includes(TEMPERATURE) ? current : [...current, TEMPERATURE]))
    },
  })
  const reservoirsOn = on(RESERVOIRS)

  // Region colours give way to the weather field, which shows the same weather in more detail.
  const fillVariable: RegionVariable | null =
    regionsOn && regionFill !== "none" && !gridOn && !observedLayers.airOn ? regionFill : null

  const rivers = useRiverLevels(
    riverLayer,
    riversOn || selection?.kind === "station" || searchLoaded,
  )
  const floods = useFloodForecast(
    floodLayer,
    floodsOn || selection?.kind === "gauge" || selection?.kind === "area",
  )
  const riverRisk = useRiverRisk(
    riversNetworkLayer,
    riverNetworkOn || selection?.kind === "river" || searchLoaded,
  )
  const basinQuery = useFloodBasins(floodLayer, floodsOn || selection?.kind === "basin")
  const shownFloods = useMemo(
    () => (floods.data ? floodsForView(floods.data, floodView) : undefined),
    [floods.data, floodView],
  )
  const basins = useMemo(
    () => ({
      ...basinQuery,
      data: basinQuery.data ? basinsForView(basinQuery.data, floodView) : undefined,
    }),
    [basinQuery, floodView],
  )
  const pastFloods = usePastFloods(pastFloodsLayer, pastFloodsOn || selection?.kind === "past")
  const reservoirs = useReservoirs(
    reservoirLayer,
    reservoirsOn ||
      selection?.kind === "reservoir" ||
      selection?.kind === "reservoirs" ||
      searchLoaded,
  )
  const reservoirAreas = useReservoirAreas(reservoirLayer, reservoirsOn)
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
  const selectedRiver =
    selection?.kind === "river"
      ? riverRisk.data?.rivers.find((river) => river.id === selection.id)
      : undefined
  const selectedBasin =
    selection?.kind === "basin"
      ? basins.data?.basins.find((basin) => basin.id === selection.id)
      : undefined
  const selectedPast =
    selection?.kind === "past"
      ? pastFloods.data?.events.find((event) => event.id === selection.id)
      : undefined
  const selectedArea =
    selection?.kind === "area"
      ? (floods.data?.features ?? []).find(
          (feature): feature is FloodAreaFeature =>
            !isGauge(feature) && feature.id === selection.id,
        )
      : undefined
  const metStation =
    selection?.kind === "met" ? observedLayers.stationFeature(selection.id) : undefined
  const rainGauge =
    selection?.kind === "raingauge" ? observedLayers.rainGaugeFeature(selection.id) : undefined
  const reservoir =
    selection?.kind === "reservoir"
      ? reservoirs.data?.features.find((feature) => feature.id === selection.id)
      : undefined

  const enable = useCallback(
    (id: string) => setEnabled((current) => (current.includes(id) ? current : [...current, id])),
    [setEnabled],
  )

  const toggle = (id: string) => {
    const turningOn = !enabled.includes(id)
    if (!turningOn && id === RIVER_LEVELS && selection?.kind === "station") setSelection(null)
    if (!turningOn && id === FLOOD_FORECAST && selection?.kind === "gauge") setSelection(null)
    if (!turningOn && id === FLOOD_FORECAST && selection?.kind === "basin") setSelection(null)
    if (!turningOn && id === FLOOD_FORECAST && selection?.kind === "area") setSelection(null)
    if (!turningOn && id === RIVERS && selection?.kind === "river") setSelection(null)
    if (!turningOn && id === PAST_FLOODS && selection?.kind === "past") setSelection(null)
    if (!turningOn && id === WEATHER_STATIONS && selection?.kind === "met") setSelection(null)
    if (!turningOn && id === RAIN_GAUGES && selection?.kind === "raingauge") setSelection(null)
    if (
      !turningOn &&
      id === RESERVOIRS &&
      (selection?.kind === "reservoir" || selection?.kind === "reservoirs")
    ) {
      setSelection(null)
    }
    setEnabled((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    )
  }

  // Camera: keep the selected place clear of the cards floating over the map.
  const overviewShown = overviewOpen && !selection
  const insets = useCallback(
    () =>
      layoutInsets({
        desktop,
        sidebarCollapsed,
        detailOpen: true,
        viewportHeight: window.innerHeight,
        smallSheet: overviewShown,
      }),
    [desktop, sidebarCollapsed, overviewShown],
  )
  // The map opens on Sri Lanka in what the cards leave clear, whatever the pan limit
  // allows. A link to a place moves the map itself.
  // The reset button above the zoom buttons brings the same view back.
  const showSriLanka = useCallback(
    (animate: boolean) => {
      if (!map) return
      const padding = clearPadding(insets())
      map.fitBounds(sriLankaBounds, {
        padding: { ...padding, top: padding.top + openingTopRoom },
        bearing: 0,
        pitch: 0,
        duration: animate ? 1200 : 0,
      })
    },
    [map, insets],
  )
  const resetView = useCallback(() => showSriLanka(true), [showSriLanka])
  const opened = useRef(false)
  useEffect(() => {
    if (!map || opened.current) return
    opened.current = true
    if (selection) return
    showSriLanka(false)
  }, [map, selection, showSriLanka])
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
    } else if (selection.kind === "river") {
      if (!selectedRiver) return
      bbox = selectedRiver.bbox
      lngLat = [(bbox[0] + bbox[2]) / 2, (bbox[1] + bbox[3]) / 2]
    } else if (selection.kind === "basin") {
      if (!selectedBasin) return
      bbox = selectedBasin.bbox
      lngLat = selectedBasin.labelPoint
    } else if (selection.kind === "past") {
      if (!selectedPast) return
      bbox = selectedPast.bbox
      lngLat = [(bbox[0] + bbox[2]) / 2, (bbox[1] + bbox[3]) / 2]
    } else if (selection.kind === "met") {
      if (!metStation) return
      lngLat = metStation.geometry.coordinates
    } else if (selection.kind === "raingauge") {
      if (!rainGauge) return
      lngLat = rainGauge.geometry.coordinates
    } else if (selection.kind === "reservoir") {
      if (!reservoir) return
      // A reservoir with no known position opens without moving the map.
      lngLat = reservoir.geometry?.coordinates ?? null
    }
    pending.current = null
    if (!lngLat) return
    if (move.target === "fit" && bbox) fitBox(bbox)
    else if (move.target === "fit" || move.target === "zoom") reveal(lngLat, { zoom: 10 })
    else reveal(lngLat)
  }, [
    map,
    selection,
    regionDetail.data,
    station,
    gauge,
    metStation,
    rainGauge,
    reservoir,
    selectedRiver,
    selectedBasin,
    selectedPast,
    reveal,
    fitBox,
  ])

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

  const openRiver = useCallback(
    (id: string, reach?: number, target: "fit" | "none" = "fit") => {
      enable(RIVERS)
      const next: Selection = reach ? { kind: "river", id, reach } : { kind: "river", id }
      if (target === "fit") pendingCamera(next, "fit")
      else pending.current = null
      setSelection(next)
    },
    [enable, pendingCamera, setSelection],
  )
  const openArea = useCallback(
    (id: string) => {
      enable(FLOOD_FORECAST)
      pending.current = null
      setSelection({ kind: "area", id })
    },
    [enable, setSelection],
  )
  const openPast = useCallback(
    (id: string, target: "fit" | "none" = "fit") => {
      enable(PAST_FLOODS)
      const next: Selection = { kind: "past", id }
      if (target === "fit") pendingCamera(next, "fit")
      else pending.current = null
      setSelection(next)
    },
    [enable, pendingCamera, setSelection],
  )
  const openRegion = useCallback(
    (id: string) => {
      const next: Selection = { kind: "region", id }
      pendingCamera(next, "fit")
      setSelection(next)
    },
    [pendingCamera, setSelection],
  )
  const weatherAt = useCallback(
    (at: { lat: number; lon: number }) => {
      const next: Selection = {
        kind: "point",
        lat: roundCoordinate(at.lat),
        lon: roundCoordinate(at.lon),
      }
      pendingCamera(next, "reveal")
      setSelection(next)
    },
    [pendingCamera, setSelection],
  )

  const openMetStation = useCallback(
    (id: string) => {
      enable(WEATHER_STATIONS)
      const next: Selection = { kind: "met", id }
      pendingCamera(next, "reveal")
      setSelection(next)
    },
    [enable, pendingCamera, setSelection],
  )

  const openRainGauge = useCallback(
    (id: string) => {
      enable(RAIN_GAUGES)
      const next: Selection = { kind: "raingauge", id }
      pendingCamera(next, "reveal")
      setSelection(next)
    },
    [enable, pendingCamera, setSelection],
  )
  const openCanals = useCallback(() => {
    pending.current = null
    setSelection({ kind: "canals" })
  }, [setSelection])

  const openReservoir = useCallback(
    (id: string) => {
      enable(RESERVOIRS)
      const next: Selection = { kind: "reservoir", id }
      pendingCamera(next, "reveal")
      setSelection(next)
    },
    [enable, pendingCamera, setSelection],
  )
  const openReservoirList = useCallback(() => {
    enable(RESERVOIRS)
    pending.current = null
    setSelection({ kind: "reservoirs" })
  }, [enable, setSelection])

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

  // Clicks: the first thing under the pointer that can open, in the order of
  // features/map/hits, else the place itself.
  useMapClick(map, (event: MapMouseEvent) => {
    if (!map) return
    setOverviewOpen(false)
    const order = hitOrder(hitLayerIds({ hazards: hazards.hover, observed: observedPointLayers }))
    for (const { feature } of orderedHits(order, queryMap(map, event.point, pointSlack))) {
      const layer = feature.layer.id
      const observed = observedTarget(feature)
      if (observed) {
        if (observed.kind === "raingauge") return openRainGauge(observed.id)
        return observed.kind === "met" ? openMetStation(observed.id) : openStation(observed.id)
      }
      if (layer === RIVER_STATIONS) return openStation(String(feature.properties.id))
      if (layer === FLOOD_GAUGES) return openGauge(String(feature.properties.gaugeId))
      if (layer === RESERVOIR_POINTS || layer === RESERVOIR_AREAS) {
        return openReservoir(String(feature.properties.id))
      }
      if (hazards.handleHit(feature)) return
      if (layer === FLOOD_AREAS) {
        // Overlapping areas come in no useful order; the most specific one opens.
        const top = topArea(queryMap(map, event.point, pointSlack)([FLOOD_AREAS], false)) ?? feature
        if (typeof top.properties.id === "string") return openArea(top.properties.id)
      }
      if (RIVER_LINES.includes(layer) && feature.properties.river) {
        const reach =
          feature.sourceLayer === "reaches" && typeof feature.id === "number"
            ? feature.id
            : undefined
        return openRiver(String(feature.properties.river), reach, "none")
      }
      if (layer === PAST_FLOOD_FILL && typeof feature.properties.id === "string") {
        return openPast(feature.properties.id, "none")
      }
      if (layer === BASIN_FILL || layer === SUBBASIN_FILL) {
        // The basin holds a link to the weather at the clicked place.
        const fid = layer === SUBBASIN_FILL ? Number(feature.properties.basin) : Number(feature.id)
        const basin = basins.data?.basins.find((item) => item.fid === fid)
        if (basin) {
          setClickedAt({ lat: event.lngLat.lat, lon: event.lngLat.lng })
          pending.current = null
          return setSelection({ kind: "basin", id: basin.id })
        }
      }
      if (layer === REGIONS_FILL && typeof feature.properties.id === "string") {
        const next: Selection = { kind: "region", id: feature.properties.id }
        pendingCamera(next, "reveal")
        return setSelection(next)
      }
    }

    const { lat, lng } = event.lngLat
    if (!inPointBounds(lat, lng)) {
      toast(t("app.outside"), {
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
      setOverviewOpen(false)
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
      } else if (choice.type === "river") {
        enable(RIVERS)
        pending.current = null
        setSelection({ kind: "river", id: choice.id })
        fitBox(choice.bbox)
      } else if (choice.type === "reservoir") {
        enable(RESERVOIRS)
        pending.current = null
        setSelection({ kind: "reservoir", id: choice.id })
        if (choice.lat !== null && choice.lon !== null) {
          reveal([choice.lon, choice.lat], { zoom: 11 })
        }
      } else {
        enable(RIVER_LEVELS)
        pending.current = null
        setSelection({ kind: "station", id: choice.id })
        reveal([choice.lon, choice.lat], { zoom: 11 })
      }
    },
    [enabled, enable, fitBox, pendingCamera, reveal, setLevel, setSelection],
  )

  // A line of the overview opens its layer and panel. The overview stays behind it,
  // so closing the panel comes back to the list.
  const openOverviewTarget = useCallback(
    (target: OverviewTarget) => {
      const action = overviewAction(target)
      action.enable.forEach(enable)
      if (action.fill) setRegionFill(action.fill)
      if (action.level) setLevel(action.level)
      if (action.selection) {
        if (action.camera) pendingCamera(action.selection, action.camera)
        else pending.current = null
        setSelection(action.selection)
      } else if (action.reveal) {
        reveal(action.reveal)
      }
    },
    [enable, pendingCamera, reveal, setLevel, setRegionFill, setSelection],
  )
  // "Show on map" in a notice: its link read the way the address is read on load.
  const openLink = useCallback(
    (url: string) => {
      const target = linkTarget(url)
      setOverviewOpen(false)
      target.enable.forEach(enable)
      if (!target.selection) return
      pendingCamera(target.selection, "fit")
      setSelection(target.selection)
    },
    [enable, pendingCamera, setSelection],
  )
  const closeOverview = useCallback(() => setOverviewOpen(false), [])
  // The "Now" button: shows the overview, or closes it when it is what is on show.
  // Over another panel it brings the overview back rather than closing anything.
  const toggleOverview = useCallback(() => {
    if (overviewOpen && !selection) {
      setOverviewOpen(false)
      return
    }
    setSelection(null)
    setOverviewOpen(true)
  }, [overviewOpen, selection, setSelection])

  // The house button: back to the home place and its weather, or how to set one.
  const goHome = useCallback(() => {
    if (!home) {
      toast(t("map.home.hint"), { id: "home" })
      return
    }
    setOverviewOpen(false)
    const next = homeSelection(home)
    if (next.kind === "region") {
      pendingCamera(next, "fit")
      setSelection(next)
    } else {
      pending.current = null
      setSelection(next)
      reveal([home.lon, home.lat], { zoom: 12 })
    }
  }, [home, pendingCamera, reveal, setSelection, t])
  const changeHome = useCallback(
    (next: HomePlace | null) => {
      setHome(next)
      toast(next ? t("map.home.saved", { name: next.name }) : t("map.home.removed"), {
        id: "home",
      })
    },
    [setHome, t],
  )
  const homeControl = useMemo(
    () => ({
      label: home ? t("map.home.goHome", { name: home.name }) : t("map.home.unset"),
      onClick: goHome,
    }),
    [home, goHome, t],
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
      if (event.key !== "Escape" || (!selection && !overviewOpen)) return
      if (document.querySelector('[role="dialog"][data-state="open"]')) return
      if (selection) setSelection(null)
      else setOverviewOpen(false)
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [selection, overviewOpen, setSelection])

  const close = useCallback(() => setSelection(null), [setSelection])

  // Layer list: one-line status, details under layers that are on, stale marks.
  const statuses: Record<string, LayerStatus> = { ...hazards.statuses, ...observedLayers.statuses }
  const details: Record<string, ReactNode> = { ...hazards.details, ...observedLayers.details }

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
            t("app.rivers.stations", { count: rivers.data.features.length }),
            raised === 0
              ? t("app.rivers.noneAboveAlert")
              : t("app.rivers.aboveAlert", { count: raised }),
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
          label={t("app.rivers.byStatus")}
          segments={statusOrder.map((status) => ({
            label: statusLabels[status],
            value: riverCounts[status],
            color: statusColors[status],
          }))}
        />
        <RiverLegend bare counts={riverCounts} />
        <Button size="sm" variant="secondary" onClick={openCanals} className="w-full">
          <ListIcon />
          {t("app.rivers.canals")}
        </Button>
      </div>
    )
  }

  const floodGauges = floods.data?.features.filter(isGauge) ?? []
  const shownGauges = shownFloods?.features.filter(isGauge) ?? []
  const floodCounts = shownFloods ? countBySeverity(shownGauges) : undefined
  if (floodLayer) {
    const issued = floodGauges
      .map((feature) => feature.properties.issuedAt)
      .sort()
      .at(-1)
    statuses[FLOOD_FORECAST] = layerStatus({
      layer: floodLayer,
      parts: shownFloods
        ? shownFloods.meta.configured
          ? [floodSummary(shownFloods)]
          : [t("app.floods.notConfigured")]
        : [],
      observedAt: issued ?? floods.data?.meta.fetchedAt,
      error: floods.error,
      pending: floods.isPending,
    })
    if (floods.data && floodGauges.length === 0 && floods.data.meta.configured) {
      statuses[FLOOD_FORECAST] = {
        state: "waiting",
        summary: t("app.floods.noneYet"),
        stale: false,
      }
    }
  }
  const basinCounts = basins.data ? countBasins(basins.data.basins) : undefined
  const basinsUnverified = basins.data ? anyUnverified(basins.data.basins) : false
  const floodLegendProps = {
    view: floodView,
    onViewChange: setFloodView,
    counts: floodCounts,
    basinCounts,
    unverified: basinsUnverified,
    flashAreas: floods.data?.meta.flashAreas ?? 0,
    checkedAt: floods.data?.meta.flashFetchedAt ?? floods.data?.meta.fetchedAt,
    lowerConfidenceGauges: floods.data?.meta.lowerConfidenceGauges ?? 0,
  }
  if (floodCounts) details[FLOOD_FORECAST] = <FloodLegend bare {...floodLegendProps} />

  if (riversNetworkLayer) {
    const atRiskCount = riverRisk.data ? riversAtRisk(riverRisk.data.rivers).length : 0
    statuses[RIVERS] = layerStatus({
      layer: riversNetworkLayer,
      parts: riverRisk.data
        ? [
            t("app.riverRisk.rivers", {
              count: riverRisk.data.rivers.filter((river) => river.named).length,
            }),
            atRiskCount === 0
              ? t("app.riverRisk.noneAtRisk")
              : t("app.riverRisk.atRisk", { count: atRiskCount }),
          ]
        : [],
      observedAt: riverRisk.data?.meta.gaugesObservedAt,
      ageLabel: "read",
      error: riverRisk.error,
      pending: riverRisk.isPending,
    })
  }
  details[RIVERS] = <RiversAtRisk data={riverRisk.data} onSelect={(id) => openRiver(id)} />

  if (pastFloodsLayer) {
    statuses[PAST_FLOODS] = {
      state: "ready",
      summary: pastFloods.data
        ? t("floods.past.listSummary", { count: pastFloods.data.events.length })
        : t("common.loading"),
      stale: false,
    }
  }
  details[PAST_FLOODS] = (
    <PastFloodsList
      data={pastFloods.data}
      selectedId={selection?.kind === "past" ? selection.id : null}
      onSelect={(id) => openPast(id)}
    />
  )

  const reservoirCounts = reservoirs.data?.meta.counts
  const activeCountOfReservoirs = activeReservoirs(reservoirs.data?.features ?? []).length
  if (reservoirLayer) {
    const status = layerStatus({
      layer: reservoirLayer,
      parts: [],
      observedAt: reservoirs.data?.meta.fetchedAt,
      error: reservoirs.error,
      pending: reservoirs.isPending,
    })
    const outOfDate =
      status.stale ||
      (reservoirCounts !== undefined &&
        reservoirCounts.total > 0 &&
        !hasCurrentFigures(reservoirs.data?.features ?? []))
    statuses[RESERVOIRS] = reservoirCounts
      ? {
          ...status,
          stale: outOfDate,
          summary: [
            reservoirSummary(reservoirCounts, activeCountOfReservoirs, outOfDate),
            status.summary,
          ].join(t("common.separator")),
        }
      : status
  }
  if (reservoirCounts) {
    details[RESERVOIRS] = (
      <div className="flex flex-col gap-3">
        <SegmentedBar
          label={t("app.reservoirs.byStatus")}
          segments={reservoirStatusOrder.map((status) => ({
            label: reservoirStatusLabels[status],
            value: reservoirCounts[status],
            color: reservoirColors[status],
          }))}
        />
        <ReservoirLegend bare counts={reservoirCounts} />
        <Button size="sm" variant="secondary" onClick={openReservoirList} className="w-full">
          <ListIcon />
          <span className="min-w-0 truncate">
            {activeCountOfReservoirs === 0
              ? t("app.reservoirs.list")
              : t("app.reservoirs.active", { count: activeCountOfReservoirs })}
          </span>
        </Button>
        <p className="text-xs text-muted-foreground">
          {t("app.reservoirs.mapped", {
            mapped: reservoirCounts.mapped,
            total: reservoirCounts.total,
          })}{" "}
          {t("terms.notOfficial")} {t("terms.emergency")}
        </p>
      </div>
    )
  }

  if (gridLayer) {
    const values = grid.data?.values[gridVariables[gridVariable].key]
    statuses[TEMPERATURE] = layerStatus({
      layer: gridLayer,
      parts: grid.data
        ? [t("app.grid.points", { count: countValues(values) }), gridVariables[gridVariable].label]
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
        aria-label={t("app.grid.shows")}
        className="w-full"
      >
        {gridVariableOrder.map((variable) => (
          <ToggleGroupItem
            key={variable}
            value={variable}
            aria-label={gridVariables[variable].label}
            title={gridVariables[variable].label}
            className="min-w-0 flex-1"
          >
            <span className="truncate">{gridVariables[variable].short}</span>
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      <GridLegend variable={gridVariable} bare className="md:hidden" />
      <p className="text-xs text-muted-foreground">
        {t("app.grid.note")} <span lang="en">Weather data by Open-Meteo.com (CC BY 4.0).</span>
      </p>
    </div>
  )

  if (windLayer) {
    const field = wind.field
    statuses[WIND] = layerStatus({
      layer: windLayer,
      parts: field ? ["NOAA GFS, 10 m", t("app.wind.times", { count: wind.steps.length })] : [],
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
  details[WIND] = <p className="text-xs text-muted-foreground">{t("app.wind.note")}</p>

  if (regionsLayer) {
    const levelCount = regionList.length
    statuses[ADMIN_REGIONS] = layerStatus({
      layer: gridLayer ?? regionsLayer,
      parts:
        levelCount > 0
          ? [
              t(`terms.adminCount.${level}`, { count: levelCount }),
              fillVariable ? regionVariables[fillVariable].label : t("app.regions.boundaries"),
            ]
          : [
              level === "gn" && zoom < levelMinZoom.gn
                ? t("app.regions.zoomForGn")
                : t("app.regions.boundaries"),
            ],
      observedAt: regionValidAt,
      error: level === "gn" ? gnWeather.error : levelWeather.error,
      pending: false,
    })
  }
  details[ADMIN_REGIONS] = (
    <div className="flex flex-col gap-3">
      <AdminLevelSelector level={level} onLevelChange={setLevel} />
      <div className="flex flex-col gap-1.5">
        <span className="text-xs text-muted-foreground">{t("app.regions.colourBy")}</span>
        <ToggleGroup
          type="single"
          size="sm"
          value={regionFill}
          onValueChange={(value) => {
            if (value) setRegionFill(value as RegionFill)
          }}
          aria-label={t("app.regions.colourBy")}
          className="w-full"
          disabled={gridOn}
        >
          <ToggleGroupItem value="none" className="min-w-0 flex-1" title={t("common.none")}>
            <span className="truncate">{t("common.none")}</span>
          </ToggleGroupItem>
          {(Object.keys(regionVariables) as RegionVariable[]).map((variable) => (
            <ToggleGroupItem
              key={variable}
              value={variable}
              aria-label={regionVariables[variable].label}
              title={regionVariables[variable].label}
              className="min-w-0 flex-1"
            >
              <span className="truncate">{regionVariables[variable].short}</span>
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        {gridOn && <p className="text-xs text-muted-foreground">{t("app.regions.hiddenByGrid")}</p>}
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
      <span>{t("app.smallPrint")}</span>
    </div>
  )

  // The notice under the toolbar: a level that needs a closer zoom, or data still on its way.
  const waiting = [
    gridOn && isNotReady(grid.error),
    windOn && isNotReady(wind.query.error),
    regionsOn && isNotReady(level === "gn" ? gnWeather.error : levelWeather.error),
    observedLayers.waiting,
  ].some(Boolean)
  const offlineNotice = useOfflineNotice()
  let notice: ReactNode = null
  if (riverRisk.data?.meta.seeded || floods.data?.meta.seeded || basins.data?.meta.seeded) {
    // Local test data must never pass as a real flood.
    notice = (
      <p
        role="alert"
        className="pointer-events-auto rounded-full border border-destructive bg-card px-3.5 py-1.5 text-sm font-medium text-destructive shadow-panel"
      >
        {t("app.testData")}
      </p>
    )
  } else if (regionsOn && zoom < levelMinZoom[level] && map && !hazards.farAway && zoom >= 4.5) {
    notice = (
      <div className="pointer-events-auto flex items-center gap-2 rounded-full border border-border bg-card/95 py-1 pr-1 pl-3.5 text-sm shadow-panel backdrop-blur-md">
        <span className="min-w-0">{t(`app.regions.zoomToSee.${level}`)}</span>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => map.easeTo({ zoom: levelMinZoom[level] + 0.3, duration: 800 })}
        >
          <ZoomInIcon />
          {t("app.regions.zoomIn")}
        </Button>
      </div>
    )
  } else if (waiting) {
    notice = (
      <p
        role="status"
        className="pointer-events-auto rounded-full border border-border bg-card/95 px-3.5 py-1.5 text-sm text-muted-foreground shadow-panel backdrop-blur-md"
      >
        {t("app.stillLoading")}
      </p>
    )
  }

  // The detail panel for the selection.
  let detail: ReactNode = hazards.detail
  let detailLabel = hazards.detailLabel ?? t("common.details")
  if (selection?.kind === "point" || selection?.kind === "region") {
    detailLabel = t("app.detail.weatherHere")
    detail = (
      <Suspense
        fallback={
          <DetailPlaceholder
            icon={<MapPinIcon />}
            title={t("app.detail.weather")}
            onClose={close}
          />
        }
      >
        <LocationPanel
          selection={selection}
          onClose={close}
          onSelectStation={openStation}
          onSelectGauge={openGauge}
          onSelectRiver={openRiver}
          onSelectMetStation={openMetStation}
          onSelectReservoir={openReservoir}
          onSelectRainGauge={openRainGauge}
          onOpenCanals={openCanals}
          home={home}
          onHomeChange={changeHome}
        />
      </Suspense>
    )
  } else if (selection?.kind === "river") {
    detailLabel = t("app.detail.river")
    detail = (
      <Suspense
        fallback={
          <DetailPlaceholder icon={<WavesIcon />} title={t("app.detail.river")} onClose={close} />
        }
      >
        <RiverPanel
          id={selection.id}
          reachFid={selection.reach}
          onClose={close}
          onSelectRiver={openRiver}
          onSelectStation={openStation}
          onSelectRegion={openRegion}
        />
      </Suspense>
    )
  } else if (selection?.kind === "basin") {
    detailLabel = t("app.detail.basin")
    detail =
      selectedBasin && basins.data ? (
        <BasinPanel
          basin={selectedBasin}
          meta={basins.data.meta}
          clickedAt={clickedAt}
          onClose={close}
          onSelectStation={openStation}
          onSelectGauge={openGauge}
          onSelectRiver={openRiver}
          onSelectArea={openArea}
          onWeatherAt={weatherAt}
        />
      ) : (
        <DetailPlaceholder
          icon={<WavesIcon />}
          title={t("app.detail.basin")}
          message={basins.isPending ? undefined : t("app.detail.basinMissing")}
          onClose={close}
        />
      )
  } else if (selection?.kind === "area") {
    detailLabel = t("app.detail.floodArea")
    detail = selectedArea ? (
      <FloodAreaPanel
        area={selectedArea}
        seeded={floods.data?.meta.seeded}
        onClose={close}
        onSelectGauge={openGauge}
      />
    ) : (
      <DetailPlaceholder
        icon={<WavesIcon />}
        title={t("app.detail.floodArea")}
        message={floods.isPending ? undefined : t("app.detail.floodAreaMissing")}
        onClose={close}
      />
    )
  } else if (selection?.kind === "past") {
    detailLabel = t("app.detail.pastFlood")
    detail = (
      <PastFloodPanel
        event={selectedPast}
        loading={pastFloods.isPending}
        note={pastFloods.data?.note}
        attribution={pastFloods.data?.meta.attribution}
        onClose={close}
      />
    )
  } else if (selection?.kind === "station") {
    detailLabel = t("app.detail.riverGauge")
    detail = station ? (
      <StationPanel station={station.properties} onClose={close} />
    ) : (
      <DetailPlaceholder
        icon={<WavesIcon />}
        title={t("app.detail.riverGauge")}
        message={rivers.isPending ? undefined : t("app.detail.riverGaugeMissing")}
        onClose={close}
      />
    )
  } else if (selection?.kind === "met") {
    detailLabel = t("app.detail.weatherStation")
    detail = <MetStationPanel id={selection.id} onClose={close} />
  } else if (selection?.kind === "raingauge") {
    detailLabel = t("app.detail.rainGauge")
    detail = <RainGaugePanel id={selection.id} onClose={close} />
  } else if (selection?.kind === "canals") {
    detailLabel = t("app.rivers.canals")
    detail = <CanalsPanel onClose={close} />
  } else if (selection?.kind === "gauge") {
    detailLabel = t("app.detail.floodGauge")
    const areas = (floods.data?.features ?? []).filter(
      (feature): feature is FloodAreaFeature =>
        !isGauge(feature) && feature.properties.gaugeId === selection.id,
    )
    detail = gauge ? (
      <FloodGaugePanel gauge={gauge.properties} areas={areas} onClose={close} />
    ) : (
      <DetailPlaceholder
        icon={<WavesIcon />}
        title={t("app.detail.floodGauge")}
        message={floods.isPending ? undefined : t("app.detail.floodGaugeMissing")}
        onClose={close}
      />
    )
  }

  if (selection?.kind === "reservoir") {
    detailLabel = t("app.detail.reservoir")
    detail = reservoir ? (
      <ReservoirPanel reservoir={reservoir.properties} onClose={close} />
    ) : (
      <DetailPlaceholder
        icon={<ContainerIcon />}
        title={t("app.detail.reservoir")}
        message={reservoirs.isPending ? undefined : t("app.detail.reservoirMissing")}
        onClose={close}
      />
    )
  } else if (selection?.kind === "reservoirs") {
    detailLabel = t("app.detail.reservoirsActive")
    detail = (
      <ReservoirListPanel
        data={reservoirs.data}
        pending={reservoirs.isPending}
        stale={statuses[RESERVOIRS]?.stale ?? false}
        onSelect={openReservoir}
        onClose={close}
      />
    )
  }

  if (!selection && overviewOpen) {
    detailLabel = t("app.detail.overview")
    detail = (
      <OverviewPanel
        overview={overview.data}
        pending={overview.isPending}
        failed={overview.isError}
        onOpen={openOverviewTarget}
        onClose={closeOverview}
      />
    )
  }

  // Legends for the layers that are on (wide screens; phones show them in the layer sheet).
  // What colours the whole map keeps its own card; the rest share the map key.
  const legendClass = "w-60"
  const keySections: MapKeySection[] = [
    ...(riverNetworkOn
      ? [
          {
            id: "river-risk",
            title: t("app.key.riverRisk"),
            unit: t("terms.estimate"),
            content: <RiverRiskLegend />,
          },
        ]
      : []),
    ...(floodsOn && floodGauges.length > 0
      ? [
          {
            id: "flood-forecast",
            title: t("layers.title.flood-forecast"),
            unit: "Google",
            content: <FloodLegend bare {...floodLegendProps} />,
          },
        ]
      : []),
    ...(riversOn
      ? [
          {
            id: "river-levels",
            title: t("layers.title.river-levels"),
            unit: t("app.key.status"),
            content: <RiverLegend bare counts={riverCounts} />,
          },
        ]
      : []),
    ...(reservoirsOn
      ? [
          {
            id: "reservoirs",
            title: t("app.key.reservoirs"),
            unit: t("app.key.status"),
            content: <ReservoirLegend bare counts={reservoirCounts} />,
          },
        ]
      : []),
    ...observedLayers.keySections,
    ...hazards.keySections,
  ]
  const legends = (
    <>
      {windTimeline}
      {observedLayers.legends}
      {gridOn && <GridLegend variable={gridVariable} className={legendClass} />}
      {fillVariable && (
        <RegionLegend variable={fillVariable} level={level} className={legendClass} />
      )}
      <MapKey sections={keySections} />
    </>
  )

  const riverHover = useMemo(() => {
    const list = riverRisk.data?.rivers ?? []
    return {
      byId: new Map(list.map((river) => [river.id, river])),
      reaches: new Map(
        list.flatMap((river) => river.reaches.map((reach) => [reach.fid, reach] as const)),
      ),
    }
  }, [riverRisk.data])
  const basinHover = useMemo(
    () => ({
      byFid: new Map((basins.data?.basins ?? []).map((basin) => [basin.fid, basin])),
      subbasins: new Map((basins.data?.subbasins ?? []).map((sub) => [sub.fid, sub.state])),
    }),
    [basins.data],
  )

  // A selected area stands out: the rest of the map goes grey around it.
  const focusGeometry = useMemo(
    () => hazards.focus ?? asFocusGeometry(selectedArea?.geometry),
    [hazards.focus, selectedArea],
  )

  const markerAt =
    (selection?.kind === "point" || selection?.kind === "region") && location ? location : null

  return (
    <TooltipProvider>
      <AppShell
        sidebarCollapsed={sidebarCollapsed}
        map={
          <MapView theme={theme} onReady={setMap} onResetView={resetView} home={homeControl}>
            {gridOn && <GridLayer grid={grid.data} variable={gridVariable} />}
            {observedLayers.mapLayers}
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
            {pastFloodsOn && pastFloodsLayer && (
              <PastFloodsLayer layer={pastFloodsLayer} selectedFid={selectedPast?.fid ?? null} />
            )}
            {floodsOn && floodLayer && (
              <FloodBasinsLayer
                layer={floodLayer}
                data={basins.data}
                selectedFid={selectedBasin?.fid ?? null}
              />
            )}
            {riverNetworkOn && riversNetworkLayer && (
              <RiverNetworkLayer
                layer={riversNetworkLayer}
                rivers={riverRisk.data?.rivers}
                version={riverRisk.data?.meta.networkVersion}
                selectedId={selection?.kind === "river" ? selection.id : null}
              />
            )}
            {floodsOn && (
              <FloodForecastLayer
                data={shownFloods}
                selectedId={selection?.kind === "gauge" ? selection.id : null}
              />
            )}
            {windOn && (
              <Suspense fallback={null}>
                <WindLayer field={wind.field} />
              </Suspense>
            )}
            {reservoirsOn && (
              <ReservoirsLayer
                data={reservoirs.data}
                areas={reservoirAreas.data}
                selectedId={selection?.kind === "reservoir" ? selection.id : null}
              />
            )}
            {riversOn && (
              <RiverLevelsLayer
                data={rivers.data}
                selectedId={selection?.kind === "station" ? selection.id : null}
              />
            )}
            {hazards.points}
            <FocusMask geometry={focusGeometry} />
            {markerAt && (
              <LocationMarker lat={markerAt.lat} lon={markerAt.lon} label={detailLabel} />
            )}
            <HoverChip
              regions={regionsOn ? { variable: fillVariable, lookup: regionLookup } : null}
              wind={windOn ? wind.field : null}
              rivers={riverNetworkOn ? riverHover : null}
              basins={floodsOn ? basinHover : null}
              hazards={hazards.hover}
              extra={{
                layers: observedPointLayers,
                describe: observedChip,
                lineAt: observedLayers.lineAt,
              }}
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
                <OverviewButton compact open={overviewShown} onClick={toggleOverview} />
                {hazards.button(true)}
                <AlertsButton compact onShowOnMap={openLink} />
              </>
            }
          />
        }
        toolbar={
          // One row of pills of one height. The search box gives way first: it
          // narrows to its icon when the labels (Sinhala, Tamil) leave little room.
          <div className="@container flex w-full max-w-3xl items-center justify-center gap-2">
            <SearchTrigger
              className="pointer-events-auto max-w-sm flex-1"
              onClick={openSearch}
              onPointerEnter={loadSearch}
              onFocus={loadSearch}
            />
            <OverviewButton open={overviewShown} onClick={toggleOverview} className={toolbarPill} />
            {hazards.button(false)}
            <AlertsButton className={toolbarPill} onShowOnMap={openLink} />
            {regionsOn && (
              <div className="pointer-events-auto hidden h-10 shrink-0 items-center rounded-full border border-border bg-card/95 px-[3px] shadow-panel backdrop-blur-md @3xl:flex">
                <AdminLevelSelector level={level} onLevelChange={setLevel} />
              </div>
            )}
          </div>
        }
        notice={
          hazards.banner || offlineNotice || notice ? (
            <div className="flex w-full flex-col items-center gap-2">
              {hazards.banner}
              {offlineNotice}
              {notice}
            </div>
          ) : null
        }
        detail={detail}
        detailLabel={detailLabel}
        detailFull={hazards.detailFull}
        detailSmall={overviewShown}
        onDetailClose={selection ? close : closeOverview}
        legends={legends}
        timeline={
          windTimeline && observedLayers.rainTimeline ? (
            <div className="flex w-full flex-col items-center gap-2">
              {observedLayers.rainTimeline}
              {windTimeline}
            </div>
          ) : (
            windTimeline || observedLayers.rainTimeline
          )
        }
        className={
          windTimeline && observedLayers.rainTimeline
            ? "max-md:[--map-controls-bottom:17rem]"
            : undefined
        }
      />
      {searchLoaded && (
        <Suspense fallback={null}>
          <SearchPalette
            open={searchOpen}
            onOpenChange={setSearchOpen}
            onChoose={choose}
            rivers={rivers.data}
            riverNetwork={riverRisk.data?.rivers}
            reservoirs={reservoirs.data}
          />
        </Suspense>
      )}
      <Toaster position={desktop ? "bottom-center" : "top-center"} />
      <InstallHost />
    </TooltipProvider>
  )
}
