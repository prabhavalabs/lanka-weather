import { MapLegend } from "@/components/weather/MapLegend"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { riskLabels } from "./present"
import type { RiskLevel } from "./types"

/** A short piece of river as drawn on the map: the line and its non-colour cue. */
function Line({
  color,
  width,
  dashed = false,
  mark,
}: {
  color: string
  width: number
  dashed?: boolean
  mark?: "dot" | "triangle" | "chevron"
}) {
  return (
    <svg aria-hidden="true" viewBox="0 0 28 12" className="h-3 w-7 shrink-0 overflow-visible">
      <line
        x1={1}
        x2={27}
        y1={6}
        y2={6}
        stroke={color}
        strokeWidth={width}
        strokeLinecap={dashed ? "butt" : "round"}
        strokeDasharray={dashed ? "4 3" : undefined}
      />
      {mark === "dot" && <circle cx={14} cy={6} r={1.4} fill="var(--river-casing)" />}
      {mark === "triangle" && <path d="M18 6 11 2.8v6.4Z" fill="var(--river-casing)" />}
      {mark === "chevron" && (
        <path
          d="m9 2.5 4 3.5-4 3.5M14 2.5l4 3.5-4 3.5"
          fill="none"
          stroke="var(--river-casing)"
          strokeWidth={1.6}
        />
      )}
    </svg>
  )
}

const entries: { level: RiskLevel; line: Parameters<typeof Line>[0] }[] = [
  {
    level: "severe",
    line: { color: "var(--status-major-flood)", width: 6, mark: "chevron" },
  },
  {
    level: "high",
    line: { color: "var(--status-minor-flood)", width: 5, mark: "triangle" },
  },
  {
    level: "watch",
    line: { color: "var(--status-alert)", width: 4, mark: "dot" },
  },
  { level: "normal", line: { color: "var(--status-normal)", width: 3 } },
  {
    level: "unknown",
    line: { color: "var(--river-unassessed)", width: 3, dashed: true },
  },
]

interface RiverRiskLegendProps {
  /** A card over the map; inside the layer list it goes without one. */
  card?: boolean
  className?: string
}

/** Key to the river lines. A dashed line is not a safe one: nothing measures it. */
export function RiverRiskLegend({ card = false, className }: RiverRiskLegendProps) {
  const t = useT()
  const list = (
    <ul className="flex flex-col gap-1 text-xs">
      {entries.map((entry) => {
        const note = t(`rivers.risk.legend.${entry.level}`)
        return (
          <li key={entry.level} className="flex min-w-0 items-center gap-2">
            <Line {...entry.line} />
            <span className="shrink-0 font-medium text-foreground">{riskLabels[entry.level]}</span>
            <span className="min-w-0 truncate text-muted-foreground" title={note}>
              {note}
            </span>
          </li>
        )
      })}
    </ul>
  )
  if (!card) return <div className={className}>{list}</div>
  return (
    <MapLegend
      title={t("rivers.risk.legend.title")}
      unit={t("terms.estimate")}
      className={cn("w-72", className)}
    >
      {list}
    </MapLegend>
  )
}
