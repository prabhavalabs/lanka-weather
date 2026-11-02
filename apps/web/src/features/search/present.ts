/**
 * Search results as the palette shows them, recent searches, and matching
 * river gauges by name. Pure, so it is tested without a browser.
 */
import type { SearchResult } from "@/components/weather/types"
import { levelWords, regionLabel } from "../location/present"
import type { RegionLevel } from "../location/types"
import { riskLabels, riverSearchNames } from "../river-risk/present"
import type { RiverRisk } from "../river-risk/types"
import type { StationFeature } from "../rivers/types"
import type { PlaceKind, PlaceSearchResult, SearchChoice } from "./types"

/** A palette entry with what choosing it does. */
export interface SearchEntry extends SearchResult {
  choice: SearchChoice
}

export const placeKindWords: Record<PlaceKind, string> = {
  city: "City",
  town: "Town",
  suburb: "Suburb",
  village: "Village",
  quarter: "Quarter",
  neighbourhood: "Neighbourhood",
  hamlet: "Hamlet",
  locality: "Locality",
  isolated_dwelling: "Dwelling",
}

/** How close to zoom for a place: towns closer than cities, hamlets closer still. */
export const placeZoom: Record<PlaceKind, number> = {
  city: 11.5,
  town: 12,
  suburb: 13,
  village: 13,
  quarter: 14,
  neighbourhood: 14,
  hamlet: 14,
  locality: 14,
  isolated_dwelling: 14.5,
}

const regionKinds = new Set<string>(["province", "district", "ds", "gn"])

/** "City · Gangawata Korale, Kandy District" or "District · Central Province". */
function subtitle(result: PlaceSearchResult): string {
  const isRegion = result.type === "region" && regionKinds.has(result.kind)
  const kind = isRegion
    ? levelWords[result.kind as RegionLevel]
    : (placeKindWords[result.kind as PlaceKind] ?? "Place")
  // Places list their GN division first; the DS division and district say enough.
  const context = [...result.hierarchy]
    .reverse()
    .filter((region) => region.level !== "gn")
    .slice(0, 2)
    .map(regionLabel)
    .join(", ")
  return context ? `${kind} · ${context}` : kind
}

/** A search API result as a palette entry. */
export function searchEntry(result: PlaceSearchResult): SearchEntry {
  const isRegion = result.type === "region" && regionKinds.has(result.kind)
  const choice: SearchChoice = isRegion
    ? {
        type: "region",
        id: result.id,
        name: result.name,
        level: result.kind as RegionLevel,
        lat: result.lat,
        lon: result.lon,
        bbox: result.bbox,
      }
    : {
        type: "place",
        id: result.id,
        name: result.name,
        kind: result.kind as PlaceKind,
        lat: result.lat,
        lon: result.lon,
      }
  const local = result.names.si ?? undefined
  return {
    id: result.id,
    kind: isRegion ? "area" : "place",
    title: isRegion
      ? regionLabel({ level: result.kind as RegionLevel, name: result.name })
      : result.name,
    localName: local && local !== result.name ? local : undefined,
    subtitle: subtitle(result),
    choice,
  }
}

/** River gauges whose name or basin has a word starting with the query. */
export function stationEntries(
  stations: StationFeature[],
  query: string,
  limit = 4,
): SearchEntry[] {
  const folded = query.trim().toLowerCase()
  if (folded.length < 2) return []
  const words = (text: string) => text.toLowerCase().split(/[^a-z0-9]+/)
  return stations
    .filter((station) =>
      [...words(station.properties.name), ...words(station.properties.basin)].some((word) =>
        word.startsWith(folded),
      ),
    )
    .slice(0, limit)
    .map((station) => {
      const [lon, lat] = station.geometry.coordinates
      return {
        id: `station:${station.id}`,
        kind: "station" as const,
        title: station.properties.name,
        subtitle: `${station.properties.basin} · river gauge`,
        meta:
          station.properties.waterLevel === null
            ? undefined
            : `${station.properties.waterLevel.toFixed(station.properties.unit === "ft" ? 1 : 2)} ${station.properties.unit}`,
        choice: { type: "station", id: station.id, name: station.properties.name, lat, lon },
      }
    })
}

/**
 * Named rivers whose name, Sinhala name or other name has a word starting with
 * the query, largest first.
 */
export function riverEntries(rivers: RiverRisk[], query: string, limit = 5): SearchEntry[] {
  const folded = query.trim().toLowerCase()
  if (folded.length < 2) return []
  const matches = (name: string) =>
    name
      .toLowerCase()
      .split(/[\s,()&-]+/)
      .some((word) => word.startsWith(folded))
  return rivers
    .filter((river) => river.named && riverSearchNames(river).some(matches))
    .sort((a, b) => b.uplandKm2 - a.uplandKm2)
    .slice(0, limit)
    .map((river) => ({
      id: `river:${river.id}`,
      kind: "river" as const,
      title: river.name,
      localName: river.nameSi ?? undefined,
      subtitle: river.parentId ? "Tributary" : "River",
      meta: riskLabels[river.level],
      choice: {
        type: "river" as const,
        id: river.id,
        name: river.name,
        lon: (river.bbox[0] + river.bbox[2]) / 2,
        lat: (river.bbox[1] + river.bbox[3]) / 2,
        bbox: river.bbox,
      },
    }))
}

export const maxRecent = 6

/** Recent searches with `entry` first, without duplicates, at most `maxRecent`. */
export function addRecent(recent: SearchEntry[], entry: SearchEntry): SearchEntry[] {
  return [entry, ...recent.filter((item) => item.id !== entry.id)].slice(0, maxRecent)
}

/** Recent searches read back from storage; anything malformed is dropped. */
export function validRecent(value: unknown): SearchEntry[] {
  if (!Array.isArray(value)) return []
  return value
    .filter(
      (item): item is SearchEntry =>
        typeof item === "object" &&
        item !== null &&
        typeof item.id === "string" &&
        typeof item.title === "string" &&
        ["place", "station", "area", "river"].includes(item.kind) &&
        typeof item.choice === "object" &&
        item.choice !== null &&
        Number.isFinite(item.choice.lat) &&
        Number.isFinite(item.choice.lon),
    )
    .slice(0, maxRecent)
}
