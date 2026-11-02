import { groupByCategory } from "../../lib/layers"
import { useLayers } from "./useLayers"

interface LayerPanelProps {
  enabled: string[]
  onToggle: (id: string) => void
}

/** Lists the layer catalogue grouped by category, with a switch per layer. */
export function LayerPanel({ enabled, onToggle }: LayerPanelProps) {
  const { data, error, isPending, isFetching, dataUpdatedAt } = useLayers()

  return (
    <aside className="pointer-events-auto flex max-h-full w-72 flex-col overflow-hidden rounded-xl border border-white/10 bg-slate-950/85 text-slate-100 shadow-xl backdrop-blur">
      <header className="border-b border-white/10 px-4 py-3">
        <h1 className="text-base font-semibold">open-weather</h1>
        <p className="text-xs text-slate-400">Weather and hazard map of Sri Lanka</p>
      </header>

      <div className="flex-1 overflow-y-auto px-2 py-2">
        {isPending && <p className="px-2 py-3 text-sm text-slate-400">Loading layers…</p>}
        {error && !data && (
          <p className="px-2 py-3 text-sm text-amber-300" role="alert">
            Layers are unavailable: {error.message}
          </p>
        )}
        {data &&
          groupByCategory(data).map((group) => (
            <section key={group.category} className="mb-3">
              <h2 className="px-2 pb-1 text-[11px] font-medium tracking-wide text-slate-400 uppercase">
                {group.label}
              </h2>
              <ul>
                {group.layers.map((layer) => {
                  const planned = layer.status === "planned"
                  return (
                    <li key={layer.id}>
                      <label
                        className={`flex items-start gap-3 rounded-lg px-2 py-2 ${planned ? "opacity-60" : "cursor-pointer hover:bg-white/5"}`}
                        title={layer.description}
                      >
                        <input
                          type="checkbox"
                          className="mt-0.5 accent-sky-400"
                          checked={enabled.includes(layer.id)}
                          disabled={planned}
                          onChange={() => onToggle(layer.id)}
                        />
                        <span className="flex-1">
                          <span className="block text-sm">{layer.title}</span>
                          <span className="block text-xs text-slate-400">
                            {planned ? "Coming soon" : layer.attribution}
                          </span>
                        </span>
                      </label>
                    </li>
                  )
                })}
              </ul>
            </section>
          ))}
      </div>

      <footer className="border-t border-white/10 px-4 py-2 text-[11px] text-slate-500">
        {isFetching
          ? "Updating…"
          : dataUpdatedAt > 0
            ? `Catalogue updated ${new Date(dataUpdatedAt).toLocaleTimeString()}`
            : null}
        <span className="block">Official warnings: Department of Meteorology and DMC.</span>
      </footer>
    </aside>
  )
}
