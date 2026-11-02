/**
 * What a station says, in words and numbers: which reading to draw and how to
 * say a feed is old. Pure, so it is tested without a browser.
 */
import { formatAge, formatNumber } from "@/lib/format"
import type { MetMeta, MetStationProperties, ThreeHourlyReading } from "./types"

/**
 * The number drawn on a station: the reading, or the daily maximum when only
 * the daily report is there.
 */
export function markerTemperature(
  station: Pick<MetStationProperties, "temperatureC" | "tempMaxC">,
): number | null {
  return station.temperatureC ?? station.tempMaxC
}

/** "Rain, 3 h" or "Rain, 24 h to 08:30". */
export function rainLabel(station: Pick<MetStationProperties, "reading" | "rainHours">): string {
  return station.reading === "daily" ? "Rain, 24 h to 08:30" : `Rain, ${station.rainHours} h`
}

/** Which feed a reading came from, in words. */
export function readingLabel(station: Pick<MetStationProperties, "reading">): string {
  if (station.reading === "3h") return "3-hourly report"
  if (station.reading === "daily") return "Daily report"
  return "No readings"
}

/**
 * The warning for when the Met Department readings are not current, naming how
 * old the copy is. Null while a feed is current.
 */
export function feedNotice(meta: MetMeta, now: Date = new Date()): string | null {
  if (!meta.stale) {
    if (meta.threeHourly.stale && !meta.daily.stale) {
      return "The 3-hourly reports have stopped updating, so stations show the daily report."
    }
    return null
  }
  const newest = [meta.threeHourly.newestObservedAt, meta.daily.newestObservedAt]
    .filter((value): value is string => value !== null)
    .sort()
    .at(-1)
  if (!newest) return "No Department of Meteorology readings have been collected yet."
  return `Not current: the newest Department of Meteorology reading is ${formatAge(newest, now)}. Our source for these reports has stopped updating.`
}

/** A gap longer than this breaks the line: the source has days with no reports. */
const gapMs = 7 * 60 * 60 * 1000

/**
 * Temperature history as SVG paths, one per unbroken run of readings, so a day
 * the source missed shows as a gap and not as a straight line across it.
 */
export function temperaturePaths(
  readings: ThreeHourlyReading[],
  width: number,
  height: number,
): { paths: string[]; dots: { x: number; y: number }[]; yMin: number; yMax: number } {
  const times = readings.map((reading) => new Date(reading.observedAt).getTime())
  const points = readings
    .map((reading, index) => ({ time: times[index]!, value: reading.temperatureC }))
    .filter((point): point is { time: number; value: number } => point.value !== null)
  if (points.length === 0) return { paths: [], dots: [], yMin: 0, yMax: 1 }
  const values = points.map((point) => point.value)
  const pad = Math.max((Math.max(...values) - Math.min(...values)) * 0.08, 0.5)
  const yMin = Math.min(...values) - pad
  const yMax = Math.max(...values) + pad
  const start = Math.min(...times)
  const span = Math.max(...times) - start || 1
  const x = (time: number) => ((time - start) / span) * width
  const y = (value: number) => height - ((value - yMin) / (yMax - yMin)) * height
  const runs: (typeof points)[] = []
  points.forEach((point, index) => {
    const previous = points[index - 1]
    if (!previous || point.time - previous.time > gapMs) runs.push([point])
    else runs.at(-1)!.push(point)
  })
  return {
    paths: runs
      .filter((run) => run.length > 1)
      .map((run) =>
        run
          .map(
            (point, index) =>
              `${index === 0 ? "M" : "L"}${x(point.time).toFixed(1)},${y(point.value).toFixed(1)}`,
          )
          .join(""),
      ),
    // A reading with no neighbour on either side has no line to be on.
    dots: runs
      .filter((run) => run.length === 1)
      .map(([point]) => ({
        x: x(point!.time),
        y: y(point!.value),
      })),
    yMin,
    yMax,
  }
}

/** Each day's rain from 3-hourly readings as bars: position 0–1 and height 0–1 across the span. */
export function rainBars(readings: ThreeHourlyReading[]) {
  const withRain = readings.filter((reading) => reading.rainMm !== null)
  const times = readings.map((reading) => new Date(reading.observedAt).getTime())
  if (withRain.length === 0 || times.length < 2) return { bars: [], peak: 0 }
  const start = Math.min(...times)
  const span = Math.max(...times) - start || 1
  const peak = Math.max(...withRain.map((reading) => reading.rainMm!))
  const bars = withRain
    .filter((reading) => reading.rainMm! > 0)
    .map((reading) => ({
      x: (new Date(reading.observedAt).getTime() - start) / span,
      height: peak > 0 ? reading.rainMm! / peak : 0,
      mm: reading.rainMm!,
    }))
  return { bars, peak }
}

/** "16.1 °C", or a dash without a value. */
export function formatTemperature(value: number | null): string {
  return value === null ? "–" : `${formatNumber(value, 1)} °C`
}

/** "2.4 mm", or a dash without a value. */
export function formatRain(value: number | null): string {
  return value === null ? "–" : `${formatNumber(value, 1)} mm`
}
