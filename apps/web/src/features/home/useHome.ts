import { useCallback, useMemo } from "react"
import { useLocalStorage } from "@/lib/useLocalStorage"
import { homeStorageKey, parseHome, type HomePlace } from "./home"

/** The home place saved in this browser, and a way to set or clear it. */
export function useHome() {
  const [stored, setStored] = useLocalStorage<unknown>(homeStorageKey, null)
  const home = useMemo(() => parseHome(stored), [stored])
  const setHome = useCallback((next: HomePlace | null) => setStored(next), [setStored])
  return [home, setHome] as const
}
