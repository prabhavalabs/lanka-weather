import type { AdminLevel } from "./types"

export const adminLevelLabels: Record<AdminLevel, { short: string; long: string }> = {
  province: { short: "Province", long: "Province" },
  district: { short: "District", long: "District" },
  ds: { short: "DS", long: "Divisional Secretariat division" },
  gn: { short: "GN", long: "Grama Niladhari division" },
}

export const adminLevels: AdminLevel[] = ["province", "district", "ds", "gn"]
