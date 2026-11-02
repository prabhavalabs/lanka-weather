import { useState, type ReactNode } from "react"
import { AppShell } from "@/components/AppShell"
import { AppSidebar } from "@/components/AppSidebar"
import { MobileBar } from "@/components/MobileBar"
import { SegmentedBar } from "@/components/SegmentedBar"
import { TooltipProvider } from "@/components/ui/tooltip"
import { formatClock } from "@/lib/format"
import { useTheme } from "@/lib/theme"
import { LayerPanel } from "./features/layers/LayerPanel"
import { useLayers } from "./features/layers/useLayers"
import { MapView } from "./features/map/MapView"
import {
  countByStatus,
  formatAge,
  statusColors,
  statusLabels,
  statusOrder,
} from "./features/rivers/present"
import { useRiverLevels } from "./features/rivers/queries"
import { RiverLegend } from "./features/rivers/RiverLegend"
import { RiverLevelsLayer } from "./features/rivers/RiverLevelsLayer"
import { StationPanel } from "./features/rivers/StationPanel"
import { useLocalStorage } from "./lib/useLocalStorage"

const RIVER_LEVELS = "river-levels"

export function App() {
  const [enabled, setEnabled] = useLocalStorage<string[]>("lanka-weather:enabled-layers", [
    RIVER_LEVELS,
  ])
  const [sidebarCollapsed, setSidebarCollapsed] = useLocalStorage(
    "lanka-weather:sidebar-collapsed",
    false,
  )
  const [selectedStation, setSelectedStation] = useState<string | null>(null)
  const { resolved: theme } = useTheme()

  const catalogue = useLayers()
  const riverLayer = catalogue.data?.find((layer) => layer.id === RIVER_LEVELS)
  const riversOn = enabled.includes(RIVER_LEVELS)
  const rivers = useRiverLevels(riverLayer, riversOn)
  const station = rivers.data?.features.find(
    (feature) => feature.id === selectedStation,
  )?.properties

  const activeCount = enabled.filter((id) =>
    catalogue.data?.some((layer) => layer.id === id && layer.status === "available"),
  ).length

  const toggle = (id: string) => {
    if (id === RIVER_LEVELS && riversOn) setSelectedStation(null)
    setEnabled((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    )
  }

  const summaries: Record<string, string> = {}
  const details: Record<string, ReactNode> = {}
  const counts = rivers.data
    ? countByStatus(rivers.data.features.map((feature) => feature.properties))
    : undefined
  if (rivers.data && counts) {
    const raised = counts.alert + counts.minor_flood + counts.major_flood
    summaries[RIVER_LEVELS] =
      `${rivers.data.features.length} stations · ${raised === 0 ? "none above alert" : `${raised} above alert`}` +
      ` · ${formatAge(rivers.data.meta.fetchedAt)}`
    details[RIVER_LEVELS] = (
      <div className="flex flex-col gap-3">
        <SegmentedBar
          label="Stations by flood status"
          segments={statusOrder.map((status) => ({
            label: statusLabels[status],
            value: counts[status],
            color: statusColors[status],
          }))}
        />
        <RiverLegend bare counts={counts} />
      </div>
    )
  } else if (rivers.error) {
    summaries[RIVER_LEVELS] = "River levels are unavailable"
  }

  const layerList = (
    <LayerPanel enabled={enabled} onToggle={toggle} summaries={summaries} details={details} />
  )
  const smallPrint = (
    <div className="flex flex-col gap-1">
      {catalogue.isFetching ? (
        <span>Updating…</span>
      ) : catalogue.dataUpdatedAt > 0 ? (
        <span>
          Catalogue updated{" "}
          <span className="numeric text-foreground">
            {formatClock(new Date(catalogue.dataUpdatedAt).toISOString())}
          </span>
        </span>
      ) : null}
      <span>Official warnings: Department of Meteorology and DMC.</span>
    </div>
  )

  return (
    <TooltipProvider>
      <AppShell
        sidebarCollapsed={sidebarCollapsed}
        map={
          <MapView theme={theme}>
            {riversOn && (
              <RiverLevelsLayer
                data={rivers.data}
                selectedId={selectedStation}
                onSelect={setSelectedStation}
              />
            )}
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
        mobileBar={<MobileBar layers={layerList} activeCount={activeCount} footer={smallPrint} />}
        detail={
          riversOn &&
          station && <StationPanel station={station} onClose={() => setSelectedStation(null)} />
        }
        legends={riversOn && sidebarCollapsed && <RiverLegend counts={counts} />}
      />
    </TooltipProvider>
  )
}
