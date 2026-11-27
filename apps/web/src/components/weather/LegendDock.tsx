import { ChevronUpIcon, GripVerticalIcon, LayersIcon, Undo2Icon } from "lucide-react"
import { useCallback, useEffect, useId, useRef, type ReactNode } from "react"
import { Button } from "@/components/ui/button"
import { useT } from "@/lib/i18n"
import { useLocalStorage } from "@/lib/useLocalStorage"
import { cn } from "@/lib/utils"
import { clampOffset, home, type Offset } from "./dockOffset"

interface LegendDockProps {
  /** One legend per item; nothing is drawn when there are none. */
  items: ReactNode[]
}

/**
 * The legends of the layers that are on, together over the map. The bar
 * under them folds them all away, and its grip moves them anywhere on the
 * map; both are kept in this browser. Each legend also folds on its own,
 * from its title.
 */
export function LegendDock({ items }: LegendDockProps) {
  const t = useT()
  const id = useId()
  const root = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useLocalStorage("lanka-weather:legends-open", true)
  const [offset, setOffset] = useLocalStorage<Offset>("lanka-weather:legends-offset", home)
  const drag = useRef<{ pointer: number; startX: number; startY: number; from: Offset } | null>(
    null,
  )

  /** The offset that keeps the dock on screen, were it drawn at `next`. */
  const clamped = useCallback(
    (next: Offset): Offset => {
      const element = root.current
      if (!element) return next
      const rect = element.getBoundingClientRect()
      const dx = next.x - offset.x
      const dy = next.y - offset.y
      return clampOffset(
        next,
        {
          left: rect.left + dx,
          right: rect.right + dx,
          top: rect.top + dy,
          bottom: rect.bottom + dy,
        },
        { width: window.innerWidth, height: window.innerHeight },
      )
    },
    [offset],
  )

  // A smaller window, or legends that grew, can leave the dock part off screen.
  useEffect(() => {
    const settle = () => {
      const next = clamped(offset)
      if (next.x !== offset.x || next.y !== offset.y) setOffset(next)
    }
    settle()
    window.addEventListener("resize", settle)
    return () => window.removeEventListener("resize", settle)
  }, [clamped, offset, setOffset, open, items.length])

  if (items.length === 0) return null
  const moved = offset.x !== 0 || offset.y !== 0

  return (
    <div
      ref={root}
      className="flex flex-col items-start gap-2"
      style={moved ? { transform: `translate(${offset.x}px, ${offset.y}px)` } : undefined}
    >
      {open && (
        <div id={id} className="flex max-w-full flex-wrap items-end gap-3">
          {items}
        </div>
      )}
      <div className="pointer-events-auto flex h-9 items-center rounded-full border border-border bg-card/95 p-0.5 text-card-foreground shadow-panel backdrop-blur-md">
        <button
          type="button"
          title={t("weather.legendDock.move")}
          aria-label={t("weather.legendDock.move")}
          className="flex h-8 w-6 cursor-grab touch-none items-center justify-center rounded-full text-muted-foreground outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring active:cursor-grabbing"
          onPointerDown={(event) => {
            if (event.button !== 0) return
            try {
              event.currentTarget.setPointerCapture(event.pointerId)
            } catch {
              // A pointer the browser no longer tracks; the drag then follows moves over the grip only.
            }
            drag.current = {
              pointer: event.pointerId,
              startX: event.clientX,
              startY: event.clientY,
              from: offset,
            }
          }}
          onPointerMove={(event) => {
            const state = drag.current
            if (!state || state.pointer !== event.pointerId) return
            setOffset(
              clamped({
                x: state.from.x + event.clientX - state.startX,
                y: state.from.y + event.clientY - state.startY,
              }),
            )
          }}
          onPointerUp={() => {
            drag.current = null
          }}
          onPointerCancel={() => {
            drag.current = null
          }}
          onKeyDown={(event) => {
            // Arrow keys move the legends too, for anyone not using a pointer.
            const step = event.shiftKey ? 48 : 16
            const moves: Record<string, Offset> = {
              ArrowLeft: { x: -step, y: 0 },
              ArrowRight: { x: step, y: 0 },
              ArrowUp: { x: 0, y: -step },
              ArrowDown: { x: 0, y: step },
            }
            const move = moves[event.key]
            if (!move) return
            event.preventDefault()
            setOffset(clamped({ x: offset.x + move.x, y: offset.y + move.y }))
          }}
        >
          <GripVerticalIcon aria-hidden="true" className="size-4" />
        </button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-expanded={open}
          aria-controls={open ? id : undefined}
          onClick={() => setOpen((value) => !value)}
          className="h-8 rounded-full px-2.5"
        >
          <LayersIcon />
          {open ? t("weather.legendDock.hide") : t("weather.legendDock.show")}
          <span className="numeric rounded-full bg-muted px-1.5 text-[0.6875rem] leading-4 text-muted-foreground">
            {items.length}
          </span>
          <ChevronUpIcon
            aria-hidden="true"
            className={cn("transition-transform", open && "rotate-180")}
          />
        </Button>
        {moved && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            title={t("weather.legendDock.reset")}
            aria-label={t("weather.legendDock.reset")}
            onClick={() => setOffset(home)}
            className="size-8 rounded-full"
          >
            <Undo2Icon />
          </Button>
        )}
      </div>
    </div>
  )
}
