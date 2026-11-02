import type { WeatherCondition } from "./types"

/** Words for each condition, for labels and screen readers. */
export const conditionLabels: Record<WeatherCondition, string> = {
  clear: "Clear",
  "clear-night": "Clear night",
  "partly-cloudy": "Partly cloudy",
  cloudy: "Cloudy",
  drizzle: "Drizzle",
  rain: "Rain",
  "heavy-rain": "Heavy rain",
  thunderstorm: "Thunderstorm",
  fog: "Fog",
  windy: "Windy",
}
