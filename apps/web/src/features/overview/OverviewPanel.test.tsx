import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it } from "vitest"
import { OverviewPanel } from "./OverviewPanel"
import type { Overview, OverviewSection } from "./types"

function section(change: Partial<OverviewSection>): OverviewSection {
  return {
    id: "rivers",
    group: "water",
    title: "Rivers at risk",
    summary: "Estimate: 4 stretches of 233 at watch or above.",
    official: false,
    estimate: true,
    available: true,
    stale: false,
    at: "2026-11-02T20:00:00Z",
    source: "Estimate by lanka-weather",
    counts: [],
    items: [],
    target: { kind: "layer", layer: "rivers" },
    ...change,
  }
}

const overview: Overview = {
  generatedAt: "2026-11-02T21:00:00Z",
  status: "1 cyclone active north of Sri Lanka. 10 reservoirs spilling.",
  stale: [],
  unavailable: [],
  sections: [
    section({
      id: "warnings",
      group: "official",
      title: "Warnings in force",
      official: true,
      estimate: false,
      items: [
        {
          id: "w",
          title: "Warning for strong winds and rough seas",
          detail: "Weather",
          tone: "severe",
          badge: "Warning: Red",
          estimate: false,
          stale: false,
          at: "2026-11-02T05:00:00Z",
          source: "Disaster Management Centre",
          target: { kind: "warnings", id: "weather" },
          href: "https://dmc.gov.lk/w.pdf",
        },
      ],
    }),
    section({
      items: [
        {
          id: "deduru-oya",
          title: "Deduru Oya",
          detail: "Kimbulwana Oya reservoir is spilling 1,551 cusecs upstream.",
          tone: "watch",
          badge: "Watch",
          estimate: true,
          stale: false,
          at: "2026-11-02T20:00:00Z",
          source: "Estimate by lanka-weather",
          target: { kind: "river", id: "deduru-oya", reach: 1 },
        },
      ],
    }),
    section({
      id: "cyclones",
      group: "hazards",
      title: "Cyclones",
      summary: "Could not be read just now, so nothing can be said from here.",
      available: false,
      estimate: false,
    }),
    section({
      id: "gauges",
      title: "River gauges",
      summary: "Only 2 of 60 gauges have a current reading.",
      stale: true,
      estimate: false,
    }),
  ],
}

const props = { pending: false, failed: false, onOpen: () => {}, onClose: () => {} }

function render(change: Partial<Parameters<typeof OverviewPanel>[0]> = {}) {
  return renderToStaticMarkup(<OverviewPanel overview={overview} {...props} {...change} />)
}

describe("OverviewPanel", () => {
  it("leads with the sentence, and official warnings before anything else", () => {
    const html = render()
    expect(html).toContain("1 cyclone active north of Sri Lanka. 10 reservoirs spilling.")
    expect(html.indexOf("Warning for strong winds")).toBeLessThan(html.indexOf("Deduru Oya"))
  })

  it("keeps the issuer's words, its badge and a link to the PDF", () => {
    const html = render()
    expect(html).toContain("Warning for strong winds and rough seas")
    expect(html).toContain("Warning: Red")
    expect(html).toContain("https://dmc.gov.lk/w.pdf")
    expect(html).toContain("Official")
  })

  it("labels what is derived as an estimate, on the section and the line", () => {
    const line = render().split("Deduru Oya")[1]!
    expect(line).toContain("Estimate")
  })

  it("marks a feed it could not read and one out of date, in words", () => {
    const html = render()
    expect(html).toContain("Not read")
    expect(html).toContain("Out of date")
  })

  it("gives every line its age and source, and a real button", () => {
    const html = render()
    expect(html).toContain("Disaster Management Centre")
    expect(html).toContain("<button")
    expect(html).toContain('aria-label="Close the current situation"')
    expect(html).toMatch(/ago|hour|min|day|now/i)
  })

  it("never tells anyone they are safe", () => {
    const html = render().toLowerCase()
    expect(html).not.toContain("all clear")
    expect(html).not.toMatch(/\bsafe\b/)
  })

  it("sends people to the DMC when it cannot load", () => {
    const html = render({ overview: undefined, failed: true })
    expect(html).toContain("could not be loaded")
    expect(html).toContain("dmc.gov.lk")
    expect(html).toContain("117")
  })

  it("says it is loading while it loads", () => {
    expect(render({ overview: undefined, pending: true })).toContain("Loading the overview")
  })
})
