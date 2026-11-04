import * as React from "react"
import { ContainerIcon, MapPinIcon, RadioTowerIcon, ShapesIcon, WavesIcon } from "lucide-react"

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
import { useT, scriptLang } from "@/lib/i18n"

import type { SearchResult, SearchResultKind } from "./types"

// The trigger lives apart so the page can show it without loading the palette.
export { SearchTrigger } from "./SearchTrigger"

/** Result groups in order; headings are `weather.search.groups.*` and `weather.search.recent.*`. */
const groups: { kind: SearchResultKind; Icon: typeof MapPinIcon }[] = [
  { kind: "river", Icon: WavesIcon },
  { kind: "place", Icon: MapPinIcon },
  { kind: "station", Icon: RadioTowerIcon },
  { kind: "reservoir", Icon: ContainerIcon },
  { kind: "area", Icon: ShapesIcon },
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
  const t = useT()
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
      title={t("weather.search.title")}
      description={t("weather.search.description")}
      className="sm:max-w-lg"
    >
      {/* Filtering is done by the parent, so cmdk's own filter is off. */}
      <Command shouldFilter={false}>
        <CommandInput
          value={query}
          onValueChange={onQueryChange}
          placeholder={t("weather.search.placeholder")}
        />
        <CommandList className="max-h-[min(60dvh,24rem)] px-1.5 pb-1.5">
          {!loading && (
            <CommandEmpty>
              {query ? t("weather.search.nothingFound") : t("weather.search.typeToSearch")}
            </CommandEmpty>
          )}
          {loading && (
            <p className="py-6 text-center text-sm text-muted-foreground" role="status">
              {t("weather.search.searching")}
            </p>
          )}
          {!loading &&
            groups.map(({ kind, Icon }) => {
              const items = shown.filter((result) => result.kind === kind)
              if (items.length === 0) return null
              return (
                <CommandGroup
                  key={kind}
                  heading={
                    query.trim() === ""
                      ? t(`weather.search.recent.${kind}`)
                      : t(`weather.search.groups.${kind}`)
                  }
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
                            <span
                              className="ml-2 text-muted-foreground"
                              lang={result.localNameLang ?? scriptLang(result.localName)}
                            >
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
