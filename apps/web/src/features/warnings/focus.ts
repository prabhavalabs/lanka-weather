/**
 * Which districts the warnings panel is about, for the focus mask: the map
 * goes grey around them. Pure, so it is tested without a browser.
 */
import type { WarningsOverview } from "./types"

export interface WarningsFilter {
  type?: string
  district?: string
}

/**
 * The districts to keep clear of the veil, or an empty list for none:
 *
 * - one district chosen: that district;
 * - one kind of warning: the districts its bulletins in force name, and for
 *   landslides also the districts with a level in force;
 * - all warnings: every highlighted district.
 *
 * A warning that names no district (one for the whole island) focuses on
 * nothing, so the map is left as it is.
 */
export function warningFocus(
  filter: WarningsFilter,
  data: Pick<WarningsOverview, "inForce" | "districts"> | undefined,
): string[] {
  if (filter.district) return [filter.district]
  if (!data) return []
  if (!filter.type) return [...new Set(data.districts.map((district) => district.id))]
  const named = data.inForce
    .filter((bulletin) => bulletin.type === filter.type)
    .flatMap((bulletin) => bulletin.districts.map((district) => district.id))
  const levels =
    filter.type === "landslide"
      ? data.districts.filter((district) => district.landslideLevel > 0).map((d) => d.id)
      : []
  return [...new Set([...named, ...levels])].sort()
}
