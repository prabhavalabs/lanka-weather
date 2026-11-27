/** The width of the fade at an edge with more to see, in pixels. */
const fade = 28

/** Which ends of a scroller have more content past them. */
export function overflowEnds(scrollLeft: number, clientWidth: number, scrollWidth: number) {
  return {
    start: scrollLeft > 1,
    end: scrollLeft + clientWidth < scrollWidth - 1,
  }
}

/** A mask that fades the content out at the ends with more past them. */
export function edgeMask(start: boolean, end: boolean): string | undefined {
  if (!start && !end) return undefined
  const from = start ? `transparent, black ${fade}px` : "black, black"
  const to = end ? `black calc(100% - ${fade}px), transparent` : "black, black"
  return `linear-gradient(to right, ${from}, ${to})`
}
