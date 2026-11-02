import { useState } from "react"
import { LayerPanel } from "./features/layers/LayerPanel"
import { useLayers } from "./features/layers/useLayers"
import { MapView } from "./features/map/MapView"
import { countByStatus, formatAge } from "./features/rivers/present"
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
  const [selectedStation, setSelectedStation] = useState<string | null>(null)
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches

  const catalogue = useLayers()
  const riverLayer = catalogue.data?.find((layer) => layer.id === RIVER_LEVELS)
  const riversOn = enabled.includes(RIVER_LEVELS)
  const rivers = useRiverLevels(riverLayer, riversOn)
  const station = rivers.data?.features.find(
    (feature) => feature.id === selectedStation,
  )?.properties

  const toggle = (id: string) => {
    if (id === RIVER_LEVELS && riversOn) setSelectedStation(null)
    setEnabled((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    )
  }

  const summaries: Record<string, string> = {}
  if (rivers.data) {
    const counts = countByStatus(rivers.data.features.map((feature) => feature.properties))
    const raised = counts.alert + counts.minor_flood + counts.major_flood
    summaries[RIVER_LEVELS] =
      `${rivers.data.features.length} stations · ${raised === 0 ? "none above alert" : `${raised} above alert`}` +
      ` · updated ${formatAge(rivers.data.meta.fetchedAt)}`
  } else if (rivers.error) {
    summaries[RIVER_LEVELS] = "River levels are unavailable"
  }

  return (
    <main className="relative h-dvh w-full overflow-hidden bg-slate-900">
      <MapView theme={prefersDark ? "dark" : "light"}>
        {riversOn && (
          <RiverLevelsLayer
            data={rivers.data}
            selectedId={selectedStation}
            onSelect={setSelectedStation}
          />
        )}
      </MapView>
      <div className="pointer-events-none absolute inset-y-0 left-0 flex p-3">
        <LayerPanel enabled={enabled} onToggle={toggle} summaries={summaries} />
      </div>
      {riversOn && station && (
        <div className="pointer-events-none absolute inset-x-0 bottom-0 flex justify-center p-3 md:inset-x-auto md:top-0 md:right-0 md:bottom-auto">
          <StationPanel station={station} onClose={() => setSelectedStation(null)} />
        </div>
      )}
      {riversOn && (
        <div className="pointer-events-none absolute bottom-10 left-3 hidden md:block">
          <RiverLegend />
        </div>
      )}
    </main>
  )
}
