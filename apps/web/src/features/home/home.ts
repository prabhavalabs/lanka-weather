/**
 * The home place: one place a person keeps, saved in this browser only, that
 * the house button on the map brings back. Pure, so it is tested without a
 * browser; nothing here leaves the device.
 */
import { inPointBounds, roundCoordinate, type Selection } from "../location/selection"

export const homeStorageKey = "lanka-weather:home"

export interface HomePlace {
  lat: number
  lon: number
  /** As the place's panel named it when it was saved. */
  name: string
  /** Set when home is a district or other region, which reopens as that region. */
  region?: string
}

const maxNameLength = 120

/** A stored value read back, or null when it is missing or not a usable place. */
export function parseHome(value: unknown): HomePlace | null {
  if (!value || typeof value !== "object") return null
  const { lat, lon, name, region } = value as Record<string, unknown>
  if (typeof lat !== "number" || typeof lon !== "number" || !Number.isFinite(lat + lon)) {
    return null
  }
  if (!inPointBounds(lat, lon)) return null
  const label = typeof name === "string" ? name.trim().slice(0, maxNameLength) : ""
  return {
    lat: roundCoordinate(lat),
    lon: roundCoordinate(lon),
    name: label,
    ...(typeof region === "string" && region ? { region } : {}),
  }
}

/** The home place for the open panel: a point with its name, or the region itself. */
export function homeFor(
  selection: Extract<Selection, { kind: "point" } | { kind: "region" }>,
  location: { lat: number; lon: number },
  name: string,
): HomePlace | null {
  return parseHome({
    lat: location.lat,
    lon: location.lon,
    name,
    ...(selection.kind === "region" ? { region: selection.id } : {}),
  })
}

/** What going home opens. */
export function homeSelection(home: HomePlace): Selection {
  if (home.region) return { kind: "region", id: home.region }
  return {
    kind: "point",
    lat: home.lat,
    lon: home.lon,
    ...(home.name ? { name: home.name } : {}),
  }
}

/** Whether the open panel is the home place. */
export function isHome(
  home: HomePlace | null,
  selection: Extract<Selection, { kind: "point" } | { kind: "region" }>,
  location: { lat: number; lon: number } | null,
): boolean {
  if (!home) return false
  if (selection.kind === "region") return home.region === selection.id
  if (home.region || !location) return false
  return roundCoordinate(location.lat) === home.lat && roundCoordinate(location.lon) === home.lon
}
