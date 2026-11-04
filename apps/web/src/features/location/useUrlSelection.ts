import { useCallback, useEffect, useRef, useState } from "react"
import { parseSelection, sameSelection, selectionSearch, type Selection } from "./selection"

/**
 * The detail panel's selection, kept in the URL (`?at=`, `?region=`,
 * `?station=`, `?gauge=`) so a view can be shared and the back button closes
 * what was opened.
 */
export function useUrlSelection() {
  const [selection, setState] = useState<Selection | null>(() =>
    parseSelection(window.location.search),
  )
  const current = useRef(selection)

  useEffect(() => {
    const onPop = () => {
      const next = parseSelection(window.location.search)
      current.current = next
      setState(next)
    }
    window.addEventListener("popstate", onPop)
    return () => window.removeEventListener("popstate", onPop)
  }, [])

  const setSelection = useCallback((next: Selection | null) => {
    const previous = current.current
    if (sameSelection(previous, next)) return
    current.current = next
    setState(next)
    const { pathname, search, hash } = window.location
    const url = `${pathname}${selectionSearch(next, search)}${hash}`
    // Opening from nothing adds a history entry, so "back" closes the panel;
    // switching between places replaces it.
    if (previous === null && next !== null) window.history.pushState(null, "", url)
    else window.history.replaceState(null, "", url)
  }, [])

  return [selection, setSelection] as const
}
