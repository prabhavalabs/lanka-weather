import { statusColors, statusLabels, statusOrder } from "./present"

/** Key to the river-levels symbols. */
export function RiverLegend() {
  return (
    <div className="pointer-events-auto rounded-lg border border-white/10 bg-slate-950/85 px-3 py-2 text-[11px] text-slate-200 shadow-lg backdrop-blur">
      <p className="mb-1 font-medium text-slate-300">River levels</p>
      <ul className="grid grid-cols-2 gap-x-3 gap-y-0.5">
        {statusOrder.map((status) => (
          <li key={status} className="flex items-center gap-1.5">
            <span
              className="inline-block size-2.5 rounded-full border border-white"
              style={{ backgroundColor: statusColors[status] }}
            />
            {statusLabels[status]}
          </li>
        ))}
        <li className="flex items-center gap-1.5">
          <span className="inline-block size-2.5 rounded-full bg-sky-500 ring-4 ring-sky-500/30" />
          Rising
        </li>
        <li className="flex items-center gap-1.5">
          <span className="inline-block size-2.5 rounded-full border border-white bg-sky-500/40" />
          Out of date
        </li>
      </ul>
    </div>
  )
}
