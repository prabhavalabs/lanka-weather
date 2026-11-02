/** Words and colours for cyclones. Pure, so it is tested without a browser. */
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

export function cycloneSummary(data: CycloneCollection): string[] {
  const systems = positions(data)
  if (systems.length > 0) {
    return [
      systems.length === 1
        ? `${systems[0]!.name} active`
        : `${systems.length} active: ${systems.map((system) => system.name).join(", ")}`,
    ]
  }
  return [data.meta.note]
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

export const evaluationNote =
  "JTWC and IMD RSMC New Delhi advisories remain under evaluation as direct sources. Track data here is GDACS's, which draws on JTWC."
