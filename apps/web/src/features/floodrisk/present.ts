/**
 * The flood-risk estimate, pure: which district's entry to show, the badge
 * for its level, and each signal as a message key with its values.
 */
import type { MessageKey } from "@/lib/i18n"
import type { FloodRiskDistrict, FloodRiskDocument, FloodRiskSignal } from "./types"

/** A district's entry; level 0 with nothing known when it is not listed. */
export function districtRisk(
  document: FloodRiskDocument | undefined,
  id: string,
): FloodRiskDistrict | null {
  if (!document) return null
  return (
    document.districts.find((entry) => entry.district === id) ?? {
      district: id,
      level: 0,
      variant: "",
      score: 0,
      families: 0,
      scenario: "general",
      signals: [],
    }
  )
}

/** Badge variants by level; level 0 is an outline, never the green of "normal". */
export const riskBadge = {
  0: "outline",
  1: "alert",
  2: "minor_flood",
  3: "major_flood",
} as const

export const levelKey = {
  0: "floodRisk.level.none",
  1: "floodRisk.level.moderate",
  2: "floodRisk.level.high",
  3: "floodRisk.level.very_high",
} as const satisfies Record<FloodRiskDistrict["level"], MessageKey>

const officialKeys = ["", "advisory", "warning", "red"] as const
const flashKeys = ["", "likely", "highly_likely"] as const
const gaugeKeys = ["", "warning", "danger", "extreme"] as const
const riverKeys = ["", "alert", "minor_flood", "major_flood"] as const

/** One signal's value as a message key and its parameters. */
export function signalValue(signal: FloodRiskSignal): {
  key: MessageKey
  params?: Record<string, string | number>
} {
  const level = Math.max(0, Math.min(3, Math.round(signal.value)))
  const mm = Math.round(signal.value)
  switch (signal.family) {
    case "official":
      return { key: `floodRisk.value.official.${officialKeys[level] || "advisory"}` }
    case "google_flash":
      return { key: `floodRisk.value.flash.${flashKeys[Math.min(level, 2)] || "likely"}` }
    case "google_gauge":
      return { key: `floodRisk.value.gauge.${gaugeKeys[level] || "warning"}` }
    case "river":
      return { key: `floodRisk.value.river.${riverKeys[level] || "alert"}` }
    case "rain_now":
      return {
        key: signal.window === "3h" ? "floodRisk.value.rain3h" : "floodRisk.value.rain1h",
        params: { mm },
      }
    case "wet_ground":
      return { key: "floodRisk.value.prior", params: { mm } }
    case "rain_forecast":
      return {
        key: signal.window === "24h" ? "floodRisk.value.forecast24h" : "floodRisk.value.forecast6h",
        params: { mm },
      }
    case "reservoir":
      return { key: "floodRisk.value.reservoir", params: { count: mm } }
  }
}
