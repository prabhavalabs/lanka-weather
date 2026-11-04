import { useSyncExternalStore } from "react"

export type ThemeMode = "light" | "dark" | "system"
export type ResolvedTheme = "light" | "dark"

/** Must match the inline script in index.html, which applies the theme before first paint. */
export const themeStorageKey = "lanka-weather:theme"

/** Browser chrome colour per theme: the page background. Must match index.html. */
export const themeColors: Record<ResolvedTheme, string> = {
  light: "#efefed",
  dark: "#050505",
}

const darkQuery = "(prefers-color-scheme: dark)"
const listeners = new Set<() => void>()

export function isThemeMode(value: unknown): value is ThemeMode {
  return value === "light" || value === "dark" || value === "system"
}

function readStoredMode(): ThemeMode {
  try {
    const stored = localStorage.getItem(themeStorageKey)
    return isThemeMode(stored) ? stored : "system"
  } catch {
    return "system"
  }
}

let mode: ThemeMode = typeof window === "undefined" ? "system" : readStoredMode()

function systemPrefersDark() {
  return typeof window !== "undefined" && window.matchMedia(darkQuery).matches
}

/** "system" resolved against the device setting. */
export function resolveTheme(value: ThemeMode, prefersDark = systemPrefersDark()): ResolvedTheme {
  if (value === "system") return prefersDark ? "dark" : "light"
  return value
}

/** Sets the `dark` class, `color-scheme` and the browser chrome colour. */
function apply() {
  const resolved = resolveTheme(mode)
  const root = document.documentElement
  root.classList.toggle("dark", resolved === "dark")
  root.style.colorScheme = resolved
  for (const meta of document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')) {
    // With an explicit choice both tags show it; with "system" each keeps its media query.
    meta.content =
      mode === "system"
        ? themeColors[meta.media.includes("dark") ? "dark" : "light"]
        : themeColors[resolved]
  }
}

function emit() {
  apply()
  for (const listener of listeners) listener()
}

/** Stores and applies a theme. The `dark` class changes before React re-renders. */
export function setThemeMode(next: ThemeMode) {
  mode = next
  try {
    localStorage.setItem(themeStorageKey, next)
  } catch {
    // Storage may be disabled; the choice then lasts for the session.
  }
  emit()
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

if (typeof window !== "undefined") {
  // Follow the device while the mode is "system", and choices made in other tabs.
  window.matchMedia(darkQuery).addEventListener("change", () => {
    if (mode === "system") emit()
  })
  window.addEventListener("storage", (event) => {
    if (event.key === themeStorageKey) {
      mode = isThemeMode(event.newValue) ? event.newValue : "system"
      emit()
    }
  })
  apply()
}

const getMode = () => mode
const getResolved = () => resolveTheme(mode)

/** The chosen mode, the theme in effect and a setter. */
export function useTheme() {
  const current = useSyncExternalStore(subscribe, getMode, () => "system" as ThemeMode)
  const resolved = useSyncExternalStore(subscribe, getResolved, () => "light" as ResolvedTheme)
  return { mode: current, resolved, setMode: setThemeMode }
}
