/** Mirrors /v1/warnings and /v1/warnings/landslides (docs/api/warnings-and-hazards.md). */
import type { Feature, FeatureCollection, MultiPolygon } from "geojson"

export type BulletinType = "landslide" | "water_level" | "weather" | "situation"
export type BulletinSeverity = "warning" | "advisory" | "withdrawal" | "routine"
export type TextStatus = "pending" | "clean" | "unusable" | "unavailable"
export type LandslideStatus = "waiting" | "active" | "lapsed" | "pending" | "none" | "unknown"

export interface LevelWords {
  level: 1 | 2 | 3
  /** As NBRO's reports print it, e.g. "Level 2 (Amber)". */
  label: string
  colour: string
  /** NBRO's word for what to do: Watch, Alert or Evacuate. */
  action: string
}

export interface DistrictRef {
  id: string
  name: string
}

export interface Bulletin {
  id: string
  type: BulletinType
  typeLabel: string
  /** The issuer's title, unchanged. */
  title: string
  severity: BulletinSeverity
  /** The colour the bulletin states for itself. */
  colour: string | null
  messageId: string | null
  issuedAt: string
  validFrom: string | null
  /** The period it states; null if none. */
  validUntil: string | null
  current: boolean
  pdfUrl: string
  sourcePageUrl: string
  language: string
  /** English sentences copied unchanged from the bulletin, or null. */
  text: string | null
  textStatus: TextStatus
  districts: DistrictRef[]
  source: string
  sourceTime: string
  fetchedAt: string
}

export interface LandslideMeta {
  source: string
  attribution: string
  status: LandslideStatus
  /** The sentence to show about the status. */
  note: string
  stale: boolean
  reportDate: string | null
  issuedAt: string | null
  validUntil: string | null
  validityStated: boolean
  messageId: string | null
  pdfUrl: string | null
  sourcePageUrl: string | null
  counts: Record<"1" | "2" | "3", number>
  unresolved: string[]
  fetchedAt: string | null
  upstreamAt: string | null
  generatedAt: string
}

export interface DivisionProperties extends LevelWords {
  id: string
  name: string
  nameSi: string | null
  districtId: string
  district: string
  labelLon: number
  labelLat: number
}

export type DivisionFeature = Feature<MultiPolygon, DivisionProperties>

export type LandslideLayerData = FeatureCollection<MultiPolygon, DivisionProperties> & {
  meta: LandslideMeta
}

export interface HistoryEntry extends LevelWords {
  date: string
}

export interface Division {
  id: string
  name: string
  nameSi: string | null
  districtId: string
  district: string
  lon: number
  lat: number
  level: LevelWords | null
  report: LandslideMeta
  history: HistoryEntry[]
}

export interface BannerItem {
  key: string
  kind: "landslide" | "bulletin"
  title: string
  detail: string
  level?: number
  colour: string | null
  issuedAt: string | null
  pdfUrl: string | null
}

export interface DistrictHighlight {
  id: string
  name: string
  landslideLevel: number
  mentioned: { id: string; title: string }[]
}

export interface WarningsOverview {
  bulletins: Bulletin[]
  inForce: Bulletin[]
  activeCount: number
  banner: BannerItem[]
  districts: DistrictHighlight[]
  landslide: LandslideMeta
  meta: {
    source: string
    attribution: string
    note: string
    fetchedAt: string | null
    upstreamAt: string | null
    generatedAt: string
    stale: boolean
  }
}

/** The landslide warning at a place. */
export interface LandslideHere {
  status: LandslideStatus
  note: string
  stale: boolean
  level: LevelWords | null
  dsId: string | null
  ds: string | null
  district: string | null
  reportDate: string | null
  issuedAt: string | null
  validUntil: string | null
  pdfUrl: string | null
}

/** Everything official about a place: the `official` part of /v1/summary. */
export interface OfficialAt {
  status: "available" | "not_available"
  landslide: LandslideHere | null
  warnings: Bulletin[]
  general: Bulletin[]
  note: string
  stale: boolean
  fetchedAt: string | null
}
