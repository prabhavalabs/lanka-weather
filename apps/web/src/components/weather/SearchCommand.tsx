import * as React from "react"
import { MapPinIcon, RadioTowerIcon, ShapesIcon } from "lucide-react"

import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "@/components/ui/command"

import type { SearchResult, SearchResultKind } from "./types"

// The trigger lives apart so the page can show it without loading the palette.
export { SearchTrigger } from "./SearchTrigger"

const groups: { kind: SearchResultKind; heading: string; Icon: typeof MapPinIcon }[] = [
  { kind: "place", heading: "Places", Icon: MapPinIcon },
  { kind: "station", heading: "River gauges", Icon: RadioTowerIcon },
  { kind: "area", heading: "Administrative areas", Icon: ShapesIcon },
]

interface SearchCommandProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Controlled query; the parent fetches results for it. */
  query: string
  onQueryChange: (query: string) => void
  /** Results for the query, in any order; they are grouped by kind. */
  results: SearchResult[]
  /** Shown before anything is typed. */
  recent?: SearchResult[]
  onSelect: (result: SearchResult) => void
  /** True while results load. */
  loading?: boolean
  /** Opens the palette with ⌘K or Ctrl+K. */
  shortcut?: boolean
  /** Under the results, e.g. the data's attribution. */
  footer?: React.ReactNode
}

/**
 * The search palette: places, river gauges and administrative areas.
 * Results come from the parent, so filtering is the parent's (server's) job.
 */
export function SearchCommand({
  open,
  onOpenChange,
  query,
  onQueryChange,
  results,
  recent = [],
  onSelect,
  loading = false,
  shortcut = true,
  footer,
}: SearchCommandProps) {
  React.useEffect(() => {
    if (!shortcut) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() === "k" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault()
        onOpenChange(!open)
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [shortcut, open, onOpenChange])

  const shown = query.trim() === "" ? recent : results

  return (
    <CommandDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Search"
      description="Find a place, a river gauge or an administrative area"
      className="sm:max-w-lg"
    >
      {/* Filtering is done by the parent, so cmdk's own filter is off. */}
      <Command shouldFilter={false}>
        <CommandInput
          value={query}
          onValueChange={onQueryChange}
          placeholder="Search places, rivers, areas…"
        />
        <CommandList className="max-h-[min(60dvh,24rem)] px-1.5 pb-1.5">
          {!loading && <CommandEmpty>{query ? "Nothing found." : "Type to search."}</CommandEmpty>}
          {loading && (
            <p className="py-6 text-center text-sm text-muted-foreground" role="status">
              Searching…
            </p>
          )}
          {!loading &&
            groups.map(({ kind, heading, Icon }) => {
              const items = shown.filter((result) => result.kind === kind)
              if (items.length === 0) return null
              return (
                <CommandGroup
                  key={kind}
                  heading={query.trim() === "" ? `Recent ${heading.toLowerCase()}` : heading}
                >
                  {items.map((result) => (
                    <CommandItem
                      key={result.id}
                      value={result.id}
                      onSelect={() => onSelect(result)}
                    >
                      <span className="flex size-8 items-center justify-center rounded-lg border border-border">
                        <Icon />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block truncate">
                          {result.title}
                          {result.localName && (
                            <span className="ml-2 text-muted-foreground" lang="si">
                              {result.localName}
                            </span>
                          )}
                        </span>
                        {result.subtitle && (
                          <span className="block truncate text-xs text-muted-foreground">
                            {result.subtitle}
                          </span>
                        )}
                      </span>
                      {result.meta && (
                        <CommandShortcut className="numeric tracking-normal">
                          {result.meta}
                        </CommandShortcut>
                      )}
                    </CommandItem>
                  ))}
                </CommandGroup>
              )
            })}
        </CommandList>
        {footer && (
          <div className="border-t border-border px-4 py-2.5 text-[0.6875rem] text-muted-foreground">
            {footer}
          </div>
        )}
      </Command>
    </CommandDialog>
  )
}
