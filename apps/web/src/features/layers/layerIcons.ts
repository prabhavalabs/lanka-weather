import {
  ActivityIcon,
  CloudFogIcon,
  CloudRainIcon,
  CloudSunIcon,
  ContainerIcon,
  HistoryIcon,
  MapIcon,
  MountainIcon,
  RadioTowerIcon,
  SplineIcon,
  ShipIcon,
  ThermometerIcon,
  TornadoIcon,
  TriangleAlertIcon,
  WavesIcon,
  WindIcon,
  type LucideIcon,
} from "lucide-react"
import type { Layer, LayerCategory } from "@/lib/layers"

const byId: Record<string, LucideIcon> = {
  "river-levels": WavesIcon,
  "flood-forecast": WavesIcon,
  rivers: SplineIcon,
  "past-floods": HistoryIcon,
  reservoirs: ContainerIcon,
  wind: WindIcon,
  temperature: ThermometerIcon,
  rainfall: CloudRainIcon,
  "weather-stations": RadioTowerIcon,
  "air-quality": CloudFogIcon,
  "sea-conditions": ShipIcon,
  cyclones: TornadoIcon,
  "landslide-warnings": MountainIcon,
  "official-warnings": TriangleAlertIcon,
  earthquakes: ActivityIcon,
  "admin-regions": MapIcon,
}

const byCategory: Record<LayerCategory, LucideIcon> = {
  hazard: TriangleAlertIcon,
  hydrology: WavesIcon,
  weather: CloudSunIcon,
  geography: MapIcon,
}

/** The line icon for a layer: its own if known, else its category's. */
export function layerIcon(layer: Pick<Layer, "id" | "category">): LucideIcon {
  return byId[layer.id] ?? byCategory[layer.category]
}
