import { useCallback, useState } from "react"

/** State mirrored to localStorage, for small UI preferences only. */
export function useLocalStorage<T>(key: string, initial: T) {
  const [value, setValue] = useState<T>(() => {
    try {
      const stored = localStorage.getItem(key)
      return stored === null ? initial : (JSON.parse(stored) as T)
    } catch {
      return initial
    }
  })

  const update = useCallback(
    (next: T | ((previous: T) => T)) => {
      setValue((previous) => {
        const resolved = next instanceof Function ? next(previous) : next
        try {
          localStorage.setItem(key, JSON.stringify(resolved))
        } catch {
          // Storage may be full or disabled; the preference then lasts for the session only.
        }
        return resolved
      })
    },
    [key],
  )

  return [value, update] as const
}
