/** Mirrors `/v1/rivers/canals` (docs/api/rivers-and-basins.md). */
export interface CanalGauge {
  id: string
  name: string
  group: string
  /** The source's own word: normal, rising, higher or offline. */
  state: string
  levelM: number | null
  observedAt: string | null
  change3hM: number | null
  last24h: { t: string; level_m: number }[]
  stale: boolean
}

export interface CanalsDocument {
  gauges: CanalGauge[]
  sea: { levelM: number | null; trend: string | null; observedAt: string | null } | null
  tide: { note: string; events: { kind: string; at: string }[] }
  meta: {
    source: string
    credit: string
    attribution: string
    via: string
    licence: string
    upstreamUrl: string
    note: string
    fetchedAt: string | null
    sourceTime: string | null
    generatedAt: string
    total: number
    reporting: number
    raised: number
    stale: boolean
  }
}

/** What a place's summary says of the canals, in the Colombo area only. */
export interface CanalsBrief {
  total: number
  reporting: number
  raised: number
  fetchedAt: string | null
  stale: boolean
  attribution: string
  note: string
}
