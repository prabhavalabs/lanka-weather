/**
 * The one-line status of a layer in the layer list: what it shows and how
 * old its data is, or that the data is still being collected. Pure, so it
 * is tested without a browser.
 */
import { isNotReady } from "@/lib/api"
import { formatAge } from "@/lib/format"
import { translate } from "@/lib/i18n"
import { isStale, type Layer } from "@/lib/layers"

/** The word before a layer's age: "updated 5 min ago", "run 9 h ago". */
export type AgeLabel = "updated" | "read" | "run" | "image" | "newest" | "issued"

export type LayerState = "loading" | "waiting" | "error" | "ready"

export interface LayerStatus {
  state: LayerState
  summary: string
  /** Older than the catalogue's `staleAfterSeconds`: shown, but marked out of date. */
  stale: boolean
}

export function layerStatus({
  layer,
  parts,
  observedAt,
  error,
  pending,
  ageLabel = "updated",
  now = new Date(),
}: {
  layer: Pick<Layer, "staleAfterSeconds">
  /** What the data holds, e.g. ["214 grid points"]. */
  parts: string[]
  /** When the data was observed or issued (ISO 8601). */
  observedAt: string | null | undefined
  error?: unknown
  pending?: boolean
  /** The word before the age: "updated", or "run" for a model run. */
  ageLabel?: AgeLabel
  now?: Date
}): LayerStatus {
  const hasData = parts.length > 0 || Boolean(observedAt)
  if (!hasData) {
    if (isNotReady(error)) {
      return {
        state: "waiting",
        summary: translate("layers.status.waiting"),
        stale: false,
      }
    }
    if (error) return { state: "error", summary: translate("layers.status.error"), stale: false }
    if (pending) return { state: "loading", summary: translate("common.loading"), stale: false }
  }
  const stale = observedAt ? isStale(layer, new Date(observedAt), now) : false
  const age = observedAt
    ? translate(`layers.age.${ageLabel}`, { age: formatAge(observedAt, now) })
    : null
  return {
    state: "ready",
    summary: [...parts, age].filter(Boolean).join(translate("common.separator")),
    stale,
  }
}

/**
 * "1 grid point", "214 grid points": English only. Use a plural message,
 * `t("ns.key", { count })`, for text people read.
 */
export function count(value: number, singular: string, plural = `${singular}s`) {
  return `${value} ${value === 1 ? singular : plural}`
}
