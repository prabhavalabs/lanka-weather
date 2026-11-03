import { translate } from "@/lib/i18n"
import type { AlertSeverity } from "./types"

/** Alert severities share the flood status colours, from informative to severe. */
export const severityColors: Record<AlertSeverity, string> = {
  advisory: "var(--status-normal)",
  watch: "var(--status-alert)",
  warning: "var(--status-minor-flood)",
  emergency: "var(--status-major-flood)",
}

/** An issuer's level in the current language. */
export function severityLabel(severity: AlertSeverity): string {
  return translate(`weather.severity.${severity}`)
}

export const severityBadge = {
  advisory: "normal",
  watch: "alert",
  warning: "minor_flood",
  emergency: "major_flood",
} as const satisfies Record<AlertSeverity, string>

const estimateKeys = {
  advisory: "aware",
  watch: "aware",
  warning: "prepared",
  emergency: "action",
} as const satisfies Record<AlertSeverity, string>

/**
 * Words for the level of an estimate: lanka-weather's own reading of the
 * forecast, so it says what to do ("Be aware", "Be prepared", "Take action")
 * rather than borrowing an official word such as NBRO's "Watch" or "Alert".
 */
export function estimateLabel(severity: AlertSeverity): string {
  return translate(`weather.estimateLevel.${estimateKeys[severity]}`)
}
