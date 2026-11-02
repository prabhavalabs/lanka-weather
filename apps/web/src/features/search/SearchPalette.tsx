/**
 * The search palette with its data: places and regions from
 * `/v1/places/search`, river gauges by name, and recent searches. Loaded on
 * demand (React.lazy) the first time search is opened.
 */
import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { SearchCommand } from "@/components/weather/SearchCommand"
import { apiGet } from "@/lib/api"
import { useDebouncedValue } from "@/lib/useDebouncedValue"
import { useLocalStorage } from "@/lib/useLocalStorage"
import type { RiverRisk } from "../river-risk/types"
import type { ReservoirsCollection } from "../reservoirs/types"
import type { RiverLevelsCollection } from "../rivers/types"
import {
  addRecent,
  reservoirEntries,
  riverEntries,
  searchEntry,
  stationEntries,
  validRecent,
  type SearchEntry,
} from "./present"
import type { PlaceSearchResponse, SearchChoice } from "./types"

export interface SearchPaletteProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onChoose: (choice: SearchChoice) => void
  /** River gauges, when loaded, to search by name. */
  rivers?: RiverLevelsCollection
  /** Rivers themselves, when loaded, to search by name. */
  riverNetwork?: RiverRisk[]
  /** Reservoirs, when loaded, to search by name. */
  reservoirs?: ReservoirsCollection
}

const recentKey = "lanka-weather:recent-searches"

export default function SearchPalette({
  open,
  onOpenChange,
  onChoose,
  rivers,
  riverNetwork,
  reservoirs,
}: SearchPaletteProps) {
  const [query, setQuery] = useState("")
  const [storedRecent, setRecent] = useLocalStorage<unknown>(recentKey, [])
  const recent = useMemo(() => validRecent(storedRecent), [storedRecent])
  const term = useDebouncedValue(query.trim(), 160)

  const search = useQuery({
    queryKey: ["places", "search", term.toLowerCase()],
    queryFn: ({ signal }) =>
      apiGet<PlaceSearchResponse>(
        `/v1/places/search?q=${encodeURIComponent(term)}&limit=12`,
        signal,
      ),
    enabled: term.length > 0,
    // The API caches results for an hour.
    staleTime: 60 * 60 * 1000,
    placeholderData: keepPreviousData,
    meta: { label: "search results" },
  })

  const entries = useMemo<SearchEntry[]>(() => {
    if (!term) return []
    const places = (search.data?.results ?? []).map(searchEntry)
    const stations = rivers ? stationEntries(rivers.features, term) : []
    const named = riverNetwork ? riverEntries(riverNetwork, term) : []
    const dams = reservoirs ? reservoirEntries(reservoirs.features, term) : []
    return [...named, ...places, ...stations, ...dams]
  }, [search.data, rivers, riverNetwork, reservoirs, term])

  const byId = useMemo(() => {
    const map = new Map<string, SearchEntry>()
    for (const entry of [...entries, ...recent]) map.set(entry.id, entry)
    return map
  }, [entries, recent])

  const typing = query.trim() !== term
  const loading = query.trim() !== "" && (typing || search.isPending) && entries.length === 0

  return (
    <SearchCommand
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next)
        if (!next) setQuery("")
      }}
      query={query}
      onQueryChange={setQuery}
      results={entries}
      recent={recent}
      loading={loading}
      shortcut={false}
      onSelect={(result) => {
        const entry = byId.get(result.id)
        if (!entry) return
        setRecent((current: unknown) => addRecent(validRecent(current), entry))
        onOpenChange(false)
        setQuery("")
        onChoose(entry.choice)
      }}
      footer={
        <>
          Places © OpenStreetMap contributors (ODbL). Regions: Survey Department and Department of
          Census and Statistics. Reservoirs: Sri Lanka Water Levels (waterlevels.lk).
        </>
      }
    />
  )
}
