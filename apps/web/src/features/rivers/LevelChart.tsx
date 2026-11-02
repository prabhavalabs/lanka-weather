import { DottedGrid } from "@/components/DottedGrid"
import { formatClock, formatDay } from "@/lib/format"
import { chartGeometry, type ChartReading, type ChartThreshold } from "./chart"

interface LevelChartProps {
  readings: ChartReading[]
  thresholds: ChartThreshold[]
  unit: string
  hours: number
}

const width = 300
const height = 120

/** Water level over time with the station's flood thresholds as reference lines. */
export function LevelChart({ readings, thresholds, unit, hours }: LevelChartProps) {
  const geometry = chartGeometry(readings, thresholds, width, height)
  if (readings.length === 0) {
    return (
      <DottedGrid className="px-4 py-8 text-center text-xs text-muted-foreground">
        No readings in the last {hours} hours.
      </DottedGrid>
    )
  }
  const first = readings[0]!
  const last = readings.at(-1)!
  const area =
    geometry.path && geometry.last
      ? `${geometry.path}L${geometry.last.x},${height}L0,${height}Z`
      : ""

  return (
    <figure className="flex flex-col gap-2">
      <DottedGrid className="px-3 pt-4 pb-3">
        <svg
          viewBox={`0 0 ${width} ${height}`}
          className="h-32 w-full overflow-visible text-foreground"
          role="img"
          aria-label={`Water level over the last ${hours} hours, in ${unit}`}
          preserveAspectRatio="none"
        >
          {area && <path d={area} fill="currentColor" opacity={0.06} />}
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
        {geometry.last && (
          // A dot drawn in HTML so the stretched SVG does not turn it into an ellipse.
          <span
            aria-hidden="true"
            className="pointer-events-none absolute size-2.5 -translate-1/2 rounded-full bg-brand ring-4 ring-brand/25"
            style={{
              left: `calc(0.75rem + ${(geometry.last.x / width).toFixed(4)} * (100% - 1.5rem))`,
              top: `calc(1rem + ${(geometry.last.y / height).toFixed(4)} * 8rem)`,
            }}
          />
        )}
      </DottedGrid>
      <figcaption className="numeric flex justify-between text-[0.6875rem] text-muted-foreground">
        <time dateTime={first.observedAt}>
          {formatDay(first.observedAt)} {formatClock(first.observedAt)}
        </time>
        <span>
          {geometry.yMin.toFixed(1)}–{geometry.yMax.toFixed(1)} {unit}
        </span>
        <time dateTime={last.observedAt}>
          {formatDay(last.observedAt)} {formatClock(last.observedAt)}
        </time>
      </figcaption>
    </figure>
  )
}
