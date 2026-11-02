import { MapLegend } from "@/components/weather/MapLegend"
import { cn } from "@/lib/utils"
import {
  landslideLevels,
  levelActions,
  levelColors,
  levelLabels,
  type LandslideLevel,
} from "./present"

/** The hatches on the map, in CSS: dots, lines, cross-hatch. */
const hatches: Record<LandslideLevel, string> = {
  1: "radial-gradient(circle at 50% 50%, rgb(10 10 10 / 0.55) 1px, transparent 1.2px)",
  2: "repeating-linear-gradient(45deg, rgb(10 10 10 / 0.55) 0 1px, transparent 1px 5px)",
  3: "repeating-linear-gradient(45deg, rgb(10 10 10 / 0.6) 0 1px, transparent 1px 3px), repeating-linear-gradient(-45deg, rgb(10 10 10 / 0.6) 0 1px, transparent 1px 3px)",
}

export function LevelSwatch({ level }: { level: LandslideLevel }) {
  return (
    <span
      aria-hidden="true"
      className="inline-block size-3.5 shrink-0 rounded-[3px] ring-1 ring-black/40 dark:ring-black/60"
      style={{
        backgroundColor: levelColors[level],
        backgroundImage: hatches[level],
        backgroundSize: level === 1 ? "6px 6px" : undefined,
      }}
    />
  )
}

interface LandslideLegendProps {
  counts?: Record<"1" | "2" | "3", number>
  bare?: boolean
  className?: string
}

/** Key to landslide levels: NBRO's words, with colour and hatch. */
export function LandslideLegend({ counts, bare = false, className }: LandslideLegendProps) {
  const entries = (
    <ul className="flex flex-col gap-1.5 text-xs">
      {landslideLevels.map((level) => (
        <li key={level} className="flex items-center justify-between gap-2">
          <span className="flex items-center gap-2">
            <LevelSwatch level={level} />
            <span className="text-muted-foreground">
              {levelLabels[level]} · {levelActions[level]}
            </span>
          </span>
          {counts && <span className="numeric text-foreground">{counts[level]}</span>}
        </li>
      ))}
    </ul>
  )
  if (bare) return <div className={cn(className)}>{entries}</div>
  return (
    <MapLegend title="Landslide warnings" unit="NBRO" className={cn("w-64", className)}>
      {entries}
    </MapLegend>
  )
}

/** Key to the district outlines. */
export function OfficialDistrictsLegend({ className }: { className?: string }) {
  return (
    <ul className={cn("flex flex-col gap-1.5 text-xs text-muted-foreground", className)}>
      <li className="flex items-center gap-2">
        <span aria-hidden="true" className="inline-block h-0 w-5 border-t-2 border-brand" />
        Landslide warning in force
      </li>
      <li className="flex items-center gap-2">
        <span
          aria-hidden="true"
          className="inline-block h-0 w-5 border-t-2 border-dashed border-brand"
        />
        Mentioned in bulletin
      </li>
    </ul>
  )
}
