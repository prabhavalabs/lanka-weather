import { useId, type ReactNode } from "react"
import { Switch } from "@/components/ui/switch"
import { MapLegend } from "@/components/weather/MapLegend"
import { formatAge } from "@/lib/format"
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
  floodSeverityColors,
  floodSeverityGlyphs,
  floodSeverityLabels,
  floodSeverityOrder,
  inundationWords,
} from "./present"
import type { FloodAreaKind, FloodSeverity } from "./types"
import type { FloodView } from "./view"

interface FloodLegendProps {
  /** What is shown, and how to change it. */
  view: FloodView
  onViewChange: (view: FloodView) => void
  /** Gauges per severity on the map, shown beside each entry. */
  counts?: Record<FloodSeverity, number>
  /** Basins per state, shown beside each entry. */
  basinCounts?: Record<BasinState, number>
  /** Some basins rest only on lower-confidence gauges. */
  unverified?: boolean
  /** How many flash-flood areas were read, and how old that reading is. */
  flashAreas?: number
  checkedAt?: string | null
  /** Lower-confidence gauges read, shown on the Extended coverage toggle. */
  lowerConfidenceGauges?: number
  /** Without the card, e.g. inside the layer list. */
  bare?: boolean
  className?: string
}

function GaugeDot({ color }: { color: string }) {
  return (
    <span
      aria-hidden="true"
      className="inline-block size-3 shrink-0 rounded-full border-2 border-[var(--flood-ring)] shadow-[0_0_0_1px_rgb(0_0_0/0.25)]"
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
        backgroundColor: `color-mix(in srgb, ${color} ${basinFillOpacity[state] * 100}%, transparent)`,
      }}
    />
  )
}

function AreaSwatch({ kind }: { kind: FloodAreaKind }) {
  const { color, opacity } = areaStyle[kind]
  return (
    <span
      aria-hidden="true"
      className="inline-block size-3 shrink-0 border-[1.5px]"
      style={{
        borderColor: color,
        backgroundColor: `color-mix(in srgb, ${color} ${opacity * 100}%, transparent)`,
      }}
    />
  )
}

/** A heading with its one toggle, as Flood Hub's View options has them. */
function Group({
  title,
  note,
  checked,
  onCheckedChange,
  label,
  children,
}: {
  title: string
  note?: string
  checked?: boolean
  onCheckedChange?: (checked: boolean) => void
  label?: string
  children?: ReactNode
}) {
  const id = useId()
  return (
    <section aria-labelledby={id} className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0">
          <h3 id={id} className="text-xs font-medium text-foreground">
            {title}
          </h3>
          {note && <p className="text-[0.6875rem] text-muted-foreground">{note}</p>}
        </div>
        {onCheckedChange && (
          <Switch
            size="sm"
            checked={checked}
            onCheckedChange={onCheckedChange}
            aria-label={label ?? title}
          />
        )}
      </div>
      {children}
    </section>
  )
}

/** Google's flood forecast, laid out like Flood Hub's View options. */
export function FloodLegend({
  view,
  onViewChange,
  counts,
  basinCounts,
  unverified,
  flashAreas = 0,
  checkedAt,
  lowerConfidenceGauges = 0,
  bare = false,
  className,
}: FloodLegendProps) {
  const set = (key: keyof FloodView) => (checked: boolean) =>
    onViewChange({ ...view, [key]: checked })
  const entries = (
    <div className="flex flex-col gap-3">
      <Group title="Riverine floods" note="Expected severity">
        <ul className="flex flex-col gap-1.5 text-xs">
          {floodSeverityOrder.map((severity) => (
            <li key={severity} className="flex items-center justify-between gap-2">
              <span className="flex items-center gap-2">
                <GaugeDot color={floodSeverityColors[severity]} />
                <span className="text-muted-foreground">{floodSeverityLabels[severity]}</span>
                {floodSeverityGlyphs[severity] && (
                  <span aria-hidden="true" className="numeric text-[0.6875rem] text-foreground">
                    {floodSeverityGlyphs[severity]}
                  </span>
                )}
              </span>
              <span className="flex items-center gap-2">
                {counts && <span className="numeric text-foreground">{counts[severity]}</span>}
                {severity === "no_flooding" && (
                  <Switch
                    size="sm"
                    checked={view.normal}
                    onCheckedChange={set("normal")}
                    aria-label="Show gauges forecast normal"
                  />
                )}
              </span>
            </li>
          ))}
          <li className="flex items-center gap-2 text-muted-foreground">
            <span aria-hidden="true" className="numeric text-foreground">
              ↑↓
            </span>
            Forecast trend
          </li>
        </ul>
      </Group>

      <Group
        title="Flash floods"
        note="Within 24 h"
        checked={view.flash}
        onCheckedChange={set("flash")}
        label="Show flash-flood areas"
      >
        {view.flash && (
          <ul className="flex flex-col gap-1.5 text-xs text-muted-foreground">
            <li className="flex items-center gap-2">
              <AreaSwatch kind="flash_highly_likely" />
              Highly likely
            </li>
            <li className="flex items-center gap-2">
              <AreaSwatch kind="flash_likely" />
              Likely
            </li>
          </ul>
        )}
        {flashAreas === 0 && (
          <p role="status" className="text-xs text-foreground">
            No flash-flood areas for Sri Lanka right now
            {checkedAt && (
              <span className="numeric text-muted-foreground"> · read {formatAge(checkedAt)}</span>
            )}
          </p>
        )}
      </Group>

      <div className="rounded-xl border border-border bg-inset p-2.5">
        <Group
          title={
            lowerConfidenceGauges > 0
              ? `Extended coverage (${lowerConfidenceGauges})`
              : "Extended coverage"
          }
          note="Lower-confidence gauges only"
          checked={view.extended}
          onCheckedChange={set("extended")}
          label="Show lower-confidence gauges"
        >
          {view.extended && (
            <p className="flex items-start gap-2 text-xs text-muted-foreground">
              <span
                aria-hidden="true"
                className="mt-0.5 inline-block size-3 shrink-0 rounded-full border-2 bg-[var(--flood-ring)]"
                style={{ borderColor: floodSeverityColors.no_flooding }}
              />
              Hollow dots. Google has not verified these models. They do not raise river risk,
              alerts or the overview.
            </p>
          )}
        </Group>
      </div>

      <Group title="Significant flood events" checked={view.events} onCheckedChange={set("events")}>
        {view.events && (
          <p className="flex items-center gap-2 text-xs text-muted-foreground">
            <AreaSwatch kind="significant_event" />
            Event present
          </p>
        )}
      </Group>

      <Group
        title="Inundation probability"
        checked={view.inundation}
        onCheckedChange={set("inundation")}
      >
        {view.inundation && (
          <ul className="flex flex-col gap-1.5 text-xs text-muted-foreground">
            {(Object.keys(inundationWords) as (keyof typeof inundationWords)[]).map((kind) => (
              <li key={kind} className="flex items-center gap-2">
                <AreaSwatch kind={kind} />
                {inundationWords[kind].label}
              </li>
            ))}
          </ul>
        )}
      </Group>

      <Group title="River basins" note="The worst forecast or gauge in each">
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
        </ul>
        <p className="text-xs text-muted-foreground">
          No forecast means nothing is known there, not that it is calm.
          {unverified && " Some rest only on lower-confidence gauges; each basin's panel says so."}
        </p>
      </Group>

      <p className="text-[0.6875rem] text-muted-foreground">
        Google Flood Hub (Flood Forecasting API). Forecasts, not official warnings.
      </p>
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
