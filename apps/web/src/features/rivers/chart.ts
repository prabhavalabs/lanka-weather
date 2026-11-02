export interface ChartReading {
  observedAt: string
  waterLevel: number
}

export interface ChartThreshold {
  label: string
  value: number
  color: string
}

export interface ChartGeometry {
  /** SVG path of the level line, or empty with fewer than two readings. */
  path: string
  thresholds: (ChartThreshold & { y: number })[]
  /** Level at the top and bottom of the plot. */
  yMax: number
  yMin: number
  /** Position of the latest reading. */
  last: { x: number; y: number } | null
}

/**
 * Scales a level series and its thresholds into a plot of the given size.
 * The vertical range always includes the thresholds, so the distance to
 * flooding is visible even when the river is low.
 */
export function chartGeometry(
  readings: ChartReading[],
  thresholds: ChartThreshold[],
  width: number,
  height: number,
): ChartGeometry {
  const levels = readings.map((reading) => reading.waterLevel)
  const values = [...levels, ...thresholds.map((threshold) => threshold.value)]
  if (values.length === 0) return { path: "", thresholds: [], yMax: 1, yMin: 0, last: null }

  let yMin = Math.min(...values)
  let yMax = Math.max(...values)
  const pad = Math.max((yMax - yMin) * 0.08, 0.1)
  yMin -= pad
  yMax += pad
  const y = (value: number) => height - ((value - yMin) / (yMax - yMin)) * height

  const times = readings.map((reading) => new Date(reading.observedAt).getTime())
  const start = Math.min(...times)
  const span = Math.max(...times) - start || 1
  const x = (time: number) => ((time - start) / span) * width

  const points = readings.map(
    (reading, index) => [x(times[index]!), y(reading.waterLevel)] as const,
  )
  const path =
    points.length < 2
      ? ""
      : points
          .map(([px, py], index) => `${index === 0 ? "M" : "L"}${px.toFixed(1)},${py.toFixed(1)}`)
          .join("")
  const lastPoint = points.at(-1)

  return {
    path,
    thresholds: thresholds.map((threshold) => ({ ...threshold, y: y(threshold.value) })),
    yMax,
    yMin,
    last: lastPoint ? { x: lastPoint[0], y: lastPoint[1] } : null,
  }
}
