import { describe, expect, it } from "vitest"
import { parseSelection } from "@/features/location/selection"
import { stripLaunchSource } from "./launch"

// Files are read through Vite, so the test needs no Node types: text as `?raw`,
// images as base64 data URLs (`?inline`).
const texts = import.meta.glob<string>(["/public/site.webmanifest", "/index.html"], {
  query: "?raw",
  import: "default",
  eager: true,
})
const images = import.meta.glob<string>("/public/**/*.{png,webp}", {
  query: "?inline",
  import: "default",
  eager: true,
})

interface Image {
  src: string
  sizes: string
  type: string
  purpose?: string
  form_factor?: string
  label?: string
}

interface Manifest {
  id: string
  name: string
  short_name: string
  description: string
  start_url: string
  scope: string
  display: string
  display_override: string[]
  lang: string
  dir: string
  categories: string[]
  theme_color: string
  background_color: string
  icons: Image[]
  shortcuts: { name: string; short_name: string; url: string; icons?: Image[] }[]
  screenshots: Image[]
}

const manifest = JSON.parse(texts["/public/site.webmanifest"] ?? "{}") as Manifest
const html = texts["/index.html"] ?? ""

const origin = "https://lankaweather.example"
const inScope = (url: string) =>
  new URL(url, origin).href.startsWith(new URL(manifest.scope, origin).href)

/** The bytes of a file in public/, or null when it does not exist. */
function bytes(src: string): Uint8Array | null {
  const url = images[`/public${src}`]
  if (!url) return null
  const binary = atob(url.slice(url.indexOf(",") + 1))
  return Uint8Array.from(binary, (char) => char.charCodeAt(0))
}

const exists = (src: string) => bytes(src) !== null

/** Width and height of a PNG or a WebP (VP8, VP8L or VP8X) file. */
function imageSize(src: string): [number, number] {
  const data = bytes(src)
  if (!data) throw new Error(`Missing: ${src}`)
  const view = new DataView(data.buffer)
  const ascii = (start: number, end: number) => String.fromCharCode(...data.slice(start, end))
  const uint24 = (at: number) => view.getUint16(at, true) + (view.getUint8(at + 2) << 16)
  if (ascii(1, 4) === "PNG") return [view.getUint32(16), view.getUint32(20)]
  if (ascii(0, 4) === "RIFF" && ascii(8, 12) === "WEBP") {
    const chunk = ascii(12, 16)
    if (chunk === "VP8X") return [uint24(24) + 1, uint24(27) + 1]
    if (chunk === "VP8L") {
      const bits = view.getUint32(21, true)
      return [(bits & 0x3fff) + 1, ((bits >> 14) & 0x3fff) + 1]
    }
    if (chunk === "VP8 ")
      return [view.getUint16(26, true) & 0x3fff, view.getUint16(28, true) & 0x3fff]
  }
  throw new Error(`Unknown image format: ${src}`)
}

describe("web manifest", () => {
  it("is linked from index.html", () => {
    expect(html).toContain('<link rel="manifest" href="/site.webmanifest" />')
  })

  it("names the app and how it opens", () => {
    expect(manifest.name).toBe("Lanka Weather")
    expect(manifest.short_name.length).toBeLessThanOrEqual(15)
    expect(manifest.description.length).toBeGreaterThan(20)
    expect(manifest.id).toBe("/")
    expect(manifest.scope).toBe("/")
    expect(manifest.display).toBe("standalone")
    expect(manifest.display_override).toEqual(["standalone", "minimal-ui"])
    expect(manifest.categories).toEqual(["weather", "news", "utilities"])
    expect(manifest.lang).toBe("en")
    expect(manifest.dir).toBe("ltr")
    for (const colour of [manifest.theme_color, manifest.background_color])
      expect(colour).toMatch(/^#[0-9a-f]{6}$/i)
  })

  it("starts inside its scope, with a launch marker the app removes", () => {
    expect(inScope(manifest.start_url)).toBe(true)
    expect(stripLaunchSource(new URL(manifest.start_url, origin).href)).toBe("/")
  })

  it("has the icons install sheets need, and every file exists at its size", () => {
    const sizes = manifest.icons.filter((icon) => icon.type === "image/png")
    expect(sizes.some((icon) => icon.sizes === "192x192" && icon.purpose === "any")).toBe(true)
    expect(sizes.some((icon) => icon.sizes === "512x512" && icon.purpose === "any")).toBe(true)
    expect(sizes.some((icon) => icon.sizes === "512x512" && icon.purpose === "maskable")).toBe(true)
    for (const icon of manifest.icons) {
      if (icon.type !== "image/png") continue
      expect(exists(icon.src), icon.src).toBe(true)
      const [width, height] = icon.sizes.split("x").map(Number)
      expect(imageSize(icon.src), icon.src).toEqual([width, height])
    }
  })

  it("has shortcuts to views the app opens from the URL", () => {
    expect(manifest.shortcuts.length).toBeGreaterThanOrEqual(2)
    for (const shortcut of manifest.shortcuts) {
      expect(inScope(shortcut.url), shortcut.url).toBe(true)
      const cleaned = stripLaunchSource(new URL(shortcut.url, origin).href)
      expect(cleaned, shortcut.url).not.toBeNull()
      const search = new URL(cleaned ?? "/", origin).search
      expect(parseSelection(search), shortcut.url).not.toBeNull()
      expect(shortcut.short_name.length).toBeLessThanOrEqual(12)
    }
  })

  it("has a narrow and a wide screenshot, at the sizes it states and kept small", () => {
    const factors = manifest.screenshots.map((shot) => shot.form_factor).sort()
    expect(factors).toEqual(["narrow", "wide"])
    for (const shot of manifest.screenshots) {
      expect(exists(shot.src), shot.src).toBe(true)
      const [width, height] = shot.sizes.split("x").map(Number)
      expect(imageSize(shot.src), shot.src).toEqual([width, height])
      expect(shot.label?.length ?? 0).toBeGreaterThan(10)
      expect(bytes(shot.src)?.length ?? 0, shot.src).toBeLessThan(400 * 1024)
      if (shot.form_factor === "narrow") expect(height).toBeGreaterThan(width ?? 0)
      else expect(width).toBeGreaterThan(height ?? 0)
    }
  })
})

describe("index.html", () => {
  it("asks for the installed-app behaviour on Apple devices", () => {
    expect(html).toContain('name="apple-mobile-web-app-capable" content="yes"')
    expect(html).toContain('name="mobile-web-app-capable" content="yes"')
    expect(html).toContain('name="apple-mobile-web-app-status-bar-style"')
    for (const size of ["180x180", "167x167", "152x152"]) {
      const match = html.match(new RegExp(`rel="apple-touch-icon" sizes="${size}" href="([^"]+)"`))
      expect(match, size).not.toBeNull()
      expect(imageSize(match?.[1] ?? "").join("x")).toBe(size)
    }
  })

  it("names the notification badge the service worker uses", () => {
    expect(imageSize("/badge-96.png")).toEqual([96, 96])
  })
})
