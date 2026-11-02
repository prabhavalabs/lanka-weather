#!/usr/bin/env node
/**
 * Generates the favicon set, touch icon, PWA icons and web manifest in
 * public/ from the SVG sources in /brand. Run after changing the brand:
 *
 *   pnpm --filter @lanka-weather/web icons
 *
 * The outputs are committed, so a build needs neither this script nor resvg.
 */
import { Resvg } from "@resvg/resvg-js"
import { copyFileSync, mkdirSync, readFileSync, writeFileSync } from "node:fs"
import path from "node:path"

const web = path.resolve(import.meta.dirname, "..")
const brand = path.resolve(web, "../../brand")
const publicDir = path.join(web, "public")

const source = (name) => readFileSync(path.join(brand, name), "utf8")

/** The same artwork on a square, full-bleed tile, for platforms that apply their own mask. */
const fullBleed = (svg) =>
  svg.replace(/<rect width="64" height="64" rx="14"/, '<rect width="64" height="64"')

function png(svg, size) {
  const resvg = new Resvg(svg, {
    fitTo: { mode: "width", value: size },
    font: { loadSystemFonts: false },
    shapeRendering: 2,
  })
  return resvg.render().asPng()
}

/** An ICO file holding PNG images (supported by every current browser). */
function ico(images) {
  const header = Buffer.alloc(6)
  header.writeUInt16LE(0, 0) // reserved
  header.writeUInt16LE(1, 2) // icon
  header.writeUInt16LE(images.length, 4)
  const entries = []
  let offset = 6 + 16 * images.length
  for (const { size, data } of images) {
    const entry = Buffer.alloc(16)
    entry.writeUInt8(size >= 256 ? 0 : size, 0)
    entry.writeUInt8(size >= 256 ? 0 : size, 1)
    entry.writeUInt8(0, 2) // no palette
    entry.writeUInt8(0, 3)
    entry.writeUInt16LE(1, 4) // colour planes
    entry.writeUInt16LE(32, 6) // bits per pixel
    entry.writeUInt32LE(data.length, 8)
    entry.writeUInt32LE(offset, 12)
    offset += data.length
    entries.push(entry)
  }
  return Buffer.concat([header, ...entries, ...images.map((image) => image.data)])
}

const icon = source("app-icon-dark.svg")
const small = source("app-icon-small.svg")
const maskable = source("app-icon-maskable.svg")

mkdirSync(publicDir, { recursive: true })
const write = (name, data) => {
  writeFileSync(path.join(publicDir, name), data)
  console.log(`  public/${name}`)
}

copyFileSync(path.join(brand, "favicon.svg"), path.join(publicDir, "favicon.svg"))
console.log("  public/favicon.svg")
write(
  "favicon.ico",
  ico([
    { size: 16, data: png(small, 16) },
    { size: 32, data: png(icon, 32) },
    { size: 48, data: png(icon, 48) },
  ]),
)
write("apple-touch-icon.png", png(fullBleed(icon), 180))
write("icon-192.png", png(icon, 192))
write("icon-512.png", png(icon, 512))
write("icon-maskable-512.png", png(maskable, 512))

const manifest = {
  name: "Lanka Weather",
  short_name: "Lanka Weather",
  description: "Weather and disaster early-warning map for Sri Lanka.",
  start_url: "/",
  scope: "/",
  display: "standalone",
  orientation: "any",
  background_color: "#050505",
  theme_color: "#050505",
  lang: "en",
  icons: [
    { src: "/icon-192.png", sizes: "192x192", type: "image/png", purpose: "any" },
    { src: "/icon-512.png", sizes: "512x512", type: "image/png", purpose: "any" },
    { src: "/icon-maskable-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
    { src: "/favicon.svg", sizes: "any", type: "image/svg+xml", purpose: "any" },
  ],
}
write("site.webmanifest", JSON.stringify(manifest, null, 2) + "\n")

// resvg's native module can keep the event loop alive.
process.exit(0)
