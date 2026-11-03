/** Words and colours for cyclones. Pure, so it is tested without a browser. */
import { getLocale, translate, translateIn, type Locale } from "@/lib/i18n"
import type { CycloneCollection, CyclonePosition, CycloneStatus } from "./types"

export function positions(data: CycloneCollection | undefined): CyclonePosition[] {
  return (data?.features ?? [])
    .map((feature) => feature.properties)
    .filter((properties): properties is CyclonePosition => properties.kind === "position")
}

/** GDACS's alert colours; a word is always shown with them. */
export const alertTokens: Record<string, `--${string}`> = {
  Green: "--wind-3",
  Orange: "--status-minor-flood",
  Red: "--status-major-flood",
}

export const alertBadge: Record<string, "normal" | "minor_flood" | "major_flood" | "unknown"> = {
  Green: "normal",
  Orange: "minor_flood",
  Red: "major_flood",
}

type AlertKey = "green" | "orange" | "red"

/** GDACS's alert colour in the reader's language; an unknown word stays as GDACS gives it. */
export function alertWord(level: string, locale: Locale = getLocale()): string {
  const key = level.trim().toLowerCase()
  return key === "green" || key === "orange" || key === "red"
    ? translateIn(locale, `terms.colour.${key as AlertKey}`)
    : level
}

/** "GDACS Red alert" */
export function alertText(level: string, locale: Locale = getLocale()): string {
  return translateIn(locale, "cyclones.alert", { level: alertWord(level, locale) })
}

/**
 * The sentence about the feed, from its status: what the server says in
 * `meta.note`, in the reader's language. Empty when systems are listed and the
 * feed is fresh.
 */
export function cycloneStatusNote(
  meta: Pick<CycloneCollection["meta"], "status" | "stale">,
  locale: Locale = getLocale(),
): string {
  switch (meta.status) {
    case "waiting":
      return translateIn(locale, "cyclones.status.waiting")
    case "active":
      return meta.stale ? translateIn(locale, "cyclones.status.activeStale") : ""
    case "unknown":
      return translateIn(locale, "cyclones.status.unknown")
    case "none":
      return translateIn(locale, "cyclones.status.none")
  }
  return ""
}

export function cycloneSummary(data: CycloneCollection): string[] {
  const systems = positions(data)
  if (systems.length > 0) {
    return [
      systems.length === 1
        ? translate("cyclones.summary.one", { name: systems[0]!.name })
        : translate("cyclones.summary.many", {
            count: systems.length,
            names: systems.map((system) => system.name).join(", "),
          }),
    ]
  }
  return [cycloneStatusNote(data.meta) || data.meta.note]
}

/** Only this state may read as calm. */
export function isCalm(status: CycloneStatus): boolean {
  return status === "none"
}

/**
 * Where a system is drawn, with Sri Lanka, for fitting the map to both. Extra
 * room to the north, where the toolbar and the warnings banner lie over the map.
 */
export function boundsWithSriLanka(
  system: Pick<CyclonePosition, "lon" | "lat">,
): [number, number, number, number] {
  const west = 79.52
  const south = 5.85
  const east = 81.95
  const north = 9.9
  return [
    Math.min(west, system.lon) - 0.5,
    Math.min(south, system.lat) - 0.5,
    Math.max(east, system.lon) + 0.5,
    Math.max(north, system.lat) + 3,
  ]
}

/** Which sources are still being weighed: JTWC and IMD are not used directly yet. */
export function evaluationNote(): string {
  return translate("cyclones.evaluation")
}
