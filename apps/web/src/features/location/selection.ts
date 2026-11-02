/**
 * What the detail panel shows, and how it is written in the URL so a view
 * can be shared or bookmarked:
 *
 * - `?at=7.2931,80.6350&name=Kandy`: weather at a point, optionally named
 *   (a searched place);
 * - `?region=LK-21`: an administrative region, at its label point;
 * - `?station=peradeniya`: a river gauge;
 * - `?gauge=hybas_4121155030`: a Google flood forecast gauge.
 */
export type Selection =
  | { kind: "point"; lat: number; lon: number; name?: string }
  | { kind: "region"; id: string }
  | { kind: "station"; id: string }
  | { kind: "gauge"; id: string }

/** The area the API answers for (`/v1/summary`, `/v1/regions/lookup`). */
export const pointBounds = { south: 5.5, north: 10.5, west: 79, east: 82.5 }

const selectionKeys = ["at", "name", "region", "station", "gauge"]
const maxNameLength = 80

/** Coordinates rounded to four decimals (about 11 m), enough for weather and short in URLs. */
export function roundCoordinate(value: number): number {
  return Math.round(value * 10_000) / 10_000
}

export function inPointBounds(lat: number, lon: number): boolean {
  return (
    lat >= pointBounds.south &&
    lat <= pointBounds.north &&
    lon >= pointBounds.west &&
    lon <= pointBounds.east
  )
}

/** A region code as the API writes it: "lk21", "LK-21" and "LK-2-1" are all "LK-21". */
export function normaliseRegionId(value: string): string | null {
  const digits = value.trim().toUpperCase().replace(/^LK/, "").replace(/-/g, "")
  if (!/^\d{1,7}$/.test(digits) || ![1, 2, 4, 7].includes(digits.length)) return null
  return `LK-${digits}`
}

/** A short identifier from the URL: letters, digits, `_`, `-` and `.` only. */
function safeId(value: string | null): string | null {
  if (!value) return null
  const trimmed = value.trim()
  return /^[\w.-]{1,120}$/.test(trimmed) ? trimmed : null
}

/** Reads the selection from a query string; anything malformed is ignored. */
export function parseSelection(search: string): Selection | null {
  const params = new URLSearchParams(search)

  const at = params.get("at")
  if (at) {
    const [lat, lon] = at.split(",").map((part) => Number(part.trim()))
    if (
      lat !== undefined &&
      lon !== undefined &&
      Number.isFinite(lat) &&
      Number.isFinite(lon) &&
      inPointBounds(lat, lon)
    ) {
      const name = params.get("name")?.trim().slice(0, maxNameLength)
      return {
        kind: "point",
        lat: roundCoordinate(lat),
        lon: roundCoordinate(lon),
        ...(name ? { name } : {}),
      }
    }
  }

  const region = params.get("region")
  if (region) {
    const id = normaliseRegionId(region)
    if (id) return { kind: "region", id }
  }

  const station = safeId(params.get("station"))
  if (station) return { kind: "station", id: station }

  const gauge = safeId(params.get("gauge"))
  if (gauge) return { kind: "gauge", id: gauge }

  return null
}

/**
 * The query string for a selection, keeping parameters that are not about
 * the selection (for example `kit`). Returns "" or "?…".
 */
export function selectionSearch(selection: Selection | null, current = ""): string {
  const params = new URLSearchParams(current)
  for (const key of selectionKeys) params.delete(key)
  if (selection?.kind === "point") {
    params.set(
      "at",
      `${roundCoordinate(selection.lat).toFixed(4)},${roundCoordinate(selection.lon).toFixed(4)}`,
    )
    if (selection.name) params.set("name", selection.name.slice(0, maxNameLength))
  } else if (selection?.kind === "region") {
    params.set("region", selection.id)
  } else if (selection?.kind === "station") {
    params.set("station", selection.id)
  } else if (selection?.kind === "gauge") {
    params.set("gauge", selection.id)
  }
  const text = params.toString().replace(/%2C/g, ",")
  return text ? `?${text}` : ""
}

/** Whether two selections show the same thing. */
export function sameSelection(a: Selection | null, b: Selection | null): boolean {
  if (a === null || b === null) return a === b
  if (a.kind !== b.kind) return false
  if (a.kind === "point" && b.kind === "point") {
    return (
      roundCoordinate(a.lat) === roundCoordinate(b.lat) &&
      roundCoordinate(a.lon) === roundCoordinate(b.lon) &&
      a.name === b.name
    )
  }
  return "id" in a && "id" in b && a.id === b.id
}
