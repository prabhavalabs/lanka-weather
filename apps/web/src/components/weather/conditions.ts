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

/** Words for WMO weather interpretation codes, as Open-Meteo reports them. */
const wmoWords: Record<number, string> = {
  0: "Clear",
  1: "Mainly clear",
  2: "Partly cloudy",
  3: "Overcast",
  45: "Fog",
  48: "Freezing fog",
  51: "Light drizzle",
  53: "Drizzle",
  55: "Dense drizzle",
  56: "Freezing drizzle",
  57: "Freezing drizzle",
  61: "Light rain",
  63: "Rain",
  65: "Heavy rain",
  66: "Freezing rain",
  67: "Freezing rain",
  71: "Light snow",
  73: "Snow",
  75: "Heavy snow",
  77: "Snow grains",
  80: "Light showers",
  81: "Showers",
  82: "Violent showers",
  85: "Snow showers",
  86: "Snow showers",
  95: "Thunderstorm",
  96: "Thunderstorm with hail",
  99: "Thunderstorm with hail",
}

/** Sustained wind from which calm skies are called windy, in km/h. */
const windyKmh = 40

/**
 * The condition for a WMO weather code (Open-Meteo's `weatherCode`), the
 * time of day and, optionally, the wind: clear skies at night are
 * "clear-night", and dry weather in a strong wind is "windy".
 */
export function conditionFromWmo(
  code: number | null | undefined,
  isDay: number | boolean | null | undefined = 1,
  windSpeedKmh?: number | null,
): { condition: WeatherCondition; summary: string } {
  const day = isDay === null || isDay === undefined ? true : Boolean(isDay)
  if (code === null || code === undefined || !Number.isFinite(code)) {
    return { condition: day ? "partly-cloudy" : "clear-night", summary: "Unknown" }
  }
  const summary = wmoWords[code] ?? "Unknown"
  let condition: WeatherCondition
  if (code >= 95) condition = "thunderstorm"
  else if (code === 65 || code === 82 || code === 67) condition = "heavy-rain"
  else if ((code >= 61 && code <= 66) || (code >= 80 && code <= 81)) condition = "rain"
  else if (code >= 51 && code <= 57) condition = "drizzle"
  else if (code === 45 || code === 48) condition = "fog"
  else if (code >= 71 && code <= 86) condition = "cloudy"
  else if (code === 3) condition = "cloudy"
  else if (code === 2) condition = "partly-cloudy"
  else condition = day ? (code === 0 ? "clear" : "partly-cloudy") : "clear-night"
  if (
    code <= 3 &&
    windSpeedKmh !== undefined &&
    windSpeedKmh !== null &&
    windSpeedKmh >= windyKmh
  ) {
    return { condition: "windy", summary: `${summary}, windy` }
  }
  return { condition, summary }
}
