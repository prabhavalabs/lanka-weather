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
      <p className="py-6 text-center text-xs text-slate-400">
        No readings in the last {hours} hours.
      </p>
    )
  }
  return (
    <figure>
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="h-32 w-full overflow-visible"
        role="img"
        aria-label={`Water level over the last ${hours} hours, in ${unit}`}
        preserveAspectRatio="none"
      >
        {geometry.thresholds.map((threshold) => (
          <g key={threshold.label}>
            <line
              x1={0}
              x2={width}
              y1={threshold.y}
              y2={threshold.y}
              stroke={threshold.color}
              strokeWidth={1}
              strokeDasharray="4 3"
              vectorEffect="non-scaling-stroke"
            />
          </g>
        ))}
        <path
          d={geometry.path}
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          strokeLinejoin="round"
          vectorEffect="non-scaling-stroke"
        />
        {geometry.last && (
          <circle cx={geometry.last.x} cy={geometry.last.y} r={3} fill="currentColor" />
        )}
      </svg>
      <figcaption className="mt-1 flex justify-between text-[11px] text-slate-400">
        <span>earliest</span>
        <span>
          {geometry.yMin.toFixed(1)}–{geometry.yMax.toFixed(1)} {unit}
        </span>
        <span>latest</span>
      </figcaption>
    </figure>
  )
}
