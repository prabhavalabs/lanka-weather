import type { Map as MapLibreMap } from "maplibre-gl"

import { sampleInto, type WindField } from "./field"
import { windBand, windBands } from "./scale"

/*
 * The wind layer's renderer (ADR 0007), ported from TerraGuard: particles
 * carried by the wind, leaving fading trails coloured by speed, drawn on a
 * 2D canvas the size of the map.
 *
 * Particles live in longitude and latitude and are placed on screen with
 * map.project every frame. Trails fade by keeping a share of the canvas each
 * frame, which only holds while the map is still: when it moves, the canvas
 * is cleared and the particles wait, and they are sown again where it stops.
 *
 * The canvas is not added to the page: `WindParticlesLayer` (layer.ts)
 * draws it inside the map, between the boundaries and the labels, so
 * gauges and labels stay on top. `onFrame` tells it a new frame is ready.
 *
 * With prefers-reduced-motion nothing moves: short streamlines with an
 * arrowhead are drawn once, and again after each move.
 */

const DEGREES = Math.PI / 180
/** How fast particles cross the screen, in CSS pixels per frame for each m/s. */
const PIXELS_PER_MS = 0.12
/** The share of the trails kept each frame: they fade over about half a second. */
const TRAIL_KEEP = 0.92
/** A particle lives this many frames, and up to LIFE_SPREAD more, before it is reborn elsewhere. */
const LIFE = 50
const LIFE_SPREAD = 70
/** One particle per this many square CSS pixels, at most MAX_PARTICLES. */
const AREA_PER_PARTICLE = 650
const MAX_PARTICLES = 6000
/** A step longer than this (a projection jump) is not drawn. */
const MAX_STEP = 40
const LINE_WIDTH = 1.25
/** Reduced motion: one streamline per this many CSS pixels, of STREAM_STEPS steps of STREAM_STEP pixels. */
const STREAM_SPACING = 34
const STREAM_STEPS = 9
const STREAM_STEP = 2.6
const ARROW = 4

/** A little short of 90°: the camera sees slightly less than a hemisphere, and particles at the rim are unreadable. */
const HORIZON_COSINE = 0.1

export interface WindRendererOptions {
  /** Called after each frame is drawn (or the canvas cleared). */
  onFrame?: () => void
}

export class WindRenderer {
  /** The drawing; composited by the map layer. */
  readonly canvas: HTMLCanvasElement
  private readonly map: MapLibreMap
  private readonly context: CanvasRenderingContext2D
  private readonly motion: MediaQueryList
  private readonly onFrame: () => void
  private field: WindField | null = null
  private colors: string[]
  private reduced: boolean
  private moving = false
  private frame = 0
  private width = 0
  private height = 0
  private count = 0
  private lon = new Float64Array(0)
  private lat = new Float64Array(0)
  private x = new Float32Array(0)
  private y = new Float32Array(0)
  private age = new Int16Array(0)
  private shown = new Uint8Array(0)
  private readonly wind = new Float32Array(2)
  /** Each band's segments this frame, as x0, y0, x1, y1, …: one stroke per colour. */
  private readonly segments: number[][] = windBands.map(() => [])

  constructor(map: MapLibreMap, colors: string[], options: WindRendererOptions = {}) {
    this.map = map
    this.colors = colors
    this.onFrame = options.onFrame ?? (() => {})
    this.canvas = document.createElement("canvas")
    const context = this.canvas.getContext("2d")
    if (!context) throw new Error("No 2D canvas")
    this.context = context
    this.motion = window.matchMedia("(prefers-reduced-motion: reduce)")
    this.reduced = this.motion.matches
    this.motion.addEventListener("change", this.onMotion)
    map.on("movestart", this.onMoveStart)
    map.on("moveend", this.onMoveEnd)
    map.on("resize", this.onResize)
    this.resize()
  }

  /** Whether particles are still (prefers-reduced-motion). */
  get still() {
    return this.reduced
  }

  /** Shows a field: a new forecast time or model run sows the particles afresh. */
  setField(field: WindField | null) {
    if (field === this.field) return
    this.field = field
    this.restart()
  }

  /** The bands' colours, resolved from the theme (an appearance switch changes them). */
  setColors(colors: string[]) {
    this.colors = colors
    if (this.reduced) this.restart()
  }

  /** Stops drawing: the layer was switched off or the map is going. */
  destroy() {
    this.stop()
    this.motion.removeEventListener("change", this.onMotion)
    this.map.off("movestart", this.onMoveStart)
    this.map.off("moveend", this.onMoveEnd)
    this.map.off("resize", this.onResize)
  }

  private readonly onMotion = () => {
    this.reduced = this.motion.matches
    this.restart()
  }

  private readonly onMoveStart = () => {
    this.moving = true
    this.stop()
    this.clear()
    this.onFrame()
  }

  private readonly onMoveEnd = () => {
    this.moving = false
    this.restart()
  }

  private readonly onResize = () => {
    this.resize()
    this.restart()
  }

  private resize() {
    const ratio = Math.min(window.devicePixelRatio || 1, 2)
    const element = this.map.getCanvas()
    this.width = element.clientWidth
    this.height = element.clientHeight
    this.canvas.width = Math.max(1, Math.round(this.width * ratio))
    this.canvas.height = Math.max(1, Math.round(this.height * ratio))
    this.context.setTransform(ratio, 0, 0, ratio, 0, 0)
  }

  private clear() {
    this.context.clearRect(0, 0, this.width, this.height)
  }

  private stop() {
    if (this.frame) cancelAnimationFrame(this.frame)
    this.frame = 0
  }

  private restart() {
    this.stop()
    this.clear()
    if (!this.field || this.moving || !this.width || !this.height) {
      this.onFrame()
      return
    }
    if (this.reduced) {
      this.drawStreamlines()
      this.onFrame()
      return
    }
    this.sow()
    this.frame = requestAnimationFrame(this.tick)
  }

  /** Degrees of latitude a particle moves per frame for each m/s, at this zoom. */
  private degreesPerStep(pixels: number) {
    return (pixels * 360) / (512 * 2 ** this.map.getZoom())
  }

  /**
   * On the globe, whether a location faces away from the camera: more than
   * HORIZON from the view's centre along the Earth. The flat map hides nothing.
   */
  private occluded(location: { lng: number; lat: number }) {
    if (this.map.getProjection()?.type !== "globe") return false
    const centre = this.map.getCenter()
    const cosine =
      Math.sin(location.lat * DEGREES) * Math.sin(centre.lat * DEGREES) +
      Math.cos(location.lat * DEGREES) *
        Math.cos(centre.lat * DEGREES) *
        Math.cos((location.lng - centre.lng) * DEGREES)
    return cosine < HORIZON_COSINE
  }

  /** A random point of the map that is on the Earth, or null. */
  private randomLocation(): { lng: number; lat: number } | null {
    for (let attempt = 0; attempt < 6; attempt += 1) {
      const x = Math.random() * this.width
      const y = Math.random() * this.height
      const location = this.map.unproject([x, y])
      if (this.occluded(location)) continue
      // Off the globe, MapLibre answers with the horizon: projecting back shows it.
      const point = this.map.project(location)
      if (Math.abs(point.x - x) > 2 || Math.abs(point.y - y) > 2) continue
      return location
    }
    return null
  }

  /** As many particles as the view holds, with staggered lives so they are not reborn together. */
  private sow() {
    const zoom = this.map.getZoom()
    // A little denser zoomed out, where the pattern is the point; the field is smooth close up.
    const density = Math.max(0.7, Math.min(1.3, 1.3 - 0.07 * zoom))
    this.count = Math.round(
      Math.min(
        MAX_PARTICLES,
        Math.max(200, ((this.width * this.height) / AREA_PER_PARTICLE) * density),
      ),
    )
    this.lon = new Float64Array(this.count)
    this.lat = new Float64Array(this.count)
    this.x = new Float32Array(this.count)
    this.y = new Float32Array(this.count)
    this.age = new Int16Array(this.count)
    this.shown = new Uint8Array(this.count)
    for (let index = 0; index < this.count; index += 1) {
      this.spawn(index)
      this.age[index] = 1 + Math.floor(Math.random() * (LIFE + LIFE_SPREAD))
    }
  }

  private spawn(index: number) {
    const location = this.randomLocation()
    this.shown[index] = 0
    if (!location) {
      this.age[index] = 1
      return
    }
    this.lon[index] = location.lng
    this.lat[index] = location.lat
    this.age[index] = LIFE + Math.floor(Math.random() * LIFE_SPREAD)
  }

  private readonly tick = () => {
    this.frame = requestAnimationFrame(this.tick)
    const field = this.field
    if (!field) return
    const context = this.context
    // Keep TRAIL_KEEP of what was drawn: destination-in multiplies by the fill's alpha; its colour is unused.
    context.globalCompositeOperation = "destination-in"
    context.globalAlpha = TRAIL_KEEP
    context.fillStyle = "#000"
    context.fillRect(0, 0, this.width, this.height)
    context.globalCompositeOperation = "source-over"
    context.globalAlpha = 1

    const step = this.degreesPerStep(PIXELS_PER_MS)
    for (const band of this.segments) band.length = 0
    const location = { lng: 0, lat: 0 }
    for (let index = 0; index < this.count; index += 1) {
      this.age[index] = this.age[index]! - 1
      const lon = this.lon[index]!
      const lat0 = this.lat[index]!
      if (this.age[index]! <= 0 || !sampleInto(field, lon, lat0, this.wind)) {
        this.spawn(index)
        continue
      }
      const u = this.wind[0]!
      const v = this.wind[1]!
      const lat = lat0 + v * step
      if (lat > 89 || lat < -89) {
        this.spawn(index)
        continue
      }
      location.lng = lon + (u * step) / Math.max(0.05, Math.cos(lat0 * DEGREES))
      location.lat = lat
      if (this.occluded(location)) {
        this.spawn(index)
        continue
      }
      const point = this.map.project(location)
      if (
        point.x < -MAX_STEP ||
        point.y < -MAX_STEP ||
        point.x > this.width + MAX_STEP ||
        point.y > this.height + MAX_STEP
      ) {
        this.spawn(index)
        continue
      }
      const x = this.x[index]!
      const y = this.y[index]!
      if (
        this.shown[index] &&
        Math.abs(point.x - x) < MAX_STEP &&
        Math.abs(point.y - y) < MAX_STEP
      ) {
        this.segments[windBand(Math.hypot(u, v))]!.push(x, y, point.x, point.y)
      }
      this.lon[index] = location.lng
      this.lat[index] = lat
      this.x[index] = point.x
      this.y[index] = point.y
      this.shown[index] = 1
    }
    this.stroke()
    this.onFrame()
  }

  /** Draws this frame's segments, one path per colour. */
  private stroke() {
    const context = this.context
    context.lineWidth = LINE_WIDTH
    context.lineCap = "round"
    this.segments.forEach((band, index) => {
      if (!band.length) return
      context.strokeStyle = this.colors[index] ?? "#888"
      context.beginPath()
      for (let at = 0; at < band.length; at += 4) {
        context.moveTo(band[at]!, band[at + 1]!)
        context.lineTo(band[at + 2]!, band[at + 3]!)
      }
      context.stroke()
    })
  }

  /**
   * Reduced motion: a still picture of the flow. From points spread evenly
   * over the view (jittered the same way each time, so a redraw does not
   * shimmer), a short line follows the wind, with an arrowhead where it
   * ends, in the colour of the wind where it starts.
   */
  private drawStreamlines() {
    const field = this.field
    if (!field) return
    const step = this.degreesPerStep(STREAM_STEP)
    for (const band of this.segments) band.length = 0
    const location = { lng: 0, lat: 0 }
    let seed = 1
    const jitter = () => {
      seed = (seed * 16807) % 2147483647
      return (seed / 2147483647 - 0.5) * STREAM_SPACING * 0.6
    }
    for (let row = STREAM_SPACING / 2; row < this.height; row += STREAM_SPACING) {
      for (let column = STREAM_SPACING / 2; column < this.width; column += STREAM_SPACING) {
        const x = column + jitter()
        const y = row + jitter()
        const start = this.map.unproject([x, y])
        if (this.occluded(start)) continue
        const first = this.map.project(start)
        if (Math.abs(first.x - x) > 2 || Math.abs(first.y - y) > 2) continue
        if (!sampleInto(field, start.lng, start.lat, this.wind)) continue
        const speed = Math.hypot(this.wind[0]!, this.wind[1]!)
        if (speed < 0.5) continue
        const band = this.segments[windBand(speed)]!
        let lng = start.lng
        let lat = start.lat
        let previous = first
        let heading: [number, number] | null = null
        for (let index = 0; index < STREAM_STEPS; index += 1) {
          if (!sampleInto(field, lng, lat, this.wind)) break
          const magnitude = Math.hypot(this.wind[0]!, this.wind[1]!)
          if (magnitude < 0.1) break
          lat += (this.wind[1]! / magnitude) * step
          lng += ((this.wind[0]! / magnitude) * step) / Math.max(0.05, Math.cos(lat * DEGREES))
          location.lng = lng
          location.lat = lat
          if (lat > 89 || lat < -89 || this.occluded(location)) break
          const point = this.map.project(location)
          if (
            Math.abs(point.x - previous.x) > MAX_STEP ||
            Math.abs(point.y - previous.y) > MAX_STEP
          )
            break
          band.push(previous.x, previous.y, point.x, point.y)
          heading = [point.x - previous.x, point.y - previous.y]
          previous = point
        }
        if (!heading) continue
        // The arrowhead: two short strokes back from the tip, 30° either side.
        const angle = Math.atan2(heading[1], heading[0])
        for (const side of [-1, 1]) {
          const back = angle + Math.PI + (side * Math.PI) / 6
          band.push(
            previous.x,
            previous.y,
            previous.x + ARROW * Math.cos(back),
            previous.y + ARROW * Math.sin(back),
          )
        }
      }
    }
    this.clear()
    this.stroke()
  }
}
