import {
  CloudDrizzleIcon,
  CloudFogIcon,
  CloudIcon,
  CloudLightningIcon,
  CloudRainIcon,
  CloudSunIcon,
  MoonIcon,
  SunIcon,
  WindIcon,
  type LucideProps,
} from "lucide-react"

import type { WeatherCondition } from "./types"

const icons: Record<WeatherCondition, typeof SunIcon> = {
  clear: SunIcon,
  "clear-night": MoonIcon,
  "partly-cloudy": CloudSunIcon,
  cloudy: CloudIcon,
  drizzle: CloudDrizzleIcon,
  rain: CloudRainIcon,
  "heavy-rain": CloudRainIcon,
  thunderstorm: CloudLightningIcon,
  fog: CloudFogIcon,
  windy: WindIcon,
}

/** A line icon for a weather condition; decorative unless given a title. */
export function ConditionIcon({
  condition,
  title,
  ...props
}: LucideProps & { condition: WeatherCondition; title?: string }) {
  const Icon = icons[condition]
  return (
    <Icon
      aria-hidden={title ? undefined : true}
      aria-label={title}
      role={title ? "img" : undefined}
      strokeWidth={1.75}
      {...props}
    />
  )
}
