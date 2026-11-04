/**
 * WCAG 2 contrast of the theme tokens in theme.css, light and dark. Text
 * pairs need 4.5:1 (AA); focus rings, controls and map symbols need 3:1.
 */
import { describe, expect, it } from "vitest"
import source from "./theme.css?raw"

const css = source.replace(/\/\*[\s\S]*?\*\//g, "")

function block(selector: string) {
  const match = css.match(new RegExp(`${selector.replace(".", "\\.")}\\s*\\{([^}]*)\\}`))
  if (!match) throw new Error(`No ${selector} block in theme.css`)
  const tokens = new Map<string, string>()
  for (const [, name, value] of match[1]!.matchAll(/--([\w-]+):\s*(#[0-9a-fA-F]{6})\s*;/g)) {
    tokens.set(name!, value!.toLowerCase())
  }
  return tokens
}

const light = block(":root")
const themes = { light, dark: new Map([...light, ...block(".dark")]) }

function luminance(hex: string) {
  const channel = (offset: number) => {
    const value = parseInt(hex.slice(offset, offset + 2), 16) / 255
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * channel(1) + 0.7152 * channel(3) + 0.0722 * channel(5)
}

function contrast(a: string, b: string) {
  const [high, low] = [luminance(a), luminance(b)].sort((x, y) => y - x) as [number, number]
  return (high + 0.05) / (low + 0.05)
}

const text = 4.5
const nonText = 3
const statuses = ["normal", "alert", "minor-flood", "major-flood", "unknown"]
const reservoirStatuses = ["spilling", "releasing", "holding", "nodata"]

/** [foreground token, background token, minimum ratio] */
const pairs: [string, string, number][] = [
  ...["background", "card", "popover", "muted", "secondary", "accent", "inset"].map(
    (surface) => ["foreground", surface, text] as [string, string, number],
  ),
  ...["background", "card", "popover", "muted", "accent", "inset"].map(
    (surface) => ["muted-foreground", surface, text] as [string, string, number],
  ),
  ["card-foreground", "card"],
  ["popover-foreground", "popover"],
  ["primary-foreground", "primary"],
  ["secondary-foreground", "secondary"],
  ["accent-foreground", "accent"],
  ["sidebar-foreground", "sidebar"],
  ["sidebar-primary-foreground", "sidebar-primary"],
  ["sidebar-accent-foreground", "sidebar-accent"],
  ["brand-foreground", "brand"],
  ["brand-soft-foreground", "brand-soft"],
  ["brand-text", "card"],
  ["brand-text", "muted"],
  ["destructive", "card"],
  ["tooltip-foreground", "tooltip"],
  ...statuses.map(
    (status) =>
      [`status-${status}-foreground`, `status-${status}`, text] as [string, string, number],
  ),
  ...reservoirStatuses.map(
    (status) =>
      [`reservoir-${status}-foreground`, `reservoir-${status}`, text] as [string, string, number],
  ),
  // Map labels on land and water.
  ["map-label", "map-land"],
  ["map-label-strong", "map-land"],
  ["map-water-label", "map-water"],
].map(([fg, bg, min]) => [fg, bg, (min as number | undefined) ?? text] as [string, string, number])

const nonTextPairs: [string, string][] = [
  ["ring", "card"],
  ["ring", "background"],
  ["brand", "card"],
  ["brand", "muted"],
  ["brand", "brand-soft"],
  ["muted-foreground", "secondary"],
  // Map symbols stand off their outline; the selected outline stands off the land.
  ...statuses.map((status) => [`status-${status}`, "status-stroke"] as [string, string]),
  ["status-stroke-selected", "map-land"],
  ["river-unassessed", "map-land"],
  ["basin-outline", "map-land"],
  ["past-flood", "map-land"],
  ["status-normal", "map-land"],
  ...reservoirStatuses.map(
    (status) => [`reservoir-${status}`, "status-stroke"] as [string, string],
  ),
]

for (const [name, tokens] of Object.entries(themes)) {
  describe(`${name} theme contrast`, () => {
    const get = (token: string) => {
      const value = tokens.get(token)
      if (!value) throw new Error(`--${token} is not a hex colour in the ${name} theme`)
      return value
    }

    it.each(pairs)("%s on %s ≥ %s", (fg, bg, min) => {
      expect(contrast(get(fg), get(bg))).toBeGreaterThanOrEqual(min)
    })

    it.each(nonTextPairs)("%s against %s ≥ 3", (fg, bg) => {
      expect(contrast(get(fg), get(bg))).toBeGreaterThanOrEqual(nonText)
    })
  })
}
