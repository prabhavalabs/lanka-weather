import type { AlertSeverity } from "./types"

/** Alert severities share the flood status colours, from informative to severe. */
export const severityColors: Record<AlertSeverity, string> = {
  advisory: "var(--status-normal)",
  watch: "var(--status-alert)",
  warning: "var(--status-minor-flood)",
  emergency: "var(--status-major-flood)",
}

export const severityLabels: Record<AlertSeverity, string> = {
  advisory: "Advisory",
  watch: "Watch",
  warning: "Warning",
  emergency: "Emergency",
}

export const severityBadge = {
  advisory: "normal",
  watch: "alert",
  warning: "minor_flood",
  emergency: "major_flood",
} as const satisfies Record<AlertSeverity, string>

/**
 * Words for the level of an estimate: lanka-weather's own reading of the
 * forecast, so it says what to do rather than borrowing an official word.
 */
export const estimateLabels: Record<AlertSeverity, string> = {
  advisory: "Be aware",
  watch: "Be aware",
  warning: "Be prepared",
  emergency: "Take action",
}
