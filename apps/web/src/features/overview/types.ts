/** Mirrors internal/overview in the Go service (docs/api/overview.md). */
export type OverviewTone = "neutral" | "watch" | "high" | "severe"

export interface OverviewTarget {
  kind:
    | "warnings"
    | "landslide"
    | "river"
    | "station"
    | "raingauge"
    | "reservoir"
    | "reservoirs"
    | "cyclone"
    | "quake"
    | "region"
    | "place"
    | "layer"
  id?: string
  reach?: number
  /** A layer to switch on first. */
  layer?: string
  /** What to colour a region by. */
  fill?: "temperature" | "rain" | "wind"
  lon?: number
  lat?: number
}

export interface OverviewItem {
  id: string
  title: string
  detail: string
  tone: OverviewTone
  badge?: string
  estimate: boolean
  stale: boolean
  at: string | null
  source: string
  target: OverviewTarget | null
  /** The official document, when there is one. */
  href?: string
}

export interface OverviewCount {
  key: string
  label: string
  value: number
}

export type OverviewGroup = "official" | "water" | "hazards" | "now"

export interface OverviewSection {
  id: string
  group: OverviewGroup
  title: string
  summary: string
  note?: string
  official: boolean
  estimate: boolean
  /** False when this part could not be read; it then never reads as calm. */
  available: boolean
  stale: boolean
  at: string | null
  source: string
  counts: OverviewCount[]
  items: OverviewItem[]
  target: OverviewTarget | null
}

export interface Overview {
  generatedAt: string
  /** One sentence built from fixed rules; it never says all is well. */
  status: string
  stale: string[]
  unavailable: string[]
  sections: OverviewSection[]
}
