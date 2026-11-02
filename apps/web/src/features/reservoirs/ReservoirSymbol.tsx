import { cn } from "@/lib/utils"
import { reservoirColors, reservoirShapes } from "./present"
import type { ReservoirStatus } from "./types"

/** The status symbol, for legends and lists; the label carries the meaning. */
export function ReservoirSymbol({
  status,
  className,
}: {
  status: ReservoirStatus
  className?: string
}) {
  const shape = status === "nodata" ? reservoirShapes.holding : reservoirShapes[status]
  const hollow = status === "nodata"
  return (
    <svg viewBox="0 0 40 40" aria-hidden="true" className={cn("size-3.5 shrink-0", className)}>
      <polygon
        points={shape.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(" ")}
        fill={hollow ? "none" : reservoirColors[status]}
        stroke={hollow ? reservoirColors[status] : "var(--status-stroke)"}
        strokeOpacity={hollow ? 1 : 0.45}
        strokeWidth={hollow ? 6 : 2}
        strokeLinejoin="round"
      />
    </svg>
  )
}
