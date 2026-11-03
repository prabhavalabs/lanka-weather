import { ListIcon } from "lucide-react"
import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from "react"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { useT } from "@/lib/i18n"
import { reducedMotionQuery, useMediaQuery } from "@/lib/useMediaQuery"
import { useLocalStorage } from "@/lib/useLocalStorage"
import { cn } from "@/lib/utils"

/** One layer's key inside the map key. */
export interface MapKeySection {
  id: string
  title: string
  /** Unit or source after the title, e.g. "Estimate" or "NBRO". */
  unit?: string
  /** The legend without its own card. */
  content: ReactNode
}

/** How far below the top of the list a section counts as the one being read. */
const READING_LINE = 12

/**
 * The key to what is on the map: one button over the map, opening the legends
 * of the layers that are on in a single card that scrolls. A stack of open
 * legend cards would cover the map it explains, and a card that scrolls inside
 * a list that scrolls is hard to move through. The row of names on top jumps
 * to a legend and shows which one is in view.
 */
export function MapKey({ sections }: { sections: MapKeySection[] }) {
  const t = useT()
  const [open, setOpen] = useLocalStorage("lanka-weather:map-key-open", false)
  const reducedMotion = useMediaQuery(reducedMotionQuery)
  const id = useId()
  const viewport = useRef<HTMLDivElement>(null)
  const [reading, setReading] = useState<string | null>(null)
  const current = sections.some((section) => section.id === reading) ? reading : sections[0]?.id

  const sectionElement = (sectionId: string) =>
    viewport.current?.querySelector<HTMLElement>(`[data-key-section="${CSS.escape(sectionId)}"]`)

  const track = useCallback(() => {
    const root = viewport.current
    if (!root) return
    const top = root.getBoundingClientRect().top
    const elements = [...root.querySelectorAll<HTMLElement>("[data-key-section]")]
    // At the bottom the last legend may be too short to reach the top: it is the one in view.
    const atEnd = root.scrollTop + root.clientHeight >= root.scrollHeight - 2
    let found = elements[0]?.dataset.keySection ?? null
    if (atEnd && root.scrollTop > 0) found = elements.at(-1)?.dataset.keySection ?? found
    else
      for (const element of elements) {
        if (element.getBoundingClientRect().top - top <= READING_LINE)
          found = element.dataset.keySection ?? found
      }
    setReading(found)
  }, [])

  // Legends come and go as layers are switched; keep the marker on one that exists.
  const sectionIds = sections.map((section) => section.id).join(" ")
  useEffect(() => {
    if (open) track()
  }, [open, sectionIds, track])

  const jump = (sectionId: string) => {
    const root = viewport.current
    const element = sectionElement(sectionId)
    if (!root || !element) return
    const offset = element.getBoundingClientRect().top - root.getBoundingClientRect().top
    root.scrollTo({ top: root.scrollTop + offset, behavior: reducedMotion ? "auto" : "smooth" })
    setReading(sectionId)
  }

  if (sections.length === 0) return null

  return (
    <div className="pointer-events-auto flex flex-col items-start gap-2">
      {open && (
        <div
          id={id}
          className="flex max-h-[min(62dvh,36rem)] w-80 flex-col overflow-hidden rounded-2xl border border-border bg-card/95 text-card-foreground shadow-panel backdrop-blur-md"
        >
          {sections.length > 1 && (
            <nav
              aria-label={t("weather.mapKey.sections")}
              className="shrink-0 border-b border-border p-2"
            >
              <ul className="flex flex-wrap gap-1">
                {sections.map((section) => (
                  <li key={section.id} className="min-w-0 max-w-full">
                    <button
                      type="button"
                      aria-current={section.id === current ? "true" : undefined}
                      onClick={() => jump(section.id)}
                      title={section.title}
                      className={cn(
                        "h-7 max-w-full truncate rounded-full px-2.5 text-xs text-muted-foreground transition-colors outline-none hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring",
                        section.id === current && "bg-accent font-medium text-foreground",
                      )}
                    >
                      {section.title}
                    </button>
                  </li>
                ))}
              </ul>
            </nav>
          )}
          <ScrollArea
            type="auto"
            className="flex min-h-0 flex-1 flex-col"
            viewportClassName="min-h-0 flex-1"
            viewportProps={{ ref: viewport, onScroll: track }}
          >
            <div className="flex flex-col divide-y divide-border">
              {sections.map((section) => (
                <section
                  key={section.id}
                  data-key-section={section.id}
                  aria-labelledby={`${id}-${section.id}`}
                  className="py-3 pr-4 pl-3.5"
                >
                  <header className="mb-2.5 flex items-baseline justify-between gap-2">
                    <h2
                      id={`${id}-${section.id}`}
                      className="min-w-0 text-sm font-medium break-words"
                    >
                      {section.title}
                    </h2>
                    {section.unit && (
                      <span className="numeric shrink-0 text-[0.6875rem] text-muted-foreground">
                        {section.unit}
                      </span>
                    )}
                  </header>
                  {section.content}
                </section>
              ))}
            </div>
          </ScrollArea>
        </div>
      )}
      <Button
        variant="secondary"
        size="sm"
        aria-expanded={open}
        aria-controls={open ? id : undefined}
        className="rounded-full bg-card/95 shadow-panel backdrop-blur-md"
        onClick={() => setOpen((current) => !current)}
      >
        <ListIcon />
        {open ? t("weather.mapKey.hide") : t("weather.mapKey.show")}
      </Button>
    </div>
  )
}
