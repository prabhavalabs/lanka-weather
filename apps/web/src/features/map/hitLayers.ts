import { BASIN_FILL, SUBBASIN_FILL } from "../floods/FloodBasinsLayer"
import { FLOOD_AREAS, FLOOD_GAUGES } from "../floods/FloodForecastLayer"
import { PAST_FLOOD_FILL } from "../floods/PastFloods"
import { REGIONS_FILL } from "../regions/RegionsLayer"
import { RESERVOIR_AREAS, RESERVOIR_POINTS } from "../reservoirs/ReservoirsLayer"
import { RIVER_LINES } from "../river-risk/RiverNetworkLayer"
import { RIVER_STATIONS } from "../rivers/RiverLevelsLayer"
import type { HitLayerIds } from "./hits"

/** Every layer a click or hover can land on, for `hitOrder`. */
export function hitLayerIds({
  hazards,
  observed,
}: {
  hazards?: { points: string[]; areas: string[] } | null
  observed: string[]
}): HitLayerIds {
  return {
    points: [
      RIVER_STATIONS,
      FLOOD_GAUGES,
      RESERVOIR_POINTS,
      ...observed,
      ...(hazards?.points ?? []),
    ],
    rivers: RIVER_LINES,
    water: [RESERVOIR_AREAS],
    areas: [FLOOD_AREAS, ...(hazards?.areas ?? [])],
    past: [PAST_FLOOD_FILL],
    basins: [BASIN_FILL, SUBBASIN_FILL],
    regions: [REGIONS_FILL],
  }
}
