import { translate } from "@/lib/i18n"
import type { AdminLevel } from "./types"

const longKeys = {
  province: "terms.adminLevel.province",
  district: "terms.adminLevel.district",
  ds: "terms.adminLevel.dsLong",
  gn: "terms.adminLevel.gnLong",
} as const satisfies Record<AdminLevel, string>

/** A level's short name (for the segmented control) and full name, in the current language. */
export function adminLevelLabel(level: AdminLevel): { short: string; long: string } {
  return { short: translate(`weather.adminShort.${level}`), long: translate(longKeys[level]) }
}

export const adminLevels: AdminLevel[] = ["province", "district", "ds", "gn"]
