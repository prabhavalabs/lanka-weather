import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react"
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
  type RefObject,
} from "react"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { edgeMask, overflowEnds } from "./scrollEdges"

/** How far a pointer moves before a press becomes a drag, in pixels. */
const dragThreshold = 4

interface ScrollRowProps {
  children: ReactNode
  className?: string
  /** Classes for the scrolling element itself. */
  viewportClassName?: string
  /** The scrolling element, e.g. to bring a chosen item into view. */
  viewportRef?: RefObject<HTMLDivElement | null>
}

/**
 * A row that scrolls sideways without a scroll bar. Where there is more past
 * an edge, the content fades out there, so a half-shown item says the row goes
 * on, and a small arrow scrolls it. A mouse can drag the row; touch swipes it
 * natively, and keyboard focus brings an item into view.
 */
export function ScrollRow({ children, className, viewportClassName, viewportRef }: ScrollRowProps) {
  const t = useT()
  const own = useRef<HTMLDivElement | null>(null)
  const viewport = viewportRef ?? own
  const [ends, setEnds] = useState({ start: false, end: false })
  const drag = useRef<{ x: number; left: number; moved: boolean } | null>(null)
  const dragged = useRef(false)

  const measure = useCallback((element: HTMLElement) => {
    const next = overflowEnds(element.scrollLeft, element.clientWidth, element.scrollWidth)
    setEnds((current) =>
      current.start === next.start && current.end === next.end ? current : next,
    )
  }, [])

  useEffect(() => {
    const element = viewport.current
    if (!element) return
    // The row's own size and its content's both change what overflows (a language switch).
    // An observer reports each element once when it starts watching, which measures the row.
    const observer = new ResizeObserver(() => measure(element))
    observer.observe(element)
    for (const child of element.children) observer.observe(child)
    return () => observer.disconnect()
  }, [measure, children, viewport])

  const page = (direction: 1 | -1) => {
    const element = viewport.current
    if (!element) return
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches
    element.scrollBy({
      left: direction * element.clientWidth * 0.7,
      behavior: reduced ? "auto" : "smooth",
    })
  }

  const mask = edgeMask(ends.start, ends.end)
  const style: CSSProperties | undefined = mask
    ? { maskImage: mask, WebkitMaskImage: mask }
    : undefined

  const arrow =
    "absolute top-1/2 z-10 flex size-6 -translate-y-1/2 items-center justify-center rounded-full border border-border bg-card text-foreground shadow-sm transition-opacity hover:bg-accent [&_svg]:size-3.5"

  return (
    <div className={cn("relative min-w-0", className)}>
      <div
        ref={viewport}
        style={style}
        onScroll={(event) => measure(event.currentTarget)}
        className={cn(
          "overflow-x-auto overscroll-x-contain [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
          viewportClassName,
        )}
        onPointerDown={(event) => {
          // Touch and pens scroll natively; a mouse drags.
          if (event.pointerType !== "mouse" || event.button !== 0) return
          const element = event.currentTarget
          if (element.scrollWidth <= element.clientWidth) return
          drag.current = { x: event.clientX, left: element.scrollLeft, moved: false }
          dragged.current = false
        }}
        onPointerMove={(event) => {
          const state = drag.current
          const element = event.currentTarget
          if (!state) return
          const dx = event.clientX - state.x
          if (!state.moved && Math.abs(dx) < dragThreshold) return
          if (!state.moved) {
            state.moved = true
            try {
              element.setPointerCapture(event.pointerId)
            } catch {
              // A pointer the browser no longer tracks; the drag follows moves over the row.
            }
          }
          element.scrollLeft = state.left - dx
        }}
        onPointerUp={() => {
          dragged.current = drag.current?.moved ?? false
          drag.current = null
        }}
        onPointerCancel={() => {
          drag.current = null
        }}
        // A drag ends with a click on whatever it started over; that click is not a choice.
        onClickCapture={(event) => {
          if (!dragged.current) return
          dragged.current = false
          event.preventDefault()
          event.stopPropagation()
        }}
      >
        {children}
      </div>
      <button
        type="button"
        tabIndex={-1}
        aria-hidden="true"
        title={t("common.scrollLeft")}
        onClick={() => page(-1)}
        className={cn(arrow, "left-0.5", !ends.start && "pointer-events-none opacity-0")}
      >
        <ChevronLeftIcon />
      </button>
      <button
        type="button"
        tabIndex={-1}
        aria-hidden="true"
        title={t("common.scrollRight")}
        onClick={() => page(1)}
        className={cn(arrow, "right-0.5", !ends.end && "pointer-events-none opacity-0")}
      >
        <ChevronRightIcon />
      </button>
    </div>
  )
}
