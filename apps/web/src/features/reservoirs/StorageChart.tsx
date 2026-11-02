import { DottedGrid } from "@/components/DottedGrid"
import { chartGeometry } from "../rivers/chart"
import { formatReadingDay } from "./present"
import type { ReservoirReading } from "./types"

const width = 300
const height = 96

/** Storage over the days it was read, with full capacity as a reference line. */
export function StorageChart({ readings, days }: { readings: ReservoirReading[]; days: number }) {
  const series = readings.filter(
    (reading): reading is ReservoirReading & { storagePercent: number } =>
      reading.storagePercent !== null,
  )
  if (series.length < 2) {
    return (
      <DottedGrid className="px-4 py-6 text-center text-xs text-muted-foreground">
        {series.length === 0
          ? `No current readings collected in the last ${days} days.`
          : `One reading so far, on ${formatReadingDay(series[0]!.day)}. History builds up day by day.`}
      </DottedGrid>
    )
  }
  const geometry = chartGeometry(
    series.map((reading) => ({
      observedAt: `${reading.day}T12:00:00+05:30`,
      waterLevel: reading.storagePercent,
    })),
    [{ label: "Full", value: 100, color: "var(--muted-foreground)" }],
    width,
    height,
  )
  const first = series[0]!
  const last = series.at(-1)!
  return (
    <figure className="flex flex-col gap-2">
      <DottedGrid className="px-3 pt-4 pb-3">
        <svg
          viewBox={`0 0 ${width} ${height}`}
          className="h-24 w-full overflow-visible text-foreground"
          role="img"
          aria-label={`Storage from ${first.storagePercent}% on ${formatReadingDay(first.day)} to ${last.storagePercent}% on ${formatReadingDay(last.day)}, over ${series.length} readings`}
          preserveAspectRatio="none"
        >
          {geometry.thresholds.map((threshold) => (
            <line
              key={threshold.label}
              x1={0}
              x2={width}
              y1={threshold.y}
              y2={threshold.y}
              stroke={threshold.color}
              strokeWidth={1.5}
              strokeDasharray="5 4"
              vectorEffect="non-scaling-stroke"
            />
          ))}
          <path
            d={geometry.path}
            fill="none"
            stroke="currentColor"
            strokeWidth={2}
            strokeLinejoin="round"
            strokeLinecap="round"
            vectorEffect="non-scaling-stroke"
          />
        </svg>
      </DottedGrid>
      <figcaption className="numeric flex justify-between text-[0.6875rem] text-muted-foreground">
        <span>{formatReadingDay(first.day)}</span>
        <span>dashed line: full</span>
        <span>{formatReadingDay(last.day)}</span>
      </figcaption>
    </figure>
  )
}
