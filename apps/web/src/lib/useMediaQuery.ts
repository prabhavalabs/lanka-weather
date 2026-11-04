import { useSyncExternalStore } from "react"

/** Whether a media query matches, kept current as the window changes. */
export function useMediaQuery(query: string, serverValue = false): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const list = window.matchMedia(query)
      list.addEventListener("change", onChange)
      return () => list.removeEventListener("change", onChange)
    },
    () => window.matchMedia(query).matches,
    () => serverValue,
  )
}

/** Wide screens get the sidebar and detail column; phones the top bar and sheets (Tailwind md). */
export const desktopQuery = "(min-width: 48rem)"

export const reducedMotionQuery = "(prefers-reduced-motion: reduce)"
