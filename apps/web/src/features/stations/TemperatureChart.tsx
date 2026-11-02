import { DottedGrid } from "@/components/DottedGrid"
import { formatClock, formatDay } from "@/lib/format"
import { rainBars, temperaturePaths } from "./present"
import type { ThreeHourlyReading } from "./types"

const width = 300
const height = 120
const rainHeight = 36

/** Observed temperature as a line over rain as bars, with the ranges in words. */
export function TemperatureChart({
  readings,
  days,
}: {
  readings: ThreeHourlyReading[]
  days: number
}) {
  const geometry = temperaturePaths(readings, width, height)
  const rain = rainBars(readings)
  if (geometry.paths.length === 0 && geometry.dots.length === 0) {
    return (
      <DottedGrid className="px-4 py-8 text-center text-xs text-muted-foreground">
        No 3-hourly readings in the {days} days to the newest report.
      </DottedGrid>
    )
  }
  const first = readings[0]!
  const last = readings.at(-1)!
  return (
    <figure className="flex flex-col gap-2">
      <DottedGrid className="px-3 pt-4 pb-3">
        <svg
          viewBox={`0 0 ${width} ${height}`}
          className="h-32 w-full overflow-visible text-foreground"
          role="img"
          aria-label={`Observed temperature from ${geometry.yMin.toFixed(1)} to ${geometry.yMax.toFixed(1)} °C over the ${days} days to the newest report${rain.peak > 0 ? `, with rain up to ${rain.peak.toFixed(1)} mm in 3 hours` : ""}`}
          preserveAspectRatio="none"
        >
          {rain.bars.map((bar) => (
            <rect
              key={bar.x}
              x={bar.x * width - 1.5}
              width={3}
              y={height - bar.height * rainHeight}
              height={bar.height * rainHeight}
              fill="var(--rain-2)"
            />
          ))}
          {geometry.paths.map((path) => (
            <path
              key={path}
              d={path}
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              strokeLinejoin="round"
              strokeLinecap="round"
              vectorEffect="non-scaling-stroke"
            />
          ))}
          {geometry.dots.map((dot) => (
            <line
              key={dot.x}
              x1={dot.x}
              x2={dot.x + 0.01}
              y1={dot.y}
              y2={dot.y}
              stroke="currentColor"
              strokeWidth={4}
              strokeLinecap="round"
              vectorEffect="non-scaling-stroke"
            />
          ))}
        </svg>
      </DottedGrid>
      <figcaption className="numeric flex justify-between gap-2 text-[0.6875rem] text-muted-foreground">
        <time dateTime={first.observedAt}>
          {formatDay(first.observedAt)} {formatClock(first.observedAt)}
        </time>
        <span>
          {geometry.yMin.toFixed(0)}–{geometry.yMax.toFixed(0)} °C
          {rain.peak > 0 && ` · rain to ${rain.peak.toFixed(1)} mm`}
        </span>
        <time dateTime={last.observedAt}>
          {formatDay(last.observedAt)} {formatClock(last.observedAt)}
        </time>
      </figcaption>
    </figure>
  )
}
