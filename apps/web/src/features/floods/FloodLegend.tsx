import { MapLegend } from "@/components/weather/MapLegend"
import { cn } from "@/lib/utils"
import type { BasinState } from "../river-risk/types"
import {
  areaStyle,
  basinColors,
  basinFillOpacity,
  basinStateLabels,
  basinStateOrder,
} from "./basins"
import {
  floodAreaLabels,
  floodSeverityColors,
  floodSeverityLabels,
  floodSeverityOrder,
} from "./present"
import type { FloodAreaKind, FloodSeverity } from "./types"

interface FloodLegendProps {
  /** Gauges per severity, shown beside each entry. */
  counts?: Record<FloodSeverity, number>
  /** Basins per state, shown beside each entry. */
  basinCounts?: Record<BasinState, number>
  /** Some basins rest only on gauges Google has not verified. */
  unverified?: boolean
  /** How many of Google's flood areas are on the map; their key shows only with some. */
  areaCount?: number
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
  className?: string
}

function Diamond({ color }: { color: string }) {
  return (
    <span
      aria-hidden="true"
      className="inline-block size-2.5 shrink-0 rotate-45 rounded-[2px] ring-1 ring-black/40 dark:ring-black/60"
      style={{ backgroundColor: color }}
    />
  )
}

function BasinSwatch({ state }: { state: BasinState }) {
  const color = basinColors[state]
  return (
    <span
      aria-hidden="true"
      className="inline-block h-2.5 w-3.5 shrink-0 rounded-[2px] border"
      style={{
        borderColor: color,
        borderStyle: state === "no_forecast" ? "dashed" : "solid",
        backgroundColor: `color-mix(in srgb, ${color} ${Math.max(basinFillOpacity[state] * 100, 0)}%, transparent)`,
      }}
    />
  )
}

function AreaSwatch({ kind }: { kind: FloodAreaKind }) {
  const { color, opacity } = areaStyle[kind]
  return (
    <span
      aria-hidden="true"
      className="inline-block h-2.5 w-3.5 shrink-0 rounded-[2px] border"
      style={{
        borderColor: color,
        backgroundColor: `color-mix(in srgb, ${color} ${opacity * 100}%, transparent)`,
      }}
    />
  )
}

const legendAreas: FloodAreaKind[] = [
  "inundation_high",
  "inundation_medium",
  "inundation_low",
  "flash_highly_likely",
  "flash_likely",
  "notification",
  "significant_event",
]

/** Key to the flood forecast: basins, Google's areas, gauges and trend arrows. */
export function FloodLegend({
  counts,
  basinCounts,
  unverified,
  areaCount = 0,
  bare = false,
  className,
}: FloodLegendProps) {
  const entries = (
    <div className="flex flex-col gap-2.5">
      <section aria-label="Flood basins">
        <h3 className="mb-1 text-[0.6875rem] font-medium text-muted-foreground">River basins</h3>
        <ul className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-xs">
          {basinStateOrder.map((state) => (
            <li key={state} className="flex items-center justify-between gap-2">
              <span className="flex items-center gap-2">
                <BasinSwatch state={state} />
                <span className="text-muted-foreground">{basinStateLabels[state]}</span>
              </span>
              {basinCounts && <span className="numeric text-foreground">{basinCounts[state]}</span>}
            </li>
          ))}
          {unverified && (
            <li className="col-span-2 text-muted-foreground">
              Some rest only on gauges Google has not verified; the basin&apos;s panel marks them.
            </li>
          )}
        </ul>
      </section>
      {areaCount > 0 && (
        <section aria-label="Google flood areas">
          <h3 className="mb-1 text-[0.6875rem] font-medium text-muted-foreground">
            Google flood areas
          </h3>
          <ul className="grid grid-cols-1 gap-y-1 text-xs">
            {legendAreas.map((kind) => (
              <li key={kind} className="flex items-center gap-2 text-muted-foreground">
                <AreaSwatch kind={kind} />
                {floodAreaLabels[kind]}
              </li>
            ))}
          </ul>
        </section>
      )}
      <section aria-label="Flood forecast gauges">
        <h3 className="mb-1 text-[0.6875rem] font-medium text-muted-foreground">Forecast gauges</h3>
        <ul className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-xs">
          {floodSeverityOrder.map((severity) => (
            <li key={severity} className="flex items-center justify-between gap-2">
              <span className="flex items-center gap-2">
                <Diamond color={floodSeverityColors[severity]} />
                <span className="text-muted-foreground">{floodSeverityLabels[severity]}</span>
              </span>
              {counts && <span className="numeric text-foreground">{counts[severity]}</span>}
            </li>
          ))}
          <li className="flex items-center gap-2 text-muted-foreground">
            <span aria-hidden="true" className="numeric text-foreground">
              ↑↓
            </span>
            Forecast trend
          </li>
        </ul>
      </section>
    </div>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend
      title="Flood forecast"
      unit="Google"
      className={cn("max-h-[50vh] w-72 overflow-y-auto", className)}
    >
      {entries}
    </MapLegend>
  )
}
