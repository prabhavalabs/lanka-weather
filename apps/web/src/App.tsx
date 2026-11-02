import { LayerPanel } from "./features/layers/LayerPanel"
import { MapView } from "./features/map/MapView"
import { useLocalStorage } from "./lib/useLocalStorage"

export function App() {
  const [enabled, setEnabled] = useLocalStorage<string[]>("lanka-weather:enabled-layers", [])
  const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches

  const toggle = (id: string) =>
    setEnabled((current) =>
      current.includes(id) ? current.filter((value) => value !== id) : [...current, id],
    )

  return (
    <main className="relative h-dvh w-full overflow-hidden bg-slate-900">
      <MapView theme={prefersDark ? "dark" : "light"} />
      <div className="pointer-events-none absolute inset-y-0 left-0 flex p-3">
        <LayerPanel enabled={enabled} onToggle={toggle} />
      </div>
    </main>
  )
}
