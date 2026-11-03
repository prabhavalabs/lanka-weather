/**
 * Search results as the palette shows them, recent searches, and matching
 * river gauges by name. Pure, so it is tested without a browser.
 *
 * Names are shown in the reader's language with the English name as the
 * fallback; the second name beside a title is the one the query matched
 * when it differs, else the name in another script. Each entry keeps what it
 * was made from, so a recent search reads in the language of the moment.
 */
import type { SearchResult } from "@/components/weather/types"
import { foldName, getLocale, localName, translateIn, type Locale } from "@/lib/i18n"
import type { RegionLevel } from "../location/types"
import { riverSearchNames } from "../river-risk/present"
import type { RiskLevel, RiverRisk } from "../river-risk/types"
import type { ReservoirFeature, ReservoirStatus } from "../reservoirs/types"
import { formatLevel } from "../rivers/present"
import type { StationFeature } from "../rivers/types"
import type { PlaceKind, PlaceSearchResult, SearchChoice } from "./types"

/** What an entry was made from, to label it again in another language. */
export type EntrySource =
  | { type: "result"; result: PlaceSearchResult }
  | { type: "station"; name: string; basin: string; meta?: string }
  | {
      type: "river"
      name: string
      nameSi: string | null
      /** The name the query matched, when it was another name. */
      matched?: string
      tributary: boolean
      level: RiskLevel
    }
  | {
      type: "reservoir"
      name: string
      district: string | null
      status: ReservoirStatus
      mapped: boolean
      meta?: string
    }

/** A palette entry with what choosing it does. */
export interface SearchEntry extends SearchResult {
  choice: SearchChoice
  from?: EntrySource
}

const placeKinds = new Set<string>([
  "city",
  "town",
  "suburb",
  "village",
  "quarter",
  "neighbourhood",
  "hamlet",
  "locality",
  "isolated_dwelling",
])

/** A kind of place in words: "City", or "Place" for a kind we do not know. */
export function placeKindWord(kind: string, locale: Locale = getLocale()): string {
  return placeKinds.has(kind)
    ? translateIn(locale, `search.kind.${kind as PlaceKind}`)
    : translateIn(locale, "search.kind.place")
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

function usable(name: string | null | undefined): name is string {
  return typeof name === "string" && name.trim() !== ""
}

const same = (a: string, b: string) => foldName(a) === foldName(b)

/** The level words that Sinhala and Tamil names of provinces and districts end with. */
const levelWords: Partial<Record<Locale, Partial<Record<RegionLevel, string>>>> = {
  si: { province: "පළාත", district: "දිස්ත්‍රික්කය" },
  ta: { province: "மாகாணம்", district: "மாவட்டம்" },
}

/**
 * A Sinhala or Tamil province or district name without its level word. The
 * data writes the word in, sometimes cut short ("මහනුවර දිස්ත්‍රික",
 * "அம்பாந்தோட்டை மாவ"), so any start of the word is taken off.
 */
function withoutLevelWord(name: string, level: RegionLevel, locale: Locale): string {
  const word = levelWords[locale]?.[level]
  const words = name.trim().split(/\s+/)
  if (!word || words.length < 2) return name.trim()
  const last = foldName(words.at(-1)!)
  return last.length >= 3 && foldName(word).startsWith(last) ? words.slice(0, -1).join(" ") : name
}

/** A region's name as people say it, in a language: "Kandy District", "மத்திய மாகாணம்". */
function regionName(
  level: RegionLevel,
  names: { en: string; si?: string | null; ta?: string | null },
  locale: Locale,
): string {
  const local = locale === "si" ? names.si : locale === "ta" ? names.ta : null
  const name = usable(local) ? withoutLevelWord(local, level, locale) : names.en
  if (level === "province" || level === "district") {
    return translateIn(locale, `search.regionName.${level}`, { name })
  }
  return name
}

/** The name beside a title: the first candidate that says something the title does not. */
function secondName(title: string, candidates: Array<string | null | undefined>) {
  return candidates.find(
    (name): name is string => usable(name) && !foldName(title).includes(foldName(name)),
  )
}

/** Words of a name for matching, in any script: "Nagalagam Street" is ["nagalagam", "street"]. */
function nameWords(text: string | null | undefined, separators = /[^\p{L}\p{M}\p{N}]+/u) {
  return foldName(text ?? "").split(separators)
}

/** "City · Gangawata Korale, Kandy District" or "District · Central Province". */
function subtitle(result: PlaceSearchResult, locale: Locale): string {
  const isRegion = result.type === "region" && regionKinds.has(result.kind)
  const kind = isRegion
    ? translateIn(locale, `terms.adminLevel.${result.kind as RegionLevel}`)
    : placeKindWord(result.kind, locale)
  // Places list their GN division first; the DS division and district say enough.
  const context = [...result.hierarchy]
    .reverse()
    .filter((region) => region.level !== "gn")
    .slice(0, 2)
    .map((region) => regionName(region.level, { en: region.name }, locale))
    .join(", ")
  return context ? translateIn(locale, "search.subtitle", { kind, context }) : kind
}

/** The words of an entry in a language, from what it was made from. */
function labels(
  source: EntrySource,
  locale: Locale,
): Pick<SearchResult, "title" | "localName" | "subtitle" | "meta"> {
  switch (source.type) {
    case "result": {
      const { result } = source
      const names = { en: result.name, si: result.names.si, ta: result.names.ta }
      const level = result.kind as RegionLevel
      const isRegion = result.type === "region" && regionKinds.has(result.kind)
      const titleIn = (language: Locale) =>
        isRegion
          ? regionName(level, names, language)
          : localName({ name: names.en, nameSi: names.si, nameTa: names.ta }, language)
      const matched = result.matchedName
      const matchedIn = usable(matched)
        ? (["en", "si", "ta"] as const).find((language) => {
            const name = names[language]
            return usable(name) && same(name, matched)
          })
        : undefined
      const title = titleIn(locale)
      return {
        title,
        localName: secondName(title, [
          matchedIn ? titleIn(matchedIn) : matched,
          titleIn(locale === "en" ? "si" : "en"),
        ]),
        subtitle: subtitle(result, locale),
      }
    }
    case "station":
      return {
        title: source.name,
        subtitle: translateIn(locale, "search.station", { basin: source.basin }),
        meta: source.meta,
      }
    case "river": {
      const title = localName({ name: source.name, nameSi: source.nameSi }, locale)
      return {
        title,
        localName: secondName(title, [
          source.matched,
          locale === "en" ? source.nameSi : source.name,
        ]),
        subtitle: translateIn(locale, source.tributary ? "search.tributary" : "search.river"),
        meta: translateIn(locale, `rivers.risk.level.${source.level}`),
      }
    }
    case "reservoir":
      return {
        title: source.name,
        subtitle: [
          source.district
            ? translateIn(locale, "search.regionName.district", { name: source.district })
            : null,
          translateIn(locale, `terms.reservoirStatus.${source.status}`),
          source.mapped ? null : translateIn(locale, "search.reservoirNotMapped"),
        ]
          .filter(Boolean)
          .join(translateIn(locale, "common.separator")),
        meta: source.meta,
      }
  }
}

/** An entry labelled again in a language, when it knows what it was made from. */
export function relabel(entry: SearchEntry, locale: Locale = getLocale()): SearchEntry {
  if (!entry.from) return entry
  try {
    return { ...entry, ...labels(entry.from, locale) }
  } catch {
    // A malformed stored source keeps the words it was saved with.
    return entry
  }
}

/** A search API result as a palette entry. */
export function searchEntry(result: PlaceSearchResult, locale: Locale = getLocale()): SearchEntry {
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
  const from: EntrySource = { type: "result", result }
  return {
    id: result.id,
    kind: isRegion ? "area" : "place",
    ...labels(from, locale),
    choice,
    from,
  }
}

/** River gauges whose name or basin has a word starting with the query. */
export function stationEntries(
  stations: StationFeature[],
  query: string,
  limit = 4,
  locale: Locale = getLocale(),
): SearchEntry[] {
  const folded = foldName(query)
  if (folded.length < 2) return []
  return stations
    .filter((station) =>
      [...nameWords(station.properties.name), ...nameWords(station.properties.basin)].some((word) =>
        word.startsWith(folded),
      ),
    )
    .slice(0, limit)
    .map((station) => {
      const [lon, lat] = station.geometry.coordinates
      const from: EntrySource = {
        type: "station",
        name: station.properties.name,
        basin: station.properties.basin,
        meta:
          station.properties.waterLevel === null
            ? undefined
            : formatLevel(station.properties.waterLevel, station.properties.unit),
      }
      return {
        id: `station:${station.id}`,
        kind: "station" as const,
        ...labels(from, locale),
        choice: { type: "station", id: station.id, name: station.properties.name, lat, lon },
        from,
      }
    })
}

/**
 * Named rivers whose name, Sinhala name or other name has a word starting with
 * the query, largest first.
 */
export function riverEntries(
  rivers: RiverRisk[],
  query: string,
  limit = 5,
  locale: Locale = getLocale(),
): SearchEntry[] {
  const folded = foldName(query)
  if (folded.length < 2) return []
  const matches = (name: string) =>
    nameWords(name, /[\s,()&-]+/).some((word) => word.startsWith(folded))
  return rivers
    .filter((river) => river.named && riverSearchNames(river).some(matches))
    .sort((a, b) => b.uplandKm2 - a.uplandKm2)
    .slice(0, limit)
    .map((river) => {
      const from: EntrySource = {
        type: "river",
        name: river.name,
        nameSi: river.nameSi,
        matched: riverSearchNames(river).find(matches),
        tributary: Boolean(river.parentId),
        level: river.level,
      }
      return {
        id: `river:${river.id}`,
        kind: "river" as const,
        ...labels(from, locale),
        choice: {
          type: "river" as const,
          id: river.id,
          name: river.name,
          lon: (river.bbox[0] + river.bbox[2]) / 2,
          lat: (river.bbox[1] + river.bbox[3]) / 2,
          bbox: river.bbox,
        },
        from,
      }
    })
}

/**
 * Reservoirs with a name, other spelling, river or district that has a word
 * starting with the query, or whose name without spaces starts with it. Those
 * spilling or releasing come first. Includes reservoirs with no position.
 */
export function reservoirEntries(
  reservoirs: ReservoirFeature[],
  query: string,
  limit = 5,
  locale: Locale = getLocale(),
): SearchEntry[] {
  const folded = foldName(query)
  if (folded.length < 2) return []
  const compact = (text: string | null | undefined) =>
    foldName(text ?? "").replace(/[^\p{L}\p{M}\p{N}]/gu, "")
  const start = compact(folded)
  const matches = reservoirs.filter(({ properties }) => {
    const names = [properties.name, properties.osmName, ...(properties.altNames ?? [])]
    return (
      [...names, properties.river, properties.district].some((text) =>
        nameWords(text).some((word) => word.startsWith(folded)),
      ) ||
      (start !== "" && names.some((name) => compact(name).startsWith(start)))
    )
  })
  const attention = (status: string) => (status === "spilling" || status === "releasing" ? 0 : 1)
  return [...matches]
    .sort((a, b) => attention(a.properties.status) - attention(b.properties.status))
    .slice(0, limit)
    .map((reservoir) => {
      const { id, name, district, storagePercent, storageFresh, status, mapped } =
        reservoir.properties
      const [lon, lat] = reservoir.geometry?.coordinates ?? [null, null]
      const from: EntrySource = {
        type: "reservoir",
        name,
        district,
        status,
        mapped,
        meta: storagePercent !== null && storageFresh ? `${storagePercent}%` : undefined,
      }
      return {
        id: `reservoir:${id}`,
        kind: "reservoir" as const,
        ...labels(from, locale),
        choice: { type: "reservoir", id, name, lat, lon },
        from,
      }
    })
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
        ["place", "station", "reservoir", "area", "river"].includes(item.kind) &&
        typeof item.choice === "object" &&
        item.choice !== null &&
        // Only a reservoir may lack a position.
        ((item.choice.type === "reservoir" &&
          item.choice.lat === null &&
          item.choice.lon === null) ||
          (Number.isFinite(item.choice.lat) && Number.isFinite(item.choice.lon))),
    )
    .slice(0, maxRecent)
}
