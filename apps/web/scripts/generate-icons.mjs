#!/usr/bin/env node
/**
 * Generates the favicon set, touch icons, PWA icons and the notification
 * badge in public/ from the SVG sources in /brand. Run after changing the
 * brand:
 *
 *   pnpm --filter @lanka-weather/web icons
 *
 * The outputs are committed, so a build needs neither this script nor resvg.
 * The web manifest (public/site.webmanifest) is written by hand and checked
 * by src/features/pwa/manifest.test.ts.
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

/** The mark alone in white on a transparent square, from the dark app icon's geometry. */
const badge = (svg) =>
  svg
    .replace(/<title>[^<]*<\/title>/, "")
    .replace(/<rect width="64" height="64" rx="14" fill="#0A0A0A"\/>/, "")
    .replace(/scale\(0\.8\)/, "scale(1.02)")
    .replace(/fill="#F2F2F2"/g, 'fill="#FFFFFF"')
    .replace(/fill="#FF5A1F"/g, 'fill="#FFFFFF"')

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
// iPad (167) and older iPad (152) Home Screen sizes; iPhones use the 180 above.
write("apple-touch-icon-167.png", png(fullBleed(icon), 167))
write("apple-touch-icon-152.png", png(fullBleed(icon), 152))
write("icon-192.png", png(icon, 192))
write("icon-512.png", png(icon, 512))
write("icon-maskable-512.png", png(maskable, 512))
// Notification badge: Android draws only its alpha, in the status bar, so the
// mark is solid white on transparent with the wave cut out.
write("badge-96.png", png(badge(icon), 96))

// resvg's native module can keep the event loop alive.
process.exit(0)
