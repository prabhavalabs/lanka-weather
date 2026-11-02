/**
 * What the detail panel shows, and how it is written in the URL so a view
 * can be shared or bookmarked:
 *
 * - `?at=7.2931,80.6350&name=Kandy`: weather at a point, optionally named
 *   (a searched place);
 * - `?region=LK-21`: an administrative region, at its label point;
 * - `?station=peradeniya`: a river gauge;
 * - `?gauge=hybas_4121155030`: a Google flood forecast gauge;
 * - `?river=kelani-ganga&reach=41405515`: a river, optionally a stretch of it;
 * - `?basin=kelani-ganga`: a flood-forecast river basin;
 * - `?area=...`: one of Google's flood polygons;
 * - `?past=kelani-ganga-flood-2016-major`: a past flood.
 * - `?landslide=LK-2136`: a DS division's landslide warning;
 * - `?quake=us7000tty7`: an earthquake;
 * - `?cyclone=1001346`: a tropical cyclone;
 * - `?warnings=all`, `?warnings=water_level`, optionally with `&district=LK-21`:
 *   the official warnings, of one type or in one district.
 * - `?met=colombo`: a Department of Meteorology station.
 * - `?reservoir=minneriya`: a reservoir;
 * - `?reservoirs=active`: the list of reservoirs spilling or releasing.
 */
export type Selection =
  | { kind: "point"; lat: number; lon: number; name?: string }
  | { kind: "region"; id: string }
  | { kind: "station"; id: string }
  | { kind: "gauge"; id: string }
  | { kind: "river"; id: string; reach?: number }
  | { kind: "basin"; id: string }
  | { kind: "area"; id: string }
  | { kind: "past"; id: string }
  | { kind: "landslide"; id: string }
  | { kind: "quake"; id: string }
  | { kind: "cyclone"; id: string }
  | { kind: "warnings"; type?: string; district?: string }
  | { kind: "met"; id: string }
  | { kind: "reservoir"; id: string }
  | { kind: "reservoirs" }

/** The area the API answers for (`/v1/summary`, `/v1/regions/lookup`). */
export const pointBounds = { south: 5.5, north: 10.5, west: 79, east: 82.5 }

const selectionKeys = [
  "at",
  "name",
  "region",
  "station",
  "gauge",
  "reservoir",
  "reservoirs",
  "met",
  "river",
  "reach",
  "basin",
  "area",
  "past",
  "landslide",
  "quake",
  "cyclone",
  "warnings",
  "district",
]

const warningTypes = ["landslide", "water_level", "weather", "situation"]
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

  const river = safeId(params.get("river"))
  if (river) {
    const reach = Number(params.get("reach"))
    return Number.isSafeInteger(reach) && reach > 0
      ? { kind: "river", id: river, reach }
      : { kind: "river", id: river }
  }

  const basin = safeId(params.get("basin"))
  if (basin) return { kind: "basin", id: basin }

  const area = safeId(params.get("area"))
  if (area) return { kind: "area", id: area }

  const past = safeId(params.get("past"))
  if (past) return { kind: "past", id: past }
  const landslide = params.get("landslide")
  if (landslide) {
    const id = normaliseRegionId(landslide)
    if (id) return { kind: "landslide", id }
  }

  const quake = safeId(params.get("quake"))
  if (quake) return { kind: "quake", id: quake }

  const cyclone = safeId(params.get("cyclone"))
  if (cyclone) return { kind: "cyclone", id: cyclone }

  const warnings = params.get("warnings")
  if (warnings) {
    const district = normaliseRegionId(params.get("district") ?? "")
    return {
      kind: "warnings",
      ...(warningTypes.includes(warnings) ? { type: warnings } : {}),
      ...(district ? { district } : {}),
    }
  }
  const met = safeId(params.get("met"))
  if (met) return { kind: "met", id: met }
  const reservoir = safeId(params.get("reservoir"))
  if (reservoir) return { kind: "reservoir", id: reservoir }

  if (params.has("reservoirs")) return { kind: "reservoirs" }

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
  } else if (selection?.kind === "river") {
    params.set("river", selection.id)
    if (selection.reach) params.set("reach", String(selection.reach))
  } else if (selection?.kind === "basin") {
    params.set("basin", selection.id)
  } else if (selection?.kind === "area") {
    params.set("area", selection.id)
  } else if (selection?.kind === "past") {
    params.set("past", selection.id)
  } else if (selection?.kind === "landslide") {
    params.set("landslide", selection.id)
  } else if (selection?.kind === "quake") {
    params.set("quake", selection.id)
  } else if (selection?.kind === "cyclone") {
    params.set("cyclone", selection.id)
  } else if (selection?.kind === "warnings") {
    params.set("warnings", selection.type ?? "all")
    if (selection.district) params.set("district", selection.district)
  } else if (selection?.kind === "met") {
    params.set("met", selection.id)
  } else if (selection?.kind === "reservoir") {
    params.set("reservoir", selection.id)
  } else if (selection?.kind === "reservoirs") {
    params.set("reservoirs", "active")
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
  if (a.kind === "river" && b.kind === "river") return a.id === b.id && a.reach === b.reach
  if (a.kind === "warnings" && b.kind === "warnings") {
    return a.type === b.type && a.district === b.district
  }
  if (a.kind === "reservoirs") return true
  return "id" in a && "id" in b && a.id === b.id
}
